package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"jas-agent/agent/core"
	"net/http"
	"strings"
)

// ESConnection Elasticsearch连接配置
type ESConnection struct {
	Host     string
	Username string
	Password string
	Client   *http.Client
}

// NewESConnection 创建ES连接
func NewESConnection(host, username, password string) *ESConnection {
	return &ESConnection{
		Host:     host,
		Username: username,
		Password: password,
		Client:   &http.Client{},
	}
}

// doRequest 执行HTTP请求
func (conn *ESConnection) doRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	url := fmt.Sprintf("%s%s", conn.Host, path)

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if conn.Username != "" && conn.Password != "" {
		req.SetBasicAuth(conn.Username, conn.Password)
	}

	resp, err := conn.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ES error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ListIndices 列出所有索引
type ListIndices struct {
	conn *ESConnection
}

func NewListIndices(conn *ESConnection) *ListIndices {
	return &ListIndices{conn: conn}
}

func (t *ListIndices) Name() string {
	return "list_indices"
}

func (t *ListIndices) Description() string {
	return "列出Elasticsearch中的所有索引。不需要参数。返回索引名称、文档数量、存储大小等信息。"
}

func (t *ListIndices) Input() any {
	return nil
}

func (t *ListIndices) Type() core.ToolType {
	return core.Normal
}

func (t *ListIndices) Handler(ctx context.Context, input string) (string, error) {
	// 使用 _cat/indices API
	respBody, err := t.conn.doRequest(ctx, "GET", "/_cat/indices?v&format=json", nil)
	if err != nil {
		return "", err
	}

	var indices []map[string]interface{}
	if err := json.Unmarshal(respBody, &indices); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(indices) == 0 {
		return "No indices found", nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d indices:\n\n", len(indices)))

	for _, index := range indices {
		indexName := index["index"]
		docsCount := index["docs.count"]
		storeSize := index["store.size"]
		health := index["health"]

		result.WriteString(fmt.Sprintf("- %s\n", indexName))
		result.WriteString(fmt.Sprintf("  Health: %s, Docs: %v, Size: %v\n", health, docsCount, storeSize))
	}

	return result.String(), nil
}

// GetIndexMapping 获取索引映射
type GetIndexMapping struct {
	conn *ESConnection
}

func NewGetIndexMapping(conn *ESConnection) *GetIndexMapping {
	return &GetIndexMapping{conn: conn}
}

func (t *GetIndexMapping) Name() string {
	return "get_index_mapping"
}

func (t *GetIndexMapping) Description() string {
	return "获取指定索引的映射（mapping）结构，包括字段类型定义。输入：索引名称（支持通配符如logs-*）。返回：索引的详细字段映射。如果索引不存在，会建议使用list_indices查找可用索引。"
}

func (t *GetIndexMapping) Input() any {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"index": map[string]interface{}{
				"type":        "string",
				"description": "索引名称",
			},
		},
		"required": []string{"index"},
	}
}

func (t *GetIndexMapping) Type() core.ToolType {
	return core.Normal
}

func (t *GetIndexMapping) Handler(ctx context.Context, input string) (string, error) {
	indexName := strings.TrimSpace(input)
	if indexName == "" {
		return "", fmt.Errorf("index name is required")
	}

	path := fmt.Sprintf("/%s/_mapping", indexName)
	respBody, err := t.conn.doRequest(ctx, "GET", path, nil)
	if err != nil {
		// 如果索引不存在，返回友好的错误信息
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "index_not_found") {
			return "", fmt.Errorf("索引 '%s' 不存在。建议：先使用 list_indices 工具查看所有可用的索引，然后选择正确的索引名称", indexName)
		}
		return "", err
	}

	// 格式化输出
	var mapping map[string]interface{}
	if err := json.Unmarshal(respBody, &mapping); err != nil {
		return "", fmt.Errorf("failed to parse mapping: %w", err)
	}

	formatted, err := json.MarshalIndent(mapping, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format mapping: %w", err)
	}

	return fmt.Sprintf("Mapping for index '%s':\n%s", indexName, string(formatted)), nil
}

