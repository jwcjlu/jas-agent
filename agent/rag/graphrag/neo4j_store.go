package graphrag

import (
	"context"
	"encoding/json"
	"fmt"
	"jas-agent/agent/rag/loader"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jStore Neo4j 图数据库存储实现
type Neo4jStore struct {
	driver neo4j.DriverWithContext
	dbName string
}

// Neo4jConfig Neo4j 配置
type Neo4jConfig struct {
	URI      string // neo4j://localhost:7687
	Username string
	Password string
	Database string // 数据库名称，默认为 "neo4j"
}

// NewNeo4jStore 创建 Neo4j 存储
func NewNeo4jStore(ctx context.Context, config Neo4jConfig) *Neo4jStore {
	if config.URI == "" {
		config.URI = "neo4j://localhost:7687"
	}
	if config.Database == "" {
		config.Database = "neo4j"
	}

	driver, err := neo4j.NewDriverWithContext(
		config.URI,
		neo4j.BasicAuth(config.Username, config.Password, ""),
	)
	if err != nil {
		panic(fmt.Errorf("create neo4j driver: %w", err))
	}

	// 验证连接
	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		panic(fmt.Errorf("verify neo4j connectivity: %w", err))
	}

	store := &Neo4jStore{
		driver: driver,
		dbName: config.Database,
	}

	// 创建索引
	if err := store.createIndexes(ctx); err != nil {
		driver.Close(ctx)
		panic(fmt.Errorf("create indexes: %w", err))
	}

	return store
}

// createIndexes 创建必要的索引
func (s *Neo4jStore) createIndexes(ctx context.Context) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
	})
	defer session.Close(ctx)

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS FOR (n:Entity) ON (n.id)",
		"CREATE INDEX IF NOT EXISTS FOR (n:Entity) ON (n.name)",
		"CREATE INDEX IF NOT EXISTS FOR ()-[r:RELATED_TO]-() ON (r.relation)",
	}

	for _, index := range indexes {
		if _, err := session.Run(ctx, index, nil); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}

	return nil
}

// Close 关闭连接
func (s *Neo4jStore) Close(ctx context.Context) error {
	return s.driver.Close(ctx)
}

