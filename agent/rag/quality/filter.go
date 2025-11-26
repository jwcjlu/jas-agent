package quality

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"jas-agent/agent/rag/loader"
	"regexp"
	"strings"
	"unicode"
)

// FilterConfig 质量控制配置
type FilterConfig struct {
	// 文档长度过滤
	MinTextLength int // 最小文本长度（字符数）
	MaxTextLength int // 最大文本长度（字符数）
	MinWordCount  int // 最小词数
	// 内容质量过滤
	MinAlphaRatio    float64 // 最小字母比例（过滤纯符号/数字）
	RemoveEmpty      bool    // 移除空文档
	RemoveDuplicates bool    // 移除重复文档
	// 文本清洗
	CleanHTML      bool // 清理 HTML 标签残留
	NormalizeSpace bool // 规范化空白字符
	RemoveNoise    bool // 移除噪音字符
	// 去重
	DedupByHash bool // 基于内容哈希去重
}

// DefaultFilterConfig 返回默认质量控制配置
func DefaultFilterConfig() *FilterConfig {
	return &FilterConfig{
		MinTextLength:    50,    // 最小 50 个字符
		MaxTextLength:    10000, // 最大 10000 个字符
		MinWordCount:     5,     // 至少 5 个词
		MinAlphaRatio:    0.5,   // 至少 50% 是字母
		RemoveEmpty:      true,
		RemoveDuplicates: true,
		CleanHTML:        true,
		NormalizeSpace:   true,
		RemoveNoise:      true,
		DedupByHash:      true,
	}
}

// FilterResult 过滤结果
type FilterResult struct {
	Total        int      `json:"total"`
	Filtered     int      `json:"filtered"`
	Valid        int      `json:"valid"`
	FilteredDocs []string `json:"filtered_docs,omitempty"` // 被过滤的文档 ID 和原因
	Duplicates   int      `json:"duplicates"`
}

// DocumentFilter 文档过滤器
type DocumentFilter struct {
	config *FilterConfig
	seen   map[string]bool // 用于去重
}

// NewDocumentFilter 创建文档过滤器
func NewDocumentFilter(config *FilterConfig) *DocumentFilter {
	if config == nil {
		config = DefaultFilterConfig()
	}
	return &DocumentFilter{
		config: config,
		seen:   make(map[string]bool),
	}
}

// FilterDocuments 过滤文档列表
func (f *DocumentFilter) FilterDocuments(docs []loader.Document) ([]loader.Document, *FilterResult) {
	result := &FilterResult{
		Total:        len(docs),
		FilteredDocs: make([]string, 0),
	}

	validDocs := make([]loader.Document, 0, len(docs))
	seenHashes := make(map[string]bool)

	for _, doc := range docs {
		// 清洗文档
		cleaned := f.cleanDocument(doc)

		// 验证文档质量
		reason := f.validateDocument(cleaned)
		if reason != "" {
			result.Filtered++
			result.FilteredDocs = append(result.FilteredDocs,
				fmt.Sprintf("%s: %s", cleaned.ID, reason))
			continue
		}

		// 去重检查
		if f.config.DedupByHash {
			hash := f.hashDocument(cleaned)
			if seenHashes[hash] {
				result.Filtered++
				result.Duplicates++
				result.FilteredDocs = append(result.FilteredDocs,
					fmt.Sprintf("%s: duplicate", cleaned.ID))
				continue
			}
			seenHashes[hash] = true
		}

		validDocs = append(validDocs, cleaned)
		result.Valid++
	}

	return validDocs, result
}

// cleanDocument 清洗文档内容
func (f *DocumentFilter) cleanDocument(doc loader.Document) loader.Document {
	text := doc.Text

	// 清理 HTML 标签残留
	if f.config.CleanHTML {
		text = cleanHTMLTags(text)
	}

	// 规范化空白字符
	if f.config.NormalizeSpace {
		text = normalizeWhitespace(text)
	}

	// 移除噪音字符
	if f.config.RemoveNoise {
		text = removeNoise(text)
	}

	cleaned := doc
	cleaned.Text = strings.TrimSpace(text)
	return cleaned
}

