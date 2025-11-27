import { useEffect, useState, useRef } from 'react';
import type { KnowledgeBaseInfo, DocumentInfo } from '../services/api';
import {
  getKnowledgeBases,
  createKnowledgeBase,
  updateKnowledgeBase,
  deleteKnowledgeBase,
  listDocuments,
  deleteDocument,
  uploadDocument,
} from '../services/api';
import './KnowledgeBaseTab.css';

interface KnowledgeBaseTabProps {
  onClose?: () => void;
  isActive?: boolean; // 是否激活
}

const KnowledgeBaseTab = ({ onClose, isActive = true }: KnowledgeBaseTabProps): JSX.Element => {
  const [knowledgeBases, setKnowledgeBases] = useState<KnowledgeBaseInfo[]>([]);
  const [selectedKB, setSelectedKB] = useState<KnowledgeBaseInfo | null>(null);
  const [documents, setDocuments] = useState<DocumentInfo[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string>('');
  const [showForm, setShowForm] = useState<boolean>(false);
  const [editingKB, setEditingKB] = useState<KnowledgeBaseInfo | null>(null);
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [allTags, setAllTags] = useState<string[]>([]);
  const [uploading, setUploading] = useState<boolean>(false);
  const [extractGraph, setExtractGraph] = useState<boolean>(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // 表单数据
  const [formData, setFormData] = useState<Partial<KnowledgeBaseInfo>>({
    name: '',
    description: '',
    tags: [],
    embedding_model: 'text-embedding-3-small',
    chunk_size: 800,
    chunk_overlap: 120,
    vector_store_type: 'memory',
    vector_store_config: '{}',
    is_active: true,
  });

  // 当 tab 激活时加载数据
  useEffect(() => {
    if (isActive) {
      void loadKnowledgeBases();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isActive]);

  // 当搜索条件变化时重新加载
  useEffect(() => {
    if (isActive) {
      void loadKnowledgeBases();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchQuery, selectedTags, isActive]);

  useEffect(() => {
    if (selectedKB) {
      void loadDocuments();
    }
    setExtractGraph(false);
  }, [selectedKB]);

  // 提取所有标签
  useEffect(() => {
    const tags = new Set<string>();
    knowledgeBases.forEach((kb) => {
      (kb.tags || []).forEach((tag) => tags.add(tag));
    });
    setAllTags(Array.from(tags).sort());
  }, [knowledgeBases]);

  const loadKnowledgeBases = async (): Promise<void> => {
    setLoading(true);
    setError('');
    try {
      const kbs = await getKnowledgeBases(searchQuery, selectedTags);
      setKnowledgeBases(kbs ?? []);
    } catch (err) {
      const message = err instanceof Error ? err.message : '未知错误';
      console.error('加载知识库失败:', err);
      setError(`加载失败: ${message}`);
    } finally {
      setLoading(false);
    }
  };

  const loadDocuments = async (kbId?: number): Promise<void> => {
    const targetKBId = kbId ?? selectedKB?.id;
    if (!targetKBId) {
      console.warn('loadDocuments: 没有知识库ID');
      return;
    }
    try {
      console.log('调用 listDocuments API，知识库ID:', targetKBId);
      // 调用 document 接口获取文档列表
      const docs = await listDocuments(targetKBId);
      console.log('listDocuments API 返回的文档列表:', docs);
      console.log('文档数量:', docs?.length ?? 0);
      
      // 直接更新状态，确保列表刷新
      const newDocs = docs ?? [];
      console.log('更新文档列表状态，新文档数量:', newDocs.length);
      setDocuments(newDocs);
      
      // 强制触发重新渲染（如果需要）
      if (newDocs.length > 0) {
        console.log('文档列表已更新，第一个文档:', {
          id: newDocs[0].id,
          name: newDocs[0].name,
          status: newDocs[0].status,
        });
      }
    } catch (err) {
      console.error('调用 listDocuments API 失败:', err);
      setDocuments([]); // 出错时清空列表
    }
  };

  const handleCreate = (): void => {
    setEditingKB(null);
    setFormData({
      name: '',
      description: '',
      tags: [],
      embedding_model: 'text-embedding-3-small',
      chunk_size: 800,
      chunk_overlap: 120,
      vector_store_type: 'memory',
      vector_store_config: '{}',
      is_active: true,
    });
    setShowForm(true);
  };

  const handleEdit = (kb: KnowledgeBaseInfo): void => {
    setEditingKB(kb);
    setFormData({
      id: kb.id,
      name: kb.name,
      description: kb.description,
      tags: kb.tags || [],
      embedding_model: kb.embedding_model || 'text-embedding-3-small',
      chunk_size: kb.chunk_size || 800,
      chunk_overlap: kb.chunk_overlap || 120,
      vector_store_type: kb.vector_store_type || 'memory',
      vector_store_config: kb.vector_store_config || '{}',
      is_active: kb.is_active ?? true,
    });
    setShowForm(true);
  };

  const handleSave = async (): Promise<void> => {
    setLoading(true);
    setError('');
    try {
      const payload: any = {
        name: formData.name,
        description: formData.description,
        tags: formData.tags || [],
        embedding_model: formData.embedding_model,
        chunk_size: formData.chunk_size,
        chunk_overlap: formData.chunk_overlap,
        vector_store_type: formData.vector_store_type,
        vector_store_config: formData.vector_store_config,
        is_active: formData.is_active,
      };
      if (editingKB) {
        payload.id = editingKB.id;
        await updateKnowledgeBase(editingKB.id, payload);
      } else {
        await createKnowledgeBase(payload);
      }
      setShowForm(false);
      await loadKnowledgeBases();
    } catch (err) {
      const message = err instanceof Error ? err.message : '未知错误';
      setError(`保存失败: ${message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleUploadClick = (): void => {
    if (!selectedKB) {
      alert('请先选择一个知识库');
      return;
    }
    fileInputRef.current?.click();
  };

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>): Promise<void> => {
    const file = e.target.files?.[0];
    if (!file) return;

    if (!selectedKB) {
      alert('请先选择一个知识库');
      return;
    }

    setUploading(true);
    try {
      console.log('开始上传文档:', file.name);
      const response = await uploadDocument(selectedKB.id, file, { extractGraph });
      console.log('文档上传成功，响应:', response);
      
      // 上传完成后，立即调用 document 接口刷新文档列表
      console.log('上传完成，调用 listDocuments 接口刷新列表...');
      await loadDocuments(selectedKB.id);
      
      // 延迟刷新，确保后端数据已完全保存
      setTimeout(async () => {
        console.log('延迟刷新文档列表（1秒后）...');
        await loadDocuments(selectedKB.id);
      }, 1000);
      
      // 如果文档正在处理，定期刷新列表以更新状态
      // 轮询检查文档状态，最多检查 20 次，每次间隔 2 秒
      let pollCount = 0;
      const maxPolls = 20;
      const pollInterval = 2000; // 2秒
      
      const pollStatus = setInterval(async () => {
        pollCount++;
        try {
          console.log(`轮询检查文档状态 (${pollCount}/${maxPolls})...`);
          // 调用 document 接口获取最新列表
          await loadDocuments(selectedKB.id);
          
          // 检查是否所有文档都已完成处理（没有 processing 状态的文档）
          const currentDocs = await listDocuments(selectedKB.id);
          const hasProcessing = currentDocs?.some(
            (doc) => doc.status === 'processing' || doc.status === 'pending'
          );
          
          // 如果没有正在处理的文档，或者达到最大轮询次数，停止轮询
          if (!hasProcessing || pollCount >= maxPolls) {
            clearInterval(pollStatus);
            // 最后一次刷新
            console.log('停止轮询，最后一次刷新文档列表...');
            await loadDocuments(selectedKB.id);
          }
        } catch (err) {
          console.error('轮询文档状态失败:', err);
          clearInterval(pollStatus);
        }
      }, pollInterval);
      
      // 40秒后强制停止轮询（防止无限轮询）
      setTimeout(() => {
        clearInterval(pollStatus);
        console.log('轮询超时，停止检查');
      }, maxPolls * pollInterval);
      
      // 重置文件输入
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : '上传失败';
      alert(`上传失败: ${message}`);
      console.error('上传文档失败:', err);
    } finally {
      setUploading(false);
    }
  };

  const handleDelete = async (id: number): Promise<void> => {
    if (!confirm('确定要删除知识库吗？这将删除所有关联的文档。')) {
      return;
    }
    setLoading(true);
    setError('');
    try {
      await deleteKnowledgeBase(id);
      if (selectedKB?.id === id) {
        setSelectedKB(null);
        setDocuments([]);
      }
      await loadKnowledgeBases();
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

  const handleTagToggle = (tag: string): void => {
    setSelectedTags((prev) =>
      prev.includes(tag) ? prev.filter((t) => t !== tag) : [...prev, tag],
    );
  };

  const handleAddTag = (tag: string): void => {
    if (tag.trim() && !formData.tags?.includes(tag.trim())) {
      setFormData((prev) => ({
        ...prev,
        tags: [...(prev.tags || []), tag.trim()],
      }));
    }
  };

  const handleRemoveTag = (tag: string): void => {
    setFormData((prev) => ({
      ...prev,
      tags: prev.tags?.filter((t) => t !== tag) || [],
    }));
  };

  if (showForm) {
    return (
      <div className="kb-form-container">
        <div className="kb-form-header">
          <h3>{editingKB ? '编辑知识库' : '创建知识库'}</h3>
          <button className="btn-close" onClick={() => setShowForm(false)}>
            ✕
          </button>
        </div>

        {error && <div className="error-message">{error}</div>}

        <div className="kb-form-body">
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

          <div className="form-group">
            <label className="optional">标签</label>
            <div className="tags-input">
              <div className="tags-list">
                {formData.tags?.map((tag) => (
                  <span key={tag} className="tag">
                    {tag}
                    <button
                      type="button"
                      className="tag-remove"
                      onClick={() => handleRemoveTag(tag)}
                    >
                      ×
                    </button>
                  </span>
                ))}
              </div>
              <input
                type="text"
                placeholder="输入标签后按回车添加"
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    handleAddTag(e.currentTarget.value);
                    e.currentTarget.value = '';
                  }
                }}
              />
            </div>
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

  if (selectedKB) {
    return (
      <div className="kb-detail-container">
        <div className="kb-detail-header">
          <button className="btn-back" onClick={() => setSelectedKB(null)}>
            ← 返回列表
          </button>
          <h3>{selectedKB.name}</h3>
        </div>

        <div className="kb-detail-info">
          <p>{selectedKB.description || '无描述'}</p>
          <div className="kb-tags">
            {(selectedKB.tags || []).map((tag) => (
              <span key={tag} className="tag">
                {tag}
              </span>
            ))}
          </div>
          <div className="kb-meta">
            <span>文档数量: {selectedKB.document_count ?? 0}</span>
            <span>嵌入模型: {selectedKB.embedding_model}</span>
            <span>向量存储: {selectedKB.vector_store_type}</span>
          </div>
        </div>

        <div className="documents-section">
          <div className="section-header">
            <h4>📄 文档列表</h4>
            <button
              className="btn-primary"
              onClick={handleUploadClick}
              disabled={uploading || !selectedKB}
            >
              {uploading ? '⏳ 上传中...' : '📤 上传文档'}
            </button>
            <input
              type="file"
              ref={fileInputRef}
              style={{ display: 'none' }}
              onChange={handleFileSelect}
              accept=".pdf,.txt,.html,.md,.xlsx,.xls,.csv,.docx,.doc,.json"
            />
            <div className="upload-options">
              <label className="checkbox-label">
                <input
                  type="checkbox"
                  checked={extractGraph}
                  onChange={(event) => setExtractGraph(event.target.checked)}
                />
                <span>提取知识图谱（存储到 Neo4j）</span>
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
                      <span>
                        图谱:{' '}
                        {doc.enable_graph_extract ?? doc.enableGraphExtract ? '已提取' : '未提取'}
                      </span>
                      {doc.processed_at && <span>处理时间: {doc.processed_at}</span>}
                    </div>
                    {doc.error_message && (
                      <div className="doc-error">错误: {doc.error_message}</div>
                    )}
                  </div>
                  <div className="doc-actions">
                    <button
                      className="btn-secondary"
                      onClick={() => alert('编辑功能开发中...')}
                    >
                      编辑
                    </button>
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
      </div>
    );
  }

  return (
    <div className="kb-tab-container">
      <div className="kb-tab-header">
        <h2>📚 知识库管理</h2>
        <button className="btn-primary" onClick={handleCreate}>
          ➕ 新建知识库
        </button>
      </div>

      <div className="kb-search-section">
        <div className="search-bar">
          <input
            type="text"
            placeholder="搜索知识库名称..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="search-input"
          />
        </div>
        {allTags.length > 0 && (
          <div className="tags-filter">
            <span className="filter-label">标签筛选:</span>
            {allTags.map((tag) => (
              <button
                key={tag}
                className={`tag-filter ${selectedTags.includes(tag) ? 'active' : ''}`}
                onClick={() => handleTagToggle(tag)}
              >
                {tag}
              </button>
            ))}
          </div>
        )}
      </div>

      {error && <div className="error-message">{error}</div>}

      {loading ? (
        <div className="loading">加载中...</div>
      ) : knowledgeBases.length === 0 ? (
        <div className="empty-state">
          <p>暂无知识库</p>
          <button className="btn-primary" onClick={handleCreate}>
            ➕ 创建知识库
          </button>
        </div>
      ) : (
        <div className="kb-list">
          {knowledgeBases.map((kb) => (
            <div key={kb.id} className="kb-card" onClick={() => setSelectedKB(kb)}>
              <div className="kb-card-header">
                <h3>{kb.name}</h3>
                <div className="kb-card-actions" onClick={(e) => e.stopPropagation()}>
                  <button className="btn-edit" onClick={() => handleEdit(kb)}>
                    编辑
                  </button>
                  <button className="btn-delete" onClick={() => handleDelete(kb.id)}>
                    删除
                  </button>
                </div>
              </div>
              {kb.description && <p className="kb-description">{kb.description}</p>}
              <div className="kb-tags">
                {(kb.tags || []).map((tag) => (
                  <span key={tag} className="tag">
                    {tag}
                  </span>
                ))}
              </div>
              <div className="kb-card-footer">
                <span>文档: {kb.document_count ?? 0}</span>
                <span>模型: {kb.embedding_model}</span>
                <span>创建: {kb.created_at}</span>
              </div>
            </div>
          ))}
        </div>
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

export default KnowledgeBaseTab;

