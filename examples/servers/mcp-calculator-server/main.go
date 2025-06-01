package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/tmc/mcp"
)

const (
	ServerName    = "mcp-calculator-server"
	ServerVersion = "0.1.0"
)

type CalculationResult struct {
	Expression string  `json:"expression"`
	Result     float64 `json:"result"`
	Operation  string  `json:"operation"`
}

type StatisticsResult struct {
	Numbers []float64 `json:"numbers"`
	Count   int       `json:"count"`
	Sum     float64   `json:"sum"`
	Mean    float64   `json:"mean"`
	Median  float64   `json:"median"`
	Min     float64   `json:"min"`
	Max     float64   `json:"max"`
	StdDev  float64   `json:"standard_deviation"`
}

func main() {
	log.SetOutput(os.Stderr)
	log.Println("Starting MCP Calculator Server...")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("A calculator server supporting basic arithmetic, advanced math functions, and statistics"),
	)

	registerCalculatorTools(server)

	log.Println("Starting protocol server via stdio...")
	if err := server.Serve(ctx, nil); err != nil {
		if err != context.Canceled {
			log.Fatalf("Error serving: %v", err)
		}
		log.Println("Server terminated.")
	}
}

func registerCalculatorTools(server *mcp.Server) {
	// Basic arithmetic operations
	basicCalcTool := mcp.Tool{
		Name:        "calculate",
		Description: "Perform basic arithmetic calculations (add, subtract, multiply, divide)",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"operation": {
					"type": "string",
					"description": "The operation to perform",
					"enum": ["add", "subtract", "multiply", "divide"]
				},
				"a": {
					"type": "number",
					"description": "First number"
				},
				"b": {
					"type": "number",
					"description": "Second number"
				}
			},
			"required": ["operation", "a", "b"]
		}`),
	}

	server.RegisterTool(basicCalcTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		operation, ok := params["operation"].(string)
		if !ok {
			return nil, fmt.Errorf("operation is required and must be a string")
		}

		a, ok := params["a"].(float64)
		if !ok {
			return nil, fmt.Errorf("parameter 'a' is required and must be a number")
		}

		b, ok := params["b"].(float64)
		if !ok {
			return nil, fmt.Errorf("parameter 'b' is required and must be a number")
		}

		result, err := performBasicCalculation(operation, a, b)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error performing calculation: %v", err),
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

	// Advanced math functions
	advancedMathTool := mcp.Tool{
		Name:        "advanced_math",
		Description: "Perform advanced mathematical operations (power, sqrt, sin, cos, tan, log, etc.)",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"function": {
					"type": "string",
					"description": "The mathematical function to perform",
					"enum": ["power", "sqrt", "sin", "cos", "tan", "asin", "acos", "atan", "log", "log10", "exp", "abs", "ceil", "floor", "round"]
				},
				"x": {
					"type": "number",
					"description": "Primary input value"
				},
				"y": {
					"type": "number",
					"description": "Secondary input value (for functions like power)",
					"optional": true
				}
			},
			"required": ["function", "x"]
		}`),
	}

	server.RegisterTool(advancedMathTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		function, ok := params["function"].(string)
		if !ok {
			return nil, fmt.Errorf("function is required and must be a string")
		}

		x, ok := params["x"].(float64)
		if !ok {
			return nil, fmt.Errorf("parameter 'x' is required and must be a number")
		}

		var y float64
		if yVal, exists := params["y"]; exists {
			if yFloat, ok := yVal.(float64); ok {
				y = yFloat
			}
		}

		result, err := performAdvancedMath(function, x, y)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error performing advanced math: %v", err),
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

	// Statistics tool
	statsTool := mcp.Tool{
		Name:        "statistics",
		Description: "Calculate statistical measures for a set of numbers",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"numbers": {
					"type": "array",
					"items": {
						"type": "number"
					},
					"description": "Array of numbers to analyze"
				}
			},
			"required": ["numbers"]
		}`),
	}

	server.RegisterTool(statsTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		numbersInterface, ok := params["numbers"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("numbers is required and must be an array")
		}

		numbers := make([]float64, len(numbersInterface))
		for i, num := range numbersInterface {
			if floatNum, ok := num.(float64); ok {
				numbers[i] = floatNum
			} else {
				return nil, fmt.Errorf("all elements in numbers array must be numbers")
			}
		}

		if len(numbers) == 0 {
			return nil, fmt.Errorf("numbers array cannot be empty")
		}

		result := calculateStatistics(numbers)
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

	// Unit conversion tool
	conversionTool := mcp.Tool{
		Name:        "convert_units",
		Description: "Convert between different units of measurement",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"value": {
					"type": "number",
					"description": "The value to convert"
				},
				"from_unit": {
					"type": "string",
					"description": "Source unit",
					"enum": ["celsius", "fahrenheit", "kelvin", "meters", "feet", "inches", "kilometers", "miles", "kilograms", "pounds", "ounces", "liters", "gallons", "ounces_fl"]
				},
				"to_unit": {
					"type": "string",
					"description": "Target unit",
					"enum": ["celsius", "fahrenheit", "kelvin", "meters", "feet", "inches", "kilometers", "miles", "kilograms", "pounds", "ounces", "liters", "gallons", "ounces_fl"]
				}
			},
			"required": ["value", "from_unit", "to_unit"]
		}`),
	}

	server.RegisterTool(conversionTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		value, ok := params["value"].(float64)
		if !ok {
			return nil, fmt.Errorf("value is required and must be a number")
		}

		fromUnit, ok := params["from_unit"].(string)
		if !ok {
			return nil, fmt.Errorf("from_unit is required and must be a string")
		}

		toUnit, ok := params["to_unit"].(string)
		if !ok {
			return nil, fmt.Errorf("to_unit is required and must be a string")
		}

		result, err := convertUnits(value, fromUnit, toUnit)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error converting units: %v", err),
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

	log.Println("Registered calculator tools: calculate, advanced_math, statistics, convert_units")
}

