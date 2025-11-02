import './Message.css';

function Message({ message }) {
  const roleNames = {
    user: '👤 用户',
    assistant: '🤖 助手',
    system: '⚙️ 系统',
    error: '❌ 错误',
  };

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

  return (
    <div className={`message ${message.role}`}>
      <div className="message-header">
        <span className="icon">{roleNames[message.role] || message.role}</span>
        <span className="timestamp">
          {message.timestamp?.toLocaleTimeString()}
        </span>
      </div>
      <div className="message-content">{message.content}</div>
      {message.metadata && (
        <div className="message-meta">{formatMetadata(message.metadata)}</div>
      )}
    </div>
  );
}

export default Message;

