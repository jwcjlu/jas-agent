@echo off
REM Windows 版本的 proto 生成脚本

REM 检查 protoc 是否安装
where protoc >nul 2>nul
if %errorlevel% neq 0 (
    echo 错误: protoc 未安装
    echo 请安装 protoc: https://grpc.io/docs/protoc-installation/
    exit /b 1
)

REM 检查 protoc-gen-go 是否安装
where protoc-gen-go >nul 2>nul
if %errorlevel% neq 0 (
    echo 安装 protoc-gen-go...
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
)

REM 检查 protoc-gen-go-grpc 是否安装
where protoc-gen-go-grpc >nul 2>nul
if %errorlevel% neq 0 (
    echo 安装 protoc-gen-go-grpc...
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
)

REM 创建输出目录（与 go_package 对齐）
if not exist api\agent\service\v1 mkdir api\agent\service\v1

REM 检查 protoc-gen-go-http 是否安装
where protoc-gen-go-http >nul 2>nul
if %errorlevel% neq 0 (
    echo 安装 protoc-gen-go-http...
    go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http@latest
)

REM 生成 Go 代码
echo 生成 gRPC/HTTP Go 代码...
protoc -I . -I third_party --go_out=. --go-grpc_out=. --go-http_out=. api/proto/agent_service.proto

echo ✅ gRPC/HTTP 代码生成完成！