func performBasicCalculation(operation string, a, b float64) (CalculationResult, error) {
	var result float64
	expression := fmt.Sprintf("%.2f %s %.2f", a, getOperatorSymbol(operation), b)

	switch operation {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return CalculationResult{}, fmt.Errorf("division by zero")
		}
		result = a / b
	default:
		return CalculationResult{}, fmt.Errorf("unsupported operation: %s", operation)
	}

	return CalculationResult{
		Expression: expression,
		Result:     result,
		Operation:  operation,
	}, nil
}

func performAdvancedMath(function string, x, y float64) (CalculationResult, error) {
	var result float64
	var expression string

	switch function {
	case "power":
		result = math.Pow(x, y)
		expression = fmt.Sprintf("%.2f^%.2f", x, y)
	case "sqrt":
		if x < 0 {
			return CalculationResult{}, fmt.Errorf("cannot take square root of negative number")
		}
		result = math.Sqrt(x)
		expression = fmt.Sprintf("sqrt(%.2f)", x)
	case "sin":
		result = math.Sin(x)
		expression = fmt.Sprintf("sin(%.2f)", x)
	case "cos":
		result = math.Cos(x)
		expression = fmt.Sprintf("cos(%.2f)", x)
	case "tan":
		result = math.Tan(x)
		expression = fmt.Sprintf("tan(%.2f)", x)
	case "asin":
		if x < -1 || x > 1 {
			return CalculationResult{}, fmt.Errorf("asin input must be between -1 and 1")
		}
		result = math.Asin(x)
		expression = fmt.Sprintf("asin(%.2f)", x)
	case "acos":
		if x < -1 || x > 1 {
			return CalculationResult{}, fmt.Errorf("acos input must be between -1 and 1")
		}
		result = math.Acos(x)
		expression = fmt.Sprintf("acos(%.2f)", x)
	case "atan":
		result = math.Atan(x)
		expression = fmt.Sprintf("atan(%.2f)", x)
	case "log":
		if x <= 0 {
			return CalculationResult{}, fmt.Errorf("cannot take natural log of non-positive number")
		}
		result = math.Log(x)
		expression = fmt.Sprintf("ln(%.2f)", x)
	case "log10":
		if x <= 0 {
			return CalculationResult{}, fmt.Errorf("cannot take log10 of non-positive number")
		}
		result = math.Log10(x)
		expression = fmt.Sprintf("log10(%.2f)", x)
	case "exp":
		result = math.Exp(x)
		expression = fmt.Sprintf("e^%.2f", x)
	case "abs":
		result = math.Abs(x)
		expression = fmt.Sprintf("abs(%.2f)", x)
	case "ceil":
		result = math.Ceil(x)
		expression = fmt.Sprintf("ceil(%.2f)", x)
	case "floor":
		result = math.Floor(x)
		expression = fmt.Sprintf("floor(%.2f)", x)
	case "round":
		result = math.Round(x)
		expression = fmt.Sprintf("round(%.2f)", x)
	default:
		return CalculationResult{}, fmt.Errorf("unsupported function: %s", function)
	}

	return CalculationResult{
		Expression: expression,
		Result:     result,
		Operation:  function,
	}, nil
}

