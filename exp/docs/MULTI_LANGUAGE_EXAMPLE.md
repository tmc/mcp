# Multi-Language MCP Example: Weather Service

This example demonstrates how to use the multi-language MCP tools to create a weather service in different programming languages from the same MCP trace.

## Step 1: Capture MCP Trace

First, we capture a trace from an existing weather service:

```bash
# Run a weather service and capture its trace
mcpspy --output=weather_trace.jsonl ./weather-server
```

The trace contains:
- Tool: `get_weather` - Gets current weather for a location
- Tool: `forecast` - Gets weather forecast
- Resource: `weather://locations` - Available weather locations
- Subscription support for weather updates

## Step 2: Analyze the Trace

```bash
# Detect the best target language based on the trace
mcp-lang-detector weather_trace.jsonl

Output:
Detected patterns suggest:
- Python (async operations, data processing)
- TypeScript (web integration, real-time updates)
- Go (performance critical, concurrent operations)
```

## Step 3: Generate Universal AST

```bash
# Convert trace to language-agnostic AST
mcp-trace-ast weather_trace.jsonl > weather_uast.json
```

weather_uast.json:
```json
{
  "type": "MCPImplementation",
  "kind": "server",
  "metadata": {
    "name": "WeatherService",
    "version": "1.0.0"
  },
  "children": [
    {
      "type": "ToolDefinition",
      "name": "get_weather",
      "inputSchema": {
        "type": "object",
        "properties": {
          "location": {"type": "string"},
          "units": {"type": "string", "enum": ["celsius", "fahrenheit"]}
        }
      },
      "implementation": {
        "type": "FunctionBody",
        "async": true,
        "children": [...]
      }
    }
  ]
}
```

## Step 4: Generate Python Implementation

```bash
# Generate Python server
mcp-py-codegen weather_trace.jsonl > weather_server.py
```

weather_server.py:
```python
from typing import Optional, Dict, Any
from dataclasses import dataclass
import asyncio
from mcp import Server, Tool, Resource

@dataclass
class WeatherRequest:
    location: str
    units: str = "celsius"

@dataclass
class WeatherResponse:
    temperature: float
    condition: str
    humidity: float
    location: str

class WeatherServer(Server):
    def __init__(self):
        super().__init__()
        self.register_tool(
            Tool(
                name="get_weather",
                description="Gets current weather for a location",
                input_schema={
                    "type": "object",
                    "properties": {
                        "location": {"type": "string"},
                        "units": {"type": "string"}
                    }
                },
                handler=self.get_weather
            )
        )
        
    async def get_weather(self, request: WeatherRequest) -> WeatherResponse:
        # Implementation from trace patterns
        weather_data = await self.fetch_weather_api(request.location)
        temperature = self.convert_units(weather_data.temp, request.units)
        
        return WeatherResponse(
            temperature=temperature,
            condition=weather_data.condition,
            humidity=weather_data.humidity,
            location=request.location
        )
    
    async def fetch_weather_api(self, location: str) -> Dict[str, Any]:
        # API call implementation
        pass
    
    def convert_units(self, temp: float, units: str) -> float:
        if units == "fahrenheit":
            return (temp * 9/5) + 32
        return temp

if __name__ == "__main__":
    server = WeatherServer()
    asyncio.run(server.run())
```

## Step 5: Generate TypeScript Implementation

```bash
# Generate TypeScript server
mcp-ts-codegen --target=node weather_trace.jsonl > weather_server.ts
```

weather_server.ts:
```typescript
import { Server, Tool, Resource } from '@mcp/server';

interface WeatherRequest {
  location: string;
  units?: 'celsius' | 'fahrenheit';
}

interface WeatherResponse {
  temperature: number;
  condition: string;
  humidity: number;
  location: string;
}

class WeatherServer extends Server {
  constructor() {
    super();
    
    this.registerTool({
      name: 'get_weather',
      description: 'Gets current weather for a location',
      inputSchema: {
        type: 'object',
        properties: {
          location: { type: 'string' },
          units: { type: 'string', enum: ['celsius', 'fahrenheit'] }
        }
      },
      handler: this.getWeather.bind(this)
    });
  }
  
  async getWeather(request: WeatherRequest): Promise<WeatherResponse> {
    const weatherData = await this.fetchWeatherAPI(request.location);
    const temperature = this.convertUnits(
      weatherData.temp,
      request.units || 'celsius'
    );
    
    return {
      temperature,
      condition: weatherData.condition,
      humidity: weatherData.humidity,
      location: request.location
    };
  }
  
  private async fetchWeatherAPI(location: string): Promise<any> {
    // API implementation
  }
  
  private convertUnits(temp: number, units: string): number {
    if (units === 'fahrenheit') {
      return (temp * 9/5) + 32;
    }
    return temp;
  }
}

const server = new WeatherServer();
server.start();
```

## Step 6: Generate Rust Implementation

```bash
# Generate Rust server
mcp-rust-codegen weather_trace.jsonl > src/main.rs
```

