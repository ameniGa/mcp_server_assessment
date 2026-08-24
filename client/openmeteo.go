package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const openMeteoBaseURL = "https://api.open-meteo.com/v1/forecast"

// DayForecast is one day's forecast, parsed from an Open-Meteo response.
type DayForecast struct {
	Date                     string
	Condition                string
	HighC                    float64
	LowC                     float64
	PrecipitationProbability int
}

// OpenMeteoClient fetches daily weather forecasts from Open-Meteo
type OpenMeteoClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewOpenMeteoClient builds an OpenMeteoClient.
func NewOpenMeteoClient(httpClient *http.Client) *OpenMeteoClient {
	return &OpenMeteoClient{httpClient: httpClient, baseURL: openMeteoBaseURL}
}

type openMeteoResponse struct {
	Daily struct {
		Time                        []string  `json:"time"`
		TemperatureMax              []float64 `json:"temperature_2m_max"`
		TemperatureMin              []float64 `json:"temperature_2m_min"`
		PrecipitationProbabilityMax []float64 `json:"precipitation_probability_max"`
		WeatherCode                 []int     `json:"weathercode"`
	} `json:"daily"`
}

// GetForecast fetches a days-long daily forecast for (lat, lon).
func (c *OpenMeteoClient) GetForecast(ctx context.Context, lat, lon float64, days int) ([]DayForecast, error) {
	q := url.Values{
		"latitude":      {strconv.FormatFloat(lat, 'f', -1, 64)},
		"longitude":     {strconv.FormatFloat(lon, 'f', -1, 64)},
		"daily":         {"temperature_2m_max,temperature_2m_min,precipitation_probability_max,weathercode"},
		"forecast_days": {strconv.Itoa(days)},
		"timezone":      {"auto"},
	}
	reqURL := c.baseURL + "?" + q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("open-meteo forecast failed: building request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, wrapDoErr("open-meteo forecast", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo forecast failed: %s", classifyStatus(resp.StatusCode))
	}

	var out openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("open-meteo forecast failed: decoding response: %w", err)
	}
	if len(out.Daily.Time) == 0 {
		return nil, fmt.Errorf("open-meteo forecast failed: empty forecast data returned for (%v, %v)", lat, lon)
	}

	forecasts := make([]DayForecast, 0, len(out.Daily.Time))
	for i, date := range out.Daily.Time {
		day := DayForecast{Date: date}
		if i < len(out.Daily.WeatherCode) {
			day.Condition = weatherCodeToCondition(out.Daily.WeatherCode[i])
		}
		if i < len(out.Daily.TemperatureMax) {
			day.HighC = out.Daily.TemperatureMax[i]
		}
		if i < len(out.Daily.TemperatureMin) {
			day.LowC = out.Daily.TemperatureMin[i]
		}
		if i < len(out.Daily.PrecipitationProbabilityMax) {
			day.PrecipitationProbability = int(out.Daily.PrecipitationProbabilityMax[i])
		}
		forecasts = append(forecasts, day)
	}
	return forecasts, nil
}

// weatherCodeToCondition maps a WMO weather interpretation code (as used by
// Open-Meteo) to a short, human-readable condition string.
func weatherCodeToCondition(code int) string {
	switch {
	case code == 0:
		return "clear sky"
	case code >= 1 && code <= 3:
		return "partly cloudy"
	case code == 45 || code == 48:
		return "fog"
	case code >= 51 && code <= 57:
		return "drizzle"
	case code >= 61 && code <= 67:
		return "rain"
	case code >= 71 && code <= 77:
		return "snow"
	case code >= 80 && code <= 82:
		return "rain showers"
	case code >= 85 && code <= 86:
		return "snow showers"
	case code >= 95 && code <= 99:
		return "thunderstorm"
	default:
		return "unknown"
	}
}
