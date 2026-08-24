package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestNominatimClient(baseURL string) *NominatimClient {
	return &NominatimClient{
		httpClient: &http.Client{Timeout: 2 * time.Second},
		userAgent:  "hospitality-scout-test/1.0",
		baseURL:    baseURL,
	}
}

func TestNominatimGeocode(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
		check   func(t *testing.T, got Coordinates)
	}{
		{
			name: "parses the first, highest-ranked result",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("User-Agent"); got != "hospitality-scout-test/1.0" {
					t.Errorf("User-Agent = %q, want hospitality-scout-test/1.0", got)
				}
				w.Write([]byte(`[
					{"lat":"38.7222524","lon":"-9.1393366","display_name":"Lisbon, Portugal"},
					{"lat":"1.0","lon":"2.0","display_name":"a different, lower-ranked match"}
				]`))
			},
			check: func(t *testing.T, got Coordinates) {
				if got.Lat != 38.7222524 || got.Lon != -9.1393366 {
					t.Errorf("Geocode = %+v, want {38.7222524 -9.1393366}", got)
				}
			},
		},
		{
			name: "no results is an error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`[]`))
			},
			wantErr: true,
		},
		{
			name: "malformed latitude is an error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`[{"lat":"not-a-number","lon":"-9.1","display_name":"bad data"}]`))
			},
			wantErr: true,
		},
		{
			name: "non-200 upstream response is an error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			c := newTestNominatimClient(server.URL)
			got, err := c.Geocode(context.Background(), "Lisbon")

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
