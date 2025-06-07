package main

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

func TestWeatherServerBasic(t *testing.T) {
	mcpscripttest.Test(t, "testdata/basic_weather_test.txt")
}

func TestWeatherServerCurrentWeather(t *testing.T) {
	mcpscripttest.Test(t, "testdata/current_weather_test.txt")
}

func TestWeatherServerForecast(t *testing.T) {
	mcpscripttest.Test(t, "testdata/weather_forecast_test.txt")
}

func TestWeatherServerErrorHandling(t *testing.T) {
	mcpscripttest.Test(t, "testdata/weather_error_handling_test.txt")
}

func TestWeatherServerUnits(t *testing.T) {
	mcpscripttest.Test(t, "testdata/weather_units_test.txt")
}

func TestWeatherServerLocationFormats(t *testing.T) {
	mcpscripttest.Test(t, "testdata/location_formats_test.txt")
}
