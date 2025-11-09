package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"jas-agent/internal/biz"

	"gorm.io/gorm"
)

type mcpRepo struct {
	data *Data
}

func NewMCPRepo(data *Data) biz.MCPRepo {
	return &mcpRepo{data: data}
}

func (r *mcpRepo) db() (*gorm.DB, error) {
	if r.data == nil || r.data.DB() == nil {
		return nil, errDBNotConfigured
	}
	return r.data.DB(), nil
}

func (r *mcpRepo) CreateMCPService(ctx context.Context, service *biz.MCPService) error {
	db, err := r.db()
	if err != nil {
		return err
	}

	model := mcpServiceModelFromBiz(service)
	if err := db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("create mcp service: %w", err)
	}
	service.ID = model.ID
	return nil
}

func (r *mcpRepo) UpdateMCPService(ctx context.Context, service *biz.MCPService) error {
	db, err := r.db()
	if err != nil {
		return err
	}

	model := mcpServiceModelFromBiz(service)
	if err := db.WithContext(ctx).Model(&MCPServiceModel{ID: model.ID}).Updates(map[string]interface{}{
		"endpoint":     model.Endpoint,
		"description":  model.Description,
		"is_active":    model.IsActive,
		"tool_count":   model.ToolCount,
		"last_refresh": model.LastRefresh,
	}).Error; err != nil {
		return fmt.Errorf("update mcp service: %w", err)
	}
	return nil
}

func (r *mcpRepo) DeleteMCPService(ctx context.Context, id int) error {
	db, err := r.db()
	if err != nil {
		return err
	}

	if err := db.WithContext(ctx).Where("id = ?", id).Delete(&MCPServiceModel{}).Error; err != nil {
		return fmt.Errorf("delete mcp service: %w", err)
	}
	return nil
}

func (r *mcpRepo) DeleteMCPServiceByName(ctx context.Context, name string) error {
	db, err := r.db()
	if err != nil {
		return err
	}

	if err := db.WithContext(ctx).Where("name = ?", name).Delete(&MCPServiceModel{}).Error; err != nil {
		return fmt.Errorf("delete mcp service by name: %w", err)
	}
	return nil
}

func (r *mcpRepo) GetMCPService(ctx context.Context, id int) (*biz.MCPService, error) {
	db, err := r.db()
	if err != nil {
		return nil, err
	}

	var model MCPServiceModel
	if err := db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("mcp service not found: %d", id)
		}
		return nil, fmt.Errorf("query mcp service: %w", err)
	}
	return model.ToBiz(), nil
}

func (r *mcpRepo) GetMCPServiceByName(ctx context.Context, name string) (*biz.MCPService, error) {
	db, err := r.db()
	if err != nil {
		return nil, err
	}

	var model MCPServiceModel
	if err := db.WithContext(ctx).Where("name = ?", name).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("query mcp service by name: %w", err)
	}
	return model.ToBiz(), nil
}

func (r *mcpRepo) ListMCPServices(ctx context.Context) ([]*biz.MCPService, error) {
	db, err := r.db()
	if err != nil {
		return nil, err
	}

	var models []MCPServiceModel
	if err := db.WithContext(ctx).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list mcp services: %w", err)
	}

	services := make([]*biz.MCPService, 0, len(models))
	for _, model := range models {
		services = append(services, model.ToBiz())
	}
	return services, nil
}

func (r *mcpRepo) UpdateMCPToolCount(ctx context.Context, name string, count int) error {
	db, err := r.db()
	if err != nil {
		return err
	}

	if err = db.WithContext(ctx).Model(&MCPServiceModel{}).
		Where("name = ?", name).
		Updates(map[string]interface{}{
			"tool_count":   count,
			"last_refresh": time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("update mcp tool count: %w", err)
	}
	return nil
}

type MCPServiceModel struct {
	ID          int       `gorm:"column:id;primaryKey"`
	Name        string    `gorm:"column:name"`
	Endpoint    string    `gorm:"column:endpoint"`
	Description string    `gorm:"column:description"`
	IsActive    bool      `gorm:"column:is_active"`
	ToolCount   int       `gorm:"column:tool_count"`
	LastRefresh time.Time `gorm:"column:last_refresh"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (MCPServiceModel) TableName() string {
	return "mcp_services"
}

func (m MCPServiceModel) ToBiz() *biz.MCPService {
	return &biz.MCPService{
		ID:          m.ID,
		Name:        m.Name,
		Endpoint:    m.Endpoint,
		Description: m.Description,
		IsActive:    m.IsActive,
		ToolCount:   m.ToolCount,
		LastRefresh: m.LastRefresh,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func mcpServiceModelFromBiz(service *biz.MCPService) *MCPServiceModel {
	return &MCPServiceModel{
		ID:          service.ID,
		Name:        service.Name,
		Endpoint:    service.Endpoint,
		Description: service.Description,
		IsActive:    service.IsActive,
		ToolCount:   service.ToolCount,
		LastRefresh: service.LastRefresh,
	}
}
