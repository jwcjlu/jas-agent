import { useEffect, useRef } from 'react';

import type { ChatMessage } from '../types';
import Message from './Message';
import WelcomeMessage from './WelcomeMessage';

import './ChatContainer.css';

interface ChatContainerProps {
  messages: ChatMessage[];
  onSetQuery: (query: string) => void;
}

const ChatContainer = ({ messages, onSetQuery }: ChatContainerProps): JSX.Element => {
  const containerRef = useRef<HTMLDivElement | null>(null);

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
        messages.map((message) => <Message key={message.id} message={message} />)
      )}
    </div>
  );
};

export default ChatContainer;