// UpsertNode 创建或更新节点
func (s *Neo4jStore) UpsertNode(ctx context.Context, node *loader.GraphNode) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
	})
	defer session.Close(ctx)

	query := `
		MERGE (n:Entity {id: $id})
		ON CREATE SET 
			n.name = $name,
			n.summary = $summary,
			n.createdAt = datetime(),
			n.updatedAt = datetime(),
			n.occurrence = 1,
			n.metadata = $metadata,
			n.sourceDocs = $sourceDocs,
			n.snippets = $snippets
		ON MATCH SET
			n.name = CASE WHEN size($name) > size(n.name) THEN $name ELSE n.name END,
			n.summary = CASE WHEN n.summary = "" THEN $summary ELSE n.summary END,
			n.updatedAt = datetime(),
			n.occurrence = n.occurrence + 1,
			n.metadata = $metadata,
			n.sourceDocs = $sourceDocs,
			n.snippets = $snippets
		RETURN n
	`

	metadataStr := marshalToJSON(node.Metadata)
	sourceDocsStr := marshalToJSON(node.SourceDocs)

	params := map[string]interface{}{
		"id":         node.ID,
		"name":       node.Name,
		"summary":    node.Summary,
		"metadata":   metadataStr,
		"sourceDocs": sourceDocsStr,
		"snippets":   node.Snippets,
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// UpsertEdge 创建或更新边（关系）
func (s *Neo4jStore) UpsertEdge(ctx context.Context, edge *loader.GraphEdge) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
	})
	defer session.Close(ctx)

	query := `
		MATCH (source:Entity {id: $sourceId})
		MATCH (target:Entity {id: $targetId})
		MERGE (source)-[r:RELATED_TO {relation: $relation}]->(target)
		ON CREATE SET
			r.evidence = $evidence,
			r.weight = $weight,
			r.createdAt = datetime(),
			r.updatedAt = datetime(),
			r.count = 1
		ON MATCH SET
			r.evidence = $evidence,
			r.weight = $weight,
			r.updatedAt = datetime(),
			r.count = r.count + 1
		RETURN r
	`

	params := map[string]interface{}{
		"sourceId": edge.Source,
		"targetId": edge.Target,
		"relation": edge.Relation,
		"evidence": edge.Evidence,
		"weight":   edge.Weight,
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// GetNode 获取节点
func (s *Neo4jStore) GetNode(ctx context.Context, nodeID string) (*loader.GraphNode, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
	})
	defer session.Close(ctx)

	query := `
		MATCH (n:Entity {id: $id})
		RETURN n.id as id, n.name as name, n.summary as summary,
		       n.metadata as metadata, n.sourceDocs as sourceDocs,
		       n.snippets as snippets, n.occurrence as occurrence
		LIMIT 1
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"id": nodeID})
	if err != nil {
		return nil, err
	}

	record, err := result.Single(ctx)
	if err != nil {
		return nil, err
	}

	metadata := convertMap(record.Values[3])
	sourceDocs := convertIntMap(record.Values[4])

	node := &loader.GraphNode{
		ID:         record.Values[0].(string),
		Name:       record.Values[1].(string),
		Summary:    record.Values[2].(string),
		Metadata:   metadata,
		SourceDocs: sourceDocs,
		Snippets:   convertStringSlice(record.Values[5]),
		Occurrence: int(record.Values[6].(int64)),
	}

	return node, nil
}

// ListNodes 列出所有节点
func (s *Neo4jStore) ListNodes(ctx context.Context) ([]*loader.GraphNode, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
	})
	defer session.Close(ctx)

	query := `
		MATCH (n:Entity)
		RETURN n.id as id, n.name as name, n.summary as summary,
		       n.metadata as metadata, n.sourceDocs as sourceDocs,
		       n.snippets as snippets, n.occurrence as occurrence
	`

	result, err := session.Run(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	var nodes []*loader.GraphNode
	for result.Next(ctx) {
		record := result.Record()
		metadata := convertMap(record.Values[3])
		sourceDocs := convertIntMap(record.Values[4])

		node := &loader.GraphNode{
			ID:         record.Values[0].(string),
			Name:       record.Values[1].(string),
			Summary:    record.Values[2].(string),
			Metadata:   metadata,
			SourceDocs: sourceDocs,
			Snippets:   convertStringSlice(record.Values[5]),
			Occurrence: int(record.Values[6].(int64)),
		}
		nodes = append(nodes, node)
	}

	return nodes, result.Err()
}

// ListEdges 列出所有边
func (s *Neo4jStore) ListEdges(ctx context.Context) ([]*loader.GraphEdge, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
	})
	defer session.Close(ctx)

	query := `
		MATCH (source:Entity)-[r:RELATED_TO]->(target:Entity)
		RETURN source.id as source, target.id as target,
		       r.relation as relation, r.evidence as evidence,
		       r.weight as weight
	`

	result, err := session.Run(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	var edges []*loader.GraphEdge
	for result.Next(ctx) {
		record := result.Record()
		edge := &loader.GraphEdge{
			Source:   record.Values[0].(string),
			Target:   record.Values[1].(string),
			Relation: record.Values[2].(string),
			Evidence: record.Values[3].(string),
			Weight:   record.Values[4].(float64),
		}
		edges = append(edges, edge)
	}

	return edges, result.Err()
}

// GetNeighbors 获取节点的邻居
func (s *Neo4jStore) GetNeighbors(ctx context.Context, nodeID string) ([]*loader.GraphEdge, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
	})
	defer session.Close(ctx)

	query := `
		MATCH (source:Entity {id: $id})-[r:RELATED_TO]->(target:Entity)
		RETURN source.id as source, target.id as target,
		       r.relation as relation, r.evidence as evidence,
		       r.weight as weight
		UNION
		MATCH (source:Entity)-[r:RELATED_TO]->(target:Entity {id: $id})
		RETURN source.id as source, target.id as target,
		       r.relation as relation, r.evidence as evidence,
		       r.weight as weight
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"id": nodeID})
	if err != nil {
		return nil, err
	}

	var edges []*loader.GraphEdge
	for result.Next(ctx) {
		record := result.Record()
		edge := &loader.GraphEdge{
			Source:   record.Values[0].(string),
			Target:   record.Values[1].(string),
			Relation: record.Values[2].(string),
			Evidence: record.Values[3].(string),
			Weight:   record.Values[4].(float64),
		}
		edges = append(edges, edge)
	}

	return edges, result.Err()
}

// Clear 清空所有数据
func (s *Neo4jStore) Clear(ctx context.Context) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
	})
	defer session.Close(ctx)

	query := "MATCH (n) DETACH DELETE n"
	_, err := session.Run(ctx, query, nil)
	return err
}

// 辅助函数：转换类型
func marshalToJSON(v interface{}) string {
	if v == nil {
		return ""
	}
	bytes, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func convertMap(v interface{}) map[string]string {
	result := make(map[string]string)
	str, ok := v.(string)
	if !ok || str == "" {
		return result
	}
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		return make(map[string]string)
	}
	return result
}

func convertIntMap(v interface{}) map[string]int {
	result := make(map[string]int)
	str, ok := v.(string)
	if !ok || str == "" {
		return result
	}
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		return make(map[string]int)
	}
	return result
}

func convertStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	slice, ok := v.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}
