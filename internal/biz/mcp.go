package biz

import (
	"context"
	"fmt"
	"jas-agent/agent/core"
	"jas-agent/agent/tools"
	pb "jas-agent/api/agent/service/v1"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type McpUsecase struct {
	mcpRepo MCPRepo
	logger  *log.Helper
}

// NewMcpUsecase 创建新的 McpUsecase。
func NewMcpUsecase(mcpRepo MCPRepo, logger log.Logger) *McpUsecase {
	uc := &McpUsecase{
		mcpRepo: mcpRepo,
		logger:  log.NewHelper(log.With(logger, "module", "biz/agent")),
	}

	return uc
}

// AddMCPService 添加MCP服务
func (s *McpUsecase) AddMCPService(ctx context.Context, req *pb.MCPServiceRequest) error {

	tm := tools.NewToolManager()
	// 创建MCP工具管理器
	mcpManager, err := tools.NewMCPToolManager(req.Name, req.Endpoint, tm, tools.TransferToMcpClientType(req.ClientType))
	if err != nil {
		return err
	}
	// 启动工具发现
	mcpManager.DiscoverAndRegisterTools()
	tm.RegisterMCPToolManager(req.Name, mcpManager)

	// 保存服务信息到内存
	serviceInfo := &MCPServiceInfo{
		Name:        req.Name,
		Endpoint:    req.Endpoint,
		Manager:     mcpManager,
		Active:      true,
		ToolCount:   len(mcpManager.GetTools()),
		CreatedAt:   time.Now(),
		LastRefresh: time.Now(),
	}

	dbService := &MCPService{
		Name:        req.Name,
		Endpoint:    req.Endpoint,
		ClientType:  req.ClientType,
		IsActive:    true,
		ToolCount:   serviceInfo.ToolCount,
		LastRefresh: time.Now(),
	}
	if err = s.mcpRepo.CreateMCPService(ctx, dbService); err != nil {
		s.logger.Errorf("save MCP service to database failed: %v", err)
		return err
	}

	s.logger.Infof("MCP service added: name=%s endpoint=%s tools=%d", req.Name, req.Endpoint, serviceInfo.ToolCount)

	return nil
}

// RemoveMCPService 移除MCP服务
func (s *McpUsecase) RemoveMCPService(ctx context.Context, req *pb.MCPServiceRequest) error {

	return s.mcpRepo.DeleteMCPServiceByName(ctx, req.Name)

}

// ListMCPServices 列出所有MCP服务
func (s *McpUsecase) ListMCPServices(ctx context.Context, req *pb.Empty) ([]*MCPService, error) {
	return s.mcpRepo.ListMCPServices(ctx)

}

// mcpServiceInfoToProto 转换MCP服务信息为Proto格式
func (s *McpUsecase) mcpServiceInfoToProto(info *MCPServiceInfo) *pb.MCPServiceInfo {
	return &pb.MCPServiceInfo{
		Name:        info.Name,
		Endpoint:    info.Endpoint,
		Active:      info.Active,
		ToolCount:   int32(info.ToolCount),
		CreatedAt:   info.CreatedAt.Format("2006-01-02 15:04:05"),
		LastRefresh: info.LastRefresh.Format("2006-01-02 15:04:05"),
	}
}

func (s *McpUsecase) ListMCPServicesWithID(ctx context.Context) ([]*MCPServiceDetail, error) {
	if s.mcpRepo == nil {
		return []*MCPServiceDetail{}, nil
	}

	mcpServices, err := s.mcpRepo.ListMCPServices(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*MCPServiceDetail, 0, len(mcpServices))
	for _, svc := range mcpServices {
		createdAt := ""
		if !svc.CreatedAt.IsZero() {
			createdAt = svc.CreatedAt.Format("2006-01-02 15:04:05")
		}
		lastRefresh := ""
		if !svc.LastRefresh.IsZero() {
			lastRefresh = svc.LastRefresh.Format("2006-01-02 15:04:05")
		}

		result = append(result, &MCPServiceDetail{
			ID:          svc.ID,
			Name:        svc.Name,
			Endpoint:    svc.Endpoint,
			Description: svc.Description,
			ClientType:  svc.ClientType,
			Active:      svc.IsActive,
			ToolCount:   svc.ToolCount,
			CreatedAt:   createdAt,
			LastRefresh: lastRefresh,
		})
	}
	return result, nil
}

func (s *McpUsecase) GetMCPToolsByID(ctx context.Context, id int) ([]*MCPToolDetail, error) {
	if s.mcpRepo == nil {
		return nil, fmt.Errorf("MCP repository not configured")
	}

	service, err := s.mcpRepo.GetMCPService(ctx, id)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, fmt.Errorf("MCP service not found: %d", id)
	}

	tm := tools.NewToolManager()
	mgr, err := tools.NewMCPToolManager(service.Name, service.Endpoint, tm, tools.TransferToMcpClientType(service.ClientType))
	if err != nil {
		return nil, err
	}
	if err = mgr.DiscoverAndRegisterTools(); err != nil {
		return nil, err
	}

	coreTools := mgr.GetTools()
	details := make([]*MCPToolDetail, 0, len(coreTools))
	for _, tool := range coreTools {
		name := tool.Name()
		if idx := strings.Index(name, tools.MCP_SEP); idx >= 0 {
			name = name[idx+1:]
		}
		toolType := "MCP"
		switch tool.Type() {
		case core.Normal:
			toolType = "Normal"
		case core.Mcp:
			toolType = "MCP"
		}
		details = append(details, &MCPToolDetail{
			Name:        name,
			Description: tool.Description(),
			Type:        toolType,
			InputSchema: tool.Input(),
		})
	}
	return details, nil
}
