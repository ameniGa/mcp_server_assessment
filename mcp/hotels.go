package mcpserver

import (
	"context"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ameni/hospitality-scout/client"
)

const hotelSearchRadiusMeters = 5000

// SearchHotelsInput is the input for the search_hotels tool.
type SearchHotelsInput struct {
	City   string `json:"city" jsonschema:"the city to search for hotels in, e.g. 'Marrakesh' or 'Austin'"`
	Region string `json:"region,omitempty" jsonschema:"optional state, province, or country used to disambiguate cities that share a name, e.g. 'Texas' for Austin"`
}

// Hotel is a lodging property returned by search_hotels.
type Hotel struct {
	Name      string  `json:"name" jsonschema:"the hotel's name as recorded in OpenStreetMap"`
	Address   string  `json:"address" jsonschema:"best-effort street address assembled from OpenStreetMap tags; empty string if none are present"`
	Latitude  float64 `json:"latitude" jsonschema:"latitude in decimal degrees (WGS84)"`
	Longitude float64 `json:"longitude" jsonschema:"longitude in decimal degrees (WGS84)"`
}

// SearchHotelsOutput is the output for the search_hotels tool.
type SearchHotelsOutput struct {
	Hotels []Hotel `json:"hotels" jsonschema:"hotels found near the resolved city center, as recorded in OpenStreetMap; empty (not omitted, not an error) if none were found within the search radius"`
}

func (d *Deps) searchHotels(ctx context.Context, _ *sdkmcp.CallToolRequest, in SearchHotelsInput) (*sdkmcp.CallToolResult, SearchHotelsOutput, error) {
	if err := validateSearchHotelsInput(in); err != nil {
		return nil, SearchHotelsOutput{}, err
	}

	query := in.City
	if strings.TrimSpace(in.Region) != "" {
		query = in.City + ", " + in.Region
	}

	ctx, cancel := context.WithTimeout(ctx, d.Timeout)
	defer cancel()

	coords, err := d.Nominatim.Geocode(ctx, query)
	if err != nil {
		return nil, SearchHotelsOutput{}, wrapUpstream(err)
	}

	hotels, err := d.Overpass.SearchHotels(ctx, coords.Lat, coords.Lon, hotelSearchRadiusMeters)
	if err != nil {
		return nil, SearchHotelsOutput{}, wrapUpstream(err)
	}

	return nil, SearchHotelsOutput{Hotels: fromClientHotels(hotels)}, nil
}

func fromClientHotels(in []client.Hotel) []Hotel {
	out := make([]Hotel, len(in))
	for i, h := range in {
		out[i] = Hotel{
			Name:      h.Name,
			Address:   h.Address,
			Latitude:  h.Latitude,
			Longitude: h.Longitude,
		}
	}
	return out
}

func validateSearchHotelsInput(in SearchHotelsInput) error {
	if strings.TrimSpace(in.City) == "" {
		return wrapValidation("city is required and cannot be empty")
	}
	return nil
}

func registerSearchHotels(server *sdkmcp.Server, d *Deps) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "search_hotels",
		Description: "Find hotels in or near a named city. Geocodes the city with OpenStreetMap Nominatim, then searches OpenStreetMap Overpass for tourism=hotel points of interest within a 5km radius of the city center. Returns an empty list (a normal, successful result) if the city geocodes fine but no hotels are found nearby — that's not an error. A tool error means the city couldn't be geocoded or an upstream service failed.",
	}, d.searchHotels)
}
