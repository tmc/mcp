# MCP Brave Search Server

A Model Context Protocol (MCP) server that provides web search capabilities using Brave's Search API, offering privacy-focused search results.

## Features

- **Web Search**: General web search with comprehensive results
- **News Search**: Dedicated news search with published dates
- **Image Search**: Image search with source URLs
- **Video Search**: Video search with publication information  
- **Privacy-Focused**: Uses Brave's privacy-focused search engine
- **Configurable Results**: Adjustable result counts and safe search settings
- **Geographic Filtering**: Country-specific search results

## Tools

### `web_search`
Perform a general web search using Brave Search.

**Parameters:**
- `query` (required): Search query string
- `type` (optional): Search type ("web", "news", "images", "videos", "all") (default: "web")
- `count` (optional): Number of results to return (1-20, default: 10)
- `country` (optional): Country code for localized results (e.g., "US", "GB", "DE")
- `safesearch` (optional): Safe search level ("off", "moderate", "strict") (default: "moderate")

### `news_search`
Search specifically for news articles.

**Parameters:**
- `query` (required): Search query string
- `count` (optional): Number of results to return (1-20, default: 10)
- `country` (optional): Country code for localized results

### `image_search`
Search for images.

**Parameters:**
- `query` (required): Search query string
- `count` (optional): Number of results to return (1-20, default: 10)
- `safesearch` (optional): Safe search level ("off", "moderate", "strict") (default: "moderate")

### `video_search`
Search for videos.

**Parameters:**
- `query` (required): Search query string
- `count` (optional): Number of results to return (1-20, default: 10)
- `safesearch` (optional): Safe search level ("off", "moderate", "strict") (default: "moderate")

## Setup

### 1. Get Brave Search API Key

1. Visit [Brave Search API](https://api.search.brave.com/)
2. Sign up for an account
3. Subscribe to a plan (Free tier available with limitations)
4. Get your API key from the dashboard

### 2. Set Environment Variable

```bash
export BRAVE_API_KEY=your_api_key_here
```

## Usage

```bash
# Set your Brave Search API key
export BRAVE_API_KEY=your_brave_api_key

# Run the server
./mcp-bravesearch-server
```

## Example Queries

### Basic web search
```json
{
  "name": "web_search",
  "arguments": {
    "query": "artificial intelligence latest developments",
    "count": 15
  }
}
```

### News search
```json
{
  "name": "news_search",
  "arguments": {
    "query": "climate change policy",
    "count": 10,
    "country": "US"
  }
}
```

### Image search with safe search
```json
{
  "name": "image_search",
  "arguments": {
    "query": "mountain landscapes",
    "count": 12,
    "safesearch": "strict"
  }
}
```

### Video search
```json
{
  "name": "video_search",
  "arguments": {
    "query": "programming tutorials python",
    "count": 8,
    "safesearch": "moderate"
  }
}
```

### Comprehensive search with all types
```json
{
  "name": "web_search",
  "arguments": {
    "query": "space exploration 2024",
    "type": "all",
    "count": 20,
    "country": "US",
    "safesearch": "moderate"
  }
}
```

## API Rate Limits

Brave Search API has the following limits:

### Free Tier
- **2,000 queries per month**
- **Rate limit**: 1 query per second

### Paid Tiers
- **Speed**: Various plans with higher rate limits
- **Base**: 2,000 queries per month + additional queries
- **Pro**: 20,000 queries per month + additional queries
- **Premium**: Unlimited queries with volume pricing

## Search Parameters

### Country Codes
Use ISO 3166-1 alpha-2 country codes:
- `US` - United States
- `GB` - United Kingdom  
- `DE` - Germany
- `FR` - France
- `JP` - Japan
- `AU` - Australia
- And more...

### Safe Search Levels
- `off` - No filtering
- `moderate` - Moderate filtering (default)
- `strict` - Strict filtering

### Search Types
- `web` - General web results
- `news` - News articles
- `images` - Image results
- `videos` - Video results
- `all` - Mix of web and news results

## Response Format

Results are formatted as human-readable text with:
- **Title**: Page/article title
- **URL**: Link to the resource
- **Description**: Brief description or snippet
- **Age/Published**: When available, shows content age
- **Type-specific info**: Additional metadata per result type

## Error Handling

The server handles various error conditions:
- **Invalid API key**: Returns authentication error
- **Rate limit exceeded**: Returns rate limit error message
- **Network issues**: Returns connection error
- **Invalid parameters**: Returns parameter validation errors

## Security Considerations

- Store API keys securely using environment variables
- Monitor API usage to avoid unexpected charges
- Use appropriate safe search settings for your use case
- Be aware of query logging policies
- Consider caching results to reduce API calls

## Dependencies

- Standard Go HTTP client
- MCP Go library: `github.com/tmc/mcp`
- No external dependencies for HTTP requests

## Privacy

Brave Search is designed with privacy in mind:
- No user tracking across searches
- No personal data collection
- Anonymous search queries
- Privacy-focused search results