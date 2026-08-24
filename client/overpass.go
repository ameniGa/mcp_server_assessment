package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const overpassBaseURL = "https://overpass-api.de/api/interpreter"

// Hotel is a lodging property parsed from an Overpass response.
type Hotel struct {
	Name      string
	Address   string
	Latitude  float64
	Longitude float64
}

// Amenity is a restaurant, bar, or cafe parsed from an Overpass response.
type Amenity struct {
	Name      string
	Type      string
	Address   string
	Latitude  float64
	Longitude float64
}

// OverpassClient queries the OpenStreetMap Overpass API for points of
// interest near a coordinate.
type OverpassClient struct {
	httpClient *http.Client
	userAgent  string
	baseURL    string
}

// NewOverpassClient builds an OverpassClient.
func NewOverpassClient(httpClient *http.Client, userAgent string) *OverpassClient {
	return &OverpassClient{httpClient: httpClient, userAgent: userAgent, baseURL: overpassBaseURL}
}

// resultLimit caps how many elements Overpass returns per query
const resultLimit = 30

type overpassResponse struct {
	Elements []overpassElement `json:"elements"`
}

type overpassElement struct {
	Lat  float64           `json:"lat"`
	Lon  float64           `json:"lon"`
	Tags map[string]string `json:"tags"`
}

// address assembles a best-effort street address from common OSM address
// tags. It returns "" if none of them are present, rather than guessing.
func (e overpassElement) address() string {
	var line string
	if n := e.Tags["addr:housenumber"]; n != "" {
		line = n
	}
	if s := e.Tags["addr:street"]; s != "" {
		if line != "" {
			line += " "
		}
		line += s
	}
	if city := e.Tags["addr:city"]; city != "" {
		if line != "" {
			line += ", "
		}
		line += city
	}
	return line
}

// query runs a raw Overpass QL string and decodes the JSON response.
func (c *OverpassClient) query(ctx context.Context, ql string) (*overpassResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, strings.NewReader("data="+ql))
	if err != nil {
		return nil, fmt.Errorf("overpass query failed: building request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, wrapDoErr("overpass query", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("overpass query failed: %s", classifyStatus(resp.StatusCode))
	}

	var out overpassResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("overpass query failed: decoding response: %w", err)
	}
	return &out, nil
}

// SearchHotels finds tourism=hotel nodes within radiusMeters of (lat, lon).
func (c *OverpassClient) SearchHotels(ctx context.Context, lat, lon float64, radiusMeters int) ([]Hotel, error) {
	ql := fmt.Sprintf(
		`[out:json][timeout:20];node["tourism"="hotel"](around:%d,%f,%f);out %d;`,
		radiusMeters, lat, lon, resultLimit,
	)
	resp, err := c.query(ctx, ql)
	if err != nil {
		return nil, fmt.Errorf("overpass hotel search failed: %w", err)
	}

	hotels := make([]Hotel, 0, len(resp.Elements))
	for _, el := range resp.Elements {
		name := el.Tags["name"]
		if name == "" {
			continue
		}
		hotels = append(hotels, Hotel{
			Name:      name,
			Address:   el.address(),
			Latitude:  el.Lat,
			Longitude: el.Lon,
		})
	}
	return hotels, nil
}

// SearchAmenities finds amenity=<amenityType> nodes within radiusMeters of (lat, lon).
func (c *OverpassClient) SearchAmenities(ctx context.Context, lat, lon float64, amenityType string, radiusMeters int) ([]Amenity, error) {
	ql := fmt.Sprintf(
		`[out:json][timeout:20];node["amenity"="%s"](around:%d,%f,%f);out %d;`,
		amenityType, radiusMeters, lat, lon, resultLimit,
	)
	resp, err := c.query(ctx, ql)
	if err != nil {
		return nil, fmt.Errorf("overpass amenity search failed: %w", err)
	}

	amenities := make([]Amenity, 0, len(resp.Elements))
	for _, el := range resp.Elements {
		name := el.Tags["name"]
		if name == "" {
			continue
		}
		amenities = append(amenities, Amenity{
			Name:      name,
			Type:      el.Tags["amenity"],
			Address:   el.address(),
			Latitude:  el.Lat,
			Longitude: el.Lon,
		})
	}
	return amenities, nil
}
