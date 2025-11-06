import React from 'react';
import './AgentSelector.css';

const AgentSelector = ({ agents, selectedAgentId, onChange }) => {
  if (!agents || agents.length === 0) {
    return (
      <div className="agent-selector-empty">
        ⚠️ 请先添加 Agent
      </div>
    );
  }

  return (
    <div className="agent-selector">
      <label>选择 Agent: *</label>
      <select value={selectedAgentId || ''} onChange={(e) => onChange(parseInt(e.target.value))}>
        <option value="">请选择一个 Agent...</option>
        {agents.map(agent => (
          <option key={agent.id} value={agent.id}>
            {agent.name} ({agent.framework.toUpperCase()})
          </option>
        ))}
      </select>
      {selectedAgentId && (
        <div className="agent-info-badge">
          {(() => {
            const agent = agents.find(a => a.id === selectedAgentId);
            return agent ? (
              <>
                <span className="badge">{agent.framework.toUpperCase()}</span>
                <span className="badge">{agent.model}</span>
                {agent.mcp_services && agent.mcp_services.length > 0 && (
                  <span className="badge">🔌 {agent.mcp_services.length} MCP</span>
                )}
              </>
            ) : null;
          })()}
        </div>
      )}
    </div>
  );
};

export default AgentSelector;

