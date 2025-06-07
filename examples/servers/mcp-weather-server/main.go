package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tmc/mcp"
)

const (
	ServerName    = "mcp-weather-server"
	ServerVersion = "0.1.0"
	BaseURL       = "https://api.openweathermap.org/data/2.5"
)

type WeatherResponse struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature"`
	FeelsLike   float64 `json:"feels_like"`
	Humidity    int     `json:"humidity"`
	Pressure    int     `json:"pressure"`
	Visibility  int     `json:"visibility"`
	UVIndex     float64 `json:"uv_index,omitempty"`
	Description string  `json:"description"`
	WindSpeed   float64 `json:"wind_speed"`
	WindDegree  int     `json:"wind_degree"`
	Cloudiness  int     `json:"cloudiness"`
	Country     string  `json:"country"`
	Sunrise     string  `json:"sunrise"`
	Sunset      string  `json:"sunset"`
	Timezone    int     `json:"timezone"`
}

type ForecastDay struct {
	Date        string  `json:"date"`
	TempMax     float64 `json:"temp_max"`
	TempMin     float64 `json:"temp_min"`
	Description string  `json:"description"`
	Humidity    int     `json:"humidity"`
	WindSpeed   float64 `json:"wind_speed"`
	Cloudiness  int     `json:"cloudiness"`
}

type ForecastResponse struct {
	Location string        `json:"location"`
	Country  string        `json:"country"`
	Days     []ForecastDay `json:"forecast"`
}

