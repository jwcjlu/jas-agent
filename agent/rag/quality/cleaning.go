package quality

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"jas-agent/agent/rag/loader"
)

// CleaningConfig 数据清洗配置
type CleaningConfig struct {
	// 去重配置
	SimilarityThreshold   float64 // 相似度阈值（0-1），超过此阈值视为重复
	UseSemanticSimilarity bool    // 是否使用语义相似度（需要embedder）

	// 噪音过滤
	RemoveNavigation     bool // 移除导航栏文本
	RemoveAds            bool // 移除广告文本
	RemoveHeadersFooters bool // 移除页眉页脚

	// 内容完整性检查
	CheckCompleteness bool // 检查内容完整性
	MinSentenceCount  int  // 最小句子数量
	MinParagraphCount int  // 最小段落数量
}

// DefaultCleaningConfig 返回默认清洗配置
func DefaultCleaningConfig() *CleaningConfig {
	return &CleaningConfig{
		SimilarityThreshold:   0.95,  // 95%相似度视为重复
		UseSemanticSimilarity: false, // 默认使用文本相似度
		RemoveNavigation:      true,
		RemoveAds:             true,
		RemoveHeadersFooters:  true,
		CheckCompleteness:     true,
		MinSentenceCount:      1,
		MinParagraphCount:     0,
	}
}

// DocumentCleaner 文档清洗器
type DocumentCleaner struct {
	config *CleaningConfig
}

// NewDocumentCleaner 创建文档清洗器
func NewDocumentCleaner(config *CleaningConfig) *DocumentCleaner {
	if config == nil {
		config = DefaultCleaningConfig()
	}
	return &DocumentCleaner{
		config: config,
	}
}

// CleanDocument 清洗单个文档
func (c *DocumentCleaner) CleanDocument(doc loader.Document) loader.Document {
	text := doc.Text

	// 移除导航栏和广告
	if c.config.RemoveNavigation {
		text = removeNavigation(text)
	}
	if c.config.RemoveAds {
		text = removeAds(text)
	}
	if c.config.RemoveHeadersFooters {
		text = removeHeadersFooters(text)
	}

	// 移除重复的空白行和多余空格
	text = normalizeTextSpacing(text)

	// 修复常见格式问题
	text = fixFormatting(text)

	cleaned := doc
	cleaned.Text = strings.TrimSpace(text)
	return cleaned
}

// CleanDocuments 批量清洗文档
func (c *DocumentCleaner) CleanDocuments(docs []loader.Document) []loader.Document {
	cleaned := make([]loader.Document, 0, len(docs))
	for _, doc := range docs {
		cleanedDoc := c.CleanDocument(doc)
		if cleanedDoc.Text != "" {
			cleaned = append(cleaned, cleanedDoc)
		}
	}
	return cleaned
}

// DetectNearDuplicates 检测近乎重复的文档
func (c *DocumentCleaner) DetectNearDuplicates(docs []loader.Document) map[string][]string {
	// 按文档ID分组的重复文档列表
	duplicateGroups := make(map[string][]string)

	// 使用文本相似度检测
	seen := make(map[int]bool)
	for i := 0; i < len(docs); i++ {
		if seen[i] {
			continue
		}

		group := []string{docs[i].ID}
		for j := i + 1; j < len(docs); j++ {
			if seen[j] {
				continue
			}

			similarity := textSimilarity(docs[i].Text, docs[j].Text)
			if similarity >= c.config.SimilarityThreshold {
				group = append(group, docs[j].ID)
				seen[j] = true
			}
		}

		if len(group) > 1 {
			duplicateGroups[docs[i].ID] = group
			seen[i] = true
		}
	}

	return duplicateGroups
}

// CheckCompleteness 检查文档完整性
func (c *DocumentCleaner) CheckCompleteness(doc loader.Document) (bool, []string) {
	issues := make([]string, 0)
	text := doc.Text

	// 检查是否为空
	if strings.TrimSpace(text) == "" {
		issues = append(issues, "文档为空")
		return false, issues
	}

	// 检查句子数量
	sentences := countSentences(text)
	if sentences < c.config.MinSentenceCount {
		issues = append(issues, fmt.Sprintf("句子数量不足（%d < %d）", sentences, c.config.MinSentenceCount))
	}

	// 检查段落数量
	paragraphs := countParagraphs(text)
	if paragraphs < c.config.MinParagraphCount {
		issues = append(issues, fmt.Sprintf("段落数量不足（%d < %d）", paragraphs, c.config.MinParagraphCount))
	}

	// 检查是否有明显的截断标记
	if hasTruncationMarkers(text) {
		issues = append(issues, "文档可能被截断")
	}

	// 检查括号/引号是否匹配
	if !areBracketsBalanced(text) {
		issues = append(issues, "括号/引号不匹配，可能内容不完整")
	}

	// 检查是否有大量连续的特殊字符（可能是格式错误）
	if hasExcessiveSpecialChars(text) {
		issues = append(issues, "包含大量特殊字符，可能格式错误")
	}

	isComplete := len(issues) == 0
	return isComplete, issues
}

// textSimilarity 计算文本相似度（使用Jaccard相似度）
func textSimilarity(text1, text2 string) float64 {
	if text1 == text2 {
		return 1.0
	}

	// 将文本转换为字符n-gram集合（n=3）
	ngrams1 := textToNGrams(text1, 3)
	ngrams2 := textToNGrams(text2, 3)

	if len(ngrams1) == 0 && len(ngrams2) == 0 {
		return 1.0
	}
	if len(ngrams1) == 0 || len(ngrams2) == 0 {
		return 0.0
	}

	// 计算交集和并集
	intersection := 0
	union := make(map[string]bool)

	for ngram := range ngrams1 {
		union[ngram] = true
		if ngrams2[ngram] {
			intersection++
		}
	}
	for ngram := range ngrams2 {
		union[ngram] = true
	}

	if len(union) == 0 {
		return 0.0
	}

	return float64(intersection) / float64(len(union))
}

