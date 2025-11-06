package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	pb "jas-agent/api/proto"

	"github.com/gorilla/mux"
)

// handleListAgents 列出所有Agent
func (gw *HTTPGateway) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	resp, err := gw.grpcServer.ListAgents(r.Context(), &pb.Empty{})
	if err != nil {
		gw.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	agents := make([]map[string]interface{}, len(resp.Agents))
	for i, agent := range resp.Agents {
		agents[i] = map[string]interface{}{
			"id":                agent.Id,
			"name":              agent.Name,
			"framework":         agent.Framework,
			"description":       agent.Description,
			"system_prompt":     agent.SystemPrompt,
			"max_steps":         agent.MaxSteps,
			"model":             agent.Model,
			"mcp_services":      agent.McpServices,
			"created_at":        agent.CreatedAt,
			"updated_at":        agent.UpdatedAt,
			"is_active":         agent.IsActive,
			"connection_config": agent.ConnectionConfig,
		}
	}

	w.WriteHeader(http.StatusOK)
	gw.sendJSON(w, map[string]interface{}{
		"agents": agents,
	})
}

// handleCreateAgent 创建Agent
func (gw *HTTPGateway) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req struct {
		Name             string            `json:"name"`
		Framework        string            `json:"framework"`
		Description      string            `json:"description"`
		SystemPrompt     string            `json:"system_prompt"`
		MaxSteps         int32             `json:"max_steps"`
		Model            string            `json:"model"`
		MCPServices      []string          `json:"mcp_services"`
		Config           map[string]string `json:"config"`
		ConnectionConfig string            `json:"connection_config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		gw.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	resp, err := gw.grpcServer.CreateAgent(r.Context(), &pb.AgentConfigRequest{
		Name:             req.Name,
		Framework:        req.Framework,
		Description:      req.Description,
		SystemPrompt:     req.SystemPrompt,
		MaxSteps:         req.MaxSteps,
		Model:            req.Model,
		McpServices:      req.MCPServices,
		Config:           req.Config,
		ConnectionConfig: req.ConnectionConfig,
	})

	if err != nil {
		gw.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !resp.Success {
		gw.sendError(w, http.StatusBadRequest, resp.Message)
		return
	}

	agent := resp.Agent
	w.WriteHeader(http.StatusOK)
	gw.sendJSON(w, map[string]interface{}{
		"success": true,
		"message": resp.Message,
		"agent": map[string]interface{}{
			"id":                agent.Id,
			"name":              agent.Name,
			"framework":         agent.Framework,
			"description":       agent.Description,
			"system_prompt":     agent.SystemPrompt,
			"max_steps":         agent.MaxSteps,
			"model":             agent.Model,
			"mcp_services":      agent.McpServices,
			"created_at":        agent.CreatedAt,
			"updated_at":        agent.UpdatedAt,
			"is_active":         agent.IsActive,
			"connection_config": agent.ConnectionConfig,
		},
	})
}

// handleGetAgent 获取单个Agent
func (gw *HTTPGateway) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		gw.sendError(w, http.StatusBadRequest, "Invalid agent ID")
		return
	}

	resp, err := gw.grpcServer.GetAgent(r.Context(), &pb.AgentGetRequest{
		Id: int32(id),
	})

	if err != nil {
		gw.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !resp.Success {
		gw.sendError(w, http.StatusNotFound, resp.Message)
		return
	}

	agent := resp.Agent
	w.WriteHeader(http.StatusOK)
	gw.sendJSON(w, map[string]interface{}{
		"agent": map[string]interface{}{
			"id":                agent.Id,
			"name":              agent.Name,
			"framework":         agent.Framework,
			"description":       agent.Description,
			"system_prompt":     agent.SystemPrompt,
			"max_steps":         agent.MaxSteps,
			"model":             agent.Model,
			"mcp_services":      agent.McpServices,
			"created_at":        agent.CreatedAt,
			"updated_at":        agent.UpdatedAt,
			"is_active":         agent.IsActive,
			"connection_config": agent.ConnectionConfig,
		},
	})
}

// handleUpdateAgent 更新Agent
func (gw *HTTPGateway) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		gw.sendError(w, http.StatusBadRequest, "Invalid agent ID")
		return
	}

	var req struct {
		Name             string            `json:"name"`
		Framework        string            `json:"framework"`
		Description      string            `json:"description"`
		SystemPrompt     string            `json:"system_prompt"`
		MaxSteps         int32             `json:"max_steps"`
		Model            string            `json:"model"`
		MCPServices      []string          `json:"mcp_services"`
		Config           map[string]string `json:"config"`
		ConnectionConfig string            `json:"connection_config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		gw.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	resp, err := gw.grpcServer.UpdateAgent(r.Context(), &pb.AgentConfigRequest{
		Id:               int32(id),
		Name:             req.Name,
		Framework:        req.Framework,
		Description:      req.Description,
		SystemPrompt:     req.SystemPrompt,
		MaxSteps:         req.MaxSteps,
		Model:            req.Model,
		McpServices:      req.MCPServices,
		Config:           req.Config,
		ConnectionConfig: req.ConnectionConfig,
	})

	if err != nil {
		gw.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !resp.Success {
		gw.sendError(w, http.StatusBadRequest, resp.Message)
		return
	}

	agent := resp.Agent
	w.WriteHeader(http.StatusOK)
	gw.sendJSON(w, map[string]interface{}{
		"success": true,
		"message": resp.Message,
		"agent": map[string]interface{}{
			"id":                agent.Id,
			"name":              agent.Name,
			"framework":         agent.Framework,
			"description":       agent.Description,
			"system_prompt":     agent.SystemPrompt,
			"max_steps":         agent.MaxSteps,
			"model":             agent.Model,
			"mcp_services":      agent.McpServices,
			"created_at":        agent.CreatedAt,
			"updated_at":        agent.UpdatedAt,
			"is_active":         agent.IsActive,
			"connection_config": agent.ConnectionConfig,
		},
	})
}

// handleDeleteAgent 删除Agent
func (gw *HTTPGateway) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		gw.sendError(w, http.StatusBadRequest, "Invalid agent ID")
		return
	}

	resp, err := gw.grpcServer.DeleteAgent(r.Context(), &pb.AgentDeleteRequest{
		Id: int32(id),
	})

	if err != nil {
		gw.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !resp.Success {
		gw.sendError(w, http.StatusBadRequest, resp.Message)
		return
	}

	w.WriteHeader(http.StatusOK)
	gw.sendJSON(w, map[string]interface{}{
		"success": true,
		"message": resp.Message,
	})
}
