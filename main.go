package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if version != "dev" {
		cfg.Server.Version = version
	}

	log, err := newLogger(cfg.Logging)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	log.Info().
		Str("server_name", cfg.Server.Name).
		Str("version", cfg.Server.Version).
		Str("log_level", cfg.Logging.Level).
		Str("elasticsearch_url", cfg.Elasticsearch.URL).
		Bool("use_api_key", cfg.Elasticsearch.APIKey != "").
		Bool("use_basic_auth", cfg.Elasticsearch.Username != "").
		Msg("Configuration loaded")

	// Initialize Elasticsearch client and handler
	esHandler, err := newElasticsearchHandler(cfg.Elasticsearch, log)
	if err != nil {
		return fmt.Errorf("failed to initialize Elasticsearch handler: %w", err)
	}

	// Create MCP server
	s := server.NewMCPServer(
		cfg.Server.Name,
		cfg.Server.Version,
		server.WithToolCapabilities(false),
	)

	// Add list_indices tool
	listIndicesTool := mcp.NewTool(
		"list_indices",
		mcp.WithDescription(
			"List all Elasticsearch indices with optional pattern filtering. Returns index names, health status, and document counts.",
		),
		mcp.WithString("pattern",
			mcp.DefaultString("*"),
			mcp.Description("Index pattern filter (e.g., 'logs-*', 'apm-*')"),
		),
	)

	// Add get_index_mappings tool
	getMappingsTool := mcp.NewTool(
		"get_index_mappings",
		mcp.WithDescription(
			"Get field mappings for one or more Elasticsearch indices. Useful for understanding index structure before querying.",
		),
		mcp.WithString("index",
			mcp.Required(),
			mcp.Description("Index name or pattern (e.g., 'logs-*', 'apm-errors-*')"),
		),
	)

	// Add search tool
	searchTool := mcp.NewTool(
		"search",
		mcp.WithDescription(
			"Execute Elasticsearch search queries with aggregations, filtering, and sorting. Returns structured search results.",
		),
		mcp.WithString("index",
			mcp.Required(),
			mcp.Description("Index name or pattern to search"),
		),
		mcp.WithString("query",
			mcp.DefaultString("{}"),
			mcp.Description("Elasticsearch query DSL as JSON string"),
		),
		mcp.WithNumber("size",
			mcp.DefaultNumber(10),
			mcp.Description("Maximum number of documents to return (0-10000)"),
		),
		mcp.WithNumber("from",
			mcp.DefaultNumber(0),
			mcp.Description("Offset from the first result (for pagination)"),
		),
		mcp.WithString("sort",
			mcp.DefaultString(""),
			mcp.Description(
				"Sort specification as JSON string (e.g., '[{\"@timestamp\": {\"order\": \"desc\"}}]')",
			),
		),
		mcp.WithString(
			"aggs",
			mcp.DefaultString(""),
			mcp.Description(
				"Aggregations specification as JSON string (e.g., '{\"avg_price\": {\"avg\": {\"field\": \"price\"}}}')",
			),
		),
		mcp.WithString(
			"_source",
			mcp.DefaultString(""),
			mcp.Description(
				"Source filtering as JSON string (e.g., '[\"field1\", \"field2\"]' or '{\"includes\": [\"field1\"], \"excludes\": [\"field2\"]}')",
			),
		),
		mcp.WithString(
			"highlight",
			mcp.DefaultString(""),
			mcp.Description(
				"Highlight specification as JSON string (e.g., '{\"fields\": {\"title\": {}}}')",
			),
		),
		mcp.WithBoolean("track_total_hits",
			mcp.DefaultBool(true),
			mcp.Description("Whether to track the total number of hits"),
		),
	)

	// Register tool handlers
	s.AddTool(listIndicesTool, esHandler.handleListIndices)
	s.AddTool(getMappingsTool, esHandler.handleGetMappings)
	s.AddTool(searchTool, esHandler.handleSearch)

	log.Info().Msg("MCP Elasticsearch server initialized, serving on stdio")

	if err := server.ServeStdio(s); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
