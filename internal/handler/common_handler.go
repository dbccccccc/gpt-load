package handler

import (
	"encoding/json"
	"fmt"
	"gpt-load/internal/channel"
	"gpt-load/internal/models"
	"gpt-load/internal/response"
	"gpt-load/internal/services"
	"strings"

	"github.com/gin-gonic/gin"
)

// CommonHandler handles common, non-grouped requests.
type CommonHandler struct {
	channelFactory *channel.Factory
	scriptService  *services.ScriptService
}

// NewCommonHandler creates a new CommonHandler.
func NewCommonHandler(channelFactory *channel.Factory, scriptService *services.ScriptService) *CommonHandler {
	return &CommonHandler{
		channelFactory: channelFactory,
		scriptService:  scriptService,
	}
}

// ChannelTypeInfo represents channel type information with metadata
type ChannelTypeInfo struct {
	Type                      string            `json:"type"`
	IsScript                  bool              `json:"is_script"`
	DisplayName               string            `json:"display_name,omitempty"`
	Description               string            `json:"description,omitempty"`
	DefaultTestModel          string            `json:"default_test_model,omitempty"`
	DefaultValidationEndpoint string            `json:"default_validation_endpoint,omitempty"`
	DefaultUpstream           string            `json:"default_upstream,omitempty"`
	SupportedModels           []string          `json:"supported_models,omitempty"`
	RequiredConfig            map[string]string `json:"required_config,omitempty"`
}

// GetChannelTypes returns a list of available channel types.
func (h *CommonHandler) GetChannelTypes(c *gin.Context) {
	// Get both static and dynamic channel types
	channelTypes := h.channelFactory.GetRegisteredChannelTypes()
	response.Success(c, channelTypes)
}

// GetChannelTypesWithMetadata returns channel types with their metadata and default values
func (h *CommonHandler) GetChannelTypesWithMetadata(c *gin.Context) {
	var channelInfos []ChannelTypeInfo

	// Get all registered channel types
	channelTypes := h.channelFactory.GetRegisteredChannelTypes()

	for _, channelType := range channelTypes {
		info := ChannelTypeInfo{
			Type: channelType,
		}

		// Check if it's a dynamic (script-based) channel
		if h.channelFactory.IsDynamicChannel(channelType) {
			info.IsScript = true

			// Get script metadata for dynamic channels
			if script, err := h.getScriptByChannelType(channelType); err == nil && script != nil {
				info.DisplayName = script.DisplayName
				info.Description = script.Description

				// Parse metadata to get default values
				if metadata, err := h.parseScriptMetadata(script); err == nil {
					info.DefaultTestModel = metadata.DefaultTestModel
					info.DefaultValidationEndpoint = metadata.DefaultValidationEndpoint
					info.SupportedModels = metadata.SupportedModels
					info.RequiredConfig = metadata.RequiredConfig

					// Set default upstream based on channel type
					info.DefaultUpstream = h.getDefaultUpstreamForScript(channelType, metadata)
				}
			}
		} else {
			// Static channel - set hardcoded defaults
			info.IsScript = false
			h.setStaticChannelDefaults(&info, channelType)
		}

		channelInfos = append(channelInfos, info)
	}

	response.Success(c, channelInfos)
}

// Helper methods

// getScriptByChannelType retrieves a script by its channel type
func (h *CommonHandler) getScriptByChannelType(channelType string) (*models.ChannelScript, error) {
	scripts, err := h.scriptService.GetEnabledScripts()
	if err != nil {
		return nil, err
	}

	for _, script := range scripts {
		if script.ChannelType == channelType {
			return script, nil
		}
	}

	return nil, fmt.Errorf("script not found for channel type: %s", channelType)
}

// parseScriptMetadata parses the JSON metadata from a script
func (h *CommonHandler) parseScriptMetadata(script *models.ChannelScript) (*models.ChannelScriptMetadata, error) {
	var metadata models.ChannelScriptMetadata
	if err := json.Unmarshal(script.Metadata, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse script metadata: %w", err)
	}
	return &metadata, nil
}

// getDefaultUpstreamForScript determines the default upstream URL for a script
func (h *CommonHandler) getDefaultUpstreamForScript(channelType string, metadata *models.ChannelScriptMetadata) string {
	// Define default upstreams for known script types
	defaultUpstreams := map[string]string{
		"grok":           "https://api.x.ai",
		"tavily_search":  "https://api.tavily.com",
		"custom_service": "https://api.example.com",
	}

	if upstream, exists := defaultUpstreams[channelType]; exists {
		return upstream
	}

	// Check if the script metadata has config hints
	if metadata.RequiredConfig != nil {
		if baseURL, exists := metadata.RequiredConfig["base_url"]; exists {
			// Extract default from description like "API base URL (default: https://api.x.ai)"
			if strings.Contains(baseURL, "default:") {
				parts := strings.Split(baseURL, "default:")
				if len(parts) > 1 {
					url := strings.TrimSpace(strings.Trim(parts[1], "()"))
					return url
				}
			}
		}
	}

	return "https://api.example.com" // Fallback
}

// setStaticChannelDefaults sets default values for static (hardcoded) channels
func (h *CommonHandler) setStaticChannelDefaults(info *ChannelTypeInfo, channelType string) {
	switch channelType {
	case "openai":
		info.DisplayName = "OpenAI"
		info.DefaultTestModel = "gpt-4"
		info.DefaultValidationEndpoint = "/v1/models"
		info.DefaultUpstream = "https://api.openai.com"
		info.SupportedModels = []string{"gpt-4", "gpt-3.5-turbo", "gpt-4-turbo"}

	case "anthropic":
		info.DisplayName = "Anthropic Claude"
		info.DefaultTestModel = "claude-3-haiku-20240307"
		info.DefaultValidationEndpoint = "/v1/messages"
		info.DefaultUpstream = "https://api.anthropic.com"
		info.SupportedModels = []string{"claude-3-haiku-20240307", "claude-3-sonnet-20240229", "claude-3-opus-20240229"}

	case "gemini":
		info.DisplayName = "Google Gemini"
		info.DefaultTestModel = "gemini-pro"
		info.DefaultValidationEndpoint = "/v1/models"
		info.DefaultUpstream = "https://generativelanguage.googleapis.com"
		info.SupportedModels = []string{"gemini-pro", "gemini-pro-vision"}

	default:
		info.DisplayName = strings.Title(channelType)
		info.DefaultTestModel = "default-model"
		info.DefaultValidationEndpoint = "/v1/models"
		info.DefaultUpstream = "https://api.example.com"
	}
}