type OpenWeatherCurrentResponse struct {
	Name string `json:"name"`
	Sys  struct {
		Country string `json:"country"`
		Sunrise int64  `json:"sunrise"`
		Sunset  int64  `json:"sunset"`
	} `json:"sys"`
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		Humidity  int     `json:"humidity"`
		Pressure  int     `json:"pressure"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
	Wind struct {
		Speed float64 `json:"speed"`
		Deg   int     `json:"deg"`
	} `json:"wind"`
	Clouds struct {
		All int `json:"all"`
	} `json:"clouds"`
	Visibility int `json:"visibility"`
	Timezone   int `json:"timezone"`
}

type OpenWeatherForecastResponse struct {
	City struct {
		Name    string `json:"name"`
		Country string `json:"country"`
	} `json:"city"`
	List []struct {
		Dt   int64 `json:"dt"`
		Main struct {
			TempMax  float64 `json:"temp_max"`
			TempMin  float64 `json:"temp_min"`
			Humidity int     `json:"humidity"`
		} `json:"main"`
		Weather []struct {
			Description string `json:"description"`
		} `json:"weather"`
		Wind struct {
			Speed float64 `json:"speed"`
		} `json:"wind"`
		Clouds struct {
			All int `json:"all"`
		} `json:"clouds"`
	} `json:"list"`
}

func main() {
	log.SetOutput(os.Stderr)
	log.Println("Starting MCP Weather Server...")

	apiKey := os.Getenv("OPENWEATHER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENWEATHER_API_KEY environment variable is required")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("A weather information server using OpenWeatherMap API"),
	)

	registerWeatherTools(server, apiKey)

	log.Println("Starting protocol server via stdio...")
	if err := server.Serve(ctx, nil); err != nil {
		if err != context.Canceled {
			log.Fatalf("Error serving: %v", err)
		}
		log.Println("Server terminated.")
	}
}

func registerWeatherTools(server *mcp.Server, apiKey string) {
	// Register current weather tool
	currentWeatherTool := mcp.Tool{
		Name:        "get_current_weather",
		Description: "Get current weather information for a location",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"location": {
					"type": "string",
					"description": "City name or 'city,country' format (e.g., 'London', 'Paris,FR', 'New York,US')"
				},
				"units": {
					"type": "string",
					"description": "Temperature units: 'metric' (Celsius), 'imperial' (Fahrenheit), or 'kelvin'",
					"enum": ["metric", "imperial", "kelvin"],
					"default": "metric"
				}
			},
			"required": ["location"]
		}`),
	}

	server.RegisterTool(currentWeatherTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		location, ok := params["location"].(string)
		if !ok || location == "" {
			return nil, fmt.Errorf("location is required and must be a string")
		}

		units := "metric"
		if u, ok := params["units"].(string); ok && u != "" {
			units = u
		}

		result, err := getCurrentWeather(apiKey, location, units)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error getting current weather: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		}, nil
	})

	// Register weather forecast tool
	forecastTool := mcp.Tool{
		Name:        "get_weather_forecast",
		Description: "Get 5-day weather forecast for a location",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"location": {
					"type": "string",
					"description": "City name or 'city,country' format (e.g., 'London', 'Paris,FR', 'New York,US')"
				},
				"units": {
					"type": "string",
					"description": "Temperature units: 'metric' (Celsius), 'imperial' (Fahrenheit), or 'kelvin'",
					"enum": ["metric", "imperial", "kelvin"],
					"default": "metric"
				}
			},
			"required": ["location"]
		}`),
	}

	server.RegisterTool(forecastTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		location, ok := params["location"].(string)
		if !ok || location == "" {
			return nil, fmt.Errorf("location is required and must be a string")
		}

		units := "metric"
		if u, ok := params["units"].(string); ok && u != "" {
			units = u
		}

		result, err := getWeatherForecast(apiKey, location, units)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error getting weather forecast: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		}, nil
	})

	log.Println("Registered weather tools: get_current_weather, get_weather_forecast")
}

func getCurrentWeather(apiKey, location, units string) (WeatherResponse, error) {
	url := fmt.Sprintf("%s/weather?q=%s&appid=%s&units=%s",
		BaseURL, url.QueryEscape(location), apiKey, units)

	resp, err := http.Get(url)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("failed to fetch weather data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WeatherResponse{}, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var owResp OpenWeatherCurrentResponse
	if err := json.NewDecoder(resp.Body).Decode(&owResp); err != nil {
		return WeatherResponse{}, fmt.Errorf("failed to decode response: %v", err)
	}

	description := "No description"
	if len(owResp.Weather) > 0 {
		description = owResp.Weather[0].Description
	}

	return WeatherResponse{
		Location:    owResp.Name,
		Temperature: owResp.Main.Temp,
		FeelsLike:   owResp.Main.FeelsLike,
		Humidity:    owResp.Main.Humidity,
		Pressure:    owResp.Main.Pressure,
		Visibility:  owResp.Visibility,
		Description: description,
		WindSpeed:   owResp.Wind.Speed,
		WindDegree:  owResp.Wind.Deg,
		Cloudiness:  owResp.Clouds.All,
		Country:     owResp.Sys.Country,
		Sunrise:     time.Unix(owResp.Sys.Sunrise, 0).Format(time.RFC3339),
		Sunset:      time.Unix(owResp.Sys.Sunset, 0).Format(time.RFC3339),
		Timezone:    owResp.Timezone,
	}, nil
}

func getWeatherForecast(apiKey, location, units string) (ForecastResponse, error) {
	url := fmt.Sprintf("%s/forecast?q=%s&appid=%s&units=%s",
		BaseURL, url.QueryEscape(location), apiKey, units)

	resp, err := http.Get(url)
	if err != nil {
		return ForecastResponse{}, fmt.Errorf("failed to fetch forecast data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ForecastResponse{}, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var owResp OpenWeatherForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&owResp); err != nil {
		return ForecastResponse{}, fmt.Errorf("failed to decode response: %v", err)
	}

	// Group forecast by day (taking daily max/min)
	dailyForecasts := make(map[string]ForecastDay)

	for _, item := range owResp.List {
		date := time.Unix(item.Dt, 0).Format("2006-01-02")

		description := "No description"
		if len(item.Weather) > 0 {
			description = item.Weather[0].Description
		}

		if existing, exists := dailyForecasts[date]; exists {
			// Update with higher max and lower min
			if item.Main.TempMax > existing.TempMax {
				existing.TempMax = item.Main.TempMax
			}
			if item.Main.TempMin < existing.TempMin {
				existing.TempMin = item.Main.TempMin
			}
			dailyForecasts[date] = existing
		} else {
			dailyForecasts[date] = ForecastDay{
				Date:        date,
				TempMax:     item.Main.TempMax,
				TempMin:     item.Main.TempMin,
				Description: description,
				Humidity:    item.Main.Humidity,
				WindSpeed:   item.Wind.Speed,
				Cloudiness:  item.Clouds.All,
			}
		}
	}

	// Convert map to slice and limit to 5 days
	var days []ForecastDay
	for _, day := range dailyForecasts {
		days = append(days, day)
		if len(days) >= 5 {
			break
		}
	}

	return ForecastResponse{
		Location: owResp.City.Name,
		Country:  owResp.City.Country,
		Days:     days,
	}, nil
}
