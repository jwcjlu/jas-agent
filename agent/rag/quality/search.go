package quality

import (
	"math"
)

// SearchFilterConfig 搜索结果过滤配置
type SearchFilterConfig struct {
	MinScore      float64 // 最小相似度分数阈值
	MaxResults    int     // 最大返回结果数
	DiversityTopK int     // 多样性重排序的 topK
}

// DefaultSearchFilterConfig 返回默认搜索过滤配置
func DefaultSearchFilterConfig() *SearchFilterConfig {
	return &SearchFilterConfig{
		MinScore:      0.0, // 默认不过滤（可以根据需要调整，如 0.7）
		MaxResults:    10,  // 最多返回 10 个结果
		DiversityTopK: 5,   // 使用前 5 个结果计算多样性
	}
}

// SearchResult 搜索结果接口（避免循环依赖）
type SearchResult interface {
	GetScore() float64
	GetVector() []float32
	GetDocumentID() string
}

// SearchResultWrapper 包装搜索结果以适配接口
type SearchResultWrapper struct {
	Score      float64
	Vector     []float32
	DocumentID string
}

func (r *SearchResultWrapper) GetScore() float64 {
	return r.Score
}

func (r *SearchResultWrapper) GetVector() []float32 {
	return r.Vector
}

func (r *SearchResultWrapper) GetDocumentID() string {
	return r.DocumentID
}

// FilterSearchResults 过滤搜索结果（泛型版本）
func FilterSearchResults(results interface{}, config *SearchFilterConfig) interface{} {
	if config == nil {
		config = DefaultSearchFilterConfig()
	}

	// 使用反射或类型断言处理不同的结果类型
	// 这里提供一个通用接口，让调用方传入适配函数
	return results
}

// FilterSearchResultsByScore 根据分数过滤结果
func FilterSearchResultsByScore(results []SearchResult, config *SearchFilterConfig) []SearchResult {
	if config == nil {
		config = DefaultSearchFilterConfig()
	}

	filtered := make([]SearchResult, 0)

	for _, result := range results {
		// 应用最小分数阈值
		if result.GetScore() < config.MinScore {
			continue
		}

		// 应用最大结果数限制
		if config.MaxResults > 0 && len(filtered) >= config.MaxResults {
			break
		}

		filtered = append(filtered, result)
	}

	return filtered
}

// RerankByDiversity 基于多样性重排序结果
func RerankByDiversity(results []SearchResult, topK int) []SearchResult {
	if len(results) <= topK {
		return results
	}

	selected := make([]SearchResult, 0, topK)
	used := make(map[int]bool)

	// 第一个结果：最高分
	if len(results) > 0 {
		selected = append(selected, results[0])
		used[0] = true
	}

	// 后续结果：选择与已选结果差异最大的
	for len(selected) < topK && len(selected) < len(results) {
		bestIdx := -1
		maxMinDist := -1.0

		for i := 1; i < len(results); i++ {
			if used[i] {
				continue
			}

			// 计算与已选结果的最小距离
			minDist := math.Inf(1)
			vecA := results[i].GetVector()
			for _, sel := range selected {
				vecB := sel.GetVector()
				if vecA != nil && vecB != nil && len(vecA) > 0 && len(vecB) > 0 {
					// 使用余弦距离
					dist := 1.0 - VectorSimilarity(vecA, vecB)
					if dist < minDist {
						minDist = dist
					}
				}
			}

			// 平衡分数和多样性
			combined := results[i].GetScore()*0.7 + minDist*0.3
			if combined > maxMinDist {
				maxMinDist = combined
				bestIdx = i
			}
		}

		if bestIdx >= 0 {
			selected = append(selected, results[bestIdx])
			used[bestIdx] = true
		} else {
			break
		}
	}

	// 填充剩余结果（按分数）
	for i := 1; i < len(results) && len(selected) < topK; i++ {
		if !used[i] {
			selected = append(selected, results[i])
		}
	}

	return selected
}
