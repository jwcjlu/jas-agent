package loader

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Option 供外部自定义加载行为
type Option func(*Options)

// Options 控制文档解析及切片行为
type Options struct {
	ChunkSize         int
	ChunkOverlap      int
	Metadata          map[string]string
	MaxFileSize       int64
	ChunkingConfig    *ChunkingConfig    // 分块配置（可选，如果为nil则使用固定大小分块）
	MetadataExtractor *MetadataExtractor // 元数据提取器（可选）
}

func defaultOptions() Options {
	return Options{
		ChunkSize:    800,
		ChunkOverlap: 120,
		Metadata:     map[string]string{},
		MaxFileSize:  32 << 20, // 32MB
	}
}

func WithChunkingConfig(ChunkingConfig *ChunkingConfig) Option {
	return func(o *Options) {
		if ChunkingConfig != nil {
			o.ChunkingConfig = ChunkingConfig
		}
	}
}

// WithChunkSize 设置每个 chunk 的最大字符数
func WithChunkSize(size int) Option {
	return func(o *Options) {
		if size > 0 {
			o.ChunkSize = size
		}
	}
}

// WithChunkOverlap 设置 chunk 之间的重叠字符数
func WithChunkOverlap(overlap int) Option {
	return func(o *Options) {
		if overlap >= 0 {
			o.ChunkOverlap = overlap
		}
	}
}

// WithDefaultMetadata 为所有文档附加默认元数据
func WithDefaultMetadata(metadata map[string]string) Option {
	return func(o *Options) {
		if metadata == nil {
			return
		}
		if o.Metadata == nil {
			o.Metadata = map[string]string{}
		}
		for k, v := range metadata {
			o.Metadata[k] = v
		}
	}
}

// WithMaxFileSize 限制可解析的最大文件体积
func WithMaxFileSize(limit int64) Option {
	return func(o *Options) {
		if limit > 0 {
			o.MaxFileSize = limit
		}
	}
}

// LoadDocuments 递归遍历输入路径并解析为 GraphRAG 文档
func LoadDocuments(ctx context.Context, inputs []string, optFns ...Option) ([]Document, error) {
	if len(inputs) == 0 {
		return nil, errors.New("no input path provided")
	}
	opts := defaultOptions()
	for _, fn := range optFns {
		fn(&opts)
	}

	var documents []Document
	for _, input := range inputs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		matches, err := expandInput(input)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				return nil, fmt.Errorf("stat %s: %w", match, err)
			}
			if info.IsDir() {
				err = filepath.WalkDir(match, func(path string, d fs.DirEntry, walkErr error) error {
					if walkErr != nil {
						return walkErr
					}
					if d.IsDir() {
						return nil
					}
					fileDocs, loadErr := loadFile(ctx, path, infoFromDirEntry(d), opts)
					if loadErr != nil {
						return loadErr
					}
					documents = append(documents, fileDocs...)
					return nil
				})
				if err != nil {
					return nil, err
				}
				continue
			}
			fileDocs, err := loadFile(ctx, match, info, opts)
			if err != nil {
				return nil, err
			}
			documents = append(documents, fileDocs...)
		}
	}
	return documents, nil
}

func loadFile(ctx context.Context, path string, info fs.FileInfo, opts Options) ([]Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if info == nil {
		var err error
		info, err = os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", path, err)
		}
	}
	if opts.MaxFileSize > 0 && info.Size() > opts.MaxFileSize {
		return nil, fmt.Errorf("file %s exceeds max size (%d > %d)", path, info.Size(), opts.MaxFileSize)
	}
	loader := findLoader(path)
	if loader == nil {
		return nil, fmt.Errorf("unsupported file type %s", filepath.Ext(path))
	}
	return loader.Load(ctx, path, opts)
}

func expandInput(input string) ([]string, error) {
	if strings.ContainsAny(input, "*?") {
		return filepath.Glob(input)
	}
	return []string{input}, nil
}

func infoFromDirEntry(entry fs.DirEntry) fs.FileInfo {
	info, err := entry.Info()
	if err != nil {
		return nil
	}
	return info
}

type documentLoader interface {
	Extensions() []string
	Load(ctx context.Context, path string, opts Options) ([]Document, error)
}

var registry []documentLoader

func registerLoader(loader documentLoader) {
	if loader == nil {
		return
	}
	registry = append(registry, loader)
}

func findLoader(path string) documentLoader {
	ext := strings.ToLower(filepath.Ext(path))
	for _, loader := range registry {
		for _, candidate := range loader.Extensions() {
			if ext == candidate {
				return loader
			}
		}
	}
	return nil
}
