import { useEffect, useMemo, useState } from 'react';

import type {
  AgentConfigPayload,
  AgentFramework,
  AgentInfo,
  MCPServiceInfo,
} from '../services/api';
import {
  createAgent,
  deleteAgent as deleteAgentApi,
  getAgents,
  updateAgent,
  getMCPServices,
} from '../services/api';
import KnowledgeBaseManage from './KnowledgeBaseManage';

import './AgentManageModal.css';

interface AgentManageModalProps {
  onClose: () => void;
  onAgentsChange?: (agents: AgentInfo[]) => void;
  mcpServices?: MCPServiceInfo[];
}

type ConnectionConfig = Record<string, string | number | undefined>;

interface AgentFormData {
  name: string;
  framework: AgentFramework;
  description: string;
  system_prompt: string;
  max_steps: number;
  model: string;
  mcp_services: string[];
  connectionConfig: ConnectionConfig;
}

const defaultFormData: AgentFormData = {
  name: '',
  framework: 'react',
  description: '',
  system_prompt: '',
  max_steps: 10,
  model: 'gpt-3.5-turbo',
  mcp_services: [],
  connectionConfig: {},
};

const resolveAgentMCP = (agent?: AgentInfo | null): string[] =>
  agent?.mcp_services ?? (agent as unknown as { mcpServices?: string[] })?.mcpServices ?? [];

