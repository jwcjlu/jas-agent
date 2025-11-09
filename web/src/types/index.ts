import type { AgentFramework, ExecutionMetadata } from '../services/api';

export type MessageRole = 'user' | 'assistant' | 'error';

export interface ChatMessage {
  id: string;
  role: MessageRole;
  content: string;
  metadata?: ExecutionMetadata;
  timestamp: Date;
}

export interface StatusState {
  text: string;
  details?: string;
}

export interface ConfigState {
  agentType: AgentFramework;
  model: string;
  maxSteps: number;
  systemPrompt: string;
  streamMode: boolean;
  enabledMCPServices: string[];
}

