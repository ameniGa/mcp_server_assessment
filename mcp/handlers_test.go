package mcpserver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// roundTripFunc lets a plain function satisfy http.RoundTripper, so tests can
// stub the shared *http.Client without hitting any real network or server.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func failOnAnyRequest(t *testing.T) roundTripFunc {
	return func(req *http.Request) (*http.Response, error) {
		t.Fatalf("network call made to %s; validation should have short-circuited before any request", req.URL)
		return nil, nil
	}
}

func TestSearchHotels(t *testing.T) {
	cases := []struct {
		name      string
		in        SearchHotelsInput
		transport roundTripFunc
		wantErr   bool
		check     func(t *testing.T, out SearchHotelsOutput)
	}{
		{
			name: "success",
			in:   SearchHotelsInput{City: "Lisbon"},
			transport: func(req *http.Request) (*http.Response, error) {
				switch req.URL.Host {
				case "nominatim.openstreetmap.org":
					return jsonResponse(200, `[{"lat":"38.72","lon":"-9.14","display_name":"Lisbon"}]`), nil
				case "overpass-api.de":
					return jsonResponse(200, `{"elements":[{"type":"node","lat":38.72,"lon":-9.14,"tags":{"name":"Hotel Test"}}]}`), nil
				default:
					return nil, fmt.Errorf("unexpected request host: %s", req.URL.Host)
				}
			},
			check: func(t *testing.T, out SearchHotelsOutput) {
				if len(out.Hotels) != 1 || out.Hotels[0].Name != "Hotel Test" {
					t.Errorf("out.Hotels = %+v, want one hotel named 'Hotel Test'", out.Hotels)
				}
			},
		},
		{
			name:    "empty city fails validation before any network call",
			in:      SearchHotelsInput{City: ""},
			wantErr: true,
		},
		{
			name: "geocoder failure returns an error",
			in:   SearchHotelsInput{City: "Lisbon"},
			transport: func(req *http.Request) (*http.Response, error) {
				return jsonResponse(504, ""), nil
			},
			wantErr: true,
		},
		{
			name: "no hotels found returns a successful empty list, not an error",
			in:   SearchHotelsInput{City: "Nowhere"},
			transport: func(req *http.Request) (*http.Response, error) {
				switch req.URL.Host {
				case "nominatim.openstreetmap.org":
					return jsonResponse(200, `[{"lat":"0","lon":"0","display_name":"Middle of nowhere"}]`), nil
				case "overpass-api.de":
					return jsonResponse(200, `{"elements":[]}`), nil
				default:
					return nil, fmt.Errorf("unexpected request host: %s", req.URL.Host)
				}
			},
			check: func(t *testing.T, out SearchHotelsOutput) {
				if out.Hotels == nil || len(out.Hotels) != 0 {
					t.Errorf("out.Hotels = %+v, want a non-nil empty slice", out.Hotels)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := tc.transport
			if transport == nil {
				transport = failOnAnyRequest(t)
			}
			deps := NewDeps(&http.Client{Transport: transport}, "test-ua", 5*time.Second)
			_, out, err := deps.searchHotels(context.Background(), nil, tc.in)
			checkHandlerResult(t, err, out, tc.wantErr, tc.check)
		})
	}
}

func TestFindNearbyAmenities(t *testing.T) {
	cases := []struct {
		name      string
		in        FindNearbyAmenitiesInput
		transport roundTripFunc
		wantErr   bool
		check     func(t *testing.T, out FindNearbyAmenitiesOutput)
	}{
		{
			name: "success",
			in:   FindNearbyAmenitiesInput{Latitude: 38.72, Longitude: -9.14, AmenityType: AmenityCafe},
			transport: func(req *http.Request) (*http.Response, error) {
				return jsonResponse(200, `{"elements":[{"type":"node","lat":38.72,"lon":-9.14,"tags":{"name":"Cafe Test","amenity":"cafe"}}]}`), nil
			},
			check: func(t *testing.T, out FindNearbyAmenitiesOutput) {
				if len(out.Amenities) != 1 || out.Amenities[0].Name != "Cafe Test" {
					t.Errorf("out.Amenities = %+v, want one amenity named 'Cafe Test'", out.Amenities)
				}
			},
		},
		{
			name:    "disallowed amenity type fails validation before any network call",
			in:      FindNearbyAmenitiesInput{Latitude: 38.7, Longitude: -9.1, AmenityType: "nightclub"},
			wantErr: true,
		},
		{
			name: "no amenities found returns a successful empty list, not an error",
			in:   FindNearbyAmenitiesInput{Latitude: 38.7, Longitude: -9.1, AmenityType: AmenityMuseum},
			transport: func(req *http.Request) (*http.Response, error) {
				return jsonResponse(200, `{"elements":[]}`), nil
			},
			check: func(t *testing.T, out FindNearbyAmenitiesOutput) {
				if out.Amenities == nil || len(out.Amenities) != 0 {
					t.Errorf("out.Amenities = %+v, want a non-nil empty slice", out.Amenities)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := tc.transport
			if transport == nil {
				transport = failOnAnyRequest(t)
			}
			deps := NewDeps(&http.Client{Transport: transport}, "test-ua", 5*time.Second)
			_, out, err := deps.findNearbyAmenities(context.Background(), nil, tc.in)
			checkHandlerResult(t, err, out, tc.wantErr, tc.check)
		})
	}
}

func TestGetWeatherForecast(t *testing.T) {
	cases := []struct {
		name      string
		in        GetWeatherForecastInput
		transport roundTripFunc
		wantErr   bool
		check     func(t *testing.T, out GetWeatherForecastOutput)
	}{
		{
			name: "defaults days when omitted",
			in:   GetWeatherForecastInput{City: "Lisbon"},
			transport: func(req *http.Request) (*http.Response, error) {
				switch req.URL.Host {
				case "nominatim.openstreetmap.org":
					return jsonResponse(200, `[{"lat":"38.72","lon":"-9.14","display_name":"Lisbon"}]`), nil
				case "api.open-meteo.com":
					if got := req.URL.Query().Get("forecast_days"); got != "3" {
						t.Errorf("forecast_days query param = %q, want \"3\" (the default)", got)
					}
					return jsonResponse(200, `{"daily":{"time":["2026-08-23"],"temperature_2m_max":[24],"temperature_2m_min":[19],"precipitation_probability_max":[10],"weathercode":[0]}}`), nil
				default:
					return nil, fmt.Errorf("unexpected request host: %s", req.URL.Host)
				}
			},
			check: func(t *testing.T, out GetWeatherForecastOutput) {
				if out.City != "Lisbon" || len(out.Days) != 1 {
					t.Errorf("out = %+v, unexpected shape", out)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := tc.transport
			if transport == nil {
				transport = failOnAnyRequest(t)
			}
			deps := NewDeps(&http.Client{Transport: transport}, "test-ua", 5*time.Second)
			_, out, err := deps.getWeatherForecast(context.Background(), nil, tc.in)
			checkHandlerResult(t, err, out, tc.wantErr, tc.check)
		})
	}
}

func checkHandlerResult[Out any](t *testing.T, err error, out Out, wantErr bool, check func(*testing.T, Out)) {
	t.Helper()
	if !wantErr {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if check != nil {
			check(t, out)
		}
		return
	}
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}
