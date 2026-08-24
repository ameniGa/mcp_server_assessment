package main

import (
	"context"
	"log"
	"net/http"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	mcpserver "github.com/ameni/hospitality-scout/mcp"
)

const (
	serverName    = "hospitality-scout"
	serverVersion = "0.1.0"

	// userAgent identifies this server to Nominatim and Overpass, both of
	// which ask unauthenticated callers to self-identify. It intentionally
	// carries no personal contact information.
	userAgent = "hospitality-scout-mcp/0.1 (take-home assessment; no-auth public APIs)"

	// perCallTimeout bounds each tool call's total upstream work (geocode
	// plus one search/forecast call). httpTimeout is a slightly larger outer
	// safety net on the shared *http.Client itself.
	perCallTimeout = 10 * time.Second
	httpTimeout    = 15 * time.Second
)

func main() {
	httpClient := &http.Client{Timeout: httpTimeout}
	deps := mcpserver.NewDeps(httpClient, userAgent, perCallTimeout)

	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}, nil)
	mcpserver.Register(server, deps)

	if err := server.Run(context.Background(), &sdkmcp.StdioTransport{}); err != nil {
		log.Fatalf("hospitality-scout: server exited: %v", err)
	}
}