func calculateStatistics(numbers []float64) StatisticsResult {
	count := len(numbers)
	
	// Calculate sum
	sum := 0.0
	for _, num := range numbers {
		sum += num
	}
	
	// Calculate mean
	mean := sum / float64(count)
	
	// Find min and max
	min := numbers[0]
	max := numbers[0]
	for _, num := range numbers {
		if num < min {
			min = num
		}
		if num > max {
			max = num
		}
	}
	
	// Calculate median
	sorted := make([]float64, len(numbers))
	copy(sorted, numbers)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	var median float64
	if count%2 == 0 {
		median = (sorted[count/2-1] + sorted[count/2]) / 2
	} else {
		median = sorted[count/2]
	}
	
	// Calculate standard deviation
	variance := 0.0
	for _, num := range numbers {
		variance += math.Pow(num-mean, 2)
	}
	variance /= float64(count)
	stdDev := math.Sqrt(variance)
	
	return StatisticsResult{
		Numbers: numbers,
		Count:   count,
		Sum:     sum,
		Mean:    mean,
		Median:  median,
		Min:     min,
		Max:     max,
		StdDev:  stdDev,
	}
}

func convertUnits(value float64, fromUnit, toUnit string) (map[string]interface{}, error) {
	// Temperature conversions
	if isTemperatureUnit(fromUnit) && isTemperatureUnit(toUnit) {
		result, err := convertTemperature(value, fromUnit, toUnit)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"original_value": value,
			"original_unit":  fromUnit,
			"converted_value": result,
			"converted_unit":  toUnit,
			"conversion_type": "temperature",
		}, nil
	}
	
	// Length conversions
	if isLengthUnit(fromUnit) && isLengthUnit(toUnit) {
		result, err := convertLength(value, fromUnit, toUnit)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"original_value": value,
			"original_unit":  fromUnit,
			"converted_value": result,
			"converted_unit":  toUnit,
			"conversion_type": "length",
		}, nil
	}
	
	// Weight conversions
	if isWeightUnit(fromUnit) && isWeightUnit(toUnit) {
		result, err := convertWeight(value, fromUnit, toUnit)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"original_value": value,
			"original_unit":  fromUnit,
			"converted_value": result,
			"converted_unit":  toUnit,
			"conversion_type": "weight",
		}, nil
	}
	
	// Volume conversions
	if isVolumeUnit(fromUnit) && isVolumeUnit(toUnit) {
		result, err := convertVolume(value, fromUnit, toUnit)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"original_value": value,
			"original_unit":  fromUnit,
			"converted_value": result,
			"converted_unit":  toUnit,
			"conversion_type": "volume",
		}, nil
	}
	
	return nil, fmt.Errorf("incompatible unit types: %s and %s", fromUnit, toUnit)
}

