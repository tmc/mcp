# MCP Weather Server

A Model Context Protocol (MCP) server that provides weather information using the OpenWeatherMap API.

## Features

- **Current Weather**: Get real-time weather conditions for any location
- **5-Day Forecast**: Get weather predictions for the next 5 days
- **Multiple Units**: Support for metric, imperial, and Kelvin temperature units
- **Comprehensive Data**: Temperature, humidity, pressure, wind, visibility, sunrise/sunset times

## Setup

1. **Get an API Key**: Sign up at [OpenWeatherMap](https://openweathermap.org/api) to get a free API key

2. **Set Environment Variable**:
   ```bash
   export OPENWEATHER_API_KEY="your_api_key_here"
   ```

3. **Build the Server**:
   ```bash
   go build -o mcp-weather-server
   ```

4. **Run the Server**:
   ```bash
   ./mcp-weather-server
   ```

## Tools

### `get_current_weather`
Get current weather information for a specific location.

**Parameters:**
- `location` (required): City name or "city,country" format (e.g., "London", "Paris,FR", "New York,US")
- `units` (optional): Temperature units - "metric" (Celsius), "imperial" (Fahrenheit), or "kelvin". Default: "metric"

**Example Response:**
```json
{
  "location": "London",
  "temperature": 15.2,
  "feels_like": 14.8,
  "humidity": 82,
  "pressure": 1013,
  "visibility": 10000,
  "description": "light rain",
  "wind_speed": 3.6,
  "wind_degree": 230,
  "cloudiness": 75,
  "country": "GB",
  "sunrise": "2024-01-15T07:40:23Z",
  "sunset": "2024-01-15T15:58:41Z",
  "timezone": 0
}
```

### `get_weather_forecast`
Get a 5-day weather forecast for a specific location.

**Parameters:**
- `location` (required): City name or "city,country" format
- `units` (optional): Temperature units - "metric", "imperial", or "kelvin". Default: "metric"

**Example Response:**
```json
{
  "location": "London",
  "country": "GB",
  "forecast": [
    {
      "date": "2024-01-15",
      "temp_max": 16.5,
      "temp_min": 12.1,
      "description": "light rain",
      "humidity": 78,
      "wind_speed": 4.2,
      "cloudiness": 85
    }
  ]
}
```

## Configuration for Claude Desktop

Add this to your Claude Desktop configuration:

```json
{
  "mcpServers": {
    "weather": {
      "command": "/path/to/mcp-weather-server",
      "env": {
        "OPENWEATHER_API_KEY": "your_api_key_here"
      }
    }
  }
}
```

## Example Usage

Once connected, you can ask Claude:
- "What's the current weather in Tokyo?"
- "Get me the 5-day forecast for San Francisco"
- "What's the temperature in London in Fahrenheit?"
- "Is it raining in New York right now?"

## Error Handling

The server includes comprehensive error handling for:
- Invalid API keys
- Non-existent locations
- Network connectivity issues
- API rate limiting
- Invalid parameters

## Rate Limits

OpenWeatherMap free tier allows:
- 1,000 calls per day
- 60 calls per minute

The server respects these limits automatically.