# MCP Time Server (Go)

A Model Context Protocol (MCP) server that provides time and timezone conversion functionality. This is a Go implementation of the time server, providing tools to get current time in any timezone and convert times between timezones.

## Features

- **get_current_time**: Get the current time in any IANA timezone
- **convert_time**: Convert a specific time from one timezone to another
- Automatic DST (Daylight Saving Time) detection
- Support for all IANA timezone names

## Tools

### get_current_time

Get the current time in a specific timezone.

**Parameters:**
- `timezone` (string, required): IANA timezone name (e.g., 'America/New_York', 'Europe/London')

**Example:**
```json
{
  "timezone": "America/New_York"
}
```

**Response:**
```json
{
  "timezone": "America/New_York",
  "datetime": "2024-01-15T14:30:00-05:00",
  "is_dst": false
}
```

### convert_time

Convert a time from one timezone to another.

**Parameters:**
- `source_timezone` (string, required): Source IANA timezone name
- `time` (string, required): Time in 24-hour format (HH:MM)
- `target_timezone` (string, required): Target IANA timezone name

**Example:**
```json
{
  "source_timezone": "America/New_York",
  "time": "14:30",
  "target_timezone": "Asia/Tokyo"
}
```

**Response:**
```json
{
  "source": {
    "timezone": "America/New_York",
    "datetime": "2024-01-15T14:30:00-05:00",
    "is_dst": false
  },
  "target": {
    "timezone": "Asia/Tokyo", 
    "datetime": "2024-01-16T04:30:00+09:00",
    "is_dst": false
  },
  "time_difference": "+14.0h"
}
```

## Usage

### Building

```bash
go build -o mcp-time-server
```

### Running

The server communicates via stdin/stdout and follows the MCP protocol:

```bash
./mcp-time-server
```

### Testing with mcp-send

You can test the server using the mcp-send tool:

```bash
# Get current time in New York
echo '{"method": "tools/call", "params": {"name": "get_current_time", "arguments": {"timezone": "America/New_York"}}}' | ./mcp-time-server

# Convert time from New York to Tokyo
echo '{"method": "tools/call", "params": {"name": "convert_time", "arguments": {"source_timezone": "America/New_York", "time": "14:30", "target_timezone": "Asia/Tokyo"}}}' | ./mcp-time-server
```

## Common Timezone Examples

- **UTC**: `UTC`
- **US Timezones**: `America/New_York`, `America/Chicago`, `America/Denver`, `America/Los_Angeles`
- **European Timezones**: `Europe/London`, `Europe/Paris`, `Europe/Berlin`, `Europe/Rome`
- **Asian Timezones**: `Asia/Tokyo`, `Asia/Shanghai`, `Asia/Kolkata`, `Asia/Dubai`
- **Australian Timezones**: `Australia/Sydney`, `Australia/Melbourne`, `Australia/Perth`

For a complete list of IANA timezone names, see: https://en.wikipedia.org/wiki/List_of_tz_database_time_zones

## Dependencies

- Standard Go library (no external dependencies)
- Uses the built-in `time` package for timezone handling