func getOperatorSymbol(operation string) string {
	switch operation {
	case "add":
		return "+"
	case "subtract":
		return "-"
	case "multiply":
		return "*"
	case "divide":
		return "/"
	default:
		return operation
	}
}

func isTemperatureUnit(unit string) bool {
	return unit == "celsius" || unit == "fahrenheit" || unit == "kelvin"
}

func isLengthUnit(unit string) bool {
	return unit == "meters" || unit == "feet" || unit == "inches" || unit == "kilometers" || unit == "miles"
}

func isWeightUnit(unit string) bool {
	return unit == "kilograms" || unit == "pounds" || unit == "ounces"
}

func isVolumeUnit(unit string) bool {
	return unit == "liters" || unit == "gallons" || unit == "ounces_fl"
}

func convertTemperature(value float64, from, to string) (float64, error) {
	// Convert to Celsius first
	var celsius float64
	switch from {
	case "celsius":
		celsius = value
	case "fahrenheit":
		celsius = (value - 32) * 5 / 9
	case "kelvin":
		celsius = value - 273.15
	default:
		return 0, fmt.Errorf("unknown temperature unit: %s", from)
	}
	
	// Convert from Celsius to target
	switch to {
	case "celsius":
		return celsius, nil
	case "fahrenheit":
		return celsius*9/5 + 32, nil
	case "kelvin":
		return celsius + 273.15, nil
	default:
		return 0, fmt.Errorf("unknown temperature unit: %s", to)
	}
}

func convertLength(value float64, from, to string) (float64, error) {
	// Convert to meters first
	var meters float64
	switch from {
	case "meters":
		meters = value
	case "feet":
		meters = value * 0.3048
	case "inches":
		meters = value * 0.0254
	case "kilometers":
		meters = value * 1000
	case "miles":
		meters = value * 1609.34
	default:
		return 0, fmt.Errorf("unknown length unit: %s", from)
	}
	
	// Convert from meters to target
	switch to {
	case "meters":
		return meters, nil
	case "feet":
		return meters / 0.3048, nil
	case "inches":
		return meters / 0.0254, nil
	case "kilometers":
		return meters / 1000, nil
	case "miles":
		return meters / 1609.34, nil
	default:
		return 0, fmt.Errorf("unknown length unit: %s", to)
	}
}

func convertWeight(value float64, from, to string) (float64, error) {
	// Convert to kilograms first
	var kg float64
	switch from {
	case "kilograms":
		kg = value
	case "pounds":
		kg = value * 0.453592
	case "ounces":
		kg = value * 0.0283495
	default:
		return 0, fmt.Errorf("unknown weight unit: %s", from)
	}
	
	// Convert from kilograms to target
	switch to {
	case "kilograms":
		return kg, nil
	case "pounds":
		return kg / 0.453592, nil
	case "ounces":
		return kg / 0.0283495, nil
	default:
		return 0, fmt.Errorf("unknown weight unit: %s", to)
	}
}

func convertVolume(value float64, from, to string) (float64, error) {
	// Convert to liters first
	var liters float64
	switch from {
	case "liters":
		liters = value
	case "gallons":
		liters = value * 3.78541
	case "ounces_fl":
		liters = value * 0.0295735
	default:
		return 0, fmt.Errorf("unknown volume unit: %s", from)
	}
	
	// Convert from liters to target
	switch to {
	case "liters":
		return liters, nil
	case "gallons":
		return liters / 3.78541, nil
	case "ounces_fl":
		return liters / 0.0295735, nil
	default:
		return 0, fmt.Errorf("unknown volume unit: %s", to)
	}
}