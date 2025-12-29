import { useEffect, useState } from 'react';
import type { KnowledgeBaseInfo, DocumentInfo } from '../services/api';
import {
  getKnowledgeBaseByAgent,
  createKnowledgeBase,
  updateKnowledgeBase,
  deleteKnowledgeBase,
  listDocuments,
  deleteDocument,
} from '../services/api';
import './KnowledgeBaseManage.css';

interface KnowledgeBaseManageProps {
  agentId: number;
  onClose?: () => void;
}

const KnowledgeBaseManage = ({ agentId, onClose }: KnowledgeBaseManageProps): JSX.Element => {
  const [knowledgeBase, setKnowledgeBase] = useState<KnowledgeBaseInfo | null>(null);
  const [documents, setDocuments] = useState<DocumentInfo[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string>('');
  const [showForm, setShowForm] = useState<boolean>(false);
  const [uploading, setUploading] = useState<boolean>(false);

  // 表单数据
  const [formData, setFormData] = useState<Partial<KnowledgeBaseInfo>>({
    name: '',
    description: '',
    embedding_model: 'text-embedding-3-small',
    chunk_size: 800,
    chunk_overlap: 120,
    vector_store_type: 'memory',
    vector_store_config: '{}',
    is_active: true,
  });

  useEffect(() => {
    if (agentId > 0) {
      void loadKnowledgeBase();
      void loadDocuments();
    }
  }, [agentId]);

  const loadKnowledgeBase = async (): Promise<void> => {
    setLoading(true);
    setError('');
    try {
      const kb = await getKnowledgeBaseByAgent(agentId);
      setKnowledgeBase(kb);
      if (kb) {
        setFormData({
          name: kb.name ?? '',
          description: kb.description ?? '',
          embedding_model: kb.embedding_model ?? 'text-embedding-3-small',
          chunk_size: kb.chunk_size ?? 800,
          chunk_overlap: kb.chunk_overlap ?? 120,
          vector_store_type: kb.vector_store_type ?? 'memory',
          vector_store_config: kb.vector_store_config ?? '{}',
          is_active: kb.is_active ?? true,
        });
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : '未知错误';
      setError(`加载失败: ${message}`);
    } finally {
      setLoading(false);
    }
  };

  const loadDocuments = async (): Promise<void> => {
    if (!knowledgeBase?.id) return;
    try {
      const docs = await listDocuments(knowledgeBase.id);
      setDocuments(docs);
    } catch (err) {
      console.error('加载文档失败:', err);
    }
  };

  useEffect(() => {
    if (knowledgeBase?.id) {
      void loadDocuments();
    }
  }, [knowledgeBase?.id]);

  const handleSave = async (): Promise<void> => {
    setLoading(true);
    setError('');
    try {
      if (knowledgeBase) {
        // 更新
        const updated = await updateKnowledgeBase(knowledgeBase.id, {
          id: knowledgeBase.id,
          agent_id: agentId,
          ...formData,
        } as any);
        setKnowledgeBase(updated.knowledge_base ?? null);
      } else {
        // 创建
        const created = await createKnowledgeBase({
          agent_id: agentId,
          ...formData,
        } as any);
        setKnowledgeBase(created.knowledge_base ?? null);
      }
      setShowForm(false);
    } catch (err) {
      const message = err instanceof Error ? err.message : '未知错误';
      setError(`保存失败: ${message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (): Promise<void> => {
    if (!knowledgeBase || !confirm('确定要删除知识库吗？这将删除所有关联的文档。')) {
      return;
    }
    setLoading(true);
    setError('');
    try {
      await deleteKnowledgeBase(knowledgeBase.id);
      setKnowledgeBase(null);
      setDocuments([]);
    } catch (err) {
      const message = err instanceof Error ? err.message : '未知错误';
      setError(`删除失败: ${message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteDocument = async (docId: number): Promise<void> => {
    if (!confirm('确定要删除该文档吗？')) {
      return;
    }
    try {
      await deleteDocument(docId);
      await loadDocuments();
    } catch (err) {
      const message = err instanceof Error ? err.message : '未知错误';
      setError(`删除文档失败: ${message}`);
    }
  };

  const handleFileUpload = async (event: React.ChangeEvent<HTMLInputElement>): Promise<void> => {
    const files = event.target.files;
    if (!files || files.length === 0 || !knowledgeBase) return;

    setUploading(true);
    setError('');

    try {
      // TODO: 实现文件上传逻辑
      // 1. 上传文件到服务器
      // 2. 解析文档
      // 3. 存储到向量数据库
      alert('文档上传功能开发中...');
    } catch (err) {
      const message = err instanceof Error ? err.message : '未知错误';
      setError(`上传失败: ${message}`);
    } finally {
      setUploading(false);
      // 重置文件输入
      event.target.value = '';
    }
  };

  if (showForm) {
    return (
      <div className="knowledge-base-form">
        <div className="form-header">
          <h3>{knowledgeBase ? '编辑知识库' : '创建知识库'}</h3>
          <button className="btn-close" onClick={() => setShowForm(false)}>✕</button>
        </div>

        {error && <div className="error-message">{error}</div>}

        <div className="form-body">
          <div className="form-group">
            <label>知识库名称 *</label>
            <input
              type="text"
              value={formData.name ?? ''}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              placeholder="例如：技术文档库"
              required
            />
          </div>

          <div className="form-group">
            <label className="optional">描述</label>
            <textarea
              value={formData.description ?? ''}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              placeholder="知识库的描述信息"
              rows={3}
            />
          </div>

          <div className="form-row">
            <div className="form-group">
              <label className="optional">嵌入模型</label>
              <select
                value={formData.embedding_model ?? 'text-embedding-3-small'}
                onChange={(e) => setFormData({ ...formData, embedding_model: e.target.value })}
              >
                <option value="text-embedding-3-small">text-embedding-3-small (1536维)</option>
                <option value="text-embedding-3-large">text-embedding-3-large (3072维)</option>
                <option value="text-embedding-ada-002">text-embedding-ada-002 (1536维)</option>
              </select>
            </div>

            <div className="form-group">
              <label className="optional">向量存储类型</label>
              <select
                value={formData.vector_store_type ?? 'memory'}
                onChange={(e) => setFormData({ ...formData, vector_store_type: e.target.value })}
              >
                <option value="memory">内存存储</option>
                <option value="milvus">Milvus</option>
              </select>
            </div>
          </div>

          <div className="form-row">
            <div className="form-group">
              <label className="optional">分块大小</label>
              <input
                type="number"
                value={formData.chunk_size ?? 800}
                onChange={(e) =>
                  setFormData({ ...formData, chunk_size: Number.parseInt(e.target.value, 10) })
                }
                min={100}
                max={5000}
                placeholder="800"
              />
            </div>

            <div className="form-group">
              <label className="optional">重叠大小</label>
              <input
                type="number"
                value={formData.chunk_overlap ?? 120}
                onChange={(e) =>
                  setFormData({ ...formData, chunk_overlap: Number.parseInt(e.target.value, 10) })
                }
                min={0}
                max={500}
                placeholder="120"
              />
            </div>
          </div>

          <div className="form-group">
            <label>
              <input
                type="checkbox"
                checked={formData.is_active ?? true}
                onChange={(e) => setFormData({ ...formData, is_active: e.target.checked })}
              />
              <span>启用知识库</span>
            </label>
          </div>

          <div className="form-actions">
            <button type="button" className="btn-secondary" onClick={() => setShowForm(false)}>
              取消
            </button>
            <button
              type="button"
              className="btn-primary"
              onClick={handleSave}
              disabled={loading || !formData.name}
            >
              {loading ? '保存中...' : '保存'}
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="knowledge-base-manage">
      <div className="kb-header">
        <h3>📚 知识库管理</h3>
        {knowledgeBase && (
          <button className="btn-primary" onClick={() => setShowForm(true)}>
            ✏️ 编辑知识库
          </button>
        )}
      </div>

      {error && <div className="error-message">{error}</div>}

      {loading && !knowledgeBase ? (
        <div className="loading">加载中...</div>
      ) : !knowledgeBase ? (
        <div className="empty-state">
          <p>该 Agent 还没有配置知识库</p>
          <button className="btn-primary" onClick={() => setShowForm(true)}>
            ➕ 创建知识库
          </button>
        </div>
      ) : (
        <>
          {/* 知识库信息 */}
          <div className="kb-info-card">
            <div className="kb-info">
              <h4>{knowledgeBase.name}</h4>
              {knowledgeBase.description && <p>{knowledgeBase.description}</p>}
              <div className="kb-meta">
                <span>嵌入模型: {knowledgeBase.embedding_model}</span>
                <span>向量存储: {knowledgeBase.vector_store_type}</span>
                <span>分块大小: {knowledgeBase.chunk_size}</span>
                <span>文档数量: {knowledgeBase.document_count ?? 0}</span>
              </div>
            </div>
            <div className="kb-actions">
              <button className="btn-secondary" onClick={() => setShowForm(true)}>
                编辑
              </button>
              <button className="btn-danger" onClick={handleDelete} disabled={loading}>
                删除
              </button>
            </div>
          </div>

          {/* 文档管理 */}
          <div className="documents-section">
            <div className="section-header">
              <h4>📄 文档管理</h4>
              <div className="upload-area">
                <input
                  type="file"
                  id="file-upload"
                  multiple
                  accept=".pdf,.txt,.md,.docx,.html,.csv,.xlsx,.json"
                  onChange={handleFileUpload}
                  style={{ display: 'none' }}
                  disabled={uploading}
                />
                <label htmlFor="file-upload" className="btn-primary">
                  {uploading ? '上传中...' : '📤 上传文档'}
                </label>
              </div>
            </div>

            {documents.length === 0 ? (
              <div className="empty-state-small">暂无文档，点击上方按钮上传</div>
            ) : (
              <div className="documents-list">
                {documents.map((doc) => (
                  <div key={doc.id} className="document-card">
                    <div className="doc-info">
                      <h5>{doc.name}</h5>
                      <div className="doc-meta">
                        <span className={`status status-${doc.status}`}>{doc.status}</span>
                        <span>类型: {doc.file_type}</span>
                        {doc.file_size && <span>大小: {formatFileSize(doc.file_size)}</span>}
                        <span>块数: {doc.chunk_count ?? 0}</span>
                        {doc.processed_at && <span>处理时间: {doc.processed_at}</span>}
                      </div>
                      {doc.error_message && (
                        <div className="doc-error">错误: {doc.error_message}</div>
                      )}
                    </div>
                    <div className="doc-actions">
                      <button
                        className="btn-danger-small"
                        onClick={() => handleDeleteDocument(doc.id)}
                      >
                        删除
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
};

// 格式化文件大小
function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

export default KnowledgeBaseManage;

