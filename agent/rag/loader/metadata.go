package loader

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MetadataExtractor 元数据提取器
type MetadataExtractor struct {
	extractFileInfo    bool // 提取文件信息
	extractContentInfo bool // 提取内容信息
	extractKeywords    bool // 提取关键词
}

// DefaultMetadataExtractor 返回默认元数据提取器
func DefaultMetadataExtractor() *MetadataExtractor {
	return &MetadataExtractor{
		extractFileInfo:    true,
		extractContentInfo: true,
		extractKeywords:    false, // 默认不提取关键词（需要NLP库）
	}
}

// ExtractMetadata 提取文档元数据
func (e *MetadataExtractor) ExtractMetadata(path string, text string, chunkIndex int, chunkType string) map[string]string {
	meta := make(map[string]string)

	if e.extractFileInfo {
		e.extractFileMetadata(meta, path)
	}

	if e.extractContentInfo {
		e.extractContentMetadata(meta, text, chunkIndex, chunkType)
	}

	if e.extractKeywords {
		// TODO: 实现关键词提取（需要NLP库或LLM）
		// keywords := e.extractKeywords(text)
		// meta["keywords"] = strings.Join(keywords, ",")
	}

	return meta
}

// extractFileMetadata 提取文件元数据
func (e *MetadataExtractor) extractFileMetadata(meta map[string]string, path string) {
	// 基础路径信息
	meta["source_path"] = path
	meta["source_name"] = filepath.Base(path)
	meta["source_ext"] = strings.ToLower(filepath.Ext(path))

	// 提取目录信息（可以作为topic）
	dir := filepath.Dir(path)
	if dir != "." && dir != "/" {
		meta["source_dir"] = dir
		// 使用目录名作为主题
		dirName := filepath.Base(dir)
		if dirName != "" {
			meta["topic"] = dirName
		}
	}

	// 提取文件修改时间
	if info, err := os.Stat(path); err == nil {
		modTime := info.ModTime()
		meta["last_updated"] = modTime.Format(time.RFC3339)
		meta["file_size"] = formatFileSize(info.Size())
	}

	// 尝试从文件名提取信息
	e.extractFromFilename(meta, filepath.Base(path))
}

// extractFromFilename 从文件名提取信息
func (e *MetadataExtractor) extractFromFilename(meta map[string]string, filename string) {
	// 移除扩展名
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// 尝试提取日期（如：2024-01-01_document.txt）
	if parts := strings.Split(name, "_"); len(parts) > 1 {
		firstPart := parts[0]
		if len(firstPart) == 10 && strings.Count(firstPart, "-") == 2 {
			// 可能是日期格式 YYYY-MM-DD
			if _, err := time.Parse("2006-01-02", firstPart); err == nil {
				meta["date"] = firstPart
			}
		}
	}

	// 尝试提取作者（如：author_document.txt）
	if strings.Contains(name, "_") {
		parts := strings.Split(name, "_")
		if len(parts) > 1 {
			// 第一部分可能是作者或日期
			firstPart := parts[0]
			if len(firstPart) < 50 && !strings.Contains(firstPart, "-") {
				meta["author"] = firstPart
			}
		}
	}
}

// extractContentMetadata 提取内容元数据
func (e *MetadataExtractor) extractContentMetadata(meta map[string]string, text string, chunkIndex int, chunkType string) {
	// Chunk信息
	meta["chunk_index"] = formatInt(chunkIndex)
	meta["chunk_type"] = chunkType
	meta["chunk_size"] = formatInt(len([]rune(text)))
	meta["word_count"] = formatInt(countWords(text))

	// 内容类型检测
	meta["content_type"] = detectContentType(text)

	// 段落和句子数量
	meta["paragraph_count"] = formatInt(countParagraphs(text))
	meta["sentence_count"] = formatInt(countSentences(text))

	// 语言检测（简单检测）
	meta["language"] = detectLanguage(text)
}

// detectContentType 检测内容类型
func detectContentType(text string) string {
	text = strings.ToLower(text)

	// 检查是否是代码
	if strings.Contains(text, "```") || strings.Contains(text, "```") {
		return "code"
	}

	// 检查是否是列表
	lines := strings.Split(text, "\n")
	listCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") ||
			strings.HasPrefix(line, "•") || strings.HasPrefix(line, "1.") {
			listCount++
		}
	}
	if listCount > len(lines)/2 {
		return "list"
	}

	// 检查是否是表格
	if strings.Count(text, "|") > strings.Count(text, "\n") {
		return "table"
	}

	// 检查是否是标题（短文本）
	if len([]rune(text)) < 100 && strings.Count(text, "\n") < 3 {
		return "title"
	}

	return "paragraph"
}

// detectLanguage 简单语言检测
func detectLanguage(text string) string {
	// 简单的中英文检测
	chineseCount := 0
	englishCount := 0

	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseCount++
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			englishCount++
		}
	}

	total := chineseCount + englishCount
	if total == 0 {
		return "unknown"
	}

	if float64(chineseCount)/float64(total) > 0.3 {
		return "zh"
	}

	return "en"
}

// countWords 统计词数
func countWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

// countParagraphs 统计段落数
func countParagraphs(text string) int {
	paragraphs := strings.Split(strings.TrimSpace(text), "\n\n")
	count := 0
	for _, para := range paragraphs {
		if strings.TrimSpace(para) != "" {
			count++
		}
	}
	if count == 0 {
		count = 1 // 至少一个段落
	}
	return count
}

// countSentences 统计句子数
func countSentences(text string) int {
	sentenceEnders := []rune{'。', '！', '？', '.', '!', '?'}
	count := 0
	for _, r := range text {
		for _, ender := range sentenceEnders {
			if r == ender {
				count++
				break
			}
		}
	}
	if count == 0 {
		count = 1 // 至少一个句子
	}
	return count
}

// formatInt 格式化整数为字符串
func formatInt(n int) string {
	return strconv.Itoa(n)
}

// formatFileSize 格式化文件大小
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return formatInt(int(size)) + "B"
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return formatInt(int(float64(size)/float64(div))) + []string{"KB", "MB", "GB", "TB"}[exp]
}
