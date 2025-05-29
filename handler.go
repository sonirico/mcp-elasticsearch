package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog"
)

type ElasticsearchHandler struct {
	client *elasticsearch.Client
	logger zerolog.Logger
}

type IndexInfo struct {
	Name         string `json:"name"`
	Health       string `json:"health,omitempty"`
	Status       string `json:"status,omitempty"`
	DocsCount    string `json:"docs_count,omitempty"`
	StoreSize    string `json:"store_size,omitempty"`
	PrimaryCount string `json:"primary_count,omitempty"`
	ReplicaCount string `json:"replica_count,omitempty"`
}

type SearchResponse struct {
	Took     int            `json:"took"`
	TimedOut bool           `json:"timed_out"`
	Shards   map[string]any `json:"_shards"`
	Hits     struct {
		Total struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		MaxScore *float64         `json:"max_score"`
		Hits     []map[string]any `json:"hits"`
	} `json:"hits"`
	Aggregations map[string]any `json:"aggregations,omitempty"`
}

func newElasticsearchHandler(
	cfg ElasticsearchConfig,
	logger zerolog.Logger,
) (*ElasticsearchHandler, error) {
	log := logger.With().Str("component", "elasticsearch").Logger()

	log.Info().Str("url", cfg.URL).Msg("Creating Elasticsearch client")

	// Configure Elasticsearch client
	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.URL},
	}

	// Set authentication method
	if cfg.APIKey != "" {
		esCfg.APIKey = cfg.APIKey
		log.Info().Msg("Using API key authentication")
	} else if cfg.Username != "" && cfg.Password != "" {
		esCfg.Username = cfg.Username
		esCfg.Password = cfg.Password
		log.Info().Str("username", cfg.Username).Msg("Using basic authentication")
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Elasticsearch client")
		return nil, fmt.Errorf("error creating elasticsearch client: %w", err)
	}

	// Test connection
	log.Debug().Msg("Testing Elasticsearch connection")
	res, err := client.Info()
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to Elasticsearch")
		return nil, fmt.Errorf("error connecting to elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		log.Error().Str("response", res.String()).Msg("Elasticsearch connection error")
		return nil, fmt.Errorf("elasticsearch connection error: %s", res.String())
	}

	log.Info().Msg("Elasticsearch connection successful")

	return &ElasticsearchHandler{
		client: client,
		logger: log,
	}, nil
}

