import { useEffect, useState } from 'react';

import AgentManageModal from './components/AgentManageModal';
import AgentSelector from './components/AgentSelector';
import ChatContainer from './components/ChatContainer';
import ConfigPanel from './components/ConfigPanel';
import Header from './components/Header';
import InputArea from './components/InputArea';
import StatusBar from './components/StatusBar';
import MCPManageModal from './components/MCPManageModal';
import {
  AGENT_TYPE_TO_ENUM,
  ChatStreamClient,
  getAgentTypes,
  getAgents,
  getMCPServices,
  getTools,
  sendChatMessage,
  type AgentInfo,
  type AgentTypeInfo,
  type ChatRequestPayload,
  type ChatStreamMessage,
  type ExecutionMetadata,
  type MCPServiceInfo,
  type ToolInfo,
} from './services/api';
import type { ChatMessage, ConfigState, StatusState } from './types';

import './App.css';

const createSessionId = (): string =>
  `session_${Date.now()}_${Math.random().toString(36).slice(2, 11)}`;

const initialConfig: ConfigState = {
  agentType: 'react',
  model: 'gpt-3.5-turbo',
  maxSteps: 10,
  systemPrompt: '',
  streamMode: true,
  enabledMCPServices: [],
};

type NonFinalStreamMessage = Extract<
  ChatStreamMessage,
  { type: 'thinking' | 'action' | 'observation' | 'metadata' }
>;

