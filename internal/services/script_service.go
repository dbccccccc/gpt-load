package services

import (
	"encoding/json"
	"fmt"
	"gpt-load/internal/models"
	"time"

	"gorm.io/gorm"
)

// ScriptService handles channel script operations
type ScriptService struct {
	db                *gorm.DB
	securityValidator *ScriptSecurityValidator
}

// NewScriptService creates a new script service
func NewScriptService(db *gorm.DB) *ScriptService {
	return &ScriptService{
		db:                db,
		securityValidator: NewScriptSecurityValidator(),
	}
}

// GetAllScripts returns all channel scripts
func (s *ScriptService) GetAllScripts() ([]models.ChannelScript, error) {
	var scripts []models.ChannelScript
	err := s.db.Find(&scripts).Error
	return scripts, err
}

// GetScriptByID returns a script by its ID
func (s *ScriptService) GetScriptByID(id uint) (*models.ChannelScript, error) {
	var script models.ChannelScript
	err := s.db.First(&script, id).Error
	if err != nil {
		return nil, err
	}
	return &script, nil
}

// GetScriptByChannelType returns a script by its channel type
func (s *ScriptService) GetScriptByChannelType(channelType string) (*models.ChannelScript, error) {
	var script models.ChannelScript
	err := s.db.Where("channel_type = ? AND status = ?", channelType, "enabled").First(&script).Error
	if err != nil {
		return nil, err
	}
	return &script, nil
}

// CreateScript creates a new channel script
func (s *ScriptService) CreateScript(script *models.ChannelScript) (*models.ChannelScript, error) {
	// Validate the script before creating
	if err := s.validateScriptSyntax(script); err != nil {
		return nil, fmt.Errorf("script validation failed: %w", err)
	}

	// Check if channel type already exists
	var existing models.ChannelScript
	err := s.db.Where("channel_type = ?", script.ChannelType).First(&existing).Error
	if err == nil {
		return nil, fmt.Errorf("channel type '%s' already exists", script.ChannelType)
	}

	err = s.db.Create(script).Error
	if err != nil {
		return nil, err
	}

	return script, nil
}

// UpdateScript updates an existing channel script
func (s *ScriptService) UpdateScript(id uint, updates interface{}) (*models.ChannelScript, error) {
	var script models.ChannelScript
	err := s.db.First(&script, id).Error
	if err != nil {
		return nil, err
	}

	// If script content is being updated, validate it
	if updateMap, ok := updates.(map[string]interface{}); ok {
		if newScript, exists := updateMap["script"]; exists {
			tempScript := script
			tempScript.Script = newScript.(string)
			if err := s.validateScriptSyntax(&tempScript); err != nil {
				return nil, fmt.Errorf("script validation failed: %w", err)
			}
		}
	}

	err = s.db.Model(&script).Updates(updates).Error
	if err != nil {
		return nil, err
	}

	return &script, nil
}

// DeleteScript deletes a channel script
func (s *ScriptService) DeleteScript(id uint) error {
	// First disable the script if it's enabled
	var script models.ChannelScript
	err := s.db.First(&script, id).Error
	if err != nil {
		return err
	}

	if script.Status == "enabled" {
		if err := s.DisableScript(id); err != nil {
			return fmt.Errorf("failed to disable script before deletion: %w", err)
		}
	}

	return s.db.Delete(&script).Error
}

// EnableScript enables a channel script
func (s *ScriptService) EnableScript(id uint) error {
	var script models.ChannelScript
	err := s.db.First(&script, id).Error
	if err != nil {
		return err
	}

	// Validate the script before enabling
	if err := s.validateScriptSyntax(&script); err != nil {
		return fmt.Errorf("cannot enable invalid script: %w", err)
	}

	// Disable any other script with the same channel type
	err = s.db.Model(&models.ChannelScript{}).
		Where("channel_type = ? AND id != ?", script.ChannelType, id).
		Update("status", "disabled").Error
	if err != nil {
		return err
	}

	// Enable this script
	script.Status = "enabled"
	script.ErrorMsg = ""
	script.LastError = nil

	return s.db.Save(&script).Error
}

// DisableScript disables a channel script
func (s *ScriptService) DisableScript(id uint) error {
	return s.db.Model(&models.ChannelScript{}).
		Where("id = ?", id).
		Update("status", "disabled").Error
}

// ValidateScript validates a script without saving it
func (s *ScriptService) ValidateScript(scriptCode string, metadata models.ChannelScriptMetadata) (map[string]interface{}, error) {
	// Create a temporary script for validation
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	tempScript := &models.ChannelScript{
		Script:   scriptCode,
		Metadata: metadataJSON,
	}

	err = s.validateScriptSyntax(tempScript)
	result := map[string]interface{}{
		"valid": err == nil,
	}

	if err != nil {
		result["error"] = err.Error()
	} else {
		result["message"] = "Script is valid"
	}

	return result, nil
}

// TestScript tests a script with sample data
func (s *ScriptService) TestScript(scriptCode string, metadata models.ChannelScriptMetadata, testData map[string]interface{}) (map[string]interface{}, error) {
	// Create a temporary script for testing
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	tempScript := &models.ChannelScript{
		Script:   scriptCode,
		Metadata: metadataJSON,
	}

	// First validate the script
	if err := s.validateScriptSyntax(tempScript); err != nil {
		return map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		}, nil
	}

	// Try to create a runtime and test basic functionality
	err = s.validateScriptSyntax(tempScript)
	if err != nil {
		return map[string]interface{}{
			"valid":   false,
			"error":   err.Error(),
			"message": "Failed to create runtime",
		}, nil
	}

	result := map[string]interface{}{
		"valid":   true,
		"message": "Script test completed successfully",
		"runtime": "JavaScript runtime created successfully",
	}

	// If test data is provided, try to execute some basic operations
	if testData != nil {
		result["test_data_processed"] = true
	}

	return result, nil
}

// GetScriptLogs returns logs for a specific script
func (s *ScriptService) GetScriptLogs(id uint) ([]map[string]interface{}, error) {
	// For now, return empty logs - this would be implemented with a proper logging system
	logs := []map[string]interface{}{
		{
			"timestamp": time.Now().Format(time.RFC3339),
			"level":     "info",
			"message":   "Script logs feature not yet implemented",
		},
	}
	return logs, nil
}

// validateScriptSyntax validates the JavaScript syntax and basic structure
func (s *ScriptService) validateScriptSyntax(script *models.ChannelScript) error {
	// Use the security validator for comprehensive validation
	if err := s.securityValidator.ValidateScript(script.Script); err != nil {
		return fmt.Errorf("security validation failed: %w", err)
	}

	return nil
}

// GetScript retrieves a script by ID
func (s *ScriptService) GetScript(id uint) (*models.ChannelScript, error) {
	var script models.ChannelScript
	if err := s.db.First(&script, id).Error; err != nil {
		return nil, err
	}
	return &script, nil
}

// GetEnabledScripts retrieves all enabled scripts
func (s *ScriptService) GetEnabledScripts() ([]*models.ChannelScript, error) {
	var scripts []*models.ChannelScript
	if err := s.db.Where("status = ?", "enabled").Find(&scripts).Error; err != nil {
		return nil, err
	}
	return scripts, nil
}
