import type { AgentInfo } from '../services/api';

import './AgentSelector.css';

interface AgentSelectorProps {
  agents: AgentInfo[];
  selectedAgentId: number | null;
  onChange: (id: number | null) => void;
  mcpToolMap?: Record<string, { name: string; description?: string }[]>;
}

const AgentSelector = ({
  agents,
  selectedAgentId,
  onChange,
  mcpToolMap = {},
}: AgentSelectorProps): JSX.Element => {
  if (!agents || agents.length === 0) {
    return <div className="agent-selector-empty">⚠️ 请先添加 Agent</div>;
  }

  const handleChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    const value = event.target.value;
    onChange(value ? Number.parseInt(value, 10) : null);
  };

  const resolveMCP = (agent?: AgentInfo) =>
    agent?.mcp_services ?? (agent as unknown as { mcpServices?: string[] })?.mcpServices ?? [];

  const currentAgent = agents.find((agent) => agent.id === selectedAgentId);
  const currentMCPs = resolveMCP(currentAgent);
  const getTooltip = (service: string): string => {
    const tools = mcpToolMap[service] ?? [];
    if (tools.length === 0) return service;
    const items = tools
      .map((tool) =>
        tool.description && tool.description.trim().length > 0
          ? `• ${tool.name}: ${tool.description}`
          : `• ${tool.name}`,
      )
      .join('\n');
    return `${service}\n${items}`;
  };

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
          {currentMCPs.length > 0 && (
            <>
              <span className="badge">🔌 {currentMCPs.length} MCP</span>
              <div className="badge-group">
                {currentMCPs.slice(0, 3).map((service) => (
                  <span key={service} className="badge" title={getTooltip(service)}>
                    {service}
                  </span>
                ))}
                {currentMCPs.length > 3 && (
                  <span className="badge" title={currentMCPs.slice(3).map(getTooltip).join('\n\n')}>
                    … {currentMCPs.length - 3} more
                  </span>
                )}
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
};

export default AgentSelector;

