package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestOpenMeteoClient(baseURL string) *OpenMeteoClient {
	return &OpenMeteoClient{
		httpClient: &http.Client{Timeout: 2 * time.Second},
		baseURL:    baseURL,
	}
}

func TestOpenMeteoGetForecast(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
		check   func(t *testing.T, got []DayForecast)
	}{
		{
			name: "maps daily arrays by index",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"daily":{
					"time":["2026-08-23","2026-08-24"],
					"temperature_2m_max":[23.5,24.9],
					"temperature_2m_min":[19.3,20.4],
					"precipitation_probability_max":[100,90],
					"weathercode":[95,1]
				}}`))
			},
			check: func(t *testing.T, got []DayForecast) {
				if len(got) != 2 {
					t.Fatalf("got %d days, want 2", len(got))
				}
				if got[0].Date != "2026-08-23" || got[0].HighC != 23.5 || got[0].LowC != 19.3 ||
					got[0].PrecipitationProbability != 100 || got[0].Condition != "thunderstorm" {
					t.Errorf("days[0] = %+v, unexpected mapping", got[0])
				}
				if got[1].Condition != "partly cloudy" {
					t.Errorf("days[1].Condition = %q, want partly cloudy", got[1].Condition)
				}
			},
		},
		{
			name: "empty daily data is an error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"daily":{"time":[]}}`))
			},
			wantErr: true,
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

			c := newTestOpenMeteoClient(server.URL)
			got, err := c.GetForecast(context.Background(), 38.72, -9.14, 2)

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
