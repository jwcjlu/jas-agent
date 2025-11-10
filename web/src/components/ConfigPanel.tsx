import type { AgentTypeInfo } from '../services/api';
import type { ConfigState } from '../types';

import './ConfigPanel.css';

type ConfigChangeHandler = <K extends keyof ConfigState>(key: K, value: ConfigState[K]) => void;

interface ConfigPanelProps {
  config: ConfigState;
  agentTypes: AgentTypeInfo[]; // 兼容保留，但当前不渲染类型选择
  onConfigChange: ConfigChangeHandler;
  onClearChat: () => void;
}

const ConfigPanel = ({
  config,
  agentTypes,
  onConfigChange,
  onClearChat,
}: ConfigPanelProps): JSX.Element => {
  return (
    <div className="config-panel">
      <div className="control-group">
        <div className="model-control">
          <span className="control-label">模型</span>
          <select
            id="model"
            className="model-select compact"
            value={config.model}
            onChange={(e) => onConfigChange('model', e.target.value)}
          >
            <option value="gpt-3.5-turbo">GPT-3.5 Turbo</option>
            <option value="gpt-4">GPT-4</option>
            <option value="gpt-4-turbo">GPT-4 Turbo</option>
          </select>
        </div>

        <label className="inline-switch">
          <input
            type="checkbox"
            checked={config.streamMode}
            onChange={(e) => onConfigChange('streamMode', e.target.checked)}
          />
          <span>流式</span>
        </label>

        <button onClick={onClearChat} className="icon-button" title="清空对话">
          🧹
        </button>
      </div>
    </div>
  );
};

export default ConfigPanel;

