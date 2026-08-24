# hospitality-scout

An MCP (Model Context Protocol) server, written in Go, that gives an LLM three
tools for hospitality scouting and guest-experience support: finding hotels in
a city, finding restaurants/bars/cafes near a point, and pulling a short-range
weather forecast for staffing and planning. It runs entirely on free, no-API-key
public APIs (OpenStreetMap Nominatim, OpenStreetMap Overpass, Open-Meteo) over
stdio, with no deployment or auth required.

## Running it locally

Requires Go 1.24+ 

```sh
go build -o hospitality-scout .
./hospitality-scout
```

Or via the included `Makefile`: `make build`, `make run` (build + run).

This starts the server on stdio, it expects to be driven by an MCP client
(Claude Code, Claude Desktop, the SDK's own client, etc.), not run
interactively. Every client just needs the absolute path to the built binary;
there are no arguments, environment variables, or API keys to configure.

### Adding it to Claude Code

```sh
claude mcp add hospitality-scout /absolute/path/to/hospitality-scout
```

### Adding it to any other MCP client

Most MCP clients read the same `mcpServers` JSON shape. Add an entry like this
to the client's config:

```json
{
  "mcpServers": {
    "hospitality-scout": {
      "command": "/absolute/path/to/hospitality-scout"
    }
  }
}
```

Restart the client afterward so it picks up the new server.

To run the automated tests (all mocked, no network access required):

```sh
go test ./...
```

## Error-handling approach

Both layers report failures the same way: the handler returns a Go `error`,
which the SDK converts into a tool result with `isError: true` and the error
text as content — never a panic, never a bare 500-style message. Validation
and upstream failures are worded so they're easy to tell apart:

- Validation failures are prefixed `invalid input: ...` and explain exactly
  what was wrong (e.g. `invalid input: amenity_type must be one of
  restaurant, bar, cafe — got "nightclub"`).
- Upstream failures — a timeout, a non-200 response, or a well-formed request
  that legitimately found nothing — are prefixed `upstream error: ...` and
  wrap the underlying client error via `%w`, so the original failure (which
  API, what status code) is preserved and traceable.
- Upstream failures also say whether retrying is likely to help, since the
  agent driving this server only ever sees the error text and has to decide
  what to do next from it: timeouts and 5xx responses are worded as safe to
  retry, 429s note that a short wait may help, and other 4xx responses say
  retrying with the same input will not help.