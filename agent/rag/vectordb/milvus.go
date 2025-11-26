package vectordb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"jas-agent/agent/rag/loader"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// MilvusConfig Milvus 连接配置
type MilvusConfig struct {
	Host        string // Milvus 服务器地址
	Port        int    // Milvus 端口
	Username    string // 用户名（可选）
	Password    string // 密码（可选）
	Database    string // 数据库名称（可选）
	Collection  string // 集合名称
	Dimensions  int    // 向量维度
	IndexType   string // 索引类型，如 "FLAT", "IVF_FLAT" 等
	MetricType  string // 距离度量类型，"L2" 或 "IP"
	CreateIfNot bool   // 如果集合不存在则创建
}

// DefaultMilvusConfig 返回默认配置
func DefaultMilvusConfig(collection string, dimensions int) *MilvusConfig {
	return &MilvusConfig{
		Host:        "10.154.20.1",
		Port:        19530,
		Collection:  collection,
		Dimensions:  dimensions,
		IndexType:   "FLAT",
		MetricType:  "L2",
		CreateIfNot: true,
	}
}

// WithAuth 设置认证信息
func (c *MilvusConfig) WithAuth(username, password string) *MilvusConfig {
	c.Username = username
	c.Password = password
	return c
}

// WithDatabase 设置数据库名称
func (c *MilvusConfig) WithDatabase(db string) *MilvusConfig {
	c.Database = db
	return c
}

// milvusStore Milvus 向量存储实现
type milvusStore struct {
	client     client.Client
	config     *MilvusConfig
	collection string
	dimensions int
}

// NewMilvusStore 创建 Milvus 向量存储
func NewMilvusStore(ctx context.Context, config *MilvusConfig) (VectorStore, error) {
	if config == nil {
		return nil, errors.New("milvus config is required")
	}

	// 构建连接地址
	address := fmt.Sprintf("%s:%d", config.Host, config.Port)

	// 创建客户端配置
	cfg := client.Config{
		Address:  address,
		Username: config.Username,
		Password: config.Password,
		DBName:   config.Database,
	}

	// 创建客户端
	milvusClient, err := client.NewClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create milvus client: %w", err)
	}

	// 如果指定了数据库，切换数据库
	if config.Database != "" {
		if err := milvusClient.UsingDatabase(ctx, config.Database); err != nil {
			milvusClient.Close()
			return nil, fmt.Errorf("failed to use database: %w", err)
		}
	}

	store := &milvusStore{
		client:     milvusClient,
		config:     config,
		collection: config.Collection,
		dimensions: config.Dimensions,
	}

	// 检查集合是否存在，如果不存在且 CreateIfNot 为 true，则创建
	exists, err := milvusClient.HasCollection(ctx, config.Collection)
	if err != nil {
		milvusClient.Close()
		return nil, fmt.Errorf("failed to check collection existence: %w", err)
	}

	if !exists {
		if config.CreateIfNot {
			if err := store.createCollection(ctx); err != nil {
				milvusClient.Close()
				return nil, fmt.Errorf("failed to create collection: %w", err)
			}
			// 加载集合到内存
			if err := milvusClient.LoadCollection(ctx, config.Collection, false); err != nil {
				milvusClient.Close()
				return nil, fmt.Errorf("failed to load collection: %w", err)
			}
		} else {
			milvusClient.Close()
			return nil, fmt.Errorf("collection %s does not exist", config.Collection)
		}
	}

	return store, nil
}

