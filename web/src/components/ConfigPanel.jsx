import './ConfigPanel.css';

function ConfigPanel({ 
  config, 
  agentTypes, 
  mcpServices = [],
  onConfigChange, 
  onClearChat, 
  onShowTools,
  onManageMCP 
}) {
  const handleMCPServiceToggle = (serviceName) => {
    const enabled = config.enabledMCPServices || [];
    const newEnabled = enabled.includes(serviceName)
      ? enabled.filter(s => s !== serviceName)
      : [...enabled, serviceName];
    onConfigChange('enabledMCPServices', newEnabled);
  };

  return (
    <div className="config-panel">
      <div className="config-section">
        <label htmlFor="agentType">Agent 类型:</label>
        <select
          id="agentType"
          value={config.agentType}
          onChange={(e) => onConfigChange('agentType', e.target.value)}
        >
          {agentTypes.map((type) => (
            <option key={type.type} value={type.type}>
              {type.name} - {type.description}
            </option>
          ))}
        </select>
      </div>

      <div className="config-section">
        <label htmlFor="model">模型:</label>
        <select
          id="model"
          value={config.model}
          onChange={(e) => onConfigChange('model', e.target.value)}
        >
          <option value="gpt-3.5-turbo">GPT-3.5 Turbo</option>
          <option value="gpt-4">GPT-4</option>
          <option value="gpt-4-turbo">GPT-4 Turbo</option>
        </select>
      </div>

      <div className="config-section">
        <label htmlFor="maxSteps">最大步数:</label>
        <input
          type="number"
          id="maxSteps"
          value={config.maxSteps}
          onChange={(e) => onConfigChange('maxSteps', parseInt(e.target.value))}
          min="1"
          max="50"
        />
      </div>

      <div className="config-section full-width">
        <label htmlFor="systemPrompt">系统提示词 (可选):</label>
        <textarea
          id="systemPrompt"
          value={config.systemPrompt}
          onChange={(e) => onConfigChange('systemPrompt', e.target.value)}
          rows="3"
          placeholder="自定义系统提示词..."
        />
      </div>

      <div className="config-section">
        <label>
          <input
            type="checkbox"
            checked={config.streamMode}
            onChange={(e) => onConfigChange('streamMode', e.target.checked)}
          />
          启用流式响应
        </label>
      </div>

      {/* MCP 服务选择 */}
      {mcpServices.length > 0 && (
        <div className="config-section full-width">
          <label>启用的 MCP 服务:</label>
          <div className="mcp-services-selector">
            {mcpServices.map((service) => (
              <label key={service.name} className="mcp-service-checkbox">
                <input
                  type="checkbox"
                  checked={(config.enabledMCPServices || []).includes(service.name)}
                  onChange={() => handleMCPServiceToggle(service.name)}
                />
                <span>{service.name}</span>
                <span className="tool-count">({service.tool_count} 工具)</span>
              </label>
            ))}
          </div>
        </div>
      )}

      <div className="config-section">
        <button onClick={onClearChat} className="btn-secondary">
          清空对话
        </button>
        <button onClick={onShowTools} className="btn-secondary">
          查看工具
        </button>
        <button onClick={onManageMCP} className="btn-secondary">
          🔌 MCP 服务
        </button>
      </div>
    </div>
  );
}

export default ConfigPanel;

