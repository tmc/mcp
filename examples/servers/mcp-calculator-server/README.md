# MCP Calculator Server

A comprehensive Model Context Protocol (MCP) server that provides mathematical calculation capabilities including basic arithmetic, advanced functions, statistics, and unit conversions.

## Features

- **Basic Arithmetic**: Addition, subtraction, multiplication, division
- **Advanced Math**: Power, square root, trigonometric functions, logarithms, exponentials
- **Statistics**: Mean, median, standard deviation, min, max for datasets
- **Unit Conversion**: Temperature, length, weight, and volume conversions

## Setup

1. **Build the Server**:
   ```bash
   go build -o mcp-calculator-server
   ```

2. **Run the Server**:
   ```bash
   ./mcp-calculator-server
   ```

## Tools

### `calculate`
Perform basic arithmetic calculations.

**Parameters:**
- `operation` (required): "add", "subtract", "multiply", or "divide"
- `a` (required): First number
- `b` (required): Second number

**Example:**
```json
{
  "operation": "multiply",
  "a": 15,
  "b": 7
}
```

**Response:**
```json
{
  "expression": "15.00 * 7.00",
  "result": 105,
  "operation": "multiply"
}
```

### `advanced_math`
Perform advanced mathematical operations.

**Parameters:**
- `function` (required): Mathematical function to perform
- `x` (required): Primary input value
- `y` (optional): Secondary input value (for functions like power)

**Supported Functions:**
- `power`: x^y (requires y parameter)
- `sqrt`: Square root
- `sin`, `cos`, `tan`: Trigonometric functions (input in radians)
- `asin`, `acos`, `atan`: Inverse trigonometric functions
- `log`: Natural logarithm
- `log10`: Base-10 logarithm
- `exp`: e^x
- `abs`: Absolute value
- `ceil`, `floor`, `round`: Rounding functions

**Example:**
```json
{
  "function": "power",
  "x": 2,
  "y": 8
}
```

**Response:**
```json
{
  "expression": "2.00^8.00",
  "result": 256,
  "operation": "power"
}
```

### `statistics`
Calculate statistical measures for a dataset.

**Parameters:**
- `numbers` (required): Array of numbers to analyze

**Example:**
```json
{
  "numbers": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
}
```

**Response:**
```json
{
  "numbers": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10],
  "count": 10,
  "sum": 55,
  "mean": 5.5,
  "median": 5.5,
  "min": 1,
  "max": 10,
  "standard_deviation": 2.8722813232690143
}
```

### `convert_units`
Convert between different units of measurement.

**Parameters:**
- `value` (required): The value to convert
- `from_unit` (required): Source unit
- `to_unit` (required): Target unit

**Supported Unit Categories:**

**Temperature:**
- `celsius`, `fahrenheit`, `kelvin`

**Length:**
- `meters`, `feet`, `inches`, `kilometers`, `miles`

**Weight:**
- `kilograms`, `pounds`, `ounces`

**Volume:**
- `liters`, `gallons`, `ounces_fl`

**Example:**
```json
{
  "value": 100,
  "from_unit": "celsius",
  "to_unit": "fahrenheit"
}
```

**Response:**
```json
{
  "original_value": 100,
  "original_unit": "celsius",
  "converted_value": 212,
  "converted_unit": "fahrenheit",
  "conversion_type": "temperature"
}
```

## Configuration for Claude Desktop

Add this to your Claude Desktop configuration:

```json
{
  "mcpServers": {
    "calculator": {
      "command": "/path/to/mcp-calculator-server"
    }
  }
}
```

## Example Usage

Once connected, you can ask Claude:
- "Calculate 15 multiplied by 7"
- "What's the square root of 144?"
- "Convert 100 degrees Celsius to Fahrenheit"
- "Calculate statistics for the numbers 1, 2, 3, 4, 5"
- "What's 2 to the power of 8?"
- "Convert 5 miles to kilometers"

## Error Handling

The server includes comprehensive error handling for:
- Division by zero
- Invalid mathematical operations (e.g., square root of negative numbers)
- Invalid function parameters
- Incompatible unit conversions
- Empty datasets for statistics

## Mathematical Notes

- Trigonometric functions expect input in radians
- Logarithms require positive input values
- Inverse trigonometric functions (asin, acos) require input between -1 and 1
- Unit conversions only work within the same measurement category (e.g., temperature to temperature)