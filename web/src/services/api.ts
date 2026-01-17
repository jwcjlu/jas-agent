import axios from 'axios';

const API_BASE = '/api';

// ---------- Types ----------
export type AgentFramework = 'react' | 'chain' | 'plan' | 'sql' | 'elasticsearch' | 'aiops';

export interface BaseResponse {
  code: number;
  message?: string;
  reason?: string;
}

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
  inputSchema?: unknown;
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
  ret: BaseResponse;
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
  ret: BaseResponse;
  agent?: AgentInfo;
}

export interface MCPServiceResponse {
  ret: BaseResponse;
  service?: MCPServiceInfo;
}

export interface AgentTypesResponse {
  ret: BaseResponse;
  types?: AgentTypeInfo[];
}

export interface ToolsResponse {
  ret: BaseResponse;
  tools?: ToolInfo[];
}

export interface MCPServicesResponse {
  ret: BaseResponse;
  services?: MCPServiceInfo[];
}

export interface MCPServicesWithIdResponse {
  ret: BaseResponse;
  services?: MCPServiceInfo[];
}

export interface MCPServiceToolsResponse {
  ret: BaseResponse;
  tools?: MCPDetailedToolInfo[];
}

export interface AgentListResponse {
  ret: BaseResponse;
  agents?: AgentInfo[];
}

// 知识库相关类型
export interface KnowledgeBaseInfo {
  id: number;
  name: string;
  description?: string;
  tags?: string[];
  // 兼容两种命名格式
  embedding_model?: string;
  embeddingModel?: string;
  chunk_size?: number;
  chunkSize?: number;
  chunk_overlap?: number;
  chunkOverlap?: number;
  vector_store_type?: string;
  vectorStoreType?: string;
  vector_store_config?: string;
  vectorStoreConfig?: string;
  is_active?: boolean;
  isActive?: boolean;
  document_count?: number;
  documentCount?: number;
  created_at?: string;
  createdAt?: string;
  updated_at?: string;
  updatedAt?: string;
}

export interface KnowledgeBaseRequest {
  id?: number;
  name: string;
  description?: string;
  embedding_model?: string;
  chunk_size?: number;
  chunk_overlap?: number;
  vector_store_type?: string;
  vector_store_config?: string;
  is_active?: boolean;
}

export interface KnowledgeBaseResponse {
  ret: BaseResponse;
  knowledge_base?: KnowledgeBaseInfo;
}

export interface KnowledgeBaseListResponse {
  ret: BaseResponse;
  knowledge_bases?: KnowledgeBaseInfo[];
  knowledgeBases?: KnowledgeBaseInfo[]; // 兼容驼峰命名
}

// 文档相关类型
export interface DocumentInfo {
  id: number;
  knowledge_base_id: number;
  name: string;
  file_path?: string;
  file_size?: number;
  file_type?: string;
  status?: string; // pending, processing, completed, failed
  chunk_count?: number;
  processed_at?: string;
  error_message?: string;
  metadata?: string;
  created_at?: string;
  updated_at?: string;
  enable_graph_extract?: boolean;
  enableGraphExtract?: boolean;
}

export interface DocumentListResponse {
  ret: BaseResponse;
  documents?: DocumentInfo[];
  Documents?: DocumentInfo[]; // 兼容驼峰命名
}

export interface DocumentResponse {
  ret: BaseResponse;
  document?: DocumentInfo;
}

// ---------- Axios client ----------
export const api = axios.create({
  baseURL: API_BASE,
  timeout: 60000,
  headers: {
    'Content-Type': 'application/json',
  },
});

export class ApiError extends Error {
  code: number;
  reason?: string;

  constructor(ret: BaseResponse) {
    super(ret.message || '请求失败');
    this.name = 'ApiError';
    this.code = ret.code;
    this.reason = ret.reason;
  }
}

const ensureSuccess = <T extends { ret?: BaseResponse }>(data: T): T => {
  // 兼容两种字段名格式：ret 和 Ret
  const ret = (data as any).ret || (data as any).Ret;
  if (!ret) {
    // 如果没有 ret 字段，但有数据内容，认为可能是成功（某些接口可能不返回 ret）
    if ((data as any).document || (data as any).Document || (data as any).knowledge_base || (data as any).KnowledgeBase) {
      return data;
    }
    throw new Error('响应格式错误：缺少 ret 字段');
  }
  
  // 兼容两种字段名格式：code 和 Code
  const code = ret.code ?? ret.Code;
  
  // 如果 code 不存在，但有成功消息且没有错误信息，认为成功
  if (code === undefined || code === null) {
    const message = ret.message ?? ret.Message ?? '';
    // 如果消息包含 "success" 或 "成功"，或者没有错误信息，认为成功
    if (message.toLowerCase().includes('success') || 
        message.includes('成功') || 
        (!message || message.trim() === '')) {
      return data;
    }
    // 否则认为是错误
    throw new ApiError({ code: 1, message: message || '请求失败', reason: ret.reason ?? ret.Reason });
  }
  
  // 如果 code 不是 0，认为是错误
  if (code !== 0) {
    const message = ret.message ?? ret.Message ?? '请求失败';
    const reason = ret.reason ?? ret.Reason;
    throw new ApiError({ code, message, reason });
  }
  
  return data;
};

