import axios from 'axios';

const API_BASE = '/api';

// 创建 axios 实例
const api = axios.create({
  baseURL: API_BASE,
  timeout: 60000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Agent 类型
export const AGENT_TYPES = {
  REACT: 'react',
  CHAIN: 'chain',
  PLAN: 'plan',
  SQL: 'sql',
};

// 获取可用的 Agent 类型
export const getAgentTypes = async () => {
  const response = await api.get('/agents');
  return response.data.agents;
};

// 获取可用的工具列表
export const getTools = async () => {
  const response = await api.get('/tools');
  return response.data.tools;
};

// MCP 服务管理
export const getMCPServices = async () => {
  const response = await api.get('/mcp/services');
  return response.data.services;
};

export const addMCPService = async (name, endpoint) => {
  const response = await api.post('/mcp/services', { name, endpoint });
  return response.data;
};

export const removeMCPService = async (name) => {
  const response = await api.delete(`/mcp/services/${name}`);
  return response.data;
};

// Agent 管理
export const getAgents = async () => {
  const response = await api.get('/agents');
  return response.data.agents;
};

export const createAgent = async (agentData) => {
  const response = await api.post('/agents', agentData);
  return response.data;
};

export const updateAgent = async (id, agentData) => {
  const response = await api.put(`/agents/${id}`, agentData);
  return response.data;
};

export const deleteAgent = async (id) => {
  const response = await api.delete(`/agents/${id}`);
  return response.data;
};

// 发送对话请求（非流式）
export const sendChatMessage = async (request) => {
  const response = await api.post('/chat', request);
  return response.data;
};

// WebSocket 流式对话
export class ChatStreamClient {
  constructor() {
    this.ws = null;
    this.messageHandlers = [];
    this.errorHandlers = [];
    this.closeHandlers = [];
  }

  connect() {
    // 使用当前 host，Vite 会自动代理 WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/chat/stream`;
    
    console.log('Connecting to WebSocket:', wsUrl);
    this.ws = new WebSocket(wsUrl);
    
    this.ws.onopen = () => {
      console.log('WebSocket connected');
    };
    
    this.ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      this.messageHandlers.forEach(handler => handler(data));
    };
    
    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      this.errorHandlers.forEach(handler => handler(error));
    };
    
    this.ws.onclose = () => {
      console.log('WebSocket closed');
      this.closeHandlers.forEach(handler => handler());
      this.ws = null;
    };
  }

  send(request) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(request));
    } else {
      console.error('WebSocket is not connected');
    }
  }

  onMessage(handler) {
    this.messageHandlers.push(handler);
  }

  onError(handler) {
    this.errorHandlers.push(handler);
  }

  onClose(handler) {
    this.closeHandlers.push(handler);
  }

  close() {
    if (this.ws) {
      this.ws.close();
    }
  }

  isConnected() {
    return this.ws && this.ws.readyState === WebSocket.OPEN;
  }
}

export default api;

