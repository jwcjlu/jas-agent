import type { ChatMessage } from '../types';

import './Message.css';

interface MessageProps {
  message: ChatMessage;
}

const roleNames: Record<string, string> = {
  user: '👤 用户',
  assistant: '🤖 助手',
  system: '⚙️ 系统',
  error: '❌ 错误',
};

const formatMetadata = (metadata: ChatMessage['metadata']): string => {
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

const Message = ({ message }: MessageProps): JSX.Element => {
  return (
    <div className={`message ${message.role}`}>
      <div className="message-header">
        <span className="icon">{roleNames[message.role] ?? message.role}</span>
        <span className="timestamp">{message.timestamp?.toLocaleTimeString()}</span>
      </div>
      <div className="message-content">{message.content}</div>
      {message.metadata && <div className="message-meta">{formatMetadata(message.metadata)}</div>}
    </div>
  );
};

export default Message;

