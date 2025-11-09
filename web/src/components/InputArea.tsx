import { useState } from 'react';

import './InputArea.css';

interface InputAreaProps {
  onSendMessage: (query: string) => void;
  isProcessing: boolean;
  disabled?: boolean;
}

const InputArea = ({ onSendMessage, isProcessing, disabled = false }: InputAreaProps) => {
  const [query, setQuery] = useState<string>('');

  const sendMessage = () => {
    if (query.trim() && !isProcessing) {
      onSendMessage(query);
      setQuery('');
    }
  };

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    sendMessage();
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Enter' && !event.shiftKey && !event.ctrlKey) {
      event.preventDefault();
      sendMessage();
    }
  };

  return (
    <form className="input-container" onSubmit={handleSubmit}>
      <textarea
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="输入您的问题..."
        rows={3}
        disabled={isProcessing}
      />
      <button
        type="submit"
        className="btn-primary"
        disabled={isProcessing || !query.trim() || disabled}
        title={disabled ? '请先选择一个 Agent' : ''}
      >
        {isProcessing ? (
          <>
            <span className="loader" />
            处理中...
          </>
        ) : disabled ? (
          '🚫 请选择 Agent'
        ) : (
          '发送'
        )}
      </button>
    </form>
  );
};

export default InputArea;

