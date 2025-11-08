import { useEffect, useState } from 'react';

import { getTools, type ToolInfo } from '../services/api';

import './ToolsModal.css';

interface ToolsModalProps {
  onClose: () => void;
}

const ToolsModal = ({ onClose }: ToolsModalProps): JSX.Element => {
  const [tools, setTools] = useState<ToolInfo[]>([]);
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    void loadTools();
  }, []);

  const loadTools = async (): Promise<void> => {
    try {
      const toolsList = await getTools();
      setTools(toolsList ?? []);
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
            tools.map((tool) => (
              <div key={tool.name} className="tool-item">
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
};

export default ToolsModal;

