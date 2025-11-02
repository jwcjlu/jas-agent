import { useEffect, useRef } from 'react';
import Message from './Message';
import WelcomeMessage from './WelcomeMessage';
import './ChatContainer.css';

function ChatContainer({ messages, onSetQuery }) {
  const containerRef = useRef(null);

  // 自动滚动到底部
  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [messages]);

  return (
    <div className="chat-container" ref={containerRef}>
      {messages.length === 0 ? (
        <WelcomeMessage onSetQuery={onSetQuery} />
      ) : (
        messages.map((message) => (
          <Message key={message.id} message={message} />
        ))
      )}
    </div>
  );
}

export default ChatContainer;

