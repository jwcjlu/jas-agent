import './WelcomeMessage.css';

interface WelcomeMessageProps {
  onSetQuery: (query: string) => void;
}

const examples: string[] = [
  '计算 (15 + 27) * 3',
  '我有一只边境牧羊犬和一只苏格兰梗，它们的总体重是多少？',
  '列出可用的工具并说明它们的用途',
];

const WelcomeMessage = ({ onSetQuery }: WelcomeMessageProps): JSX.Element => (
  <div className="welcome-message">
    <h2>👋 欢迎使用 JAS Agent</h2>
    <p>请在下方输入您的问题，AI代理将为您提供帮助</p>

    <div className="examples">
      <h3>💡 示例问题：</h3>
      {examples.map((example) => (
        <button key={example} className="example-btn" onClick={() => onSetQuery(example)}>
          {example}
        </button>
      ))}
    </div>
  </div>
);

export default WelcomeMessage;

