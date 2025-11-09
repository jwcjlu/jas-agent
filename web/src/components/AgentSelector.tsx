import type { AgentInfo } from '../services/api';

import './AgentSelector.css';

interface AgentSelectorProps {
  agents: AgentInfo[];
  selectedAgentId: number | null;
  onChange: (id: number | null) => void;
}

const AgentSelector = ({ agents, selectedAgentId, onChange }: AgentSelectorProps): JSX.Element => {
  if (!agents || agents.length === 0) {
    return <div className="agent-selector-empty">⚠️ 请先添加 Agent</div>;
  }

  const handleChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    const value = event.target.value;
    onChange(value ? Number.parseInt(value, 10) : null);
  };

  const currentAgent = agents.find((agent) => agent.id === selectedAgentId);

  return (
    <div className="agent-selector">
      <label>选择 Agent: *</label>
      <select value={selectedAgentId ?? ''} onChange={handleChange}>
        <option value="">请选择一个 Agent...</option>
        {agents.map((agent) => (
          <option key={agent.id} value={agent.id}>
            {agent.name} ({agent.framework.toUpperCase()})
          </option>
        ))}
      </select>
      {currentAgent && (
        <div className="agent-info-badge">
          <span className="badge">{currentAgent.framework.toUpperCase()}</span>
          {currentAgent.model && <span className="badge">{currentAgent.model}</span>}
          {currentAgent.mcp_services && currentAgent.mcp_services.length > 0 && (
            <span className="badge">🔌 {currentAgent.mcp_services.length} MCP</span>
          )}
        </div>
      )}
    </div>
  );
};

export default AgentSelector;

