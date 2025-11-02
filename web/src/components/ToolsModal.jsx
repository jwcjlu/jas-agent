import { useState, useEffect } from 'react';
import { getTools } from '../services/api';
import './ToolsModal.css';

function ToolsModal({ onClose }) {
  const [tools, setTools] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadTools();
  }, []);

  const loadTools = async () => {
    try {
      const toolsList = await getTools();
      setTools(toolsList);
    } catch (error) {
      console.error('加载工具列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="modal" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>🛠️ 可用工具</h2>
          <button className="close-btn" onClick={onClose}>
            &times;
          </button>
        </div>
        <div className="modal-body">
          {loading ? (
            <p>加载中...</p>
          ) : tools.length > 0 ? (
            tools.map((tool, index) => (
              <div key={index} className="tool-item">
                <h3>
                  {tool.name}
                  <span className="tool-type">{tool.type}</span>
                </h3>
                <p>{tool.description}</p>
              </div>
            ))
          ) : (
            <p>暂无可用工具</p>
          )}
        </div>
      </div>
    </div>
  );
}

export default ToolsModal;

