package service

import (
	"context"
	"fmt"

	structpb "google.golang.org/protobuf/types/known/structpb"

	"jas-agent/internal/biz"

	pb "jas-agent/api/agent/service/v1"
)

// AgentService 实现 Kratos gRPC/HTTP 服务接口，并委托现有的 AgentServer 处理核心逻辑。
type AgentService struct {
	pb.UnimplementedAgentServiceServer
	delegate   *biz.AgentUsecase
	mcpService *biz.McpUsecase
}

// NewAgentService 创建 AgentService。
func NewAgentService(delegate *biz.AgentUsecase, mcpService *biz.McpUsecase) (*AgentService, error) {

	return &AgentService{delegate: delegate, mcpService: mcpService}, nil
}

// Chat 处理单次对话请求。
func (s *AgentService) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
	return s.delegate.Chat(ctx, req)
}

// StreamChat 处理流式对话请求。
func (s *AgentService) StreamChat(req *pb.ChatRequest, stream pb.AgentService_StreamChatServer) error {
	return s.delegate.StreamChat(req, stream)
}

// ListAgentTypes 获取可用的 Agent 类型。
func (s *AgentService) ListAgentTypes(ctx context.Context, req *pb.Empty) (*pb.AgentTypesResponse, error) {
	return s.delegate.ListAgentTypes(ctx, req)
}

// ListTools 获取可用的工具列表。
func (s *AgentService) ListTools(ctx context.Context, req *pb.Empty) (*pb.ToolsResponse, error) {
	return s.delegate.ListTools(ctx, req)
}

// AddMCPService 新增 MCP 服务。
func (s *AgentService) AddMCPService(ctx context.Context, req *pb.MCPServiceRequest) (*pb.MCPServiceResponse, error) {
	return s.mcpService.AddMCPService(ctx, req)
}

// RemoveMCPService 移除 MCP 服务。
func (s *AgentService) RemoveMCPService(ctx context.Context, req *pb.MCPServiceRequest) (*pb.MCPServiceResponse, error) {
	return s.mcpService.RemoveMCPService(ctx, req)
}

// ListMCPServices 列出所有 MCP 服务。
func (s *AgentService) ListMCPServices(ctx context.Context, req *pb.Empty) (*pb.MCPServicesResponse, error) {
	return s.mcpService.ListMCPServices(ctx, req)
}

// ListMCPServicesWithId 列出带 ID 的 MCP 服务。
func (s *AgentService) ListMCPServicesWithId(ctx context.Context, req *pb.Empty) (*pb.MCPServicesWithIdResponse, error) {
	services, err := s.mcpService.ListMCPServicesWithID(ctx)
	if err != nil {
		return nil, err
	}

	resp := &pb.MCPServicesWithIdResponse{
		Services: make([]*pb.MCPServiceWithIdInfo, 0, len(services)),
	}

	for _, svc := range services {
		resp.Services = append(resp.Services, &pb.MCPServiceWithIdInfo{
			Id:          int32(svc.ID),
			Name:        svc.Name,
			Endpoint:    svc.Endpoint,
			Description: svc.Description,
			Active:      svc.Active,
			ToolCount:   int32(svc.ToolCount),
			CreatedAt:   svc.CreatedAt,
			LastRefresh: svc.LastRefresh,
		})
	}

	return resp, nil
}

// GetMCPServiceTools 查询指定 MCP 服务的工具详情。
func (s *AgentService) GetMCPServiceTools(ctx context.Context, req *pb.MCPServiceToolsRequest) (*pb.MCPServiceToolsResponse, error) {
	tools, err := s.mcpService.GetMCPToolsByID(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}

	resp := &pb.MCPServiceToolsResponse{
		Tools: make([]*pb.MCPServiceToolInfo, 0, len(tools)),
	}

	for _, tool := range tools {
		info := &pb.MCPServiceToolInfo{
			Name:        tool.Name,
			Description: tool.Description,
			Type:        tool.Type,
		}

		if tool.InputSchema != nil {
			if m, ok := tool.InputSchema.(map[string]any); ok {
				if st, err := structpb.NewStruct(m); err == nil {
					info.InputSchema = st
				}
			} else if val, err := structpb.NewValue(tool.InputSchema); err == nil {
				if st := val.GetStructValue(); st != nil {
					info.InputSchema = st
				}
			}
		}

		resp.Tools = append(resp.Tools, info)
	}

	return resp, nil
}

// CreateAgent 创建 Agent。
func (s *AgentService) CreateAgent(ctx context.Context, req *pb.AgentConfigRequest) (*pb.AgentConfigResponse, error) {
	return s.delegate.CreateAgent(ctx, req)
}

// UpdateAgent 更新 Agent。
func (s *AgentService) UpdateAgent(ctx context.Context, req *pb.AgentConfigRequest) (*pb.AgentConfigResponse, error) {
	return s.delegate.UpdateAgent(ctx, req)
}

// DeleteAgent 删除 Agent。
func (s *AgentService) DeleteAgent(ctx context.Context, req *pb.AgentDeleteRequest) (*pb.AgentConfigResponse, error) {
	return s.delegate.DeleteAgent(ctx, req)
}

// GetAgent 获取 Agent。
func (s *AgentService) GetAgent(ctx context.Context, req *pb.AgentGetRequest) (*pb.AgentConfigResponse, error) {
	return s.delegate.GetAgent(ctx, req)
}

// ListAgents 列出所有 Agent。
func (s *AgentService) ListAgents(ctx context.Context, req *pb.Empty) (*pb.AgentListResponse, error) {
	return s.delegate.ListAgents(ctx, req)
}
