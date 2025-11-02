@echo off
echo ========================================
echo JAS Agent 前后端联调启动脚本
echo ========================================
echo.

REM 检查参数
if "%1"=="" (
    echo 错误: 请提供 API Key
    echo 用法: start_all.bat YOUR_API_KEY YOUR_BASE_URL
    exit /b 1
)

if "%2"=="" (
    echo 错误: 请提供 Base URL
    echo 用法: start_all.bat YOUR_API_KEY YOUR_BASE_URL
    exit /b 1
)

set API_KEY=%1
set BASE_URL=%2

echo 📋 配置信息:
echo   API Key: %API_KEY%
echo   Base URL: %BASE_URL%
echo.

echo ========================================
echo 第1步: 编译检查
echo ========================================
go build ./...
if errorlevel 1 (
    echo ❌ 编译失败！
    pause
    exit /b 1
)
echo ✅ 编译成功
echo.

echo ========================================
echo 第2步: 启动后端服务器
echo ========================================
echo 正在启动后端服务器...
echo 后端地址: http://localhost:8080
echo API端点: http://localhost:8080/api
echo.
start "JAS Agent 后端" cmd /k "cd cmd\server && go run main.go -apiKey %API_KEY% -baseUrl %BASE_URL%"

REM 等待后端启动
echo 等待后端启动（5秒）...
timeout /t 5 /nobreak >nul

echo ========================================
echo 第3步: 测试 API 端点
echo ========================================
curl http://localhost:8080/api/agents
if errorlevel 1 (
    echo ⚠️ API 可能未就绪，请等待片刻后手动测试
) else (
    echo ✅ API 端点正常
)
echo.

echo ========================================
echo 第4步: 启动 React 前端
echo ========================================
echo 正在启动前端开发服务器...
echo 前端地址: http://localhost:3000
echo.
start "JAS Agent 前端" cmd /k "cd web && npm run dev"

echo.
echo ========================================
echo ✅ 启动完成！
echo ========================================
echo.
echo 📡 服务地址:
echo   后端服务: http://localhost:8080
echo   前端界面: http://localhost:3000
echo   API端点: http://localhost:8080/api
echo.
echo 📖 使用说明:
echo   1. 访问 http://localhost:3000 使用 React 前端
echo   2. 选择 Agent 类型和配置参数
echo   3. 输入问题并发送
echo   4. 查看执行过程和结果
echo.
echo 💡 测试建议:
echo   - 尝试不同的 Agent 类型
echo   - 测试流式和非流式响应
echo   - 查看工具列表
echo   - 测试多轮对话
echo.
echo 📝 查看详细测试指南: scripts\test_integration.md
echo.
echo 按任意键退出此窗口（服务将继续运行）...
pause >nul