// validateDocument 验证文档质量
func (f *DocumentFilter) validateDocument(doc loader.Document) string {
	text := doc.Text

	// 检查空文档
	if f.config.RemoveEmpty && strings.TrimSpace(text) == "" {
		return "empty document"
	}

	// 检查最小长度
	if f.config.MinTextLength > 0 && len([]rune(text)) < f.config.MinTextLength {
		return fmt.Sprintf("too short (min %d chars)", f.config.MinTextLength)
	}

	// 检查最大长度
	if f.config.MaxTextLength > 0 && len([]rune(text)) > f.config.MaxTextLength {
		return fmt.Sprintf("too long (max %d chars)", f.config.MaxTextLength)
	}

	// 检查词数
	if f.config.MinWordCount > 0 {
		words := strings.Fields(text)
		if len(words) < f.config.MinWordCount {
			return fmt.Sprintf("too few words (min %d)", f.config.MinWordCount)
		}
	}

	// 检查字母比例
	if f.config.MinAlphaRatio > 0 {
		alphaCount := 0
		totalCount := 0
		for _, r := range text {
			if unicode.IsLetter(r) {
				alphaCount++
			}
			if !unicode.IsSpace(r) && !unicode.IsControl(r) {
				totalCount++
			}
		}
		if totalCount > 0 {
			ratio := float64(alphaCount) / float64(totalCount)
			if ratio < f.config.MinAlphaRatio {
				return fmt.Sprintf("low alpha ratio (%.2f < %.2f)", ratio, f.config.MinAlphaRatio)
			}
		}
	}

	return "" // 文档质量合格
}

// hashDocument 计算文档内容哈希
func (f *DocumentFilter) hashDocument(doc loader.Document) string {
	// 使用文本内容和关键元数据进行哈希
	content := doc.Text
	if doc.Metadata != nil {
		// 排除 chunk_index，因为同一文档的不同 chunk 不应该被认为是重复
		if source, ok := doc.Metadata["source_path"]; ok {
			content += source
		}
	}
	hash := md5.Sum([]byte(content))
	return hex.EncodeToString(hash[:])
}

// cleanHTMLTags 清理 HTML 标签残留
func cleanHTMLTags(text string) string {
	// 移除 HTML 标签
	re := regexp.MustCompile(`<[^>]+>`)
	text = re.ReplaceAllString(text, "")
	// 解码 HTML 实体
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	return text
}

// normalizeWhitespace 规范化空白字符
func normalizeWhitespace(text string) string {
	// 统一换行符
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	// 移除多余空白行
	lines := strings.Split(text, "\n")
	filtered := make([]string, 0, len(lines))
	lastEmpty := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if !lastEmpty {
				filtered = append(filtered, "")
				lastEmpty = true
			}
		} else {
			filtered = append(filtered, line)
			lastEmpty = false
		}
	}
	return strings.Join(filtered, "\n")
}

// removeNoise 移除噪音字符
func removeNoise(text string) string {
	var builder strings.Builder
	for _, r := range text {
		// 保留字母、数字、标点、空白字符
		if unicode.IsLetter(r) || unicode.IsNumber(r) ||
			unicode.IsPunct(r) || unicode.IsSpace(r) ||
			unicode.IsSymbol(r) {
			builder.WriteRune(r)
		} else if unicode.IsControl(r) && r != '\n' && r != '\t' {
			// 移除控制字符（保留换行和制表符）
			continue
		} else {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// ScoreDocument 评分文档质量（0-1）
func ScoreDocument(doc loader.Document) float64 {
	text := doc.Text
	if text == "" {
		return 0.0
	}

	score := 1.0
	runes := []rune(text)

	// 长度评分：适中长度得分更高
	length := len(runes)
	if length < 50 {
		score *= 0.5 // 太短
	} else if length > 5000 {
		score *= 0.8 // 太长
	}

	// 内容多样性评分
	uniqueChars := make(map[rune]bool)
	alphaCount := 0
	totalNonSpace := 0

	for _, r := range runes {
		if !unicode.IsSpace(r) {
			totalNonSpace++
			uniqueChars[r] = true
			if unicode.IsLetter(r) {
				alphaCount++
			}
		}
	}

	// 字母比例
	if totalNonSpace > 0 {
		alphaRatio := float64(alphaCount) / float64(totalNonSpace)
		if alphaRatio < 0.3 {
			score *= 0.5 // 字母太少
		}
	}

	// 字符多样性
	uniqueRatio := float64(len(uniqueChars)) / float64(length)
	if uniqueRatio < 0.1 {
		score *= 0.7 // 字符重复度太高
	}

	// 标点符号比例（适中的标点表示内容完整）
	punctCount := 0
	for _, r := range runes {
		if unicode.IsPunct(r) {
			punctCount++
		}
	}
	punctRatio := float64(punctCount) / float64(length)
	if punctRatio < 0.01 {
		score *= 0.8 // 标点太少，可能是不完整的内容
	} else if punctRatio > 0.2 {
		score *= 0.7 // 标点太多，可能是噪音
	}

	return score
}
