import { useState } from 'react';
import './InputArea.css';

function InputArea({ onSendMessage, isProcessing, disabled = false }) {
  const [query, setQuery] = useState('');

  const handleSubmit = (e) => {
    e.preventDefault();
    if (query.trim() && !isProcessing) {
      onSendMessage(query);
      setQuery('');
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey && !e.ctrlKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  return (
    <form className="input-container" onSubmit={handleSubmit}>
      <textarea
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="输入您的问题..."
        rows="3"
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
            <span className="loader"></span>
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
}

export default InputArea;