const AgentManageModal = ({
  onClose,
  onAgentsChange,
  mcpServices = [],
}: AgentManageModalProps): JSX.Element => {
  const [agents, setAgents] = useState<AgentInfo[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string>('');
  const [showForm, setShowForm] = useState<boolean>(false);
  const [editingAgent, setEditingAgent] = useState<AgentInfo | null>(null);
  const [formData, setFormData] = useState<AgentFormData>(defaultFormData);
  const [availableServices, setAvailableServices] = useState<MCPServiceInfo[]>(mcpServices);
  const [showKnowledgeBase, setShowKnowledgeBase] = useState<boolean>(false);
  const [selectedAgentId, setSelectedAgentId] = useState<number | null>(null);

  useEffect(() => {
    void loadAgents();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    // 优先使用外部传入；若为空则自行兜底拉取一次
    if (mcpServices && mcpServices.length > 0) {
      setAvailableServices(mcpServices);
      return;
    }
    void (async () => {
      try {
        const list = await getMCPServices();
        setAvailableServices(list ?? []);
      } catch {
        // ignore
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mcpServices?.length]);
  const loadAgents = async (): Promise<void> => {
    setLoading(true);
    setError('');
    try {
      const list = await getAgents();
      const normalized = (list ?? []).map((agent) => {
        const anyAgent = agent as unknown as {
          system_prompt?: string;
          systemPrompt?: string;
          max_steps?: number;
          maxSteps?: number;
          connection_config?: string;
          connectionConfig?: string;
        };
        return {
          ...agent,
          system_prompt: anyAgent.system_prompt ?? anyAgent.systemPrompt ?? '',
          max_steps: anyAgent.max_steps ?? anyAgent.maxSteps ?? 10,
          connection_config:
            anyAgent.connection_config ?? anyAgent.connectionConfig ?? '',
          mcp_services: resolveAgentMCP(agent),
        } as AgentInfo;
      });
      setAgents(normalized);
      onAgentsChange?.(normalized);
    } catch (err) {
      const message = err instanceof Error ? err.message : '未知错误';
      setError(`加载失败: ${message}`);
    } finally {
      setLoading(false);
    }
  };

  const resetForm = (data: Partial<AgentFormData> = defaultFormData): void => {
    setFormData({
      ...defaultFormData,
      ...data,
      connectionConfig: data.connectionConfig ?? {},
      mcp_services: data.mcp_services ?? [],
    });
  };

  const handleAdd = (): void => {
    setEditingAgent(null);
    resetForm();
    setShowForm(true);
  };

  const parseConnectionConfig = (agent: AgentInfo): ConnectionConfig => {
    const anyAgent = agent as unknown as {
      connection_config?: string | ConnectionConfig;
      connectionConfig?: string | ConnectionConfig;
    };
    const conn = anyAgent.connection_config ?? anyAgent.connectionConfig;
    if (!conn) {
      // 如果是rootcause框架，返回默认结构
      if (agent.framework === 'rootcause') {
        return {
          trace: { type: 'jaeger', baseUrl: '', username: '', password: '' },
          log: { host: '', username: '', password: '' },
        };
      }
      return {};
    }
    if (typeof conn === 'string') {
      try {
        return JSON.parse(conn) as ConnectionConfig;
      } catch (error) {
        console.error('解析连接配置失败:', error);
        return agent.framework === 'rootcause'
          ? {
              trace: { type: 'jaeger', baseUrl: '', username: '', password: '' },
              log: { host: '', username: '', password: '' },
            }
          : {};
      }
    }
    return conn as ConnectionConfig;
  };

  const handleEdit = (agent: AgentInfo): void => {
    setEditingAgent(agent);
    const anyAgent = agent as unknown as {
      system_prompt?: string;
      systemPrompt?: string;
      max_steps?: number;
      maxSteps?: number;
    };
    resetForm({
      name: agent.name,
      framework: agent.framework,
      description: agent.description ?? '',
      system_prompt: anyAgent.system_prompt ?? anyAgent.systemPrompt ?? '',
      max_steps: anyAgent.max_steps ?? anyAgent.maxSteps ?? 10,
      model: agent.model ?? 'gpt-3.5-turbo',
      mcp_services: resolveAgentMCP(agent),
      connectionConfig: parseConnectionConfig(agent),
    });
    setShowForm(true);
  };

  const handleDelete = async (id: number): Promise<void> => {
    if (!window.confirm('确定要删除这个 Agent 吗？')) return;

    setLoading(true);
    setError('');
    try {
      await deleteAgentApi(id);
      await loadAgents();
    } catch (err) {
      const message = err instanceof Error ? err.message : '未知错误';
      setError(`删除失败: ${message}`);
    } finally {
      setLoading(false);
    }
  };

  const buildPayload = (): AgentConfigPayload => {
    const connectionConfig =
      Object.keys(formData.connectionConfig).length > 0
        ? JSON.stringify(formData.connectionConfig)
        : '';

    return {
      name: formData.name,
      framework: formData.framework,
      description: formData.description,
      system_prompt: formData.system_prompt,
      max_steps: formData.max_steps,
      model: formData.model,
      mcp_services: formData.mcp_services,
      connection_config: connectionConfig,
    };
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>): Promise<void> => {
    event.preventDefault();
    setLoading(true);
    setError('');

    try {
      // 限制新增 chain/plan
      if (!editingAgent && (formData.framework === 'chain' || formData.framework === 'plan')) {
        setError('当前不支持新增 Chain 或 Plan 框架的 Agent');
        setLoading(false);
        return;
      }
      const payload = buildPayload();
      if (editingAgent) {
        await updateAgent(editingAgent.id, payload);
      } else {
        await createAgent(payload);
      }
      setShowForm(false);
      await loadAgents();
    } catch (err) {
      const message = err instanceof Error ? err.message : '未知错误';
      setError(`保存失败: ${message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleMCPToggle = (serviceName: string): void => {
    setFormData((prev) => ({
      ...prev,
      mcp_services: prev.mcp_services.includes(serviceName)
        ? prev.mcp_services.filter((s) => s !== serviceName)
        : [...prev.mcp_services, serviceName],
    }));
  };

  const connectionConfigInputs = useMemo(() => {
    if (formData.framework === 'sql') {
      return (
        <div className="connection-config-section">
          <h4>📊 MySQL 连接配置</h4>
          <div className="form-row">
            <div className="form-group">
              <label>主机</label>
              <input
                type="text"
                value={(formData.connectionConfig.host as string) ?? ''}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    connectionConfig: { ...prev.connectionConfig, host: e.target.value },
                  }))
                }
                placeholder="localhost"
                required
              />
            </div>
            <div className="form-group">
              <label>端口</label>
              <input
                type="number"
                value={Number(formData.connectionConfig.port) || 3306}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    connectionConfig: {
                      ...prev.connectionConfig,
                      port: Number.parseInt(e.target.value, 10),
                    },
                  }))
                }
                placeholder="3306"
                required
              />
            </div>
          </div>
          <div className="form-group">
            <label>数据库名称</label>
            <input
              type="text"
              value={(formData.connectionConfig.database as string) ?? ''}
              onChange={(e) =>
                setFormData((prev) => ({
                  ...prev,
                  connectionConfig: { ...prev.connectionConfig, database: e.target.value },
                }))
              }
              placeholder="mydb"
              required
            />
          </div>
          <div className="form-row">
            <div className="form-group">
              <label>用户名</label>
              <input
                type="text"
                value={(formData.connectionConfig.username as string) ?? ''}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    connectionConfig: { ...prev.connectionConfig, username: e.target.value },
                  }))
                }
                placeholder="root"
                required
              />
            </div>
            <div className="form-group">
              <label>密码</label>
              <input
                type="password"
                value={(formData.connectionConfig.password as string) ?? ''}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    connectionConfig: { ...prev.connectionConfig, password: e.target.value },
                  }))
                }
                placeholder="密码"
              />
            </div>
          </div>
        </div>
      );
    }

    if (formData.framework === 'elasticsearch') {
      return (
        <div className="connection-config-section">
          <h4>🔍 Elasticsearch 连接配置</h4>
          <div className="form-group">
            <label>ES 服务地址</label>
            <input
              type="text"
              value={(formData.connectionConfig.host as string) ?? ''}
              onChange={(e) =>
                setFormData((prev) => ({
                  ...prev,
                  connectionConfig: { ...prev.connectionConfig, host: e.target.value },
                }))
              }
              placeholder="http://localhost:9200"
              required
            />
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="optional">用户名</label>
              <input
                type="text"
                value={(formData.connectionConfig.username as string) ?? ''}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    connectionConfig: { ...prev.connectionConfig, username: e.target.value },
                  }))
                }
                placeholder="elastic (可选)"
              />
            </div>
            <div className="form-group">
              <label className="optional">密码</label>
              <input
                type="password"
                value={(formData.connectionConfig.password as string) ?? ''}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    connectionConfig: { ...prev.connectionConfig, password: e.target.value },
                  }))
                }
                placeholder="密码 (可选)"
              />
            </div>
          </div>
        </div>
      );
    }

    if (formData.framework === 'rootcause') {
      const traceConfig = (formData.connectionConfig.trace as Record<string, string>) ?? {};
      const logConfig = (formData.connectionConfig.log as Record<string, string>) ?? {};
      
      return (
        <div className="connection-config-section">
          <h4>🔍 根因分析配置</h4>
          
          {/* Trace配置 */}
          <div className="config-subsection">
            <h5>📊 Trace 配置 (Jaeger/SkyWalking)</h5>
            <div className="form-group">
              <label>Trace 类型</label>
              <select
                value={traceConfig.type ?? 'jaeger'}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    connectionConfig: {
                      ...prev.connectionConfig,
                      trace: {
                        ...traceConfig,
                        type: e.target.value,
                      },
                    },
                  }))
                }
                required
              >
                <option value="jaeger">Jaeger</option>
                <option value="skywalking">SkyWalking</option>
              </select>
            </div>
            <div className="form-group">
              <label>Trace 服务地址</label>
              <input
                type="text"
                value={traceConfig.baseUrl ?? ''}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    connectionConfig: {
                      ...prev.connectionConfig,
                      trace: {
                        ...traceConfig,
                        baseUrl: e.target.value,
                      },
                    },
                  }))
                }
                placeholder="http://localhost:16686 (Jaeger) 或 http://localhost:12800 (SkyWalking)"
                required
              />
            </div>
            <div className="form-row">
              <div className="form-group">
                <label className="optional">用户名</label>
                <input
                  type="text"
                  value={traceConfig.username ?? ''}
                  onChange={(e) =>
                    setFormData((prev) => ({
                      ...prev,
                      connectionConfig: {
                        ...prev.connectionConfig,
                        trace: {
                          ...traceConfig,
                          username: e.target.value,
                        },
                      },
                    }))
                  }
                  placeholder="用户名 (可选)"
                />
              </div>
              <div className="form-group">
                <label className="optional">密码</label>
                <input
                  type="password"
                  value={traceConfig.password ?? ''}
                  onChange={(e) =>
                    setFormData((prev) => ({
                      ...prev,
                      connectionConfig: {
                        ...prev.connectionConfig,
                        trace: {
                          ...traceConfig,
                          password: e.target.value,
                        },
                      },
                    }))
                  }
                  placeholder="密码 (可选)"
                />
              </div>
            </div>
          </div>

          {/* 日志配置 */}
          <div className="config-subsection">
            <h5>📝 日志配置 (Elasticsearch)</h5>
            <div className="form-group">
              <label>ES 服务地址</label>
              <input
                type="text"
                value={logConfig.host ?? ''}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    connectionConfig: {
                      ...prev.connectionConfig,
                      log: {
                        ...logConfig,
                        host: e.target.value,
                      },
                    },
                  }))
                }
                placeholder="http://localhost:9200"
                required
              />
            </div>
            <div className="form-row">
              <div className="form-group">
                <label className="optional">用户名</label>
                <input
                  type="text"
                  value={logConfig.username ?? ''}
                  onChange={(e) =>
                    setFormData((prev) => ({
                      ...prev,
                      connectionConfig: {
                        ...prev.connectionConfig,
                        log: {
                          ...logConfig,
                          username: e.target.value,
                        },
                      },
                    }))
                  }
                  placeholder="elastic (可选)"
                />
              </div>
              <div className="form-group">
                <label className="optional">密码</label>
                <input
                  type="password"
                  value={logConfig.password ?? ''}
                  onChange={(e) =>
                    setFormData((prev) => ({
                      ...prev,
                      connectionConfig: {
                        ...prev.connectionConfig,
                        log: {
                          ...logConfig,
                          password: e.target.value,
                        },
                      },
                    }))
                  }
                  placeholder="密码 (可选)"
                />
              </div>
            </div>
          </div>
        </div>
      );
    }

    return null;
  }, [formData.connectionConfig, formData.framework]);

  // 显示知识库管理
  if (showKnowledgeBase && selectedAgentId) {
    return (
      <div className="modal-overlay" onClick={() => setShowKnowledgeBase(false)}>
        <div className="modal-content agent-modal" onClick={(e) => e.stopPropagation()}>
          <div className="modal-header">
            <h2>📚 知识库管理</h2>
            <button className="modal-close" onClick={() => setShowKnowledgeBase(false)}>
              ×
            </button>
          </div>
          <div className="modal-body">
            <KnowledgeBaseManage
              agentId={selectedAgentId}
              onClose={() => setShowKnowledgeBase(false)}
            />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content agent-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>🤖 Agent 管理</h2>
          <button className="modal-close" onClick={onClose}>
            ×
          </button>
        </div>

        <div className="modal-body">
          {error && <div className="error-message">⚠️ {error}</div>}

          {showForm ? (
            <form className="agent-form" onSubmit={handleSubmit}>
              <div className="form-group">
                <label>Agent 名称</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData((prev) => ({ ...prev, name: e.target.value }))}
                  required
                  placeholder="例如: 默认助手、数据分析师、代码审查专家"
                />
              </div>

              <div className="form-group">
                <label>框架类型</label>
                <select
                  value={formData.framework}
                  onChange={(e) => {
                    const framework = e.target.value as AgentFramework;
                    setFormData({
                      ...defaultFormData,
                      ...formData,
                      framework,
                      connectionConfig: {},
                    });
                  }}
                  required
                >
                  <option value="react">🔄 ReAct - 推理与行动循环（适合通用对话）</option>
                  <option value="plan" disabled={!editingAgent}>📋 Plan - 规划后执行（暂不支持新增）</option>
                  <option value="chain" disabled={!editingAgent}>⛓️ Chain - 链式调用（暂不支持新增）</option>
                  <option value="sql">🗄️ SQL - MySQL数据库查询（需配置数据库）</option>
                  <option value="elasticsearch">🔍 Elasticsearch - 日志搜索分析（需配置ES）</option>
                  <option value="rootcause">🔍 Root Cause - 智能故障根因分析（需配置Trace和日志）</option>
                </select>
              </div>

              {connectionConfigInputs}

              {formData.mcp_services.length > 0 && (
                <div className="selected-mcps">
                  <span className="label">已绑定 MCP:</span>
                  <div className="chips">
                    {formData.mcp_services.map((name) => (
                      <span key={name} className="chip">🔌 {name}</span>
                    ))}
                  </div>
                </div>
              )}

              <div className="form-group">
                <label className="optional">描述</label>
                <textarea
                  value={formData.description}
                  onChange={(e) =>
                    setFormData((prev) => ({ ...prev, description: e.target.value }))
                  }
                  placeholder="简要描述这个 Agent 的功能和用途..."
                  rows={2}
                />
              </div>

              <div className="form-group">
                <label className="optional">系统提示词</label>
                <textarea
                  value={formData.system_prompt}
                  onChange={(e) =>
                    setFormData((prev) => ({ ...prev, system_prompt: e.target.value }))
                  }
                  placeholder="自定义 Agent 的行为和特性..."
                  rows={5}
                />
              </div>

              <div className="form-row">
                <div className="form-group">
                  <label>模型</label>
                  <select
                    value={formData.model}
                    onChange={(e) =>
                      setFormData((prev) => ({ ...prev, model: e.target.value }))
                    }
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
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        max_steps: Number.parseInt(e.target.value, 10) || 1,
                      }))
                    }
                    min={1}
                    max={100}
                    placeholder="10"
                  />
                </div>
              </div>

              {availableServices.length > 0 && (
                <div className="form-group">
                  <label className="optional">绑定的 MCP 服务</label>
                  <div className="mcp-checkboxes">
                    {availableServices.map((service) => (
                      <label key={service.name} className="mcp-checkbox">
                        <input
                          type="checkbox"
                          checked={formData.mcp_services.includes(service.name)}
                          onChange={() => handleMCPToggle(service.name)}
                        />
                        <span>{service.name}</span>
                        <span className="tool-count">
                          {(service as unknown as { tool_count?: number; toolCount?: number }).tool_count ??
                            (service as unknown as { tool_count?: number; toolCount?: number }).toolCount ??
                            0}{' '}
                          工具
                        </span>
                      </label>
                    ))}
                  </div>
                </div>
              )}

              {availableServices.length === 0 && (
                <div className="form-group">
                  <label className="optional">绑定的 MCP 服务</label>
                  <div className="empty-state">暂无可用 MCP 服务，请先通过“🔌 MCP 管理”添加。</div>
                </div>
              )}

              <div className="form-actions">
                <button type="button" onClick={() => setShowForm(false)} className="btn-secondary">
                  取消
                </button>
                <button type="submit" className="btn-primary" disabled={loading}>
                  {loading ? '保存中...' : '保存'}
                </button>
              </div>
            </form>
          ) : (
            <>
              <div className="actions">
                <button className="btn-primary" onClick={handleAdd}>
                  ➕ 新建 Agent
                </button>
              </div>

              {loading ? (
                <div className="loading">加载中...</div>
              ) : (
                <div className="agent-list">
                  {agents.length === 0 ? (
                    <div className="empty-state">暂无 Agent，请点击“新建 Agent”创建。</div>
                  ) : (
                    agents.map((agent) => (
                      <div key={agent.id} className="agent-card">
                        <div className="agent-info">
                          <h3>{agent.name}</h3>
                          <p className="framework">框架: {agent.framework}</p>
                          {agent.description && <p className="description">{agent.description}</p>}
                        {resolveAgentMCP(agent).length > 0 && (
                            <p className="mcp-list">
                            MCP: {resolveAgentMCP(agent).join(', ')}
                            </p>
                          )}
                        </div>
                        <div className="agent-actions">
                          <button className="btn-secondary" onClick={() => handleEdit(agent)}>
                            编辑
                          </button>
                          <button
                            className="btn-secondary"
                            onClick={() => {
                              setSelectedAgentId(agent.id);
                              setShowKnowledgeBase(true);
                            }}
                          >
                            📚 知识库
                          </button>
                          <button
                            className="btn-danger"
                            onClick={() => handleDelete(agent.id)}
                            disabled={loading}
                          >
                            删除
                          </button>
                        </div>
                      </div>
                    ))
                  )}
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default AgentManageModal;

