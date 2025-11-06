import React, { useState, useEffect } from 'react';
import './AgentManageModal.css';

const AgentManageModal = ({ onClose, onAgentsChange, mcpServices = [] }) => {
  const [agents, setAgents] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [editingAgent, setEditingAgent] = useState(null);
  
  const [formData, setFormData] = useState({
    name: '',
    framework: 'react',
    description: '',
    system_prompt: '',
    max_steps: 10,
    model: 'gpt-3.5-turbo',
    mcp_services: [],
    connection_config: {},
  });

  useEffect(() => {
    loadAgents();
  }, []);

  const loadAgents = async () => {
    setLoading(true);
    setError('');
    try {
      const response = await fetch('/api/agents');
      const data = await response.json();
      setAgents(data.agents || []);
      if (onAgentsChange) {
        onAgentsChange(data.agents || []);
      }
    } catch (err) {
      setError(`加载失败: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleAdd = () => {
    setEditingAgent(null);
    setFormData({
      name: '',
      framework: 'react',
      description: '',
      system_prompt: '',
      max_steps: 10,
      model: 'gpt-3.5-turbo',
      mcp_services: [],
      connection_config: {},
    });
    setShowForm(true);
  };

  const handleEdit = (agent) => {
    setEditingAgent(agent);
    
    // 解析 connection_config
    let connConfig = {};
    if (agent.connection_config) {
      try {
        connConfig = typeof agent.connection_config === 'string' 
          ? JSON.parse(agent.connection_config)
          : agent.connection_config;
      } catch (e) {
        console.error('解析连接配置失败:', e);
      }
    }
    
    setFormData({
      name: agent.name,
      framework: agent.framework,
      description: agent.description,
      system_prompt: agent.system_prompt || '',
      max_steps: agent.max_steps,
      model: agent.model,
      mcp_services: agent.mcp_services || [],
      connection_config: connConfig,
    });
    setShowForm(true);
  };

  const handleDelete = async (id) => {
    if (!confirm('确定要删除这个 Agent 吗？')) {
      return;
    }

    setLoading(true);
    setError('');
    try {
      const response = await fetch(`/api/agents/${id}`, {
        method: 'DELETE',
      });
      const data = await response.json();
      
      if (data.success) {
        await loadAgents();
      } else {
        setError(data.message || '删除失败');
      }
    } catch (err) {
      setError(`删除失败: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      const url = editingAgent ? `/api/agents/${editingAgent.id}` : '/api/agents';
      const method = editingAgent ? 'PUT' : 'POST';
      
      // 准备数据，将 connection_config 转为 JSON 字符串
      const submitData = {
        ...formData,
        connection_config: formData.connection_config && Object.keys(formData.connection_config).length > 0
          ? JSON.stringify(formData.connection_config)
          : '',
      };
      
      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(submitData),
      });
      
      const data = await response.json();
      
      if (data.success) {
        setShowForm(false);
        await loadAgents();
      } else {
        setError(data.message || '保存失败');
      }
    } catch (err) {
      setError(`保存失败: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleMCPToggle = (serviceName) => {
    setFormData(prev => ({
      ...prev,
      mcp_services: prev.mcp_services.includes(serviceName)
        ? prev.mcp_services.filter(s => s !== serviceName)
        : [...prev.mcp_services, serviceName]
    }));
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content agent-modal" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <h2>🤖 Agent 管理</h2>
          <button className="modal-close" onClick={onClose}>×</button>
        </div>
        
        <div className="modal-body">
          {error && (
            <div className="error-message">
              ⚠️ {error}
            </div>
          )}
          
          {showForm ? (
            <form className="agent-form" onSubmit={handleSubmit}>
              {/* 基本信息 */}
              <div className="form-group">
                <label>Agent 名称</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={e => setFormData({ ...formData, name: e.target.value })}
                  required
                  placeholder="例如: 默认助手、数据分析师、代码审查专家"
                />
              </div>

              <div className="form-group">
                <label>框架类型</label>
                <select
                  value={formData.framework}
                  onChange={e => {
                    setFormData({ 
                      ...formData, 
                      framework: e.target.value,
                      connection_config: {} // 切换框架时清空连接配置
                    });
                  }}
                  required
                >
                  <option value="react">🔄 ReAct - 推理与行动循环（适合通用对话）</option>
                  <option value="plan">📋 Plan - 规划后执行（适合复杂任务）</option>
                  <option value="chain">⛓️ Chain - 链式调用（适合工作流）</option>
                  <option value="sql">🗄️ SQL - MySQL数据库查询（需配置数据库）</option>
                  <option value="elasticsearch">🔍 Elasticsearch - 日志搜索分析（需配置ES）</option>
                </select>
              </div>

              {/* SQL 连接配置 */}
              {formData.framework === 'sql' && (
                <div className="connection-config-section">
                  <h4>📊 MySQL 连接配置</h4>
                  <div className="form-row">
                    <div className="form-group">
                      <label>主机</label>
                      <input
                        type="text"
                        value={formData.connection_config.host || ''}
                        onChange={e => setFormData({
                          ...formData,
                          connection_config: { ...formData.connection_config, host: e.target.value }
                        })}
                        placeholder="localhost"
                        required
                      />
                    </div>
                    <div className="form-group">
                      <label>端口</label>
                      <input
                        type="number"
                        value={formData.connection_config.port || 3306}
                        onChange={e => setFormData({
                          ...formData,
                          connection_config: { ...formData.connection_config, port: parseInt(e.target.value) }
                        })}
                        placeholder="3306"
                        required
                      />
                    </div>
                  </div>
                  <div className="form-group">
                    <label>数据库名称</label>
                    <input
                      type="text"
                      value={formData.connection_config.database || ''}
                      onChange={e => setFormData({
                        ...formData,
                        connection_config: { ...formData.connection_config, database: e.target.value }
                      })}
                      placeholder="mydb"
                      required
                    />
                  </div>
                  <div className="form-row">
                    <div className="form-group">
                      <label>用户名</label>
                      <input
                        type="text"
                        value={formData.connection_config.username || ''}
                        onChange={e => setFormData({
                          ...formData,
                          connection_config: { ...formData.connection_config, username: e.target.value }
                        })}
                        placeholder="root"
                        required
                      />
                    </div>
                    <div className="form-group">
                      <label>密码</label>
                      <input
                        type="password"
                        value={formData.connection_config.password || ''}
                        onChange={e => setFormData({
                          ...formData,
                          connection_config: { ...formData.connection_config, password: e.target.value }
                        })}
                        placeholder="密码"
                      />
                    </div>
                  </div>
                </div>
              )}

              {/* Elasticsearch 连接配置 */}
              {formData.framework === 'elasticsearch' && (
                <div className="connection-config-section">
                  <h4>🔍 Elasticsearch 连接配置</h4>
                  <div className="form-group">
                    <label>ES 服务地址</label>
                    <input
                      type="text"
                      value={formData.connection_config.host || ''}
                      onChange={e => setFormData({
                        ...formData,
                        connection_config: { ...formData.connection_config, host: e.target.value }
                      })}
                      placeholder="http://localhost:9200"
                      required
                    />
                  </div>
                  <div className="form-row">
                    <div className="form-group">
                      <label className="optional">用户名</label>
                      <input
                        type="text"
                        value={formData.connection_config.username || ''}
                        onChange={e => setFormData({
                          ...formData,
                          connection_config: { ...formData.connection_config, username: e.target.value }
                        })}
                        placeholder="elastic (可选)"
                      />
                    </div>
                    <div className="form-group">
                      <label className="optional">密码</label>
                      <input
                        type="password"
                        value={formData.connection_config.password || ''}
                        onChange={e => setFormData({
                          ...formData,
                          connection_config: { ...formData.connection_config, password: e.target.value }
                        })}
                        placeholder="密码 (可选)"
                      />
                    </div>
                  </div>
                </div>
              )}

              <div className="form-group">
                <label className="optional">描述</label>
                <textarea
                  value={formData.description}
                  onChange={e => setFormData({ ...formData, description: e.target.value })}
                  placeholder="简要描述这个 Agent 的功能和用途..."
                  rows={2}
                />
              </div>

              <div className="form-group">
                <label className="optional">系统提示词</label>
                <textarea
                  value={formData.system_prompt}
                  onChange={e => setFormData({ ...formData, system_prompt: e.target.value })}
                  placeholder="自定义 Agent 的行为和特性，例如：&#10;你是一个专业的数据分析师，擅长：&#10;1. 数据清洗和预处理&#10;2. 统计分析和可视化&#10;3. 洞察提取和报告撰写"
                  rows={5}
                />
              </div>

              {/* 模型配置 */}
              <div className="form-row">
                <div className="form-group">
                  <label>模型</label>
                  <select
                    value={formData.model}
                    onChange={e => setFormData({ ...formData, model: e.target.value })}
                  >
                    <option value="gpt-3.5-turbo">GPT-3.5 Turbo (快速、经济)</option>
                    <option value="gpt-4">GPT-4 (强大、准确)</option>
                    <option value="gpt-4-turbo">GPT-4 Turbo (长文本)</option>
                  </select>
                </div>

                <div className="form-group">
                  <label>最大步数</label>
                  <input
                    type="number"
                    value={formData.max_steps}
                    onChange={e => setFormData({ ...formData, max_steps: parseInt(e.target.value) || 1 })}
                    min={1}
                    max={100}
                    placeholder="10"
                  />
                </div>
              </div>

              {/* MCP 服务绑定 */}
              {mcpServices.length > 0 && (
                <div className="form-group">
                  <label className="optional">绑定的 MCP 服务</label>
                  <div className="mcp-checkboxes">
                    {mcpServices.map(service => (
                      <label key={service.name} className="mcp-checkbox">
                        <input
                          type="checkbox"
                          checked={formData.mcp_services.includes(service.name)}
                          onChange={() => handleMCPToggle(service.name)}
                        />
                        <span>{service.name}</span>
                        <span className="tool-count">{service.tool_count} 工具</span>
                      </label>
                    ))}
                  </div>
                </div>
              )}

              <div className="form-actions">
                <button type="button" onClick={() => setShowForm(false)} className="btn-secondary">
                  取消
                </button>
                <button type="submit" disabled={loading} className="btn-primary">
                  {loading ? '保存中...' : '保存'}
                </button>
              </div>
            </form>
          ) : (
            <>
              <div className="agents-actions">
                <button onClick={handleAdd} className="btn-primary">
                  ➕ 添加 Agent
                </button>
              </div>

              {loading && <div className="loading">⏳ 加载中...</div>}

              <div className="agents-list">
                {agents.length === 0 ? (
                  <div className="empty-state">
                    暂无 Agent，点击上方按钮添加
                  </div>
                ) : (
                  agents.map(agent => (
                    <div key={agent.id} className="agent-card">
                      <div className="agent-header">
                        <h3>{agent.name}</h3>
                        <div className="agent-actions">
                          <button onClick={() => handleEdit(agent)} className="btn-edit">
                            ✏️ 编辑
                          </button>
                          <button onClick={() => handleDelete(agent.id)} className="btn-delete">
                            🗑️ 删除
                          </button>
                        </div>
                      </div>
                      <div className="agent-info">
                        <div className="agent-meta">
                          <span className="agent-framework">{agent.framework.toUpperCase()}</span>
                          <span className="agent-model">{agent.model}</span>
                          <span className="agent-steps">最大{agent.max_steps}步</span>
                        </div>
                        {agent.description && (
                          <p className="agent-description">{agent.description}</p>
                        )}
                        {agent.mcp_services && agent.mcp_services.length > 0 && (
                          <div className="agent-mcp-services">
                            <strong>MCP服务:</strong> {agent.mcp_services.join(', ')}
                          </div>
                        )}
                      </div>
                    </div>
                  ))
                )}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default AgentManageModal;