export const AGENT_TYPES: Record<string, AgentFramework> = {
  REACT: 'react',
  CHAIN: 'chain',
  PLAN: 'plan',
  SQL: 'sql',
  ELASTICSEARCH: 'elasticsearch',
  AIOPS: 'aiops',
};

export const AGENT_TYPE_TO_ENUM: Record<AgentFramework, number> = {
  react: 0,
  chain: 1,
  plan: 2,
  sql: 3,
  elasticsearch: 4,
  aiops: 5,
};

// ---------- REST APIs ----------
export const getAgentTypes = async (): Promise<AgentTypeInfo[]> => {
  const response = await api.get<AgentTypesResponse>('/agent-types');
  const data = ensureSuccess(response.data);
  return data.types ?? [];
};

export const getTools = async (): Promise<ToolInfo[]> => {
  const response = await api.get<ToolsResponse>('/tools');
  const data = ensureSuccess(response.data);
  return data.tools ?? [];
};

export const getMCPServices = async (): Promise<MCPServiceInfo[]> => {
  const response = await api.get<MCPServicesResponse>('/mcp/services');
  const data = ensureSuccess(response.data);
  return data.services ?? [];
};

export const getMCPServicesWithId = async (): Promise<MCPServiceInfo[]> => {
  const response = await api.get<MCPServicesWithIdResponse>('/mcp/services-with-id');
  const data = ensureSuccess(response.data);
  return data.services ?? [];
};

export const addMCPService = async (
  name: string,
  endpoint: string,
  clientType: string = 'metoro',
): Promise<MCPServiceResponse> => {
  const response = await api.post<MCPServiceResponse>('/mcp/services', { name, endpoint, clientType });
  return ensureSuccess(response.data);
};

export const removeMCPService = async (name: string): Promise<MCPServiceResponse> => {
  const response = await api.delete<MCPServiceResponse>(`/mcp/services/${name}`);
  return ensureSuccess(response.data);
};

export const getMCPServiceTools = async (id: number): Promise<MCPDetailedToolInfo[]> => {
  const response = await api.get<MCPServiceToolsResponse>(`/mcp/services/${id}/tools`);
  const data = ensureSuccess(response.data);
  return data.tools ?? [];
};

export const getAgents = async (): Promise<AgentInfo[]> => {
  const response = await api.get<AgentListResponse>('/agents');
  const data = ensureSuccess(response.data);
  return data.agents ?? [];
};

export const createAgent = async (
  agentData: AgentConfigPayload,
): Promise<AgentConfigResponse> => {
  const response = await api.post<AgentConfigResponse>('/agents', agentData);
  return ensureSuccess(response.data);
};

export const updateAgent = async (
  id: number,
  agentData: AgentConfigPayload,
): Promise<AgentConfigResponse> => {
  const response = await api.put<AgentConfigResponse>(`/agents/${id}`, agentData);
  return ensureSuccess(response.data);
};

export const deleteAgent = async (id: number): Promise<AgentConfigResponse> => {
  const response = await api.delete<AgentConfigResponse>(`/agents/${id}`);
  return ensureSuccess(response.data);
};

export const sendChatMessage = async (
  request: ChatRequestPayload,
): Promise<ChatResponsePayload> => {
  const response = await api.post<ChatResponsePayload>('/chat', request);
  return ensureSuccess(response.data);
};

