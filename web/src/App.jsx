import { useState, useEffect } from 'react';
import Header from './components/Header';
import ConfigPanel from './components/ConfigPanel';
import ChatContainer from './components/ChatContainer';
import InputArea from './components/InputArea';
import StatusBar from './components/StatusBar';
import ToolsModal from './components/ToolsModal';
import { sendChatMessage, ChatStreamClient, getAgentTypes } from './services/api';
import './App.css';

function App() {
  // 状态管理
  const [messages, setMessages] = useState([]);
  const [config, setConfig] = useState({
    agentType: 'react',
    model: 'gpt-3.5-turbo',
    maxSteps: 10,
    systemPrompt: '',
    streamMode: true,
  });
  const [isProcessing, setIsProcessing] = useState(false);
  const [status, setStatus] = useState({ text: '就绪', details: '' });
  const [showToolsModal, setShowToolsModal] = useState(false);
  const [sessionId] = useState(() => `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`);
  const [agentTypes, setAgentTypes] = useState([]);

  // 加载 Agent 类型
  useEffect(() => {
    loadAgentTypes();
  }, []);

  const loadAgentTypes = async () => {
    try {
      const types = await getAgentTypes();
      setAgentTypes(types.filter(t => t.available));
      setStatus({ text: '就绪', details: `${types.length} 个代理可用` });
    } catch (error) {
      console.error('加载代理列表失败:', error);
      setStatus({ text: '错误', details: '无法加载代理列表' });
    }
  };

  // 添加消息
  const addMessage = (role, content, metadata = null) => {
    const newMessage = {
      id: `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      role,
      content,
      metadata,
      timestamp: new Date(),
    };
    setMessages(prev => [...prev, newMessage]);
    return newMessage.id;
  };

  // 更新消息
  const updateMessage = (messageId, content, metadata = null) => {
    setMessages(prev =>
      prev.map(msg =>
        msg.id === messageId
          ? { ...msg, content, metadata: metadata || msg.metadata }
          : msg
      )
    );
  };

  // 发送消息
  const handleSendMessage = async (query) => {
    if (!query.trim() || isProcessing) return;

    // 添加用户消息
    addMessage('user', query);
    setIsProcessing(true);

    const request = {
      query,
      agent_type: config.agentType,
      model: config.model,
      max_steps: config.maxSteps,
      system_prompt: config.systemPrompt,
      session_id: sessionId,
    };

    try {
      if (config.streamMode) {
        await handleStreamMessage(request);
      } else {
        await handleNormalMessage(request);
      }
    } catch (error) {
      addMessage('error', `错误: ${error.message}`);
      setStatus({ text: '错误', details: error.message });
    } finally {
      setIsProcessing(false);
    }
  };

  // 普通对话
  const handleNormalMessage = async (request) => {
    setStatus({ text: '处理中', details: '正在思考...' });

    const response = await sendChatMessage(request);

    if (response.success) {
      addMessage('assistant', response.response, response.metadata);
      setStatus({
        text: '完成',
        details: formatMetadata(response.metadata),
      });
    } else {
      addMessage('error', response.error || '未知错误');
      setStatus({ text: '错误', details: response.error });
    }
  };

  // 流式对话
  const handleStreamMessage = async (request) => {
    return new Promise((resolve, reject) => {
      const client = new ChatStreamClient();
      let messageId = null;
      let fullContent = '';
      let currentStep = 0;

      client.onMessage((data) => {
        console.log('收到消息:', data);
        
        if (data.type === 'error') {
          addMessage('error', data.error || data.content);
          client.close();
          reject(new Error(data.error || data.content));
          return;
        }

        if (data.type === 'final') {
          // 在最终结果中也保留执行过程
          if (messageId) {
            // 添加分隔线和最终答案标题
            const finalContent = fullContent + 
              '\n' + '='.repeat(60) + '\n' +
              '📊 最终答案：\n' + 
              '='.repeat(60) + '\n\n' +
              data.content;
            updateMessage(messageId, finalContent, data.metadata);
          } else {
            addMessage('assistant', data.content, data.metadata);
          }
          setStatus({ text: '完成', details: formatMetadata(data.metadata) });
          client.close();
          resolve();
        } else {
          // 其他类型的消息（thinking, action, observation）
          if (!messageId) {
            messageId = addMessage('assistant', '', null, true);
          }

          currentStep = data.step || currentStep;
          fullContent += formatStreamContent(data);

          updateMessage(messageId, fullContent, data.metadata);
          setStatus({ text: '执行中', details: `步骤 ${currentStep}` });
        }
      });

      client.onError((error) => {
        console.error('WebSocket错误:', error);
        addMessage('error', 'WebSocket连接错误。请尝试使用非流式模式。');
        setStatus({ text: '错误', details: 'WebSocket连接失败' });
        reject(error);
      });

      client.onClose(() => {
        console.log('WebSocket已关闭');
      });

      client.connect();
      
      // 等待连接建立后发送
      const sendInterval = setInterval(() => {
        if (client.isConnected()) {
          clearInterval(sendInterval);
          console.log('发送请求:', request);
          client.send(request);
          setStatus({ text: '连接成功', details: '流式响应中...' });
        }
      }, 100);

      // 超时处理
      setTimeout(() => {
        if (!client.isConnected()) {
          clearInterval(sendInterval);
          client.close();
          reject(new Error('WebSocket连接超时'));
        }
      }, 5000);
    });
  };

  // 格式化流式内容
  const formatStreamContent = (data) => {
    const typeEmojis = {
      thinking: '💭 思考',
      action: '⚙️ 执行',
      observation: '👁️ 观察',
      metadata: 'ℹ️ 信息',
    };

    const typeLabel = typeEmojis[data.type] || '📝 消息';
    
    // 格式化内容，保持原始换行
    const content = data.content || '';
    
    // 如果是多行内容，添加分隔线
    const lines = content.split('\n').length;
    const separator = lines > 2 ? '\n' + '─'.repeat(50) + '\n' : '\n';
    
    return `[${typeLabel}]\n${content}${separator}\n`;
  };

  // 格式化元数据
  const formatMetadata = (metadata) => {
    if (!metadata) return '';

    const parts = [];
    if (metadata.total_steps) parts.push(`${metadata.total_steps} 步`);
    if (metadata.tools_called) parts.push(`${metadata.tools_called} 个工具`);
    if (metadata.execution_time_ms) parts.push(`${metadata.execution_time_ms}ms`);
    if (metadata.tool_names && metadata.tool_names.length > 0) {
      parts.push(`工具: ${metadata.tool_names.join(', ')}`);
    }

    return parts.join(' | ');
  };

  // 清空对话
  const handleClearChat = () => {
    setMessages([]);
    setStatus({ text: '就绪', details: '对话已清空' });
  };

  // 配置更新
  const handleConfigChange = (key, value) => {
    setConfig(prev => ({ ...prev, [key]: value }));
  };

  return (
    <div className="app">
      <Header />
      
      <div className="main-container">
        <ConfigPanel
          config={config}
          agentTypes={agentTypes}
          onConfigChange={handleConfigChange}
          onClearChat={handleClearChat}
          onShowTools={() => setShowToolsModal(true)}
        />

        <ChatContainer
          messages={messages}
          onSetQuery={(query) => handleSendMessage(query)}
        />

        <InputArea
          onSendMessage={handleSendMessage}
          isProcessing={isProcessing}
        />

        <StatusBar status={status} />
      </div>

      {showToolsModal && (
        <ToolsModal onClose={() => setShowToolsModal(false)} />
      )}
    </div>
  );
}

export default App;

