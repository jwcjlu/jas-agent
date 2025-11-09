import { useEffect, useState } from 'react';

import {
  addMCPService,
  getMCPServices,
  removeMCPService,
  type MCPServiceInfo,
  type MCPServiceResponse,
} from '../services/api';

import './MCPManageModal.css';

interface MCPManageModalProps {
  onClose: () => void;
  onServicesChange?: (services: MCPServiceInfo[]) => void;
}

interface FeedbackMessage {
  type: 'success' | 'error' | '';
  text: string;
}

const MCPManageModal = ({ onClose, onServicesChange }: MCPManageModalProps): JSX.Element => {
  const [services, setServices] = useState<MCPServiceInfo[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [adding, setAdding] = useState<boolean>(false);
  const [newService, setNewService] = useState<{ name: string; endpoint: string }>({
    name: '',
    endpoint: '',
  });
  const [message, setMessage] = useState<FeedbackMessage>({ type: '', text: '' });
  const [editingName, setEditingName] = useState<string | null>(null);
  const [editValues, setEditValues] = useState<{ name: string; endpoint: string }>({
    name: '',
    endpoint: '',
  });

  useEffect(() => {
    void loadServices();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const loadServices = async (): Promise<void> => {
    try {
      const servicesList = await getMCPServices();
      setServices(servicesList ?? []);
      onServicesChange?.(servicesList ?? []);
    } catch (error) {
      console.error('加载MCP服务失败:', error);
      setMessage({ type: 'error', text: '加载失败' });
    } finally {
      setLoading(false);
    }
  };

  const handleAddService = async (): Promise<void> => {
    if (!newService.name.trim() || !newService.endpoint.trim()) {
      setMessage({ type: 'error', text: '请填写服务名称和端点' });
      return;
    }

    setAdding(true);
    setMessage({ type: '', text: '' });

    try {
      const result: MCPServiceResponse = await addMCPService(
        newService.name.trim(),
        newService.endpoint.trim(),
      );

      if (result.success) {
        setMessage({ type: 'success', text: result.message ?? '添加成功' });
        setNewService({ name: '', endpoint: '' });
        await loadServices();
      } else {
        setMessage({ type: 'error', text: result.message ?? '添加失败' });
      }
    } catch (error) {
      const text = error instanceof Error ? error.message : '未知错误';
      setMessage({ type: 'error', text: `添加失败: ${text}` });
    } finally {
      setAdding(false);
    }
  };

  const handleRemoveService = async (name: string): Promise<void> => {
    if (!window.confirm(`确定要移除MCP服务 "${name}" 吗？`)) return;

    try {
      const result = await removeMCPService(name);
      if (result.success) {
        setMessage({ type: 'success', text: result.message ?? '移除成功' });
        await loadServices();
      } else {
        setMessage({ type: 'error', text: result.message ?? '移除失败' });
      }
    } catch (error) {
      const text = error instanceof Error ? error.message : '未知错误';
      setMessage({ type: 'error', text: `移除失败: ${text}` });
    }
  };

  const handleStartEdit = (service: MCPServiceInfo): void => {
    setEditingName(service.name);
    setEditValues({ name: service.name, endpoint: service.endpoint });
    setMessage({ type: '', text: '' });
  };

  const handleCancelEdit = (): void => {
    setEditingName(null);
    setEditValues({ name: '', endpoint: '' });
  };

  const handleSaveEdit = async (originalName: string): Promise<void> => {
    if (!editValues.name.trim() || !editValues.endpoint.trim()) {
      setMessage({ type: 'error', text: '请填写完整的名称与端点' });
      return;
    }
    try {
      // 后端暂未提供更新接口，这里采用“移除后新增”的方式模拟更新
      if (originalName !== editValues.name) {
        const confirmRename = window.confirm(
          `将把服务名由 "${originalName}" 重命名为 "${editValues.name}"，确认继续？`,
        );
        if (!confirmRename) return;
      }
      await removeMCPService(originalName);
      const result = await addMCPService(editValues.name.trim(), editValues.endpoint.trim());
      if (result.success) {
        setMessage({ type: 'success', text: result.message ?? '更新成功' });
        setEditingName(null);
        setEditValues({ name: '', endpoint: '' });
        await loadServices();
      } else {
        setMessage({ type: 'error', text: result.message ?? '更新失败' });
      }
    } catch (error) {
      const text = error instanceof Error ? error.message : '未知错误';
      setMessage({ type: 'error', text: `更新失败: ${text}` });
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content mcp-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>🔌 MCP 服务管理</h2>
          <button className="close-btn" onClick={onClose}>
            &times;
          </button>
        </div>

        <div className="modal-body">
          {message.text && <div className={`message-alert ${message.type}`}>{message.text}</div>}

          <div className="add-service-form">
            <h3>添加 MCP 服务</h3>
            <div className="form-group">
              <label>服务名称:</label>
              <input
                type="text"
                value={newService.name}
                onChange={(e) => setNewService((prev) => ({ ...prev, name: e.target.value }))}
                placeholder="例如: weather-mcp"
              />
            </div>
            <div className="form-group">
              <label>服务端点:</label>
              <input
                type="text"
                value={newService.endpoint}
                onChange={(e) => setNewService((prev) => ({ ...prev, endpoint: e.target.value }))}
                placeholder="例如: http://localhost:8080/mcp"
              />
            </div>
            <button
              onClick={handleAddService}
              disabled={adding || !newService.name || !newService.endpoint}
              className="btn-primary"
            >
              {adding ? '添加中...' : '添加服务'}
            </button>
          </div>

          <div className="services-list">
            <h3>已添加的 MCP 服务 ({services.length})</h3>

            {loading ? (
              <p>加载中...</p>
            ) : services.length > 0 ? (
              services.map((service) => {
                const isActive =
                  service.is_active ??
                  (service as unknown as { active?: boolean }).active ??
                  false;
                return (
                  <div key={service.name} className="service-item">
                    <div className="service-header">
                      <div>
                        <h4>
                          {editingName === service.name ? (
                            <input
                              type="text"
                              value={editValues.name}
                              onChange={(e) =>
                                setEditValues((prev) => ({ ...prev, name: e.target.value }))
                              }
                              placeholder="服务名称"
                            />
                          ) : (
                            service.name
                          )}
                        </h4>
                        <span className={`status-badge ${isActive ? 'active' : 'inactive'}`}>
                          {isActive ? '✅ 活跃' : '⚠️ 未激活'}
                        </span>
                      </div>
                      <div className="service-actions">
                        {editingName === service.name ? (
                          <>
                            <button
                              className="btn-primary-small"
                              onClick={() => void handleSaveEdit(service.name)}
                            >
                              保存
                            </button>
                            <button className="btn-secondary-small" onClick={handleCancelEdit}>
                              取消
                            </button>
                          </>
                        ) : (
                          <>
                            <button
                              className="btn-secondary-small"
                              onClick={() => handleStartEdit(service)}
                            >
                              编辑
                            </button>
                            <button
                              onClick={() => void handleRemoveService(service.name)}
                              className="btn-danger-small"
                              title="移除服务"
                            >
                              移除
                            </button>
                          </>
                        )}
                      </div>
                    </div>
                    <div className="service-details">
                      {editingName === service.name ? (
                        <div className="form-group">
                          <label>服务端点:</label>
                          <input
                            type="text"
                            value={editValues.endpoint}
                            onChange={(e) =>
                              setEditValues((prev) => ({ ...prev, endpoint: e.target.value }))
                            }
                            placeholder="例如: http://localhost:8080/mcp"
                          />
                        </div>
                      ) : (
                        <p>
                          <strong>端点:</strong> {service.endpoint}
                        </p>
                      )}
                      <p>
                        <strong>工具数量:</strong> {service.tool_count ?? 0}
                      </p>
                      <p>
                        <strong>创建时间:</strong> {service.created_at ?? '-'}
                      </p>
                      <p>
                        <strong>最后刷新:</strong> {service.last_refresh ?? '-'}
                      </p>
                    </div>
                  </div>
                );
              })
            ) : (
              <div className="empty-state">
                <p>暂无MCP服务</p>
                <p className="hint">添加MCP服务后，可以使用更多外部工具</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default MCPManageModal;