func (h *ElasticsearchHandler) handleListIndices(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	pattern := request.GetString("pattern", "*")

	h.logger.Info().Str("pattern", pattern).Msg("Listing indices")

	// Use _cat/indices API for detailed index information
	res, err := h.client.Cat.Indices(
		h.client.Cat.Indices.WithContext(ctx),
		h.client.Cat.Indices.WithIndex(pattern),
		h.client.Cat.Indices.WithFormat("json"),
		h.client.Cat.Indices.WithH("index,health,status,docs.count,store.size,pri,rep"),
	)
	if err != nil {
		h.logger.Error().Err(err).Str("pattern", pattern).Msg("Failed to list indices")
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list indices: %v", err)), nil
	}
	defer res.Body.Close()

	if res.IsError() {
		h.logger.Error().Str("response", res.String()).Msg("Elasticsearch error listing indices")
		return mcp.NewToolResultError(fmt.Sprintf("Elasticsearch error: %s", res.String())), nil
	}

	var indices []IndexInfo
	if err := json.NewDecoder(res.Body).Decode(&indices); err != nil {
		h.logger.Error().Err(err).Msg("Failed to decode indices response")
		return mcp.NewToolResultError(fmt.Sprintf("Failed to decode response: %v", err)), nil
	}

	// Convert to a more readable format
	var result []map[string]any
	for _, idx := range indices {
		result = append(result, map[string]any{
			"name":          idx.Name,
			"health":        idx.Health,
			"status":        idx.Status,
			"docs_count":    idx.DocsCount,
			"store_size":    idx.StoreSize,
			"primary_count": idx.PrimaryCount,
			"replica_count": idx.ReplicaCount,
		})
	}

	response := map[string]any{
		"total_indices": len(result),
		"pattern":       pattern,
		"indices":       result,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to marshal indices response")
		return mcp.NewToolResultError("Failed to marshal result to JSON"), nil
	}

	h.logger.Info().
		Int("count", len(result)).
		Str("pattern", pattern).
		Msg("Listed indices successfully")
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (h *ElasticsearchHandler) handleGetMappings(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	index, err := request.RequireString("index")
	if err != nil {
		h.logger.Error().Err(err).Msg("Missing index parameter")
		return mcp.NewToolResultError("Missing 'index' parameter"), nil
	}

	h.logger.Info().Str("index", index).Msg("Getting index mappings")

	res, err := h.client.Indices.GetMapping(
		h.client.Indices.GetMapping.WithContext(ctx),
		h.client.Indices.GetMapping.WithIndex(index),
	)
	if err != nil {
		h.logger.Error().Err(err).Str("index", index).Msg("Failed to get mappings")
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get mappings: %v", err)), nil
	}
	defer res.Body.Close()

	if res.IsError() {
		h.logger.Error().Str("response", res.String()).Msg("Elasticsearch error getting mappings")
		return mcp.NewToolResultError(fmt.Sprintf("Elasticsearch error: %s", res.String())), nil
	}

	var mappings map[string]any
	if err := json.NewDecoder(res.Body).Decode(&mappings); err != nil {
		h.logger.Error().Err(err).Msg("Failed to decode mappings response")
		return mcp.NewToolResultError(fmt.Sprintf("Failed to decode response: %v", err)), nil
	}

	response := map[string]any{
		"index":    index,
		"mappings": mappings,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to marshal mappings response")
		return mcp.NewToolResultError("Failed to marshal result to JSON"), nil
	}

	h.logger.Info().Str("index", index).Msg("Retrieved mappings successfully")
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (h *ElasticsearchHandler) handleSearch(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	index, err := request.RequireString("index")
	if err != nil {
		h.logger.Error().Err(err).Msg("Missing index parameter")
		return mcp.NewToolResultError("Missing 'index' parameter"), nil
	}

	queryString := request.GetString("query", "{}")
	size := request.GetInt("size", 10)
	sortString := request.GetString("sort", "")
	trackTotalHits := request.GetBool("track_total_hits", true)

	h.logger.Info().
		Str("index", index).
		Str("query", queryString).
		Int("size", size).
		Str("sort", sortString).
		Bool("track_total_hits", trackTotalHits).
		Msg("Executing search")

	// Validate size parameter
	if size < 0 || size > 10000 {
		return mcp.NewToolResultError("Size parameter must be between 0 and 10000"), nil
	}

	// Parse query JSON
	var query map[string]any
	if err := json.Unmarshal([]byte(queryString), &query); err != nil {
		h.logger.Error().Err(err).Str("query", queryString).Msg("Invalid query JSON")
		return mcp.NewToolResultError(fmt.Sprintf("Invalid query JSON: %v", err)), nil
	}

	// Build search request
	searchRequest := map[string]any{
		"query": query,
		"size":  size,
	}

	if trackTotalHits {
		searchRequest["track_total_hits"] = true
	}

	// Parse and add sort if provided
	if sortString != "" {
		var sort any
		if err := json.Unmarshal([]byte(sortString), &sort); err != nil {
			h.logger.Error().Err(err).Str("sort", sortString).Msg("Invalid sort JSON")
			return mcp.NewToolResultError(fmt.Sprintf("Invalid sort JSON: %v", err)), nil
		}
		searchRequest["sort"] = sort
	}

	// Convert to JSON
	searchBody, err := json.Marshal(searchRequest)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to marshal search request")
		return mcp.NewToolResultError("Failed to create search request"), nil
	}

	h.logger.Debug().RawJSON("search_body", searchBody).Msg("Search request body")

	// Execute search
	res, err := h.client.Search(
		h.client.Search.WithContext(ctx),
		h.client.Search.WithIndex(index),
		h.client.Search.WithBody(strings.NewReader(string(searchBody))),
		h.client.Search.WithTrackTotalHits(trackTotalHits),
		h.client.Search.WithPretty(),
	)
	if err != nil {
		h.logger.Error().Err(err).Str("index", index).Msg("Failed to execute search")
		return mcp.NewToolResultError(fmt.Sprintf("Failed to execute search: %v", err)), nil
	}
	defer res.Body.Close()

	if res.IsError() {
		h.logger.Error().Str("response", res.String()).Msg("Elasticsearch search error")
		return mcp.NewToolResultError(
			fmt.Sprintf("Elasticsearch search error: %s", res.String()),
		), nil
	}

	var searchResponse SearchResponse
	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		h.logger.Error().Err(err).Msg("Failed to decode search response")
		return mcp.NewToolResultError(fmt.Sprintf("Failed to decode response: %v", err)), nil
	}

	// Build response with metadata
	response := map[string]any{
		"index":      index,
		"took":       searchResponse.Took,
		"timed_out":  searchResponse.TimedOut,
		"total_hits": searchResponse.Hits.Total.Value,
		"max_score":  searchResponse.Hits.MaxScore,
		"hits":       searchResponse.Hits.Hits,
		"shards":     searchResponse.Shards,
	}

	if searchResponse.Aggregations != nil && len(searchResponse.Aggregations) > 0 {
		response["aggregations"] = searchResponse.Aggregations
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to marshal search response")
		return mcp.NewToolResultError("Failed to marshal result to JSON"), nil
	}

	h.logger.Info().
		Str("index", index).
		Int("total_hits", searchResponse.Hits.Total.Value).
		Int("returned_hits", len(searchResponse.Hits.Hits)).
		Int("took_ms", searchResponse.Took).
		Msg("Search executed successfully")

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
