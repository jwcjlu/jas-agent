import { useState, useEffect } from 'react';
import { getMCPServices, addMCPService, removeMCPService } from '../services/api';
import './MCPManageModal.css';

function MCPManageModal({ onClose, onServicesChange }) {
  const [services, setServices] = useState([]);
  const [loading, setLoading] = useState(true);
  const [adding, setAdding] = useState(false);
  const [newService, setNewService] = useState({ name: '', endpoint: '' });
  const [message, setMessage] = useState({ type: '', text: '' });

  useEffect(() => {
    loadServices();
  }, []);

  const loadServices = async () => {
    try {
      const servicesList = await getMCPServices();
      setServices(servicesList || []);
      if (onServicesChange) {
        onServicesChange(servicesList || []);
      }
    } catch (error) {
      console.error('加载MCP服务失败:', error);
      setMessage({ type: 'error', text: '加载失败' });
    } finally {
      setLoading(false);
    }
  };

  const handleAddService = async () => {
    if (!newService.name || !newService.endpoint) {
      setMessage({ type: 'error', text: '请填写服务名称和端点' });
      return;
    }

    setAdding(true);
    setMessage({ type: '', text: '' });

    try {
      const result = await addMCPService(newService.name, newService.endpoint);
      
      if (result.success) {
        setMessage({ type: 'success', text: result.message });
        setNewService({ name: '', endpoint: '' });
        await loadServices();
      } else {
        setMessage({ type: 'error', text: result.message });
      }
    } catch (error) {
      setMessage({ type: 'error', text: `添加失败: ${error.message}` });
    } finally {
      setAdding(false);
    }
  };

  const handleRemoveService = async (name) => {
    if (!confirm(`确定要移除MCP服务 "${name}" 吗？`)) {
      return;
    }

    try {
      const result = await removeMCPService(name);
      
      if (result.success) {
        setMessage({ type: 'success', text: result.message });
        await loadServices();
      } else {
        setMessage({ type: 'error', text: result.message });
      }
    } catch (error) {
      setMessage({ type: 'error', text: `移除失败: ${error.message}` });
    }
  };

  return (
    <div className="modal" onClick={onClose}>
      <div className="modal-content mcp-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>🔌 MCP 服务管理</h2>
          <button className="close-btn" onClick={onClose}>
            &times;
          </button>
        </div>
        
        <div className="modal-body">
          {/* 消息提示 */}
          {message.text && (
            <div className={`message-alert ${message.type}`}>
              {message.text}
            </div>
          )}

          {/* 添加新服务表单 */}
          <div className="add-service-form">
            <h3>添加 MCP 服务</h3>
            <div className="form-group">
              <label>服务名称:</label>
              <input
                type="text"
                value={newService.name}
                onChange={(e) => setNewService({ ...newService, name: e.target.value })}
                placeholder="例如: weather-mcp"
              />
            </div>
            <div className="form-group">
              <label>服务端点:</label>
              <input
                type="text"
                value={newService.endpoint}
                onChange={(e) => setNewService({ ...newService, endpoint: e.target.value })}
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

          {/* MCP 服务列表 */}
          <div className="services-list">
            <h3>已添加的 MCP 服务 ({services.length})</h3>
            
            {loading ? (
              <p>加载中...</p>
            ) : services.length > 0 ? (
              services.map((service) => (
                <div key={service.name} className="service-item">
                  <div className="service-header">
                    <div>
                      <h4>{service.name}</h4>
                      <span className={`status-badge ${service.active ? 'active' : 'inactive'}`}>
                        {service.active ? '✅ 活跃' : '⚠️ 未激活'}
                      </span>
                    </div>
                    <button
                      onClick={() => handleRemoveService(service.name)}
                      className="btn-danger-small"
                      title="移除服务"
                    >
                      🗑️ 移除
                    </button>
                  </div>
                  <div className="service-details">
                    <p><strong>端点:</strong> {service.endpoint}</p>
                    <p><strong>工具数量:</strong> {service.tool_count}</p>
                    <p><strong>创建时间:</strong> {service.created_at}</p>
                    <p><strong>最后刷新:</strong> {service.last_refresh}</p>
                  </div>
                </div>
              ))
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
}

export default MCPManageModal;

