package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const nominatimBaseURL = "https://nominatim.openstreetmap.org/search"

// Coordinates is a geocoded latitude/longitude pair, as resolved by Nominatim.
type Coordinates struct {
	Lat float64
	Lon float64
}

// NominatimClient geocodes free-text place names using the OpenStreetMap Nominatim search API.
type NominatimClient struct {
	httpClient *http.Client
	userAgent  string
	baseURL    string
}

// NewNominatimClient builds a NominatimClient.
func NewNominatimClient(httpClient *http.Client, userAgent string) *NominatimClient {
	return &NominatimClient{httpClient: httpClient, userAgent: userAgent, baseURL: nominatimBaseURL}
}

type nominatimResult struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
}

// Geocode resolves a free-text place query (e.g. "Lisbon, Portugal") to
// coordinates, returning Nominatim's top-ranked match.
func (c *NominatimClient) Geocode(ctx context.Context, query string) (Coordinates, error) {
	reqURL := c.baseURL + "?" + url.Values{
		"q":      {query},
		"format": {"json"},
		"limit":  {"1"},
	}.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return Coordinates{}, fmt.Errorf("nominatim geocode failed: building request: %w", err)
	}
	httpReq.Header.Set("User-Agent", c.userAgent)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return Coordinates{}, wrapDoErr("nominatim geocode", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Coordinates{}, fmt.Errorf("nominatim geocode failed: unexpected status %d", resp.StatusCode)
	}

	var results []nominatimResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return Coordinates{}, fmt.Errorf("nominatim geocode failed: decoding response: %w", err)
	}
	if len(results) == 0 {
		return Coordinates{}, fmt.Errorf("nominatim geocode failed: no match found for %q", query)
	}

	lat, err := strconv.ParseFloat(results[0].Lat, 64)
	if err != nil {
		return Coordinates{}, fmt.Errorf("nominatim geocode failed: parsing latitude %q: %w", results[0].Lat, err)
	}
	lon, err := strconv.ParseFloat(results[0].Lon, 64)
	if err != nil {
		return Coordinates{}, fmt.Errorf("nominatim geocode failed: parsing longitude %q: %w", results[0].Lon, err)
	}

	return Coordinates{Lat: lat, Lon: lon}, nil
}
