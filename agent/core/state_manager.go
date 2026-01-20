package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// StateSnapshot Agent状态快照
type StateSnapshot struct {
	AgentID     string                 `json:"agent_id"`
	AgentType   string                 `json:"agent_type"`
	State       string                 `json:"state"`
	CurrentStep int                    `json:"current_step"`
	MaxSteps    int                    `json:"max_steps"`
	Query       string                 `json:"query,omitempty"`
	Results     []string               `json:"results,omitempty"`
	Memory      []Message              `json:"memory,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
}

// StateManager 状态管理器接口
type StateManager interface {
	// Save 保存状态快照
	Save(ctx context.Context, snapshot *StateSnapshot) error
	// Load 加载状态快照
	Load(ctx context.Context, agentID string) (*StateSnapshot, error)
	// List 列出所有状态快照
	List(ctx context.Context, agentType string) ([]*StateSnapshot, error)
	// Delete 删除状态快照
	Delete(ctx context.Context, agentID string) error
	// CreateSnapshot 从Agent执行器创建快照
	CreateSnapshot(ctx context.Context, agentID string, agentType string, state interface{}) (*StateSnapshot, error)
}

// InMemoryStateManager 内存状态管理器（用于测试和开发）
type InMemoryStateManager struct {
	mu        sync.RWMutex
	snapshots map[string]*StateSnapshot
}

// NewInMemoryStateManager 创建内存状态管理器
func NewInMemoryStateManager() *InMemoryStateManager {
	return &InMemoryStateManager{
		snapshots: make(map[string]*StateSnapshot),
	}
}

// Save 保存状态快照
func (sm *InMemoryStateManager) Save(ctx context.Context, snapshot *StateSnapshot) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	snapshot.UpdatedAt = time.Now()
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = time.Now()
	}

	// 提取TraceID和SpanID
	if span := trace.SpanFromContext(ctx); span != nil {
		spanCtx := span.SpanContext()
		if spanCtx.IsValid() {
			snapshot.TraceID = spanCtx.TraceID().String()
			snapshot.SpanID = spanCtx.SpanID().String()
		}
	}

	sm.snapshots[snapshot.AgentID] = snapshot
	return nil
}

// Load 加载状态快照
func (sm *InMemoryStateManager) Load(ctx context.Context, agentID string) (*StateSnapshot, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	snapshot, ok := sm.snapshots[agentID]
	if !ok {
		return nil, fmt.Errorf("snapshot not found: %s", agentID)
	}

	// 返回快照的副本
	return sm.cloneSnapshot(snapshot), nil
}

// List 列出所有状态快照
func (sm *InMemoryStateManager) List(ctx context.Context, agentType string) ([]*StateSnapshot, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var snapshots []*StateSnapshot
	for _, snapshot := range sm.snapshots {
		if agentType == "" || snapshot.AgentType == agentType {
			snapshots = append(snapshots, sm.cloneSnapshot(snapshot))
		}
	}

	return snapshots, nil
}

// Delete 删除状态快照
func (sm *InMemoryStateManager) Delete(ctx context.Context, agentID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.snapshots, agentID)
	return nil
}

// CreateSnapshot 从Agent执行器创建快照（需要传入执行器状态）
func (sm *InMemoryStateManager) CreateSnapshot(ctx context.Context, agentID string, agentType string, state interface{}) (*StateSnapshot, error) {
	snapshot := &StateSnapshot{
		AgentID:   agentID,
		AgentType: agentType,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// 提取TraceID和SpanID
	if span := trace.SpanFromContext(ctx); span != nil {
		spanCtx := span.SpanContext()
		if spanCtx.IsValid() {
			snapshot.TraceID = spanCtx.TraceID().String()
			snapshot.SpanID = spanCtx.SpanID().String()
		}
	}

	// 尝试从state中提取信息（需要根据实际类型进行类型断言）
	if executorState, ok := state.(map[string]interface{}); ok {
		if state, ok := executorState["state"].(string); ok {
			snapshot.State = state
		}
		if currentStep, ok := executorState["current_step"].(int); ok {
			snapshot.CurrentStep = currentStep
		}
		if maxSteps, ok := executorState["max_steps"].(int); ok {
			snapshot.MaxSteps = maxSteps
		}
		if query, ok := executorState["query"].(string); ok {
			snapshot.Query = query
		}
		if results, ok := executorState["results"].([]string); ok {
			snapshot.Results = results
		}
		if memory, ok := executorState["memory"].([]Message); ok {
			snapshot.Memory = memory
		}
	}

	return snapshot, nil
}

// cloneSnapshot 克隆快照
func (sm *InMemoryStateManager) cloneSnapshot(snapshot *StateSnapshot) *StateSnapshot {
	data, _ := json.Marshal(snapshot)
	var cloned StateSnapshot
	_ = json.Unmarshal(data, &cloned)
	return &cloned
}

// PersistentStateManager 持久化状态管理器接口（用于数据库存储）
type PersistentStateManager interface {
	StateManager
	// SaveToDB 保存到数据库
	SaveToDB(ctx context.Context, snapshot *StateSnapshot) error
	// LoadFromDB 从数据库加载
	LoadFromDB(ctx context.Context, agentID string) (*StateSnapshot, error)
}

var (
	globalStateManager StateManager
	stateManagerOnce   sync.Once
)

// GetGlobalStateManager 获取全局状态管理器
func GetGlobalStateManager() StateManager {
	stateManagerOnce.Do(func() {
		globalStateManager = NewInMemoryStateManager()
	})
	return globalStateManager
}

// SetGlobalStateManager 设置全局状态管理器
func SetGlobalStateManager(sm StateManager) {
	globalStateManager = sm
}
