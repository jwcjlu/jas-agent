#!/bin/bash

# 生成 gRPC 代码脚本

# 检查 protoc 是否安装
if ! command -v protoc &> /dev/null; then
    echo "错误: protoc 未安装"
    echo "请安装 protoc: https://grpc.io/docs/protoc-installation/"
    exit 1
fi

# 检查 protoc-gen-go 是否安装
if ! command -v protoc-gen-go &> /dev/null; then
    echo "安装 protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# 检查 protoc-gen-go-grpc 是否安装
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "安装 protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# 创建输出目录（与 go_package 对齐）
mkdir -p api/agent/service/v1

# 安装 Kratos HTTP 插件（如果缺失）
if ! command -v protoc-gen-go-http &> /dev/null; then
    echo "安装 protoc-gen-go-http..."
    go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http@latest
fi

# 生成 Go 代码
echo "生成 gRPC/HTTP Go 代码..."
protoc -I . -I third_party \
    --go_out=. \
    --go-grpc_out=. \
    --go-http_out=. \
    api/proto/agent_service.proto

echo "✅ gRPC 代码生成完成！"


