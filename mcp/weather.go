package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ameni/hospitality-scout/client"
)

const (
	defaultForecastDays = 3
	minForecastDays     = 1
	maxForecastDays     = 7
)

// GetWeatherForecastInput is the input for the get_weather_forecast tool.
type GetWeatherForecastInput struct {
	City string `json:"city" jsonschema:"the city to forecast weather for, e.g. 'Lisbon'"`
	Days int    `json:"days,omitempty" jsonschema:"number of days to forecast, from 1 to 7; defaults to 3 if omitted"`
}

// DayForecast is the forecast for a single calendar day.
type DayForecast struct {
	Date                     string  `json:"date" jsonschema:"forecast date in YYYY-MM-DD format"`
	Condition                string  `json:"condition" jsonschema:"plain-language sky and precipitation summary derived from the WMO weather code"`
	HighC                    float64 `json:"high_c" jsonschema:"forecast high temperature in degrees Celsius"`
	LowC                     float64 `json:"low_c" jsonschema:"forecast low temperature in degrees Celsius"`
	PrecipitationProbability int     `json:"precipitation_probability" jsonschema:"maximum chance of precipitation for the day, as a percentage from 0 to 100"`
}

// GetWeatherForecastOutput is the output for the get_weather_forecast tool.
type GetWeatherForecastOutput struct {
	City string        `json:"city" jsonschema:"the city the forecast was generated for"`
	Days []DayForecast `json:"days" jsonschema:"one entry per forecast day, in chronological order"`
}

func (d *Deps) getWeatherForecast(ctx context.Context, _ *sdkmcp.CallToolRequest, in GetWeatherForecastInput) (*sdkmcp.CallToolResult, GetWeatherForecastOutput, error) {
	if err := validateGetWeatherForecastInput(in); err != nil {
		return nil, GetWeatherForecastOutput{}, err
	}

	days := in.Days
	if days == 0 {
		days = defaultForecastDays
	}

	ctx, cancel := context.WithTimeout(ctx, d.Timeout)
	defer cancel()

	coords, err := d.Nominatim.Geocode(ctx, in.City)
	if err != nil {
		return nil, GetWeatherForecastOutput{}, wrapUpstream(err)
	}

	daily, err := d.OpenMeteo.GetForecast(ctx, coords.Lat, coords.Lon, days)
	if err != nil {
		return nil, GetWeatherForecastOutput{}, wrapUpstream(err)
	}

	return nil, GetWeatherForecastOutput{City: in.City, Days: fromClientDayForecasts(daily)}, nil
}

func fromClientDayForecasts(in []client.DayForecast) []DayForecast {
	out := make([]DayForecast, len(in))
	for i, d := range in {
		out[i] = DayForecast{
			Date:                     d.Date,
			Condition:                d.Condition,
			HighC:                    d.HighC,
			LowC:                     d.LowC,
			PrecipitationProbability: d.PrecipitationProbability,
		}
	}
	return out
}

func validateGetWeatherForecastInput(in GetWeatherForecastInput) error {
	if strings.TrimSpace(in.City) == "" {
		return wrapValidation("city is required and cannot be empty")
	}
	if in.Days != 0 && (in.Days < minForecastDays || in.Days > maxForecastDays) {
		return wrapValidation("days must be between %d and %d, got %d", minForecastDays, maxForecastDays, in.Days)
	}
	return nil
}

func registerGetWeatherForecast(server *sdkmcp.Server, d *Deps) {
	inSchema, err := jsonschema.For[GetWeatherForecastInput](nil)
	if err != nil {
		panic(fmt.Sprintf("mcpserver: building get_weather_forecast schema: %v", err))
	}
	if daysSchema, ok := inSchema.Properties["days"]; ok {
		daysSchema.Minimum = new(float64(minForecastDays))
		daysSchema.Maximum = new(float64(maxForecastDays))
	}

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_weather_forecast",
		Description: "Get a short-range daily weather forecast for a named city, for guest-experience and staffing planning. Geocodes the city with OpenStreetMap Nominatim, then fetches the forecast from Open-Meteo. Returns a tool error if the city can't be geocoded.",
		InputSchema: inSchema,
	}, d.getWeatherForecast)
}
