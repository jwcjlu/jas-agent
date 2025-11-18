package algorithm

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// LogCluster 表示一个日志聚类
type LogCluster struct {
	Template  string   // 日志模板（变量部分用 <*> 表示）
	Logs      []string // 属于该聚类的原始日志
	ClusterID int      // 聚类ID
	Count     int      // 该聚类的日志数量
}

// Drain3 简化的 Drain3 日志聚类算法实现
type Drain3 struct {
	clusters      []*LogCluster
	nextClusterID int
	maxDepth      int     // 前缀树最大深度
	simThreshold  float64 // 相似度阈值
}

// NewDrain3 创建新的 Drain3 实例
func NewDrain3(maxDepth int, simThreshold float64) *Drain3 {
	if maxDepth <= 0 {
		maxDepth = 4 // 默认深度
	}
	if simThreshold <= 0 {
		simThreshold = 0.5 // 默认相似度阈值
	}
	return &Drain3{
		clusters:      make([]*LogCluster, 0),
		nextClusterID: 1,
		maxDepth:      maxDepth,
		simThreshold:  simThreshold,
	}
}

// extractTextFromDoc 从文档中提取文本内容（尝试多个常见字段）
func extractTextFromDoc(doc map[string]interface{}) string {
	// 常见的日志字段名
	fields := []string{"message", "msg", "M", "log", "content", "text", "description", "error", "exception"}

	for _, field := range fields {
		if val, ok := doc[field]; ok {
			if str, ok := val.(string); ok && str != "" {
				return str
			}
		}
	}

	// 如果没有找到，尝试将整个文档转换为字符串
	// 但只取前500个字符，避免过长
	docStr := fmt.Sprintf("%v", doc)
	if len(docStr) > 500 {
		docStr = docStr[:500] + "..."
	}
	return docStr
}

// tokenize 将日志消息分词（按空格和特殊字符）
func tokenize(log string) []string {
	// 移除多余空格
	log = strings.TrimSpace(log)
	// 按空格分割
	tokens := strings.Fields(log)
	return tokens
}

// replaceNumbers 将数字替换为通配符
func replaceNumbers(token string) string {
	matched, _ := regexp.MatchString(`^\d+$`, token)
	if matched {
		return "<*>"
	}
	// 匹配IP地址、版本号等
	ipPattern := regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	token = ipPattern.ReplaceAllString(token, "<*>")
	// 匹配十六进制
	hexPattern := regexp.MustCompile(`0x[0-9a-fA-F]+`)
	token = hexPattern.ReplaceAllString(token, "<*>")
	return token
}

// createTemplate 从日志创建模板
func createTemplate(log string) string {
	tokens := tokenize(log)
	templateTokens := make([]string, 0, len(tokens))

	for _, token := range tokens {
		replaced := replaceNumbers(token)
		templateTokens = append(templateTokens, replaced)
	}

	return strings.Join(templateTokens, " ")
}

// calculateSimilarity 计算两个模板的相似度（基于编辑距离的简化版本）
func calculateSimilarity(template1, template2 string) float64 {
	tokens1 := tokenize(template1)
	tokens2 := tokenize(template2)

	if len(tokens1) == 0 && len(tokens2) == 0 {
		return 1.0
	}
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	// 计算相同token的数量
	common := 0
	maxLen := len(tokens1)
	if len(tokens2) > maxLen {
		maxLen = len(tokens2)
	}

	// 简单的匹配：比较相同位置的token
	minLen := len(tokens1)
	if len(tokens2) < minLen {
		minLen = len(tokens2)
	}

	for i := 0; i < minLen; i++ {
		if tokens1[i] == tokens2[i] {
			common++
		}
	}

	// 相似度 = 相同token数 / 最大长度
	return float64(common) / float64(maxLen)
}

// AddLog 添加日志到聚类中
func (d *Drain3) AddLog(log string) *LogCluster {
	if log == "" {
		return nil
	}

	template := createTemplate(log)

	// 查找最相似的聚类
	bestCluster := d.findBestCluster(template)

	if bestCluster != nil {
		// 更新现有聚类
		bestCluster.Logs = append(bestCluster.Logs, log)
		bestCluster.Count++
		// 更新模板（合并两个模板）
		bestCluster.Template = d.mergeTemplates(bestCluster.Template, template)
		return bestCluster
	}

	// 创建新聚类
	newCluster := &LogCluster{
		Template:  template,
		Logs:      []string{log},
		ClusterID: d.nextClusterID,
		Count:     1,
	}
	d.clusters = append(d.clusters, newCluster)
	d.nextClusterID++
	return newCluster
}

// findBestCluster 查找最相似的聚类
func (d *Drain3) findBestCluster(template string) *LogCluster {
	var bestCluster *LogCluster
	bestSimilarity := d.simThreshold

	for _, cluster := range d.clusters {
		similarity := calculateSimilarity(cluster.Template, template)
		if similarity > bestSimilarity {
			bestSimilarity = similarity
			bestCluster = cluster
		}
	}

	return bestCluster
}

// mergeTemplates 合并两个模板（取相同位置的相同token，不同位置用<*>）
func (d *Drain3) mergeTemplates(template1, template2 string) string {
	tokens1 := tokenize(template1)
	tokens2 := tokenize(template2)

	maxLen := len(tokens1)
	if len(tokens2) > maxLen {
		maxLen = len(tokens2)
	}

	merged := make([]string, maxLen)

	for i := 0; i < maxLen; i++ {
		if i < len(tokens1) && i < len(tokens2) {
			if tokens1[i] == tokens2[i] {
				merged[i] = tokens1[i]
			} else {
				merged[i] = "<*>"
			}
		} else if i < len(tokens1) {
			merged[i] = tokens1[i]
		} else {
			merged[i] = tokens2[i]
		}
	}

	return strings.Join(merged, " ")
}

// GetClusters 获取所有聚类（按数量排序）
func (d *Drain3) GetClusters() []*LogCluster {
	// 按数量降序排序
	sort.Slice(d.clusters, func(i, j int) bool {
		return d.clusters[i].Count > d.clusters[j].Count
	})
	return d.clusters
}

// ClusterDocuments 对文档进行聚类
func ClusterDocuments(docs []map[string]interface{}) []*LogCluster {
	drain3 := NewDrain3(4, 0.5)

	// 提取文本并添加到聚类
	for _, doc := range docs {
		text := extractTextFromDoc(doc)
		if text != "" {
			drain3.AddLog(text)
		}
	}

	return drain3.GetClusters()
}
