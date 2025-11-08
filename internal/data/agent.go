package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"jas-agent/internal/biz"

	"gorm.io/gorm"
)

var errDBNotConfigured = errors.New("database not configured")

type agentRepo struct {
	data *Data
}

func NewAgentRepo(data *Data) biz.AgentRepo {
	return &agentRepo{data: data}
}

func (r *agentRepo) db() (*gorm.DB, error) {
	if r.data == nil || r.data.DB() == nil {
		return nil, errDBNotConfigured
	}
	return r.data.DB(), nil
}

func (r *agentRepo) CreateAgent(ctx context.Context, agent *biz.Agent) error {
	db, err := r.db()
	if err != nil {
		return err
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model := agentModelFromBiz(agent)
		if err := tx.Create(model).Error; err != nil {
			return fmt.Errorf("create agent: %w", err)
		}
		agent.ID = model.ID

		if err := bindMCPServicesTx(ctx, tx, agent.ID, agent.MCPServices); err != nil {
			return err
		}
		return nil
	})
}

func (r *agentRepo) UpdateAgent(ctx context.Context, agent *biz.Agent) error {
	db, err := r.db()
	if err != nil {
		return err
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model := agentModelFromBiz(agent)
		if err := tx.Model(&AgentModel{ID: model.ID}).Updates(map[string]interface{}{
			"name":              model.Name,
			"framework":         model.Framework,
			"description":       model.Description,
			"system_prompt":     model.SystemPrompt,
			"max_steps":         model.MaxSteps,
			"model":             model.Model,
			"config":            model.Config,
			"connection_config": model.ConnectionConfig,
			"is_active":         model.IsActive,
		}).Error; err != nil {
			return fmt.Errorf("update agent: %w", err)
		}

		if err := clearMCPBindingsTx(ctx, tx, agent.ID); err != nil {
			return err
		}

		if err := bindMCPServicesTx(ctx, tx, agent.ID, agent.MCPServices); err != nil {
			return err
		}

		return nil
	})
}

func (r *agentRepo) DeleteAgent(ctx context.Context, id int) error {
	db, err := r.db()
	if err != nil {
		return err
	}

	if err := db.WithContext(ctx).Where("id = ?", id).Delete(&AgentModel{}).Error; err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	return nil
}

func (r *agentRepo) GetAgent(ctx context.Context, id int) (*biz.Agent, error) {
	db, err := r.db()
	if err != nil {
		return nil, err
	}

	var model AgentModel
	if err := db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("agent not found: %d", id)
		}
		return nil, fmt.Errorf("query agent: %w", err)
	}

	services, err := fetchAgentServices(ctx, db, id)
	if err != nil {
		return nil, err
	}

	return model.ToBiz(services), nil
}

func (r *agentRepo) ListAgents(ctx context.Context) ([]*biz.Agent, error) {
	db, err := r.db()
	if err != nil {
		return nil, err
	}

	var models []AgentModel
	if err := db.WithContext(ctx).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	agents := make([]*biz.Agent, 0, len(models))
	for _, model := range models {
		services, err := fetchAgentServices(ctx, db, model.ID)
		if err != nil {
			return nil, err
		}
		agents = append(agents, model.ToBiz(services))
	}
	return agents, nil
}

type AgentModel struct {
	ID               int       `gorm:"column:id;primaryKey"`
	Name             string    `gorm:"column:name"`
	Framework        string    `gorm:"column:framework"`
	Description      string    `gorm:"column:description"`
	SystemPrompt     string    `gorm:"column:system_prompt"`
	MaxSteps         int       `gorm:"column:max_steps"`
	Model            string    `gorm:"column:model"`
	Config           string    `gorm:"column:config"`
	ConnectionConfig string    `gorm:"column:connection_config"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
	IsActive         bool      `gorm:"column:is_active"`
}

func (AgentModel) TableName() string {
	return "agents"
}

func (m AgentModel) ToBiz(services []string) *biz.Agent {
	return &biz.Agent{
		ID:               m.ID,
		Name:             m.Name,
		Framework:        m.Framework,
		Description:      m.Description,
		SystemPrompt:     m.SystemPrompt,
		MaxSteps:         m.MaxSteps,
		Model:            m.Model,
		MCPServices:      services,
		ConnectionConfig: m.ConnectionConfig,
		ConfigJSON:       m.Config,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
		IsActive:         m.IsActive,
	}
}

func agentModelFromBiz(agent *biz.Agent) *AgentModel {
	model := &AgentModel{
		ID:               agent.ID,
		Name:             agent.Name,
		Framework:        agent.Framework,
		Description:      agent.Description,
		SystemPrompt:     agent.SystemPrompt,
		MaxSteps:         agent.MaxSteps,
		Model:            agent.Model,
		Config:           agent.ConfigJSON,
		ConnectionConfig: agent.ConnectionConfig,
		IsActive:         agent.IsActive,
	}
	if model.Config == "" {
		model.Config = "{}"
	}
	if model.ConnectionConfig == "" {
		model.ConnectionConfig = "{}"
	}
	return model
}

type agentMCPBinding struct {
	AgentID      int `gorm:"column:agent_id"`
	MCPServiceID int `gorm:"column:mcp_service_id"`
}

func (agentMCPBinding) TableName() string {
	return "agent_mcp_bindings"
}

func bindMCPServicesTx(ctx context.Context, tx *gorm.DB, agentID int, names []string) error {
	if len(names) == 0 {
		return nil
	}

	var services []struct {
		ID   int
		Name string
	}
	if err := tx.WithContext(ctx).
		Table("mcp_services").
		Select("id", "name").
		Where("name IN ?", names).
		Find(&services).Error; err != nil {
		return fmt.Errorf("query mcp services: %w", err)
	}

	nameToID := make(map[string]int, len(services))
	for _, service := range services {
		nameToID[service.Name] = service.ID
	}

	for _, name := range names {
		if id, ok := nameToID[name]; ok {
			if err := tx.WithContext(ctx).Create(&agentMCPBinding{
				AgentID:      agentID,
				MCPServiceID: id,
			}).Error; err != nil {
				return fmt.Errorf("bind mcp service %s: %w", name, err)
			}
		}
	}
	return nil
}

func clearMCPBindingsTx(ctx context.Context, tx *gorm.DB, agentID int) error {
	if err := tx.WithContext(ctx).Where("agent_id = ?", agentID).Delete(&agentMCPBinding{}).Error; err != nil {
		return fmt.Errorf("clear mcp bindings: %w", err)
	}
	return nil
}

func fetchAgentServices(ctx context.Context, db *gorm.DB, agentID int) ([]string, error) {
	var names []string
	if err := db.WithContext(ctx).
		Table("mcp_services AS m").
		Joins("INNER JOIN agent_mcp_bindings b ON m.id = b.mcp_service_id").
		Where("b.agent_id = ?", agentID).
		Pluck("m.name", &names).Error; err != nil {
		return nil, fmt.Errorf("fetch agent mcp services: %w", err)
	}
	return names, nil
}
