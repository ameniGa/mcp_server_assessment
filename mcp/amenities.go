package mcpserver

import (
	"context"
	"fmt"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ameni/hospitality-scout/client"
)

const amenitySearchRadiusMeters = 1000

// AmenityType is the constrained set of amenity categories find_nearby_amenities accepts
type AmenityType string

const (
	AmenityRestaurant AmenityType = "restaurant"
	AmenityMuseum     AmenityType = "museum"
	AmenityCafe       AmenityType = "cafe"
)

var allowedAmenityTypes = map[AmenityType]bool{
	AmenityRestaurant: true,
	AmenityMuseum:     true,
	AmenityCafe:       true,
}

// FindNearbyAmenitiesInput is the input for the find_nearby_amenities tool.
type FindNearbyAmenitiesInput struct {
	Latitude    float64     `json:"latitude" jsonschema:"latitude in decimal degrees (WGS84) of the point to search around"`
	Longitude   float64     `json:"longitude" jsonschema:"longitude in decimal degrees (WGS84) of the point to search around"`
	AmenityType AmenityType `json:"amenity_type" jsonschema:"the category of amenity to search for: restaurant, museum, or cafe"`
}

// Amenity is a restaurant, bar, or cafe returned by find_nearby_amenities.
type Amenity struct {
	Name      string  `json:"name" jsonschema:"the amenity's name as recorded in OpenStreetMap"`
	Type      string  `json:"type" jsonschema:"the OpenStreetMap amenity tag value; always echoes the requested amenity_type ('restaurant', 'museum', or 'cafe')"`
	Address   string  `json:"address" jsonschema:"best-effort street address assembled from OpenStreetMap tags; empty string if none are present"`
	Latitude  float64 `json:"latitude" jsonschema:"latitude in decimal degrees (WGS84)"`
	Longitude float64 `json:"longitude" jsonschema:"longitude in decimal degrees (WGS84)"`
}

// FindNearbyAmenitiesOutput is the output for the find_nearby_amenities tool.
type FindNearbyAmenitiesOutput struct {
	Amenities []Amenity `json:"amenities" jsonschema:"amenities of the requested type found near the given point, as recorded in OpenStreetMap; empty (not omitted, not an error) if none were found within the search radius"`
}

func (d *Deps) findNearbyAmenities(ctx context.Context, _ *sdkmcp.CallToolRequest, in FindNearbyAmenitiesInput) (*sdkmcp.CallToolResult, FindNearbyAmenitiesOutput, error) {
	if err := validateFindNearbyAmenitiesInput(in); err != nil {
		return nil, FindNearbyAmenitiesOutput{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, d.Timeout)
	defer cancel()

	amenities, err := d.Overpass.SearchAmenities(ctx, in.Latitude, in.Longitude, string(in.AmenityType), amenitySearchRadiusMeters)
	if err != nil {
		return nil, FindNearbyAmenitiesOutput{}, wrapUpstream(err)
	}

	return nil, FindNearbyAmenitiesOutput{Amenities: fromClientAmenities(amenities)}, nil
}

func fromClientAmenities(in []client.Amenity) []Amenity {
	out := make([]Amenity, len(in))
	for i, a := range in {
		out[i] = Amenity{
			Name:      a.Name,
			Type:      a.Type,
			Address:   a.Address,
			Latitude:  a.Latitude,
			Longitude: a.Longitude,
		}
	}
	return out
}

func validateFindNearbyAmenitiesInput(in FindNearbyAmenitiesInput) error {
	if in.Latitude < -90 || in.Latitude > 90 {
		return wrapValidation("latitude must be between -90 and 90, got %v", in.Latitude)
	}
	if in.Longitude < -180 || in.Longitude > 180 {
		return wrapValidation("longitude must be between -180 and 180, got %v", in.Longitude)
	}
	if !allowedAmenityTypes[in.AmenityType] {
		return wrapValidation("amenity_type must be one of restaurant, museum, cafe — got %q", in.AmenityType)
	}
	return nil
}

func registerFindNearbyAmenities(server *sdkmcp.Server, d *Deps) {
	inSchema, err := jsonschema.For[FindNearbyAmenitiesInput](&jsonschema.ForOptions{
		TypeSchemas: map[reflect.Type]*jsonschema.Schema{
			reflect.TypeFor[AmenityType](): {
				Type: "string",
				Enum: []any{AmenityRestaurant, AmenityMuseum, AmenityCafe},
			},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("mcpserver: building find_nearby_amenities schema: %v", err))
	}
	if latSchema, ok := inSchema.Properties["latitude"]; ok {
		latSchema.Minimum = new(float64(-90))
		latSchema.Maximum = new(float64(90))
	}
	if lonSchema, ok := inSchema.Properties["longitude"]; ok {
		lonSchema.Minimum = new(float64(-180))
		lonSchema.Maximum = new(float64(180))
	}

	fmt.Printf("%v\n", inSchema.Properties)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "find_nearby_amenities",
		Description: "Find restaurants, museums, or cafes near a specific latitude/longitude point (e.g. a hotel) by querying OpenStreetMap Overpass within a 1km radius. amenity_type must be exactly one of \"restaurant\", \"museum\", or \"cafe\" (validated against the input schema before this tool runs — any other value is rejected with a schema error, not a tool call). Returns an empty list (a normal, successful result) if nothing is found within the radius — that's not an error. A tool error means the coordinates were invalid or an upstream service failed.",
		InputSchema: inSchema,
	}, d.findNearbyAmenities)
}