src/main.rs:
```rust
use mcp_server::{Server, Tool, ToolHandler, Result};
use serde::{Deserialize, Serialize};
use async_trait::async_trait;

#[derive(Deserialize)]
struct WeatherRequest {
    location: String,
    #[serde(default = "default_units")]
    units: String,
}

#[derive(Serialize)]
struct WeatherResponse {
    temperature: f64,
    condition: String,
    humidity: f64,
    location: String,
}

fn default_units() -> String {
    "celsius".to_string()
}

struct WeatherServer {
    server: Server,
}

impl WeatherServer {
    fn new() -> Self {
        let mut server = Server::new();
        
        server.register_tool(Tool {
            name: "get_weather".to_string(),
            description: "Gets current weather for a location".to_string(),
            input_schema: serde_json::json!({
                "type": "object",
                "properties": {
                    "location": {"type": "string"},
                    "units": {"type": "string"}
                }
            }),
            handler: Box::new(GetWeatherHandler),
        });
        
        Self { server }
    }
    
    async fn run(self) -> Result<()> {
        self.server.start().await
    }
}

struct GetWeatherHandler;

#[async_trait]
impl ToolHandler for GetWeatherHandler {
    async fn handle(&self, input: serde_json::Value) -> Result<serde_json::Value> {
        let request: WeatherRequest = serde_json::from_value(input)?;
        let weather_data = fetch_weather_api(&request.location).await?;
        let temperature = convert_units(weather_data.temp, &request.units);
        
        let response = WeatherResponse {
            temperature,
            condition: weather_data.condition,
            humidity: weather_data.humidity,
            location: request.location,
        };
        
        Ok(serde_json::to_value(response)?)
    }
}

async fn fetch_weather_api(location: &str) -> Result<WeatherData> {
    // API implementation
    todo!()
}

fn convert_units(temp: f64, units: &str) -> f64 {
    if units == "fahrenheit" {
        (temp * 9.0 / 5.0) + 32.0
    } else {
        temp
    }
}

#[tokio::main]
async fn main() -> Result<()> {
    let server = WeatherServer::new();
    server.run().await
}
```

## Step 7: Generate Tests Across Languages

```bash
# Generate cross-language test suite
mcp-polyglot-test --trace=weather_trace.jsonl --languages=python,typescript,rust
```

This generates:
- test_weather_server.py
- weather_server.test.ts
- tests/weather_test.rs

## Step 8: Generate React UI

```bash
# Generate React component for the weather service
mcp-react-generator weather_trace.jsonl > WeatherComponent.tsx
```

WeatherComponent.tsx:
```tsx
import React, { useState } from 'react';
import { useMCPClient } from '@mcp/react';

interface WeatherData {
  temperature: number;
  condition: string;
  humidity: number;
  location: string;
}

export const WeatherComponent: React.FC = () => {
  const mcp = useMCPClient();
  const [location, setLocation] = useState('');
  const [units, setUnits] = useState<'celsius' | 'fahrenheit'>('celsius');
  const [weather, setWeather] = useState<WeatherData | null>(null);
  const [loading, setLoading] = useState(false);
  
  const getWeather = async () => {
    setLoading(true);
    try {
      const result = await mcp.callTool('get_weather', {
        location,
        units
      });
      setWeather(result.content[0]);
    } catch (error) {
      console.error('Failed to get weather:', error);
    } finally {
      setLoading(false);
    }
  };
  
  return (
    <div className="weather-widget">
      <h2>Weather Service</h2>
      <div>
        <input
          type="text"
          value={location}
          onChange={(e) => setLocation(e.target.value)}
          placeholder="Enter location"
        />
        <select value={units} onChange={(e) => setUnits(e.target.value as any)}>
          <option value="celsius">Celsius</option>
          <option value="fahrenheit">Fahrenheit</option>
        </select>
        <button onClick={getWeather} disabled={loading}>
          Get Weather
        </button>
      </div>
      
      {weather && (
        <div className="weather-display">
          <h3>{weather.location}</h3>
          <p>Temperature: {weather.temperature}° {units}</p>
          <p>Condition: {weather.condition}</p>
          <p>Humidity: {weather.humidity}%</p>
        </div>
      )}
    </div>
  );
};
```

## Step 9: Generate Documentation

```bash
# Generate multi-language documentation
mcp-rosetta --from=trace --to=docs weather_trace.jsonl > WEATHER_API.md
```

This creates documentation showing how to use the weather service in each language.

## Step 10: Performance Comparison

```bash
# Compare performance across implementations
mcp-lang-bench compare \
  python:weather_server.py \
  typescript:weather_server.ts \
  rust:target/release/weather_server

Output:
┌─────────────┬──────────┬─────────┬────────────┐
│ Language    │ Latency  │ Memory  │ Throughput │
├─────────────┼──────────┼─────────┼────────────┤
│ Rust        │ 1.2ms    │ 8MB     │ 50k req/s  │
│ Go          │ 2.1ms    │ 15MB    │ 35k req/s  │
│ TypeScript  │ 3.5ms    │ 45MB    │ 20k req/s  │
│ Python      │ 4.8ms    │ 35MB    │ 15k req/s  │
└─────────────┴──────────┴─────────┴────────────┘
```

## Key Benefits

1. **Language Choice**: Developers can work in their preferred language
2. **Consistency**: All implementations follow the same MCP protocol
3. **Testing**: Cross-language testing ensures compatibility
4. **Performance**: Choose the right language for your use case
5. **UI Generation**: Automatic frontend components
6. **Documentation**: Multi-language docs from a single source

## Advanced Features

### Custom Templates
```bash
# Use custom templates for generation
mcp-universal-codegen \
  --template=my-templates/async-server.tmpl \
  --language=python \
  weather_trace.jsonl
```

### Language Bridges
```bash
# Bridge between different implementations
mcp-lang-bridge \
  --frontend=typescript:ui_server.ts \
  --backend=rust:weather_server
```

### Migration Support
```bash
# Migrate from one language to another
mcp-migrate \
  --from=python:old_weather.py \
  --to=rust \
  --trace=weather_trace.jsonl
```

This example demonstrates the power of language-agnostic MCP tooling, enabling teams to work across language boundaries while maintaining protocol compatibility and best practices.