# JAS Agent Web - React 前端

这是 JAS Agent 的 React 前端实现，提供了一个现代化、组件化的用户界面。

## 特性

- ✅ React 18 + Hooks
- ✅ 组件化架构
- ✅ 响应式设计
- ✅ WebSocket 实时通信
- ✅ Vite 构建工具
- ✅ 现代化 UI/UX
- ✅ 流式响应支持

## 技术栈

- **React 18**: UI 框架
- **Vite**: 构建工具
- **Axios**: HTTP 客户端
- **WebSocket**: 实时通信
- **CSS3**: 样式

## 快速开始

### 1. 安装依赖

```bash
cd web-react
npm install
```

### 2. 开发模式

```bash
npm run dev
```

访问 `http://localhost:3000`

### 3. 生产构建

```bash
npm run build
```

构建产物将输出到 `../web-react-dist` 目录。

## 项目结构

```
web-react/
├── public/             # 静态资源
├── src/
│   ├── components/     # React 组件
│   │   ├── Header.jsx
│   │   ├── ConfigPanel.jsx
│   │   ├── ChatContainer.jsx
│   │   ├── Message.jsx
│   │   ├── WelcomeMessage.jsx
│   │   ├── InputArea.jsx
│   │   ├── StatusBar.jsx
│   │   └── ToolsModal.jsx
│   ├── services/       # API 服务
│   │   └── api.js
│   ├── App.jsx         # 主应用组件
│   ├── App.css         # 主应用样式
│   ├── main.jsx        # 入口文件
│   └── index.css       # 全局样式
├── index.html          # HTML 模板
├── vite.config.js      # Vite 配置
├── package.json        # 依赖配置
└── README.md           # 本文档
```

## 组件说明

### Header
顶部标题栏，显示应用名称和描述。

### ConfigPanel
配置面板，用于选择 Agent 类型、模型、系统提示词等。

### ChatContainer
对话容器，显示所有对话消息。

### Message
单条消息组件，支持用户、助手、系统、错误等多种消息类型。

### WelcomeMessage
欢迎消息，显示示例问题。

### InputArea
输入区域，用于输入和发送消息。

### StatusBar
状态栏，显示当前状态和执行信息。

### ToolsModal
工具列表模态框，显示所有可用工具。

## API 服务

### api.js

提供以下功能：

- `getAgentTypes()`: 获取可用的 Agent 类型
- `getTools()`: 获取可用的工具列表
- `sendChatMessage(request)`: 发送对话请求（非流式）
- `ChatStreamClient`: WebSocket 流式对话客户端

## 配置

### Vite 配置 (vite.config.js)

```javascript
{
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: '../web-react-dist',
  },
}
```

## 使用方式

### 1. 启动后端服务

```bash
cd cmd/server
go run main.go -apiKey YOUR_API_KEY -baseUrl YOUR_BASE_URL
```

### 2. 启动前端开发服务器

```bash
cd web-react
npm run dev
```

### 3. 访问应用

打开浏览器访问 `http://localhost:3000`

## 功能使用

### 选择 Agent 类型

1. 在配置面板选择 Agent 类型（ReAct、Chain、Plan）
2. 选择模型（GPT-3.5、GPT-4）
3. 配置最大步数
4. （可选）设置系统提示词

### 发送消息

1. 在输入框输入您的问题
2. 点击发送按钮或按 Enter 键
3. 查看 AI 的响应

### 流式响应

- 勾选"启用流式响应"可以实时看到执行过程
- 不勾选则一次性返回完整结果

### 查看工具

点击"查看工具"按钮可以查看所有可用的工具列表。

## 开发

### 添加新组件

1. 在 `src/components/` 创建新组件文件
2. 创建对应的 CSS 文件
3. 在需要的地方导入使用

### 修改样式

- 全局样式：修改 `src/index.css` 或 `src/App.css`
- 组件样式：修改对应组件的 CSS 文件
- CSS 变量定义在 `App.css` 的 `:root` 中

### 调试

浏览器开发者工具：
- Console: 查看日志
- Network: 查看 API 请求
- WebSocket: 查看 WebSocket 连接

## 构建优化

### 生产构建

```bash
npm run build
```

### 预览构建结果

```bash
npm run preview
```

## 部署

### 静态部署

1. 构建生产版本
2. 将 `web-react-dist` 目录部署到静态服务器
3. 配置后端 API 代理

### 与后端集成

后端服务器可以直接提供构建后的静态文件：

```go
// 在 http_gateway.go 中
router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web-react-dist")))
```

## 故障排查

### 开发服务器启动失败

检查端口 3000 是否被占用：
```bash
lsof -i :3000  # macOS/Linux
netstat -ano | findstr :3000  # Windows
```

### API 请求失败

1. 确认后端服务器已启动
2. 检查 Vite 代理配置
3. 查看浏览器 Network 面板

### WebSocket 连接失败

1. 确认后端支持 WebSocket
2. 检查浏览器控制台错误
3. 确认防火墙设置

## 浏览器兼容性

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## 许可证

MIT License

## 更多信息

- [Vite 文档](https://vitejs.dev/)
- [React 文档](https://react.dev/)
- [JAS Agent 主文档](../README.md)

