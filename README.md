# MCP Elasticsearch Server 🔍

A Model Context Protocol (MCP) server that provides Elasticsearch integration capabilities to AI assistants and other MCP clients. This server enables querying Elasticsearch clusters through a standardized interface.

## Features

- **🔐 Multiple Authentication Methods**: Supports both API key and username/password authentication
- **📊 Index Management**: List indices with health status and document counts
- **🗺️ Schema Discovery**: Retrieve field mappings to understand index structure
- **🔍 Advanced Search**: Execute complex Elasticsearch queries with aggregations and sorting
- **📋 Structured Responses**: JSON-formatted output with search metadata
- **⚡ Performance Monitoring**: Query execution time tracking
- **🎯 Context Aware**: Supports search execution with proper context cancellation

## Tools

### list_indices
List all Elasticsearch indices with optional pattern filtering.

**Parameters:**
- `pattern` (string, optional): Index pattern filter (default: "*")

**Returns:**
- Total index count
- Index details (name, health, status, document count, size)

### get_index_mappings
Get field mappings for one or more Elasticsearch indices.

**Parameters:**
- `index` (string, required): Index name or pattern

**Returns:**
- Complete field mappings for the specified indices

### search
Execute Elasticsearch search queries with full DSL support.

**Parameters:**
- `index` (string, required): Index name or pattern to search
- `query` (string, optional): Elasticsearch query DSL as JSON (default: "{}")
- `size` (number, optional): Maximum documents to return (default: 10, max: 10000)
- `sort` (string, optional): Sort specification as JSON
- `track_total_hits` (boolean, optional): Track total hit count (default: true)

**Returns:**
- Search results with hits, aggregations, and metadata

## Configuration

### Environment Variables

#### Elasticsearch Configuration
- `ES_URL`: Elasticsearch cluster URL (required)
- `ES_API_KEY`: API key for authentication (optional)
- `ES_USERNAME`: Username for basic authentication (optional)
- `ES_PASSWORD`: Password for basic authentication (optional)

#### Server Configuration
- `MCP_ES_SERVER_NAME`: Server name (default: "mcp-elasticsearch 🔍")

#### Logging Configuration
- `MCP_ES_LOG_LEVEL`: Log level (debug, info, warn, error, fatal)
- `MCP_ES_LOG_FORMAT`: Log format (json, console)
- `MCP_ES_LOG_OUTPUT`: Log output (stdout, stderr)

### Authentication

You must provide either:
1. **API Key authentication**: Set `ES_API_KEY`
2. **Basic authentication**: Set both `ES_USERNAME` and `ES_PASSWORD`

## Installation

```bash
# Clone and build
git clone <repository>
cd mcp-elasticsearch
go mod download
go build -o bin/mcp-elasticsearch .

# Install to system
sudo install bin/mcp-elasticsearch /usr/local/bin/
```

## Usage

### Direct Execution

```bash
# With API key
ES_URL="https://your-cluster.com" ES_API_KEY="your-api-key" mcp-elasticsearch

# With basic auth
ES_URL="https://your-cluster.com" ES_USERNAME="user" ES_PASSWORD="pass" mcp-elasticsearch

# With custom logging
ES_URL="https://your-cluster.com" ES_API_KEY="key" MCP_ES_LOG_LEVEL=debug mcp-elasticsearch
```

### Integration with Claude Desktop

Add to your Claude configuration:

```json
{
  "mcpServers": {
    "elasticsearch": {
      "command": "mcp-elasticsearch",
      "env": {
        "ES_URL": "https://your-cluster.com",
        "ES_API_KEY": "your-api-key",
        "MCP_ES_LOG_LEVEL": "info"
      }
    }
  }
}
```

### Integration with ApMentor

Update your `config.json`:

```json
{
  "mcpServers": {
    "elasticsearch-go": {
      "command": "/usr/local/bin/mcp-elasticsearch",
      "env": {
        "ES_URL": "https://atani.es.eu-west-1.aws.found.io",
        "ES_API_KEY": "your-api-key",
        "MCP_ES_LOG_LEVEL": "info"
      }
    }
  }
}
```

## Example Queries

### List All Indices
```json
{
  "tool": "list_indices",
  "parameters": {
    "pattern": "*"
  }
}
```

### List Log Indices Only
```json
{
  "tool": "list_indices", 
  "parameters": {
    "pattern": "logs-*"
  }
}
```

### Get Index Mappings
```json
{
  "tool": "get_index_mappings",
  "parameters": {
    "index": "logs-apm.error-*"
  }
}
```

### Simple Search
```json
{
  "tool": "search",
  "parameters": {
    "index": "logs-*",
    "query": "{\"match\": {\"service.name\": \"broker-api-b2b\"}}",
    "size": 50
  }
}
```

### Complex Search with Aggregations
```json
{
  "tool": "search",
  "parameters": {
    "index": "logs-*",
    "query": "{\"bool\": {\"must\": [{\"term\": {\"service.name\": \"broker-api-b2b\"}}, {\"range\": {\"@timestamp\": {\"gte\": \"now-24h\"}}}]}}",
    "size": 0,
    "aggs": "{\"error_types\": {\"terms\": {\"field\": \"error.type.keyword\", \"size\": 10}}}"
  }
}
```

### Search with Sorting
```json
{
  "tool": "search",
  "parameters": {
    "index": "logs-*",
    "query": "{\"match\": {\"log.level\": \"ERROR\"}}",
    "sort": "[{\"@timestamp\": {\"order\": \"desc\"}}]",
    "size": 20
  }
}
```

## Development

```bash
# Install dependencies
go mod download

# Format code
gofmt -w .

# Run tests
go test -v ./...

# Build
go build -o bin/mcp-elasticsearch .

# Run with debug logging
MCP_ES_LOG_LEVEL=debug go run .
```

## Error Handling

The server provides detailed error messages for common issues:

- **Authentication failures**: Check your API key or credentials
- **Index not found**: Verify index names and patterns
- **Query syntax errors**: Validate your Elasticsearch query JSON
- **Connection issues**: Ensure Elasticsearch is accessible

## Security Considerations

- Store API keys and credentials securely
- Use environment variables for sensitive configuration
- Consider network security for Elasticsearch access
- Monitor query patterns and resource usage

## License

MIT License - See LICENSE file for details.