// createCollection 创建集合
func (s *milvusStore) createCollection(ctx context.Context) error {
	// 定义字段
	idField := entity.NewField().
		WithName("id").
		WithDataType(entity.FieldTypeVarChar).
		WithMaxLength(65535).
		WithIsPrimaryKey(true).
		WithIsAutoID(false)

	vectorField := entity.NewField().
		WithName("vector").
		WithDataType(entity.FieldTypeFloatVector).
		WithDim(int64(s.dimensions))

	// 元数据字段（JSON 格式存储）
	metadataField := entity.NewField().
		WithName("metadata").
		WithDataType(entity.FieldTypeJSON)

	// 文本内容字段
	textField := entity.NewField().
		WithName("text").
		WithDataType(entity.FieldTypeVarChar).
		WithMaxLength(65535)

	// 创建 schema
	schema := entity.NewSchema().
		WithName(s.collection).
		WithDescription("Vector store for RAG documents").
		WithField(idField).
		WithField(vectorField).
		WithField(metadataField).
		WithField(textField)

	// 创建集合
	err := s.client.CreateCollection(ctx, schema, entity.DefaultShardNumber)
	if err != nil {
		return fmt.Errorf("create collection failed: %w", err)
	}

	// 创建索引
	metricType := entity.L2
	if s.config.MetricType == "IP" {
		metricType = entity.IP
	} else if s.config.MetricType == "COSINE" {
		metricType = entity.COSINE
	}

	var idx entity.Index
	if s.config.IndexType == "FLAT" {
		idx, err = entity.NewIndexFlat(metricType)
		if err != nil {
			return fmt.Errorf("create flat index failed: %w", err)
		}
	} else {
		// 默认使用 FLAT 索引
		idx, err = entity.NewIndexFlat(metricType)
		if err != nil {
			return fmt.Errorf("create index failed: %w", err)
		}
	}

	err = s.client.CreateIndex(ctx, s.collection, "vector", idx, false)
	if err != nil {
		return fmt.Errorf("create index failed: %w", err)
	}

	return nil
}

// Insert 插入向量
func (s *milvusStore) Insert(ctx context.Context, vectors []Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	ids := make([]string, len(vectors))
	vecs := make([][]float32, len(vectors))
	texts := make([]string, len(vectors))
	metadatas := make([][]byte, len(vectors))

	for i, v := range vectors {
		if len(v.Vector) != s.dimensions {
			return fmt.Errorf("vector dimension mismatch at index %d: expected %d, got %d",
				i, s.dimensions, len(v.Vector))
		}

		ids[i] = v.ID

		// 复制向量
		vecCopy := make([]float32, len(v.Vector))
		copy(vecCopy, v.Vector)
		vecs[i] = vecCopy

		// 存储文本内容
		if v.Document != nil {
			texts[i] = v.Document.Text
		}

		// 序列化元数据
		metadata := map[string]interface{}{}
		if v.Metadata != nil {
			for k, v := range v.Metadata {
				metadata[k] = v
			}
		}
		if v.Document != nil && v.Document.Metadata != nil {
			for k, v := range v.Document.Metadata {
				metadata[k] = v
			}
		}
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata at index %d: %w", i, err)
		}
		metadatas[i] = metadataJSON
	}

	// 构建数据列
	columns := []entity.Column{
		entity.NewColumnVarChar("id", ids),
		entity.NewColumnFloatVector("vector", s.dimensions, vecs),
		entity.NewColumnJSONBytes("metadata", metadatas),
		entity.NewColumnVarChar("text", texts),
	}

	// 插入数据
	_, err := s.client.Insert(ctx, s.collection, "", columns...)
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}

	return nil
}