const App = (): JSX.Element => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [config, setConfig] = useState<ConfigState>(initialConfig);
  const [isProcessing, setIsProcessing] = useState<boolean>(false);
  const [status, setStatus] = useState<StatusState>({ text: '就绪', details: '' });
  const [showAgentModal, setShowAgentModal] = useState<boolean>(false);
  const [showMCPModal, setShowMCPModal] = useState<boolean>(false);
  const [sessionId] = useState<string>(() => createSessionId());
  const [agentTypes, setAgentTypes] = useState<AgentTypeInfo[]>([]);
  const [mcpServices, setMcpServices] = useState<MCPServiceInfo[]>([]);
  const [agents, setAgents] = useState<AgentInfo[]>([]);
  const [selectedAgentId, setSelectedAgentId] = useState<number | null>(null);
  const [mcpToolMap, setMcpToolMap] = useState<Record<string, ToolInfo[]>>({});

  const normalizeAgent = (agent: AgentInfo): AgentInfo => ({
    ...agent,
    mcp_services:
      agent.mcp_services ?? (agent as unknown as { mcpServices?: string[] }).mcpServices ?? [],
  });

  const syncEnabledMCPWithAgent = (agentId: number | null, list?: AgentInfo[]): void => {
    if (!agentId) {
      setConfig((prev) => ({ ...prev, enabledMCPServices: [] }));
      return;
    }
    const source = (list ?? agents).map(normalizeAgent);
    const found = source.find((a) => a.id === agentId);
    const bound = found?.mcp_services ?? [];
    setConfig((prev) => ({ ...prev, enabledMCPServices: bound }));
  };

  useEffect(() => {
    void loadAgentTypes();
    void loadMCPServices();
    void loadAgents();
    void loadTools();
  }, []);

  const loadAgentTypes = async (): Promise<void> => {
    try {
      const types = await getAgentTypes();
      setAgentTypes(types.filter((t) => t.available));
    } catch (error) {
      console.error('加载代理列表失败:', error);
    }
  };

  const loadMCPServices = async (): Promise<void> => {
    try {
      const services = await getMCPServices();
      setMcpServices(services ?? []);
      if (services && services.length > 0) {
        console.log(`📋 加载了 ${services.length} 个MCP服务`);
      }
    } catch (error) {
      console.error('加载MCP服务失败:', error);
    }
  };

  const handleMCPServicesChange = (services: MCPServiceInfo[] = []): void => {
    setMcpServices(services);
    const serviceNames = services.map((s) => s.name);
    setConfig((prev) => ({
      ...prev,
      enabledMCPServices: serviceNames,
    }));
  };

  const loadAgents = async (): Promise<void> => {
    try {
      const agentsList = await getAgents();
      const normalized = (agentsList ?? []).map(normalizeAgent);
      setAgents(normalized);
      if (
        (!selectedAgentId || !normalized.some((a) => a.id === selectedAgentId)) &&
        normalized.length > 0
      ) {
        const firstId = normalized[0].id;
        setSelectedAgentId(firstId);
        syncEnabledMCPWithAgent(firstId, normalized);
      } else if (selectedAgentId) {
        syncEnabledMCPWithAgent(selectedAgentId, normalized);
      }
    } catch (error) {
      console.error('加载Agent列表失败:', error);
    }
  };

  const loadTools = async (): Promise<void> => {
    try {
      const toolList = await getTools();
      const map: Record<string, ToolInfo[]> = {};
      (toolList ?? []).forEach((tool) => {
        const service =
          tool.mcp_service ?? (tool as unknown as { mcpService?: string }).mcpService ?? '';
        if (!service) return;
        if (!map[service]) map[service] = [];
        map[service].push(tool);
      });
      setMcpToolMap(map);
    } catch (error) {
      console.error('加载工具列表失败:', error);
    }
  };

  const handleAgentsChange = (agentsList: AgentInfo[]): void => {
    const normalized = (agentsList ?? []).map(normalizeAgent);
    setAgents(normalized);
    if (
      (!selectedAgentId || !normalized.some((a) => a.id === selectedAgentId)) &&
      normalized.length > 0
    ) {
      const firstId = normalized[0].id;
      setSelectedAgentId(firstId);
      syncEnabledMCPWithAgent(firstId, normalized);
    } else if (selectedAgentId) {
      syncEnabledMCPWithAgent(selectedAgentId, normalized);
    }
  };

  const addMessage = (
    role: ChatMessage['role'],
    content: string,
    metadata?: ExecutionMetadata,
  ): string => {
    const newMessage: ChatMessage = {
      id: `msg_${Date.now()}_${Math.random().toString(36).slice(2, 11)}`,
      role,
      content,
      metadata,
      timestamp: new Date(),
    };
    setMessages((prev) => [...prev, newMessage]);
    return newMessage.id;
  };

  const updateMessage = (
    messageId: string,
    content: string,
    metadata?: ExecutionMetadata,
  ): void => {
    setMessages((prev) =>
      prev.map((msg) =>
        msg.id === messageId ? { ...msg, content, metadata: metadata ?? msg.metadata } : msg,
      ),
    );
  };

  const handleSendMessage = async (query: string): Promise<void> => {
    if (!query.trim() || isProcessing) return;
    if (!selectedAgentId) {
      alert('请先选择一个 Agent！');
      return;
    }

    addMessage('user', query);
    setIsProcessing(true);

    const agentTypeValue = AGENT_TYPE_TO_ENUM[config.agentType];

    const request: ChatRequestPayload = {
      query,
      agent_id: selectedAgentId,
      session_id: sessionId,
      agent_type: Number.isFinite(agentTypeValue) ? agentTypeValue : undefined,
      model: config.model,
      max_steps: config.maxSteps,
      system_prompt: config.systemPrompt,
      enabled_mcp_services: config.enabledMCPServices ?? [],
    };

    try {
      if (config.streamMode) {
        await handleStreamMessage(request);
      } else {
        await handleNormalMessage(request);
      }
    } catch (error) {
      if (error instanceof Error) {
        addMessage('error', `错误: ${error.message}`);
        setStatus({ text: '错误', details: error.message });
      } else {
        addMessage('error', '未知错误');
        setStatus({ text: '错误', details: '未知错误' });
      }
    } finally {
      setIsProcessing(false);
    }
  };

  const handleNormalMessage = async (request: ChatRequestPayload): Promise<void> => {
    setStatus({ text: '处理中', details: '正在思考...' });

    const response = await sendChatMessage(request);

    addMessage('assistant', response.response, response.metadata);
    setStatus({
      text: '完成',
      details: formatMetadata(response.metadata),
    });
  };

  const handleStreamMessage = async (request: ChatRequestPayload): Promise<void> =>
    new Promise((resolve, reject) => {
      const client = new ChatStreamClient();
      let messageId: string | null = null;
      let fullContent = '';
      let currentStep = 0;
      let finished = false;

      client.onMessage((data: ChatStreamMessage) => {
        if (data.type === 'error') {
          addMessage('error', data.error ?? data.content ?? '未知错误');
          client.close();
          reject(new Error(data.error ?? data.content ?? '未知错误'));
          return;
        }

        if (data.type === 'final') {
          finished = true;
          if (messageId) {
            const finalContent =
              `${fullContent}\n${'='.repeat(60)}\n` +
              '📊 最终答案：\n' +
              `${'='.repeat(60)}\n\n` +
              data.content;
            updateMessage(messageId, finalContent, data.metadata);
          } else {
            addMessage('assistant', data.content, data.metadata);
          }
          setStatus({ text: '完成', details: formatMetadata(data.metadata) });
          // 延迟关闭，确保 resolve 先执行
          setTimeout(() => client.close(), 0);
          resolve();
          return;
        } else {
          if (!messageId) {
            messageId = addMessage('assistant', '');
          }

          currentStep = data.step ?? currentStep;
          fullContent += formatStreamContent(data);
          updateMessage(messageId, fullContent, data.metadata);
          setStatus({ text: '执行中', details: `步骤 ${currentStep}` });
        }
      });

      client.onError((error) => {
        console.error('WebSocket错误:', error);
        addMessage('error', 'WebSocket连接错误。请尝试使用非流式模式。');
        setStatus({ text: '错误', details: 'WebSocket连接失败' });
        reject(error instanceof Error ? error : new Error('WebSocket连接失败'));
      });

      client.onClose(() => {
        console.log('WebSocket已关闭');
        if (!finished) {
          reject(new Error('WebSocket 已关闭'));
        }
      });

      client.connect();

      const sendOnce = () => {
        if (client.isConnected()) {
          client.send(request);
          setStatus({ text: '连接成功', details: '流式响应中...' });
        } else {
          setTimeout(sendOnce, 100);
        }
      };
      sendOnce();
    });

  const formatStreamContent = (data: NonFinalStreamMessage): string => {
    const typeEmojis: Record<NonFinalStreamMessage['type'], string> = {
      thinking: '💭 思考',
      action: '⚙️ 执行',
      observation: '👁️ 观察',
      metadata: 'ℹ️ 信息',
    };

    const typeLabel = typeEmojis[data.type] ?? '📝 消息';
    const content = data.content ?? '';
    const lines = content.split('\n').length;
    const separator = lines > 2 ? `\n${'─'.repeat(50)}\n` : '\n';

    return `[${typeLabel}]\n${content}${separator}\n`;
  };

  const formatMetadata = (metadata?: ExecutionMetadata): string => {
    if (!metadata) return '';

    const parts: string[] = [];
    if (metadata.total_steps) parts.push(`${metadata.total_steps} 步`);
    if (metadata.tools_called) parts.push(`${metadata.tools_called} 个工具`);
    if (metadata.execution_time_ms) parts.push(`${metadata.execution_time_ms}ms`);
    if (metadata.tool_names && metadata.tool_names.length > 0) {
      parts.push(`工具: ${metadata.tool_names.join(', ')}`);
    }

    return parts.join(' | ');
  };

  const handleClearChat = (): void => {
    setMessages([]);
    setStatus({ text: '就绪', details: '对话已清空' });
  };

  const handleConfigChange = <K extends keyof ConfigState>(
    key: K,
    value: ConfigState[K],
  ): void => {
    setConfig((prev) => ({ ...prev, [key]: value }));
  };

  return (
    <div className="app">
      <Header />

      <div className="main-container">
        <div className="agent-selector-wrapper">
          <AgentSelector
            agents={agents}
            selectedAgentId={selectedAgentId}
            onChange={(id) => {
              setSelectedAgentId(id);
              syncEnabledMCPWithAgent(id);
            }}
            mcpToolMap={mcpToolMap}
          />
          <button onClick={() => setShowAgentModal(true)} className="btn-manage-agent">
            🤖 管理
          </button>
          <button onClick={() => setShowMCPModal(true)} className="btn-manage-agent">
            🔌 MCP 管理
          </button>

          <ConfigPanel
            config={config}
            agentTypes={agentTypes}
            onConfigChange={handleConfigChange}
            onClearChat={handleClearChat}
          />
        </div>

        <div className="chat-container-wrapper">
          <ChatContainer messages={messages} onSetQuery={handleSendMessage} />
        </div>

        <div className="input-area-wrapper">
          <InputArea
            onSendMessage={handleSendMessage}
            isProcessing={isProcessing}
            disabled={!selectedAgentId}
          />
        </div>

        <div className="status-bar-wrapper">
          <StatusBar status={status} />
        </div>
      </div>

      {/* 工具与 MCP 弹窗已移除以简化左侧配置 */}

      {showAgentModal && (
        <AgentManageModal
          onClose={() => setShowAgentModal(false)}
          onAgentsChange={handleAgentsChange}
          mcpServices={mcpServices}
        />
      )}

      {showMCPModal && (
        <MCPManageModal
          onClose={() => setShowMCPModal(false)}
          onServicesChange={handleMCPServicesChange}
        />
      )}
    </div>
  );
};

export default App;

