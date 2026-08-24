package mcpserver

import "testing"

func TestValidateSearchHotelsInput(t *testing.T) {
	cases := []struct {
		name    string
		in      SearchHotelsInput
		wantErr bool
	}{
		{"valid city only", SearchHotelsInput{City: "Lisbon"}, false},
		{"valid city and region", SearchHotelsInput{City: "Austin", Region: "Texas"}, false},
		{"empty city", SearchHotelsInput{City: ""}, true},
		{"whitespace-only city", SearchHotelsInput{City: "   "}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSearchHotelsInput(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateSearchHotelsInput(%+v) error = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
		})
	}
}

func TestValidateFindNearbyAmenitiesInput(t *testing.T) {
	cases := []struct {
		name    string
		in      FindNearbyAmenitiesInput
		wantErr bool
	}{
		{"valid restaurant", FindNearbyAmenitiesInput{Latitude: 38.7, Longitude: -9.1, AmenityType: AmenityRestaurant}, false},
		{"valid bar", FindNearbyAmenitiesInput{Latitude: 0, Longitude: 0, AmenityType: AmenityMuseum}, false},
		{"valid cafe at extremes", FindNearbyAmenitiesInput{Latitude: 90, Longitude: -180, AmenityType: AmenityCafe}, false},
		{"latitude too high", FindNearbyAmenitiesInput{Latitude: 90.1, Longitude: 0, AmenityType: AmenityMuseum}, true},
		{"latitude too low", FindNearbyAmenitiesInput{Latitude: -90.1, Longitude: 0, AmenityType: AmenityMuseum}, true},
		{"longitude too high", FindNearbyAmenitiesInput{Latitude: 0, Longitude: 180.1, AmenityType: AmenityMuseum}, true},
		{"longitude too low", FindNearbyAmenitiesInput{Latitude: 0, Longitude: -180.1, AmenityType: AmenityMuseum}, true},
		{"disallowed amenity type", FindNearbyAmenitiesInput{Latitude: 0, Longitude: 0, AmenityType: "nightclub"}, true},
		{"empty amenity type", FindNearbyAmenitiesInput{Latitude: 0, Longitude: 0, AmenityType: ""}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFindNearbyAmenitiesInput(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateFindNearbyAmenitiesInput(%+v) error = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
		})
	}
}

func TestValidateGetWeatherForecastInput(t *testing.T) {
	cases := []struct {
		name    string
		in      GetWeatherForecastInput
		wantErr bool
	}{
		{"valid, days omitted", GetWeatherForecastInput{City: "Lisbon"}, false},
		{"valid, days at minimum", GetWeatherForecastInput{City: "Lisbon", Days: 1}, false},
		{"valid, days at maximum", GetWeatherForecastInput{City: "Lisbon", Days: 7}, false},
		{"empty city", GetWeatherForecastInput{City: "", Days: 3}, true},
		{"days too low", GetWeatherForecastInput{City: "Lisbon", Days: -1}, true},
		{"days too high", GetWeatherForecastInput{City: "Lisbon", Days: 8}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateGetWeatherForecastInput(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateGetWeatherForecastInput(%+v) error = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
		})
	}
}
