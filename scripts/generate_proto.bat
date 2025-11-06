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

REM 创建输出目录
if not exist api\proto mkdir api\proto

REM 生成 Go 代码
echo 生成 gRPC Go 代码...
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/proto/agent_service.proto

echo ✅ gRPC 代码生成完成！