// SearchDocuments 搜索文档
type SearchDocuments struct {
	conn *ESConnection
}

func NewSearchDocuments(conn *ESConnection) *SearchDocuments {
	return &SearchDocuments{conn: conn}
}

func (t *SearchDocuments) Name() string {
	return "search_documents"
}

func (t *SearchDocuments) Description() string {
	return "在指定索引中搜索文档。输入：JSON格式的搜索查询（包含index和query）。返回：匹配的文档列表。"
}

func (t *SearchDocuments) Input() any {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"index": map[string]interface{}{
				"type":        "string",
				"description": "索引名称",
			},
			"query": map[string]interface{}{
				"type":        "object",
				"description": "ES查询DSL",
			},
			"size": map[string]interface{}{
				"type":        "integer",
				"description": "返回结果数量（默认1000）",
			},
		},
		"required": []string{"index", "query"},
	}
}

func (t *SearchDocuments) Type() core.ToolType {
	return core.Normal
}

func (t *SearchDocuments) Handler(ctx context.Context, input string) (string, error) {
	// 解析输入
	var searchReq struct {
		Index string                 `json:"index"`
		Query map[string]interface{} `json:"query"`
		Size  int                    `json:"size"`
	}

	if err := json.Unmarshal([]byte(input), &searchReq); err != nil {
		return "", fmt.Errorf("JSON解析失败: %w\n\n输入内容:\n%s\n\n请确保JSON格式正确，所有括号都已闭合", err, input)
	}

	if searchReq.Index == "" {
		return "", fmt.Errorf("index name is required")
	}

	if searchReq.Size == 0 {
		searchReq.Size = 1000
	}

	// 构建搜索请求
	searchBody := map[string]interface{}{
		"query": searchReq.Query,
		"size":  searchReq.Size,
	}

	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal search body: %w", err)
	}

	path := fmt.Sprintf("/%s/_search", searchReq.Index)
	respBody, err := t.conn.doRequest(ctx, "POST", path, bodyBytes)
	if err != nil {
		// 如果索引不存在，返回友好的错误信息
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "index_not_found") {
			return "", fmt.Errorf("索引 '%s' 不存在。建议：先使用 list_indices 工具查看所有可用的索引，或使用通配符模式如 'logs-*'", searchReq.Index)
		}
		return "", err
	}
	/*var resp searchResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to parse search response: %w", err)
	}

	if len(resp.Hits.Hits) == 0 {
		return "No documents found matching the query", nil
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}*/

	return string(respBody), nil
}

// GetDocument 获取指定文档
type GetDocument struct {
	conn *ESConnection
}

// 解析响应

func NewGetDocument(conn *ESConnection) *GetDocument {
	return &GetDocument{conn: conn}
}

func (t *GetDocument) Name() string {
	return "get_document"
}

func (t *GetDocument) Description() string {
	return "根据ID获取指定文档。输入：JSON格式包含index和id。返回：文档内容。"
}

func (t *GetDocument) Input() any {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"index": map[string]interface{}{
				"type":        "string",
				"description": "索引名称",
			},
			"id": map[string]interface{}{
				"type":        "string",
				"description": "文档ID",
			},
		},
		"required": []string{"index", "id"},
	}
}

func (t *GetDocument) Type() core.ToolType {
	return core.Normal
}

