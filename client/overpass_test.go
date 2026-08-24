package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestOverpassClient(baseURL string) *OverpassClient {
	return &OverpassClient{
		httpClient: &http.Client{Timeout: 2 * time.Second},
		userAgent:  "hospitality-scout-test/1.0",
		baseURL:    baseURL,
	}
}

func TestOverpassSearchHotels(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
		check   func(t *testing.T, got []Hotel)
	}{
		{
			name: "maps named elements and skips unnamed ones",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"elements":[
					{"type":"node","lat":38.72,"lon":-9.14,"tags":{"name":"Hotel A","addr:housenumber":"10","addr:street":"Rua X","addr:city":"Lisbon"}},
					{"type":"node","lat":38.73,"lon":-9.15,"tags":{}}
				]}`))
			},
			check: func(t *testing.T, got []Hotel) {
				if len(got) != 1 {
					t.Fatalf("got %d hotels, want 1 (unnamed element should be skipped)", len(got))
				}
				h := got[0]
				if h.Name != "Hotel A" || h.Address != "10 Rua X, Lisbon" || h.Latitude != 38.72 || h.Longitude != -9.14 {
					t.Errorf("hotel = %+v, want Name=Hotel A Address='10 Rua X, Lisbon' Lat=38.72 Lon=-9.14", h)
				}
			},
		},
		{
			name: "empty results is not an error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"elements":[]}`))
			},
			check: func(t *testing.T, got []Hotel) {
				if len(got) != 0 {
					t.Errorf("got %d hotels, want 0", len(got))
				}
			},
		},
		{
			name: "non-200 upstream response is an error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusGatewayTimeout)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedBody []byte
			handler := tc.handler
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedBody, _ = io.ReadAll(r.Body)
				handler(w, r)
			}))
			defer server.Close()

			c := newTestOverpassClient(server.URL)
			got, err := c.SearchHotels(context.Background(), 38.72, -9.14, 5000)

			if !tc.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !strings.Contains(string(capturedBody), `tourism"="hotel"`) {
					t.Errorf("query body = %q, want it to filter on tourism=hotel", capturedBody)
				}
				if strings.Contains(string(capturedBody), "way[") {
					t.Errorf("query body = %q, expected node-only query (no way clause)", capturedBody)
				}
				if tc.check != nil {
					tc.check(t, got)
				}
				return
			}
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

func TestOverpassSearchAmenities(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
		check   func(t *testing.T, got []Amenity)
	}{
		{
			name: "sets Type from the amenity tag",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"elements":[
					{"type":"node","lat":38.72,"lon":-9.14,"tags":{"name":"Cafe A","amenity":"cafe"}}
				]}`))
			},
			check: func(t *testing.T, got []Amenity) {
				if len(got) != 1 || got[0].Type != "cafe" || got[0].Name != "Cafe A" {
					t.Errorf("amenities = %+v, want one Cafe A with type cafe", got)
				}
			},
		},
		{
			name: "non-200 upstream response is an error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			c := newTestOverpassClient(server.URL)
			got, err := c.SearchAmenities(context.Background(), 38.72, -9.14, "cafe", 1000)

			if !tc.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tc.check != nil {
					tc.check(t, got)
				}
				return
			}
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}
