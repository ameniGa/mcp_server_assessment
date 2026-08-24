package mcpserver

import (
	"context"
	"net/http"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// connectInMemory builds a real *sdkmcp.Server via Register and connects a
// real *sdkmcp.Client to it over an in-memory transport.
func connectInMemory(t *testing.T, deps *Deps) *sdkmcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "hospitality-scout-test", Version: "v0.0.1"}, nil)
	Register(server, deps)

	serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func TestServer_ToolsRoundTrip(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "nominatim.openstreetmap.org":
			return jsonResponse(200, `[{"lat":"38.72","lon":"-9.14","display_name":"Lisbon"}]`), nil
		case "overpass-api.de":
			return jsonResponse(200, `{"elements":[{"type":"node","lat":38.72,"lon":-9.14,"tags":{"name":"Hotel Test","amenity":"cafe"}}]}`), nil
		case "api.open-meteo.com":
			return jsonResponse(200, `{"daily":{"time":["2026-08-23"],"temperature_2m_max":[24],"temperature_2m_min":[19],"precipitation_probability_max":[10],"weathercode":[0]}}`), nil
		default:
			t.Fatalf("unexpected request host: %s", req.URL.Host)
			return nil, nil
		}
	})
	deps := NewDeps(&http.Client{Transport: transport}, "test-ua", 5*time.Second)
	session := connectInMemory(t, deps)
	ctx := context.Background()

	cases := []struct {
		name string
		args map[string]any
	}{
		{"search_hotels", map[string]any{"city": "Lisbon"}},
		{"find_nearby_amenities", map[string]any{"latitude": 38.72, "longitude": -9.14, "amenity_type": "cafe"}},
		{"get_weather_forecast", map[string]any{"city": "Lisbon"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: tc.name, Arguments: tc.args})
			if err != nil {
				t.Fatalf("CallTool(%s) protocol error: %v", tc.name, err)
			}
			if res.IsError {
				t.Fatalf("CallTool(%s) returned a tool error: %v", tc.name, res.Content)
			}
			if res.StructuredContent == nil {
				t.Errorf("CallTool(%s) returned no structured content", tc.name)
			}
		})
	}
}

func TestServer_RejectsDisallowedAmenityTypeBeforeHandlerRuns(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatal("network call made for a request the SDK's own schema validation should have rejected")
		return nil, nil
	})
	deps := NewDeps(&http.Client{Transport: transport}, "test-ua", 5*time.Second)
	session := connectInMemory(t, deps)
	ctx := context.Background()

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "find_nearby_amenities",
		Arguments: map[string]any{"latitude": 38.7, "longitude": -9.1, "amenity_type": "nightclub"},
	})
	if err != nil {
		t.Fatalf("CallTool protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected the SDK's JSON schema validation to reject amenity_type=nightclub, got a successful result")
	}
}

func TestServer_RejectsOutOfRangeCoordinatesBeforeHandlerRuns(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatal("network call made for a request the SDK's own schema validation should have rejected")
		return nil, nil
	})
	deps := NewDeps(&http.Client{Transport: transport}, "test-ua", 5*time.Second)
	session := connectInMemory(t, deps)
	ctx := context.Background()

	cases := []struct {
		name string
		args map[string]any
	}{
		{"latitude too high", map[string]any{"latitude": 90.1, "longitude": 0, "amenity_type": "cafe"}},
		{"latitude too low", map[string]any{"latitude": -90.1, "longitude": 0, "amenity_type": "cafe"}},
		{"longitude too high", map[string]any{"latitude": 0, "longitude": 180.1, "amenity_type": "cafe"}},
		{"longitude too low", map[string]any{"latitude": 0, "longitude": -180.1, "amenity_type": "cafe"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
				Name:      "find_nearby_amenities",
				Arguments: tc.args,
			})
			if err != nil {
				t.Fatalf("CallTool protocol error: %v", err)
			}
			if !res.IsError {
				t.Fatalf("expected the SDK's JSON schema validation to reject %+v, got a successful result", tc.args)
			}
		})
	}
}
