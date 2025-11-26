package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type jsonLoader struct{}

func init() {
	registerLoader(jsonLoader{})
}

func (jsonLoader) Extensions() []string {
	return []string{".json", ".jsonl"}
}

func (jsonLoader) Load(ctx context.Context, path string, opts Options) ([]Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read json file %s: %w", path, err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	ext := strings.ToLower(path[strings.LastIndex(path, "."):])
	var documents []Document

	if ext == ".jsonl" {
		// JSONL 格式：每行一个 JSON 对象
		lines := strings.Split(string(data), "\n")
		for lineNum, line := range lines {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(line), &obj); err != nil {
				return nil, fmt.Errorf("parse jsonl line %d: %w", lineNum+1, err)
			}
			formatted, err := formatJSONObject(obj)
			if err != nil {
				return nil, fmt.Errorf("format jsonl line %d: %w", lineNum+1, err)
			}
			chunks, err := chunkContent(ctx, formatted, opts)
			if err != nil {
				return nil, fmt.Errorf("chunk text: %w", err)
			}
			meta := map[string]string{
				"source_type": "jsonl",
				"line_number": fmt.Sprintf("%d", lineNum+1),
			}
			documents = append(documents, buildDocuments(path, chunks, opts, meta)...)
		}
	} else {
		// 标准 JSON 格式
		var obj interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			return nil, fmt.Errorf("parse json %s: %w", path, err)
		}
		formatted, err := formatJSONObject(obj)
		if err != nil {
			return nil, fmt.Errorf("format json %s: %w", path, err)
		}
		chunks, err := chunkContent(ctx, formatted, opts)
		if err != nil {
			return nil, fmt.Errorf("chunk text: %w", err)
		}
		meta := map[string]string{
			"source_type": "json",
		}
		documents = append(documents, buildDocuments(path, chunks, opts, meta)...)
	}

	return documents, nil
}

func formatJSONObject(obj interface{}) (string, error) {
	var builder strings.Builder
	if err := formatJSONValue(&builder, obj, 0); err != nil {
		return "", err
	}
	return builder.String(), nil
}

func formatJSONValue(builder *strings.Builder, val interface{}, indent int) error {
	indentStr := strings.Repeat("  ", indent)
	switch v := val.(type) {
	case map[string]interface{}:
		if len(v) == 0 {
			builder.WriteString("{}")
			return nil
		}
		first := true
		for key, value := range v {
			if !first {
				builder.WriteString("\n")
			}
			builder.WriteString(fmt.Sprintf("%s%s: ", indentStr, key))
			if err := formatJSONValue(builder, value, indent+1); err != nil {
				return err
			}
			first = false
		}
	case []interface{}:
		if len(v) == 0 {
			builder.WriteString("[]")
			return nil
		}
		for i, item := range v {
			if i > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(fmt.Sprintf("%s[%d]: ", indentStr, i))
			if err := formatJSONValue(builder, item, indent+1); err != nil {
				return err
			}
		}
	case string:
		builder.WriteString(v)
	case float64:
		builder.WriteString(fmt.Sprintf("%g", v))
	case bool:
		builder.WriteString(fmt.Sprintf("%t", v))
	case nil:
		builder.WriteString("null")
	default:
		builder.WriteString(fmt.Sprintf("%v", v))
	}
	return nil
}