// Search 搜索向量
func (s *milvusStore) Search(ctx context.Context, queryVector []float32, topK int, filter map[string]string) ([]SearchResult, error) {
	if len(queryVector) != s.dimensions {
		return nil, fmt.Errorf("query vector dimension mismatch: expected %d, got %d",
			s.dimensions, len(queryVector))
	}

	// 构建过滤表达式
	var expr string
	if filter != nil && len(filter) > 0 {
		var conditions []string
		for k, v := range filter {
			// 使用 JSON 路径查询元数据字段
			conditions = append(conditions, fmt.Sprintf(`JSON_EXTRACT(metadata, "$.%s") == "%s"`, k, v))
		}
		if len(conditions) > 0 {
			expr = conditions[0]
			for i := 1; i < len(conditions); i++ {
				expr += " && " + conditions[i]
			}
		}
	}

	// 构建搜索向量
	queryVec := entity.FloatVector(queryVector)

	// 距离度量类型
	metricType := entity.L2
	if s.config.MetricType == "IP" {
		metricType = entity.IP
	} else if s.config.MetricType == "COSINE" {
		metricType = entity.COSINE
	}

	// 搜索参数
	sp, _ := entity.NewIndexFlatSearchParam()

	// 执行搜索
	searchResults, err := s.client.Search(
		ctx,
		s.collection,
		[]string{},
		expr,
		[]string{"id", "vector", "metadata", "text"},
		[]entity.Vector{queryVec},
		"vector",
		metricType,
		topK,
		sp,
	)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(searchResults) == 0 {
		return []SearchResult{}, nil
	}

	// 处理搜索结果
	results := make([]SearchResult, 0)
	for _, result := range searchResults {
		if result.Err != nil {
			continue
		}

		// 获取 ID 列
		idCol, ok := result.IDs.(*entity.ColumnVarChar)
		if !ok {
			continue
		}

		for i := 0; i < result.ResultCount && i < idCol.Len(); i++ {
			id, err := idCol.ValueByIdx(i)
			if err != nil {
				continue
			}

			// 获取文本和元数据
			var text string
			var metadata map[string]string
			var vec []float32

			// 遍历输出字段
			for _, field := range result.Fields {
				switch field.Name() {
				case "text":
					if textCol, ok := field.(*entity.ColumnVarChar); ok && i < textCol.Len() {
						text, _ = textCol.ValueByIdx(i)
					}
				case "metadata":
					if metaCol, ok := field.(*entity.ColumnJSONBytes); ok && i < metaCol.Len() {
						metaBytes, _ := metaCol.ValueByIdx(i)
						if len(metaBytes) > 0 {
							json.Unmarshal(metaBytes, &metadata)
						}
					}
				case "vector":
					if vecCol, ok := field.(*entity.ColumnFloatVector); ok {
						// ColumnFloatVector 使用 Data() 方法获取所有数据
						allVecs := vecCol.Data()
						if i < len(allVecs) {
							vecCopy := make([]float32, len(allVecs[i]))
							copy(vecCopy, allVecs[i])
							vec = vecCopy
						}
					}
				}
			}

			// 计算相似度分数（从距离转换）
			var score float64
			if i < len(result.Scores) {
				distance := result.Scores[i]
				score = s.distanceToScore(distance)
			}

			// 构建 Document 对象
			doc := &loader.Document{
				ID:       id,
				Text:     text,
				Metadata: metadata,
			}

			var distance float64
			if i < len(result.Scores) {
				distance = float64(result.Scores[i])
			}

			// 构建完整的 SearchResult，包含所有元数据
			results = append(results, SearchResult{
				Vector: &Vector{
					ID:       id,
					Document: doc,
					Vector:   vec,
					Metadata: metadata,
				},
				Score:    score,
				Distance: distance,
				ID:       id,
				Text:     text,
				Metadata: metadata, // 元数据已从 Milvus 中提取
			})
		}
	}

	return results, nil
}

// distanceToScore 将距离转换为相似度分数
func (s *milvusStore) distanceToScore(distance float32) float64 {
	// L2 距离：越小越相似，转换为 0-1 的分数
	// IP 内积：越大越相似，需要归一化
	// COSINE：已经是 0-1 范围的相似度
	if s.config.MetricType == "L2" {
		// L2 距离转相似度：使用 1 / (1 + distance) 的变换
		return float64(1.0 / (1.0 + distance))
	} else if s.config.MetricType == "IP" {
		// IP 内积，直接使用（假设已经归一化）
		return float64(distance)
	} else if s.config.MetricType == "COSINE" {
		// COSINE 相似度
		return float64(distance)
	}
	return float64(distance)
}

