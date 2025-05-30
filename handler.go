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
	Name         string `json:"index"`
	Health       string `json:"health,omitempty"`
	Status       string `json:"status,omitempty"`
	UUID         string `json:"uuid,omitempty"`
	DocsCount    string `json:"docs.count,omitempty"`
	DocsDeleted  string `json:"docs.deleted,omitempty"`
	StoreSize    string `json:"store.size,omitempty"`
	PrimarySize  string `json:"pri.store.size,omitempty"`
	PrimaryCount string `json:"pri,omitempty"`
	ReplicaCount string `json:"rep,omitempty"`
	CreationDate string `json:"creation.date,omitempty"`
	CreationTime string `json:"creation.date.string,omitempty"`
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
		h.client.Cat.Indices.WithH(
			"index,health,status,uuid,docs.count,docs.deleted,store.size,pri.store.size,pri,rep,creation.date,creation.date.string",
		),
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
			"uuid":          idx.UUID,
			"docs_count":    idx.DocsCount,
			"docs_deleted":  idx.DocsDeleted,
			"store_size":    idx.StoreSize,
			"primary_size":  idx.PrimarySize,
			"primary_count": idx.PrimaryCount,
			"replica_count": idx.ReplicaCount,
			"creation_date": idx.CreationDate,
			"creation_time": idx.CreationTime,
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
	from := request.GetInt("from", 0)
	sortString := request.GetString("sort", "")
	aggsString := request.GetString("aggs", "")
	sourceString := request.GetString("_source", "")
	highlightString := request.GetString("highlight", "")
	trackTotalHits := request.GetBool("track_total_hits", true)
	timeout := request.GetString("timeout", "")

	h.logger.Info().
		Str("index", index).
		Str("query", queryString).
		Int("size", size).
		Int("from", from).
		Str("sort", sortString).
		Str("aggs", aggsString).
		Str("_source", sourceString).
		Str("highlight", highlightString).
		Bool("track_total_hits", trackTotalHits).
		Str("timeout", timeout).
		Msg("Executing search")

	// Validate size and from parameters
	if size < 0 || size > 10000 {
		return mcp.NewToolResultError("Size parameter must be between 0 and 10000"), nil
	}
	if from < 0 {
		return mcp.NewToolResultError("From parameter must be >= 0"), nil
	}
	if from+size > 10000 {
		return mcp.NewToolResultError("from + size must not exceed 10000"), nil
	}

	// Parse query JSON
	var query map[string]any
	if queryString == "{}" || queryString == "" {
		// Default to match_all query if empty
		query = map[string]any{
			"match_all": map[string]any{},
		}
	} else {
		if err := json.Unmarshal([]byte(queryString), &query); err != nil {
			h.logger.Error().Err(err).Str("query", queryString).Msg("Invalid query JSON")
			return mcp.NewToolResultError(fmt.Sprintf("Invalid query JSON: %v", err)), nil
		}

		// Validate that query is not empty after parsing
		if len(query) == 0 {
			query = map[string]any{
				"match_all": map[string]any{},
			}
		}
	}

	// Build search request
	searchRequest := map[string]any{
		"query": query,
		"size":  size,
		"from":  from,
	}

	if trackTotalHits {
		searchRequest["track_total_hits"] = true
	}

	// Add timeout if specified
	if timeout != "" {
		searchRequest["timeout"] = timeout
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

	// Parse and add aggregations if provided
	if aggsString != "" {
		var aggs map[string]any
		if err := json.Unmarshal([]byte(aggsString), &aggs); err != nil {
			h.logger.Error().Err(err).Str("aggs", aggsString).Msg("Invalid aggregations JSON")
			return mcp.NewToolResultError(fmt.Sprintf("Invalid aggregations JSON: %v", err)), nil
		}
		searchRequest["aggs"] = aggs
	}

	// Parse and add _source if provided
	if sourceString != "" {
		var source any
		if err := json.Unmarshal([]byte(sourceString), &source); err != nil {
			h.logger.Error().Err(err).Str("_source", sourceString).Msg("Invalid _source JSON")
			return mcp.NewToolResultError(fmt.Sprintf("Invalid _source JSON: %v", err)), nil
		}
		searchRequest["_source"] = source
	}

	// Parse and add highlight if provided
	if highlightString != "" {
		var highlight map[string]any
		if err := json.Unmarshal([]byte(highlightString), &highlight); err != nil {
			h.logger.Error().
				Err(err).
				Str("highlight", highlightString).
				Msg("Invalid highlight JSON")
			return mcp.NewToolResultError(fmt.Sprintf("Invalid highlight JSON: %v", err)), nil
		}
		searchRequest["highlight"] = highlight
	}

	// Convert to JSON
	searchBody, err := json.Marshal(searchRequest)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to marshal search request")
		return mcp.NewToolResultError("Failed to create search request"), nil
	}

	h.logger.Debug().RawJSON("search_body", searchBody).Msg("Search request body")

	// Build search options
	searchOptions := []func(*elasticsearch.SearchRequest){
		h.client.Search.WithContext(ctx),
		h.client.Search.WithIndex(index),
		h.client.Search.WithBody(strings.NewReader(string(searchBody))),
		h.client.Search.WithTrackTotalHits(trackTotalHits),
		h.client.Search.WithPretty(),
	}

	// Add timeout option if specified
	if timeout != "" {
		searchOptions = append(searchOptions, h.client.Search.WithTimeout(timeout))
	}

	// Execute search
	res, err := h.client.Search(searchOptions...)
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
		"from":       from,
		"size":       size,
	}

	// Add aggregations to response if present
	if searchResponse.Aggregations != nil && len(searchResponse.Aggregations) > 0 {
		response["aggregations"] = searchResponse.Aggregations
		h.logger.Debug().
			Int("agg_count", len(searchResponse.Aggregations)).
			Msg("Aggregations found in response")
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
		Int("agg_count", len(searchResponse.Aggregations)).
		Msg("Search executed successfully")

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
