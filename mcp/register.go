package mcpserver

import (
	"net/http"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ameni/hospitality-scout/client"
)

// Deps bundles the external API clients and shared configuration used by all
// tool handlers.
type Deps struct {
	Nominatim *client.NominatimClient
	Overpass  *client.OverpassClient
	OpenMeteo *client.OpenMeteoClient

	// Timeout bounds each tool call's total upstream work (geocode + search).
	Timeout time.Duration
}

// NewDeps builds the shared Deps used by all registered tools.
func NewDeps(httpClient *http.Client, userAgent string, timeout time.Duration) *Deps {
	return &Deps{
		Nominatim: client.NewNominatimClient(httpClient, userAgent),
		Overpass:  client.NewOverpassClient(httpClient, userAgent),
		OpenMeteo: client.NewOpenMeteoClient(httpClient),
		Timeout:   timeout,
	}
}

// Register adds all three tools to server.
func Register(server *sdkmcp.Server, deps *Deps) {
	registerSearchHotels(server, deps)
}