// Delete 删除向量
func (s *milvusStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// 构建删除表达式
	expr := fmt.Sprintf(`id in [%s]`, formatStringList(ids))

	// 执行删除
	err := s.client.Delete(ctx, s.collection, "", expr)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取向量
func (s *milvusStore) GetByID(ctx context.Context, id string) (*Vector, error) {
	results, err := s.BatchGet(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.New("vector not found")
	}
	return results[0], nil
}

// BatchGet 批量获取向量
func (s *milvusStore) BatchGet(ctx context.Context, ids []string) ([]*Vector, error) {
	if len(ids) == 0 {
		return []*Vector{}, nil
	}

	// 构建查询表达式
	expr := fmt.Sprintf(`id in [%s]`, formatStringList(ids))

	// 执行查询
	result, err := s.client.Query(
		ctx,
		s.collection,
		[]string{},
		expr,
		[]string{"id", "vector", "metadata", "text"},
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	vectors := make([]*Vector, 0)

	// 找到 ID 列来确定行数
	var idCol *entity.ColumnVarChar
	var rowCount int
	for _, field := range result {
		if field.Name() == "id" {
			if col, ok := field.(*entity.ColumnVarChar); ok {
				idCol = col
				rowCount = col.Len()
				break
			}
		}
	}

	if idCol == nil {
		return vectors, nil
	}

	// 遍历每一行
	for i := 0; i < rowCount; i++ {
		id, _ := idCol.ValueByIdx(i)

		// 获取其他字段
		var text string
		var metadata map[string]string
		var vec []float32

		for _, field := range result {
			switch field.Name() {
			case "text":
				if textCol, ok := field.(*entity.ColumnVarChar); ok && i < textCol.Len() {
					text, _ = textCol.ValueByIdx(i)
				}
			case "metadata":
				if metaCol, ok := field.(*entity.ColumnJSONBytes); ok && i < metaCol.Len() {
					metaBytes, _ := metaCol.ValueByIdx(i)
					if len(metaBytes) > 0 {
						json.Unmarshal(metaBytes, &metadata)
					}
				}
			case "vector":
				if vecCol, ok := field.(*entity.ColumnFloatVector); ok {
					// ColumnFloatVector 使用 Data() 方法获取所有数据
					allVecs := vecCol.Data()
					if i < len(allVecs) {
						vecCopy := make([]float32, len(allVecs[i]))
						copy(vecCopy, allVecs[i])
						vec = vecCopy
					}
				}
			}
		}

		doc := &loader.Document{
			ID:       id,
			Text:     text,
			Metadata: metadata,
		}

		vectors = append(vectors, &Vector{
			ID:       id,
			Document: doc,
			Vector:   vec,
			Metadata: metadata,
		})
	}

	return vectors, nil
}

// Stats 获取统计信息
func (s *milvusStore) Stats(ctx context.Context) (Stats, error) {
	// 获取集合统计信息
	stats, err := s.client.GetCollectionStatistics(ctx, s.collection)
	if err != nil {
		return Stats{}, fmt.Errorf("get collection stats failed: %w", err)
	}

	// 解析行数（stats 是 map[string]string）
	var rowCount int
	if stats != nil {
		if rc, ok := stats["row_count"]; ok {
			rowCount, _ = strconv.Atoi(rc)
		}
	}

	return Stats{
		Count:      rowCount,
		Dimensions: s.dimensions,
	}, nil
}

// formatStringList 格式化字符串列表为 Milvus 表达式
func formatStringList(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	result := fmt.Sprintf(`"%s"`, ids[0])
	for i := 1; i < len(ids); i++ {
		result += fmt.Sprintf(`, "%s"`, ids[i])
	}
	return result
}

// Close 关闭 Milvus 连接（可选方法，不在接口中）
func (s *milvusStore) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
