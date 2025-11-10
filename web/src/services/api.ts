import axios from 'axios';

const API_BASE = '/api';

// ---------- Types ----------
export type AgentFramework = 'react' | 'chain' | 'plan' | 'sql' | 'elasticsearch';

export interface AgentTypeInfo {
  type: AgentFramework;
  name: string;
  description?: string;
  available: boolean;
}

export interface ToolInfo {
  name: string;
  description?: string;
  type: string;
  mcp_service?: string;
  input?: unknown;
  input_schema?: unknown;
}

export interface MCPServiceInfo {
  id?: number;
  name: string;
  endpoint: string;
  description?: string;
  is_active?: boolean;
  tool_count?: number;
  last_refresh?: string;
  created_at?: string;
  updated_at?: string;
}

export interface MCPDetailedToolInfo {
  name: string;
  description?: string;
  type?: string;
  input_schema?: unknown;
}

export interface AgentInfo {
  id: number;
  name: string;
  framework: AgentFramework;
  description?: string;
  system_prompt?: string;
  max_steps?: number;
  model?: string;
  mcp_services?: string[];
  created_at?: string;
  updated_at?: string;
  is_active?: boolean;
  connection_config?: string;
  config_json?: string;
}

export interface ExecutionMetadata {
  total_steps?: number;
  tools_called?: number;
  tool_names?: string[];
  execution_time_ms?: number;
  state?: string;
}

export interface ChatRequestPayload {
  query: string;
  agent_id: number;
  session_id?: string;
  agent_type?: number;
  model?: string;
  system_prompt?: string;
  max_steps?: number;
  enabled_mcp_services?: string[];
  config?: Record<string, string>;
}

export interface ChatResponsePayload {
  response: string;
  agent_type?: string;
  metadata?: ExecutionMetadata;
  success: boolean;
  error?: string;
}

export type ChatStreamMessage =
  | {
      type: 'thinking' | 'action' | 'observation' | 'metadata';
      content: string;
      step?: number;
      metadata?: ExecutionMetadata;
    }
  | {
      type: 'final';
      content: string;
      metadata?: ExecutionMetadata;
    }
  | {
      type: 'error';
      content?: string;
      error?: string;
    };

export interface AgentConfigPayload {
  name: string;
  framework: AgentFramework;
  description?: string;
  system_prompt?: string;
  max_steps?: number;
  model?: string;
  mcp_services?: string[];
  connection_config?: string;
  config_json?: string;
}

export interface AgentConfigResponse {
  success: boolean;
  message?: string;
  agent?: AgentInfo;
}

export interface MCPServiceResponse {
  success: boolean;
  message?: string;
  service?: MCPServiceInfo;
}

// ---------- Axios client ----------
export const api = axios.create({
  baseURL: API_BASE,
  timeout: 60000,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const AGENT_TYPES: Record<string, AgentFramework> = {
  REACT: 'react',
  CHAIN: 'chain',
  PLAN: 'plan',
  SQL: 'sql',
  ELASTICSEARCH: 'elasticsearch',
};

export const AGENT_TYPE_TO_ENUM: Record<AgentFramework, number> = {
  react: 0,
  chain: 1,
  plan: 2,
  sql: 3,
  elasticsearch: 4,
};

// ---------- REST APIs ----------
export const getAgentTypes = async (): Promise<AgentTypeInfo[]> => {
  const response = await api.get<{ types: AgentTypeInfo[] }>('/agent-types');
  return response.data.types ?? [];
};

export const getTools = async (): Promise<ToolInfo[]> => {
  const response = await api.get<{ tools: ToolInfo[] }>('/tools');
  return response.data.tools ?? [];
};

export const getMCPServices = async (): Promise<MCPServiceInfo[]> => {
  const response = await api.get<{ services: MCPServiceInfo[] }>('/mcp/services');
  return response.data.services ?? [];
};

export const getMCPServicesWithId = async (): Promise<MCPServiceInfo[]> => {
  const response = await api.get<{ services: MCPServiceInfo[] }>('/mcp/services-with-id');
  return response.data.services ?? [];
};

export const addMCPService = async (
  name: string,
  endpoint: string,
): Promise<MCPServiceResponse> => {
  const response = await api.post<MCPServiceResponse>('/mcp/services', { name, endpoint });
  return response.data;
};

export const removeMCPService = async (name: string): Promise<MCPServiceResponse> => {
  const response = await api.delete<MCPServiceResponse>(`/mcp/services/${name}`);
  return response.data;
};

export const getMCPServiceTools = async (id: number): Promise<MCPDetailedToolInfo[]> => {
  const response = await api.get<{ tools: MCPDetailedToolInfo[] }>(`/mcp/services/${id}/tools`);
  return response.data.tools ?? [];
};

export const getAgents = async (): Promise<AgentInfo[]> => {
  const response = await api.get<{ agents: AgentInfo[] }>('/agents');
  return response.data.agents ?? [];
};

export const createAgent = async (
  agentData: AgentConfigPayload,
): Promise<AgentConfigResponse> => {
  const response = await api.post<AgentConfigResponse>('/agents', agentData);
  return response.data;
};

export const updateAgent = async (
  id: number,
  agentData: AgentConfigPayload,
): Promise<AgentConfigResponse> => {
  const response = await api.put<AgentConfigResponse>(`/agents/${id}`, agentData);
  return response.data;
};

export const deleteAgent = async (id: number): Promise<AgentConfigResponse> => {
  const response = await api.delete<AgentConfigResponse>(`/agents/${id}`);
  return response.data;
};

export const sendChatMessage = async (
  request: ChatRequestPayload,
): Promise<ChatResponsePayload> => {
  const response = await api.post<ChatResponsePayload>('/chat', request);
  return response.data;
};

// ---------- WebSocket Client ----------
type MessageHandler = (message: ChatStreamMessage) => void;
type ErrorHandler = (error: Event) => void;
type CloseHandler = () => void;

export class ChatStreamClient {
  private ws: WebSocket | null = null;
  private readonly messageHandlers: MessageHandler[] = [];
  private readonly errorHandlers: ErrorHandler[] = [];
  private readonly closeHandlers: CloseHandler[] = [];

  connect(): void {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/chat/stream`;

    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      // no-op
    };

    this.ws.onmessage = (event: MessageEvent<string>) => {
      try {
        const data = JSON.parse(event.data) as ChatStreamMessage;
        this.messageHandlers.forEach((handler) => handler(data));
      } catch (error) {
        console.error('Failed to parse WebSocket message', error);
      }
    };

    this.ws.onerror = (error: Event) => {
      console.error('WebSocket error:', error);
      this.errorHandlers.forEach((handler) => handler(error));
    };

    this.ws.onclose = () => {
      this.closeHandlers.forEach((handler) => handler());
      this.ws = null;
    };
  }

  send(request: ChatRequestPayload): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(request));
    } else {
      console.error('WebSocket is not connected');
    }
  }

  onMessage(handler: MessageHandler): void {
    this.messageHandlers.push(handler);
  }

  onError(handler: ErrorHandler): void {
    this.errorHandlers.push(handler);
  }

  onClose(handler: CloseHandler): void {
    this.closeHandlers.push(handler);
  }

  close(): void {
    this.ws?.close();
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

export default api;

