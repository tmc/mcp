# MCP Google Maps Server

A Model Context Protocol server that provides tools for interacting with Google Maps services.

## Features

- Search for places
- Get directions between locations
- Geocode addresses to coordinates
- Reverse geocode coordinates to addresses
- Get detailed place information

## Tools

### search_places
Search for places using Google Maps.
- `query` (string, required): Search query for places
- `location` (string, optional): Location to search near
- `radius` (integer, optional): Search radius in meters

### get_directions
Get directions between two locations.
- `origin` (string, required): Starting location
- `destination` (string, required): Destination location
- `mode` (string, optional): Travel mode (driving, walking, transit, bicycling)

### geocode
Convert address to coordinates.
- `address` (string, required): Address to geocode

### reverse_geocode
Convert coordinates to address.
- `lat` (number, required): Latitude coordinate
- `lng` (number, required): Longitude coordinate

### get_place_details
Get detailed information about a place.
- `place_id` (string, required): Google Places ID

## Usage

```bash
go run main.go
```

## Building

```bash
go build -o mcp-google-maps-server .
```