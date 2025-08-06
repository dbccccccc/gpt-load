package handler

import (
	"encoding/json"
	"strconv"

	app_errors "gpt-load/internal/errors"
	"gpt-load/internal/models"
	"gpt-load/internal/response"
	"gpt-load/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ScriptHandler handles channel script management endpoints
type ScriptHandler struct {
	scriptService  *services.ScriptService
	scriptManager  *services.ScriptManager
}

// NewScriptHandler creates a new script handler
func NewScriptHandler(scriptService *services.ScriptService, scriptManager *services.ScriptManager) *ScriptHandler {
	return &ScriptHandler{
		scriptService: scriptService,
		scriptManager: scriptManager,
	}
}

// GetScripts returns all channel scripts
func (h *ScriptHandler) GetScripts(c *gin.Context) {
	scripts, err := h.scriptService.GetAllScripts()
	if err != nil {
		logrus.Errorf("Failed to get scripts: %v", err)
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, "Failed to get scripts"))
		return
	}

	response.Success(c, scripts)
}

// GetScript returns a specific channel script by ID
func (h *ScriptHandler) GetScript(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrBadRequest, "Invalid script ID"))
		return
	}

	script, err := h.scriptService.GetScriptByID(uint(id))
	if err != nil {
		logrus.Errorf("Failed to get script %d: %v", id, err)
		response.Error(c, app_errors.ErrResourceNotFound)
		return
	}

	response.Success(c, script)
}

// CreateScript creates a new channel script
func (h *ScriptHandler) CreateScript(c *gin.Context) {
	var req struct {
		Name        string                         `json:"name" binding:"required"`
		DisplayName string                         `json:"display_name"`
		Description string                         `json:"description"`
		Author      string                         `json:"author"`
		Version     string                         `json:"version" binding:"required"`
		ChannelType string                         `json:"channel_type" binding:"required"`
		Script      string                         `json:"script" binding:"required"`
		Metadata    models.ChannelScriptMetadata   `json:"metadata" binding:"required"`
		Config      map[string]interface{}         `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, "Invalid metadata"))
		return
	}

	// Convert config to JSON
	var configJSON []byte
	if req.Config != nil {
		configJSON, err = json.Marshal(req.Config)
		if err != nil {
			response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, "Invalid config"))
			return
		}
	}

	script := &models.ChannelScript{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Author:      req.Author,
		Version:     req.Version,
		ChannelType: req.ChannelType,
		Script:      req.Script,
		Metadata:    metadataJSON,
		Config:      configJSON,
		Status:      "disabled", // Default to disabled for safety
	}

	createdScript, err := h.scriptService.CreateScript(script)
	if err != nil {
		logrus.Errorf("Failed to create script: %v", err)
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, "Failed to create script"))
		return
	}

	response.Success(c, createdScript)
}

// UpdateScript updates an existing channel script
func (h *ScriptHandler) UpdateScript(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrBadRequest, "Invalid script ID"))
		return
	}

	var req struct {
		Name        *string                        `json:"name"`
		DisplayName *string                        `json:"display_name"`
		Description *string                        `json:"description"`
		Author      *string                        `json:"author"`
		Version     *string                        `json:"version"`
		Script      *string                        `json:"script"`
		Metadata    *models.ChannelScriptMetadata  `json:"metadata"`
		Config      *map[string]interface{}        `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	updatedScript, err := h.scriptService.UpdateScript(uint(id), req)
	if err != nil {
		logrus.Errorf("Failed to update script %d: %v", id, err)
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, "Failed to update script"))
		return
	}

	response.Success(c, updatedScript)
}

// DeleteScript deletes a channel script
func (h *ScriptHandler) DeleteScript(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrBadRequest, "Invalid script ID"))
		return
	}

	err = h.scriptService.DeleteScript(uint(id))
	if err != nil {
		logrus.Errorf("Failed to delete script %d: %v", id, err)
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, "Failed to delete script"))
		return
	}

	response.Success(c, gin.H{"message": "Script deleted successfully"})
}

// EnableScript enables a channel script
func (h *ScriptHandler) EnableScript(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrBadRequest, "Invalid script ID"))
		return
	}

	err = h.scriptService.EnableScript(uint(id))
	if err != nil {
		logrus.Errorf("Failed to enable script %d: %v", id, err)
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, "Failed to enable script"))
		return
	}

	response.Success(c, gin.H{"message": "Script enabled successfully"})
}

// DisableScript disables a channel script
func (h *ScriptHandler) DisableScript(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrBadRequest, "Invalid script ID"))
		return
	}

	err = h.scriptService.DisableScript(uint(id))
	if err != nil {
		logrus.Errorf("Failed to disable script %d: %v", id, err)
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, "Failed to disable script"))
		return
	}

	response.Success(c, gin.H{"message": "Script disabled successfully"})
}

// ValidateScript validates a channel script without saving it
func (h *ScriptHandler) ValidateScript(c *gin.Context) {
	var req struct {
		Script   string                       `json:"script" binding:"required"`
		Metadata models.ChannelScriptMetadata `json:"metadata" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	result, err := h.scriptService.ValidateScript(req.Script, req.Metadata)
	if err != nil {
		logrus.Errorf("Failed to validate script: %v", err)
		response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, "Script validation failed"))
		return
	}

	response.Success(c, result)
}

// TestScript tests a channel script with sample data
func (h *ScriptHandler) TestScript(c *gin.Context) {
	var req struct {
		Script     string                       `json:"script" binding:"required"`
		Metadata   models.ChannelScriptMetadata `json:"metadata" binding:"required"`
		TestData   map[string]interface{}       `json:"test_data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	result, err := h.scriptService.TestScript(req.Script, req.Metadata, req.TestData)
	if err != nil {
		logrus.Errorf("Failed to test script: %v", err)
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, "Failed to test script"))
		return
	}

	response.Success(c, result)
}

// GetScriptLogs returns logs for a specific script
func (h *ScriptHandler) GetScriptLogs(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrBadRequest, "Invalid script ID"))
		return
	}

	logs, err := h.scriptService.GetScriptLogs(uint(id))
	if err != nil {
		logrus.Errorf("Failed to get script logs %d: %v", id, err)
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, "Failed to get script logs"))
		return
	}

	response.Success(c, logs)
}

// ReloadScript manually reloads a specific script
func (h *ScriptHandler) ReloadScript(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrBadRequest, "Invalid script ID"))
		return
	}

	err = h.scriptManager.ReloadScript(uint(id))
	if err != nil {
		logrus.Errorf("Failed to reload script %d: %v", id, err)
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, "Failed to reload script"))
		return
	}

	response.Success(c, gin.H{"message": "Script reloaded successfully"})
}

// ReloadAllScripts reloads all enabled scripts
func (h *ScriptHandler) ReloadAllScripts(c *gin.Context) {
	err := h.scriptManager.ReloadAllScripts()
	if err != nil {
		logrus.Errorf("Failed to reload all scripts: %v", err)
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, "Failed to reload scripts"))
		return
	}

	response.Success(c, gin.H{"message": "All scripts reloaded successfully"})
}

// GetActiveScripts returns currently active script channel types
func (h *ScriptHandler) GetActiveScripts(c *gin.Context) {
	activeScripts := h.scriptManager.GetActiveScripts()
	response.Success(c, gin.H{
		"active_scripts": activeScripts,
		"count":         len(activeScripts),
	})
}
