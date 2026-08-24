package mcpserver

import "fmt"

// wrapValidation marks an error as a failure of input validation
func wrapValidation(format string, args ...any) error {
	return fmt.Errorf("invalid input: "+format, args...)
}

// wrapUpstream marks an error as a failure that happened while talking to an
// external API (geocoding, Overpass, or Open-Meteo), after validation passed.
func wrapUpstream(err error) error {
	return fmt.Errorf("upstream error: %w", err)
}