// textToNGrams 将文本转换为n-gram集合
func textToNGrams(text string, n int) map[string]bool {
	ngrams := make(map[string]bool)
	runes := []rune(strings.ToLower(text))

	for i := 0; i <= len(runes)-n; i++ {
		ngram := string(runes[i : i+n])
		ngrams[ngram] = true
	}

	return ngrams
}

// removeNavigation 移除导航栏文本
func removeNavigation(text string) string {
	// 移除常见的导航栏模式
	patterns := []string{
		`(?i)(首页|home|主页|main)[\s\n]*[|•\-\s]*[\s\n]*(关于|about|联系我们|contact)`,
		`(?i)导航|navigation|nav`,
		`(?i)菜单|menu`,
	}

	result := text
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "")
	}

	return result
}

// removeAds 移除广告文本
func removeAds(text string) string {
	// 移除常见的广告模式
	patterns := []string{
		`(?i)(点击|click|立即|now)[\s\n]*(购买|buy|下载|download|注册|register)`,
		`(?i)(广告|advertisement|ad|ads)`,
		`(?i)(限时|limited)[\s\n]*(优惠|discount|特价|sale)`,
	}

	result := text
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "")
	}

	return result
}

// removeHeadersFooters 移除页眉页脚
func removeHeadersFooters(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= 2 {
		return text
	}

	// 移除前两行和后两行（通常是页眉页脚）
	filtered := lines[2:]
	if len(filtered) > 2 {
		filtered = filtered[:len(filtered)-2]
	}

	return strings.Join(filtered, "\n")
}

// normalizeTextSpacing 规范化文本间距
func normalizeTextSpacing(text string) string {
	// 移除多余的空白行（保留最多两个连续换行）
	re := regexp.MustCompile(`\n{3,}`)
	text = re.ReplaceAllString(text, "\n\n")

	// 移除行首行尾的空白
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}

	return strings.Join(lines, "\n")
}

// fixFormatting 修复常见格式问题
func fixFormatting(text string) string {
	// 修复缺少空格的情况（中文和英文/数字之间）
	re := regexp.MustCompile(`([\p{Han}])([A-Za-z0-9])`)
	text = re.ReplaceAllString(text, "$1 $2")
	re = regexp.MustCompile(`([A-Za-z0-9])([\p{Han}])`)
	text = re.ReplaceAllString(text, "$1 $2")

	// 修复多个连续标点符号
	re = regexp.MustCompile(`([。！？])\1+`)
	text = re.ReplaceAllString(text, "$1")

	return text
}

// countSentences 统计句子数量
func countSentences(text string) int {
	// 使用标点符号作为句子分隔符
	sentenceEnders := regexp.MustCompile(`[。！？.!?]`)
	matches := sentenceEnders.FindAllString(text, -1)
	return len(matches)
}

// countParagraphs 统计段落数量
func countParagraphs(text string) int {
	paragraphs := strings.Split(strings.TrimSpace(text), "\n\n")
	count := 0
	for _, para := range paragraphs {
		if strings.TrimSpace(para) != "" {
			count++
		}
	}
	return count
}

// hasTruncationMarkers 检查是否有截断标记
func hasTruncationMarkers(text string) bool {
	truncationPatterns := []string{
		`\.\.\.$`,  // 以...结尾
		`…$`,       // 以…结尾
		`\[未完待续\]`, // [未完待续]
		`\(未完待续\)`, // (未完待续)
		`待续`,
		`to be continued`,
		`TBC`,
	}

	for _, pattern := range truncationPatterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		if re.MatchString(text) {
			return true
		}
	}

	// 检查是否在句子中间截断（最后一个字符不是句子结束符）
	trimmed := strings.TrimSpace(text)
	if len(trimmed) > 0 {
		lastChar := trimmed[len(trimmed)-1]
		lastRune := rune(lastChar)
		if !unicode.IsPunct(lastRune) || (lastRune != '。' && lastRune != '!' && lastRune != '?' && lastRune != '.' && lastRune != '？' && lastRune != '！') {
			// 检查是否可能是截断（文本很长但以非句子结束符结尾）
			if len([]rune(trimmed)) > 500 {
				return true
			}
		}
	}

	return false
}

// areBracketsBalanced 检查括号是否匹配
func areBracketsBalanced(text string) bool {
	pairs := map[rune]rune{
		'(': ')',
		'[': ']',
		'{': '}',
		'"': '"',
		'《': '》',
		'【': '】',
	}

	stack := make([]rune, 0)
	for _, r := range text {
		// 检查是否是开括号
		if closing, isOpen := pairs[r]; isOpen {
			stack = append(stack, closing)
		} else {
			// 检查是否是闭括号
			if len(stack) > 0 && r == stack[len(stack)-1] {
				stack = stack[:len(stack)-1]
			} else {
				// 检查是否是其他闭括号
				for open, close := range pairs {
					if r == close && r != open {
						// 不匹配
						return false
					}
				}
			}
		}
	}

	return len(stack) == 0
}

// hasExcessiveSpecialChars 检查是否有过多的特殊字符
func hasExcessiveSpecialChars(text string) bool {
	specialCharCount := 0
	totalChars := 0

	for _, r := range text {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && !unicode.IsSpace(r) {
			specialCharCount++
		}
		if !unicode.IsSpace(r) {
			totalChars++
		}
	}

	if totalChars == 0 {
		return false
	}

	ratio := float64(specialCharCount) / float64(totalChars)
	return ratio > 0.5 // 如果特殊字符超过50%，可能是格式错误
}
