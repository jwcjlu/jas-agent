package graphrag

import (
	"context"
	"encoding/json"
	"fmt"
	"jas-agent/agent/core"
	"jas-agent/agent/llm"
	"jas-agent/agent/rag/loader"
	"strings"
)

// EntityRelation 实体和关系结构
type EntityRelation struct {
	Entities  []EntityInfo   `json:"entities"`
	Relations []RelationInfo `json:"relations"`
}

// EntityInfo 实体信息
type EntityInfo struct {
	Name     string            `json:"name"`
	Type     string            `json:"type"` // 实体类型，如：Person, Organization, Location, Concept 等
	Summary  string            `json:"summary"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RelationInfo 关系信息
type RelationInfo struct {
	Source   string `json:"source"`   // 源实体名称
	Target   string `json:"target"`   // 目标实体名称
	Relation string `json:"relation"` // 关系类型
	Evidence string `json:"evidence"` // 证据文本
}

// LLMExtractor 使用大模型提取实体和关系
type LLMExtractor struct {
	chat  llm.Chat
	model string
}

// NewLLMExtractor 创建 LLM 提取器
func NewLLMExtractor(chat llm.Chat, model string) *LLMExtractor {
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	return &LLMExtractor{
		chat:  chat,
		model: model,
	}
}

// ExtractEntitiesAndRelations 从文本中提取实体和关系
func (e *LLMExtractor) ExtractEntitiesAndRelations(ctx context.Context, text string) (*EntityRelation, error) {
	// 构建提示词
	prompt := e.buildExtractionPrompt(text)

	// 调用 LLM
	messages := []core.Message{
		{
			Role:    "system",
			Content: "你是一个专业的知识图谱构建助手。你的任务是从文本中提取实体和关系，并以 JSON 格式返回。",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	req := llm.NewChatRequest(e.model, messages)
	resp, err := e.chat.Completions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("llm completion error: %w", err)
	}

	content := resp.Content()
	if content == "" {
		return nil, fmt.Errorf("empty response from llm")
	}

	// 解析 JSON 响应
	var result EntityRelation

	// 尝试提取 JSON（可能包含 markdown 代码块）
	jsonStr := extractJSON(content)
	if jsonStr == "" {
		jsonStr = content
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse llm response as json: %w, content: %s", err, content)
	}

	return &result, nil
}

// buildExtractionPrompt 构建提取提示词
func (e *LLMExtractor) buildExtractionPrompt(text string) string {
	return fmt.Sprintf(`请从以下文本中提取实体和关系，并以 JSON 格式返回。

要求：
1. 提取所有重要的实体（人物、组织、地点、概念、事件等）
2. 提取实体之间的关系
3. 为每个实体提供简要描述
4. 为每个关系提供证据文本（原文中的相关句子）

文本内容：
%s

请以以下 JSON 格式返回：
{
  "entities": [
    {
      "name": "实体名称",
      "type": "实体类型（Person/Organization/Location/Concept/Event等）",
      "summary": "实体简要描述",
      "metadata": {}
    }
  ],
  "relations": [
    {
      "source": "源实体名称",
      "target": "目标实体名称",
      "relation": "关系类型（如：属于、位于、影响、导致等）",
      "evidence": "证据文本（原文中的相关句子）"
    }
  ]
}

只返回 JSON，不要包含其他文字说明。`, text)
}

// extractJSON 从文本中提取 JSON（处理 markdown 代码块）
func extractJSON(text string) string {
	// 移除 markdown 代码块标记
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}
	return text
}

// IngestDocumentWithLLM 使用 LLM 提取实体和关系并存储到 Neo4j
func (e *LLMExtractor) IngestDocumentWithLLM(ctx context.Context, doc *loader.Document, neo4jStore *Neo4jStore) (int, int, error) {
	// 将文档分割成句子或段落
	sentences := splitSentences(doc.Text)

	var totalNodes, totalEdges int
	seenNodes := make(map[string]*loader.GraphNode)

	// 批量处理句子（每批处理多个句子以提高效率）
	batchSize := 5
	for i := 0; i < len(sentences); i += batchSize {
		end := i + batchSize
		if end > len(sentences) {
			end = len(sentences)
		}

		batch := sentences[i:end]
		batchText := strings.Join(batch, " ")

		// 使用 LLM 提取实体和关系
		result, err := e.ExtractEntitiesAndRelations(ctx, batchText)
		if err != nil {
			// 如果提取失败，跳过这一批，继续处理下一批
			fmt.Printf("Failed to extract from batch %d-%d: %v\n", i, end, err)
			continue
		}

		// 处理实体
		for _, entityInfo := range result.Entities {
			nodeID := normalizeEntity(entityInfo.Name)
			if nodeID == "" {
				continue
			}

			// 检查是否已存在
			node, exists := seenNodes[nodeID]
			if !exists {
				node = &loader.GraphNode{
					ID:         nodeID,
					Name:       entityInfo.Name,
					Summary:    entityInfo.Summary,
					Metadata:   entityInfo.Metadata,
					SourceDocs: map[string]int{doc.ID: 1},
					Snippets:   []string{},
					Occurrence: 1,
				}
				if node.Metadata == nil {
					node.Metadata = make(map[string]string)
				}
				node.Metadata["type"] = entityInfo.Type
				seenNodes[nodeID] = node
				totalNodes++
			} else {
				node.Occurrence++
				node.SourceDocs[doc.ID]++
			}

			// 添加摘要和片段
			if node.Summary == "" {
				node.Summary = entityInfo.Summary
			}
			if len(node.Snippets) < 10 {
				node.Snippets = appendUniqueSnippet(node.Snippets, batchText)
			}

			// 合并元数据
			for k, v := range doc.Metadata {
				if _, exists := node.Metadata[k]; !exists {
					node.Metadata[k] = v
				}
			}

			// 存储到 Neo4j
			if err = neo4jStore.UpsertNode(ctx, node); err != nil {
				return totalNodes, totalEdges, fmt.Errorf("upsert node to neo4j: %w", err)
			}
		}

		// 处理关系
		for _, relationInfo := range result.Relations {
			sourceID := normalizeEntity(relationInfo.Source)
			targetID := normalizeEntity(relationInfo.Target)

			if sourceID == "" || targetID == "" || sourceID == targetID {
				continue
			}

			// 确保源和目标节点存在
			if _, exists := seenNodes[sourceID]; !exists {
				sourceNode := &loader.GraphNode{
					ID:         sourceID,
					Name:       relationInfo.Source,
					Metadata:   make(map[string]string),
					SourceDocs: map[string]int{doc.ID: 1},
					Occurrence: 1,
				}
				seenNodes[sourceID] = sourceNode
				if err := neo4jStore.UpsertNode(ctx, sourceNode); err != nil {
					return totalNodes, totalEdges, fmt.Errorf("upsert source node: %w", err)
				}
			}

			if _, exists := seenNodes[targetID]; !exists {
				targetNode := &loader.GraphNode{
					ID:         targetID,
					Name:       relationInfo.Target,
					Metadata:   make(map[string]string),
					SourceDocs: map[string]int{doc.ID: 1},
					Occurrence: 1,
				}
				seenNodes[targetID] = targetNode
				if err := neo4jStore.UpsertNode(ctx, targetNode); err != nil {
					return totalNodes, totalEdges, fmt.Errorf("upsert target node: %w", err)
				}
			}

			// 创建边
			edge := &loader.GraphEdge{
				Source:   sourceID,
				Target:   targetID,
				Relation: relationInfo.Relation,
				Evidence: relationInfo.Evidence,
				Weight:   computeEdgeWeight(relationInfo.Evidence),
			}

			// 存储到 Neo4j
			if err := neo4jStore.UpsertEdge(ctx, edge); err != nil {
				return totalNodes, totalEdges, fmt.Errorf("upsert edge to neo4j: %w", err)
			}

			totalEdges++
		}
	}

	return totalNodes, totalEdges, nil
}
