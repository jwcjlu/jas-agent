package loader

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

func normalizeWhitespace(text string) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			lines[i] = ""
			continue
		}
		lines[i] = collapseSpaces(line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func collapseSpaces(text string) string {
	var builder strings.Builder
	builder.Grow(len(text))
	var last rune
	for _, r := range text {
		if unicode.IsSpace(r) {
			if last == ' ' {
				continue
			}
			builder.WriteRune(' ')
			last = ' '
			continue
		}
		builder.WriteRune(r)
		last = r
	}
	return builder.String()
}

func chunkText(text string, size, overlap int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if size <= 0 {
		return []string{text}
	}
	runes := []rune(text)
	if len(runes) <= size {
		return []string{text}
	}
	step := size - overlap
	if step <= 0 {
		step = size
	}
	var chunks []string
	for start := 0; start < len(runes); start += step {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		if end == len(runes) {
			break
		}
	}
	return chunks
}

func buildDocuments(path string, chunks []string, opts Options, extraMeta map[string]string) []Document {
	if len(chunks) == 0 {
		return nil
	}

	// 使用元数据提取器
	extractor := opts.MetadataExtractor
	if extractor == nil {
		extractor = DefaultMetadataExtractor()
	}

	meta := mergeMetadata(baseMetadata(path), opts.Metadata)
	meta = mergeMetadata(meta, extraMeta)

	baseID := shortHash(path + fmt.Sprint(extraMeta))
	docs := make([]Document, 0, len(chunks))
	for idx, chunk := range chunks {
		// 检测chunk类型
		chunkType := detectChunkType(chunk)

		// 提取丰富的元数据
		chunkMeta := extractor.ExtractMetadata(path, chunk, idx, chunkType)

		// 合并基础元数据
		chunkMeta = mergeMetadata(meta, chunkMeta)

		docs = append(docs, Document{
			ID:       fmt.Sprintf("%s-%d", baseID, idx),
			Text:     chunk,
			Metadata: chunkMeta,
		})
	}
	return docs
}

// detectChunkType 检测chunk类型（简化版，避免循环依赖）
func detectChunkType(text string) string {
	if len([]rune(text)) < 100 && strings.Count(text, "\n") < 3 {
		return "title"
	}
	return "paragraph"
}

func baseMetadata(path string) map[string]string {
	ext := strings.ToLower(filepath.Ext(path))
	return map[string]string{
		"source_path": path,
		"source_name": filepath.Base(path),
		"source_ext":  ext,
	}
}

func mergeMetadata(base map[string]string, extra map[string]string) map[string]string {
	if len(extra) == 0 {
		return cloneMetadata(base)
	}
	result := cloneMetadata(base)
	if result == nil {
		result = map[string]string{}
	}
	for k, v := range extra {
		result[k] = v
	}
	return result
}

func cloneMetadata(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func shortHash(text string) string {
	sum := sha1.Sum([]byte(text))
	return hex.EncodeToString(sum[:8])
}