// ---------- 知识库管理 APIs ----------
export const getKnowledgeBases = async (
  searchQuery: string = '',
  tags: string[] = [],
): Promise<KnowledgeBaseInfo[]> => {
  const params = new URLSearchParams();
  if (searchQuery) {
    params.append('search', searchQuery);
  }
  tags.forEach((tag) => params.append('tags', tag));

  const url = `/knowledge-bases${params.toString() ? `?${params.toString()}` : ''}`;
  console.log('请求知识库列表 URL:', url);
  const response = await api.get<KnowledgeBaseListResponse>(url);
  console.log('知识库列表原始响应:', response.data);
  const data = ensureSuccess(response.data);
  console.log('解析后的数据:', data);
  console.log('knowledge_bases 字段:', data.knowledge_bases);
  console.log('knowledgeBases 字段:', data.knowledgeBases);
  // 兼容两种字段名格式：knowledge_bases (下划线) 和 knowledgeBases (驼峰)
  const kbs = data.knowledge_bases ?? data.knowledgeBases ?? [];
  // 标准化字段名（将驼峰转换为下划线）
  const normalizedKbs = kbs.map((kb: any) => ({
    id: kb.id,
    name: kb.name,
    description: kb.description,
    tags: kb.tags,
    embedding_model: kb.embedding_model ?? kb.embeddingModel,
    chunk_size: kb.chunk_size ?? kb.chunkSize,
    chunk_overlap: kb.chunk_overlap ?? kb.chunkOverlap,
    vector_store_type: kb.vector_store_type ?? kb.vectorStoreType,
    vector_store_config: kb.vector_store_config ?? kb.vectorStoreConfig,
    is_active: kb.is_active ?? kb.isActive,
    document_count: kb.document_count ?? kb.documentCount,
    created_at: kb.created_at ?? kb.createdAt,
    updated_at: kb.updated_at ?? kb.updatedAt,
  }));
  if (normalizedKbs && normalizedKbs.length > 0) {
    console.log('第一个知识库的完整数据:', JSON.stringify(normalizedKbs[0], null, 2));
  }
  return normalizedKbs;
};

export const getKnowledgeBaseByAgent = async (agentId: number): Promise<KnowledgeBaseInfo | null> => {
  try {
    const response = await api.get<KnowledgeBaseResponse>(`/agents/${agentId}/knowledge-base`);
    const data = ensureSuccess(response.data);
    return data.knowledge_base ?? null;
  } catch (error) {
    if (error instanceof ApiError && error.code !== 0) {
      return null;
    }
    throw error;
  }
};

export const createKnowledgeBase = async (
  kbData: KnowledgeBaseRequest,
): Promise<KnowledgeBaseResponse> => {
  const response = await api.post<KnowledgeBaseResponse>('/knowledge-bases', kbData);
  return ensureSuccess(response.data);
};

export const updateKnowledgeBase = async (
  id: number,
  kbData: KnowledgeBaseRequest,
): Promise<KnowledgeBaseResponse> => {
  const response = await api.put<KnowledgeBaseResponse>(`/knowledge-bases/${id}`, kbData);
  return ensureSuccess(response.data);
};

export const deleteKnowledgeBase = async (id: number): Promise<KnowledgeBaseResponse> => {
  const response = await api.delete<KnowledgeBaseResponse>(`/knowledge-bases/${id}`);
  return ensureSuccess(response.data);
};

export const listDocuments = async (knowledgeBaseId: number): Promise<DocumentInfo[]> => {
  const response = await api.get<DocumentListResponse>(`/knowledge-bases/${knowledgeBaseId}/documents`);
  const data = ensureSuccess(response.data);
  // 兼容两种字段名格式
  const docs = data.documents ?? data.Documents ?? [];
  console.log('listDocuments API 返回:', { data, docs });
  return docs.map((doc) => ({
    ...doc,
    enable_graph_extract: doc.enable_graph_extract ?? doc.enableGraphExtract ?? false,
  }));
};

export const deleteDocument = async (id: number): Promise<DocumentResponse> => {
  const response = await api.delete<DocumentResponse>(`/documents/${id}`);
  return ensureSuccess(response.data);
};

export const uploadDocument = async (
  knowledgeBaseId: number,
  file: File,
  options?: { extractGraph?: boolean },
): Promise<DocumentResponse> => {
  const formData = new FormData();
  formData.append('file', file);
  formData.append('extractGraph', options?.extractGraph ? 'true' : 'false');

  const response = await api.post<DocumentResponse>(
    `/knowledge-bases/${knowledgeBaseId}/documents/upload`,
    formData,
    {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    },
  );
  
  console.log('uploadDocument 原始响应:', response.data);
  
  const data = response.data;
  const ret = (data as any).ret || (data as any).Ret;
  
  // 检查响应：如果有 document 字段且 ret 中没有错误 code，认为成功
  if ((data as any).document || (data as any).Document) {
    if (!ret || ret.code === undefined || ret.code === null || ret.code === 0) {
      console.log('文档上传成功，返回响应');
      return data;
    }
  }
  
  // 使用 ensureSuccess 进行标准检查
  return ensureSuccess(data);
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