func (t *GetDocument) Handler(ctx context.Context, input string) (string, error) {
	var req struct {
		Index string `json:"index"`
		ID    string `json:"id"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("JSON解析失败: %w\n\n输入内容:\n%s\n\n请确保JSON格式正确，所有括号都已闭合", err, input)
	}

	if req.Index == "" || req.ID == "" {
		return "", fmt.Errorf("index and id are required")
	}

	path := fmt.Sprintf("/%s/_doc/%s", req.Index, req.ID)
	respBody, err := t.conn.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return "", err
	}

	var docResp struct {
		Found  bool                   `json:"found"`
		Source map[string]interface{} `json:"_source"`
	}

	if err := json.Unmarshal(respBody, &docResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !docResp.Found {
		return fmt.Sprintf("Document with ID '%s' not found in index '%s'", req.ID, req.Index), nil
	}

	formatted, _ := json.MarshalIndent(docResp.Source, "", "  ")
	return fmt.Sprintf("Document (ID: %s):\n%s", req.ID, string(formatted)), nil
}

// AggregateData 执行聚合查询
type AggregateData struct {
	conn *ESConnection
}

func NewAggregateData(conn *ESConnection) *AggregateData {
	return &AggregateData{conn: conn}
}

func (t *AggregateData) Name() string {
	return "aggregate_data"
}

func (t *AggregateData) Description() string {
	return "执行聚合查询分析数据。输入：JSON格式包含index、aggs（聚合定义）和可选的query（查询过滤）。例如：过滤ERROR级别的日志再聚合，使用{\"index\":\"logs\",\"query\":{\"term\":{\"L\":\"ERROR\"}},\"aggs\":{...}}。返回：聚合结果。"
}

func (t *AggregateData) Input() any {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"index": map[string]interface{}{
				"type":        "string",
				"description": "索引名称",
			},
			"query": map[string]interface{}{
				"type":        "object",
				"description": "查询过滤条件（可选，用于过滤要聚合的数据）",
			},
			"aggs": map[string]interface{}{
				"type":        "object",
				"description": "聚合定义（ES aggregations DSL）",
			},
		},
		"required": []string{"index", "aggs"},
	}
}

func (t *AggregateData) Type() core.ToolType {
	return core.Normal
}

func (t *AggregateData) Handler(ctx context.Context, input string) (string, error) {
	var aggReq struct {
		Index string                 `json:"index"`
		Query map[string]interface{} `json:"query"` // 支持查询过滤
		Aggs  map[string]interface{} `json:"aggs"`
	}

	if err := json.Unmarshal([]byte(input), &aggReq); err != nil {
		return "", fmt.Errorf("JSON解析失败: %w\n\n输入内容:\n%s\n\n请确保JSON格式正确，所有括号都已闭合", err, input)
	}

	if aggReq.Index == "" {
		return "", fmt.Errorf("index name is required")
	}

	// 构建聚合请求
	aggBody := map[string]interface{}{
		"size": 0, // 不返回文档，只返回聚合结果
	}

	// 如果有查询条件，添加到请求中
	if aggReq.Query != nil && len(aggReq.Query) > 0 {
		aggBody["query"] = aggReq.Query
	}

	aggBody["aggs"] = aggReq.Aggs

	bodyBytes, err := json.Marshal(aggBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal aggregation body: %w", err)
	}

	path := fmt.Sprintf("/%s/_search", aggReq.Index)
	respBody, err := t.conn.doRequest(ctx, "POST", path, bodyBytes)
	if err != nil {
		return "", err
	}

	// 解析响应
	var aggResp map[string]interface{}
	if err := json.Unmarshal(respBody, &aggResp); err != nil {
		return "", fmt.Errorf("failed to parse aggregation response: %w", err)
	}

	// 提取聚合结果
	if aggs, ok := aggResp["aggregations"].(map[string]interface{}); ok {
		formatted, _ := json.MarshalIndent(aggs, "", "  ")
		return fmt.Sprintf("Aggregation results:\n%s", string(formatted)), nil
	}

	return "No aggregation results found", nil
}

// RegisterESTools 注册所有Elasticsearch工具
func RegisterESTools(conn *ESConnection, toolManager *ToolManager) {
	toolManager.RegisterTool(NewListIndices(conn))
	toolManager.RegisterTool(NewGetIndexMapping(conn))
	toolManager.RegisterTool(NewSearchDocuments(conn))
	toolManager.RegisterTool(NewGetDocument(conn))
	toolManager.RegisterTool(NewAggregateData(conn))
	toolManager.RegisterTool(NewSearchIndices(conn)) // 新增：索引模糊搜索
}
