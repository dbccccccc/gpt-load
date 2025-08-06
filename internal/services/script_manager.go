package services

import (
	"crypto/sha256"
	"fmt"
	"gpt-load/internal/channel"
	"gpt-load/internal/models"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ScriptManager handles hot-reloading of channel scripts
type ScriptManager struct {
	db              *gorm.DB
	channelFactory  *channel.Factory
	scriptService   *ScriptService
	activeScripts   map[string]*channel.ScriptRuntime
	scriptVersions  map[string]string // script_id -> version hash
	mutex           sync.RWMutex
	reloadInterval  time.Duration
	stopChan        chan struct{}
	running         bool
}

// NewScriptManager creates a new script manager
func NewScriptManager(db *gorm.DB, channelFactory *channel.Factory, scriptService *ScriptService) *ScriptManager {
	return &ScriptManager{
		db:             db,
		channelFactory: channelFactory,
		scriptService:  scriptService,
		activeScripts:  make(map[string]*channel.ScriptRuntime),
		scriptVersions: make(map[string]string),
		reloadInterval: 30 * time.Second, // Check for updates every 30 seconds
		stopChan:       make(chan struct{}),
	}
}

// Start begins the hot-reloading process
func (sm *ScriptManager) Start() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.running {
		return fmt.Errorf("script manager is already running")
	}

	// Load all enabled scripts initially
	if err := sm.loadAllEnabledScripts(); err != nil {
		return fmt.Errorf("failed to load initial scripts: %w", err)
	}

	sm.running = true
	go sm.reloadLoop()

	logrus.Info("Script manager started with hot-reloading enabled")
	return nil
}

// Stop stops the hot-reloading process
func (sm *ScriptManager) Stop() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if !sm.running {
		return
	}

	close(sm.stopChan)
	sm.running = false

	logrus.Info("Script manager stopped")
}

// ReloadScript manually reloads a specific script
func (sm *ScriptManager) ReloadScript(scriptID uint) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	script, err := sm.scriptService.GetScript(scriptID)
	if err != nil {
		return fmt.Errorf("failed to get script: %w", err)
	}

	if script.Status != "enabled" {
		// Remove from active scripts if disabled
		delete(sm.activeScripts, script.ChannelType)
		delete(sm.scriptVersions, fmt.Sprintf("%d", script.ID))
		sm.channelFactory.UnregisterDynamicChannel(script.ChannelType)
		logrus.WithField("script", script.Name).Info("Script disabled and unregistered")
		return nil
	}

	return sm.loadScript(script)
}

// ReloadAllScripts reloads all enabled scripts
func (sm *ScriptManager) ReloadAllScripts() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	return sm.loadAllEnabledScripts()
}

// GetActiveScripts returns a list of currently active script channel types
func (sm *ScriptManager) GetActiveScripts() []string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var channelTypes []string
	for channelType := range sm.activeScripts {
		channelTypes = append(channelTypes, channelType)
	}
	return channelTypes
}

// IsScriptActive checks if a script is currently active
func (sm *ScriptManager) IsScriptActive(channelType string) bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	_, exists := sm.activeScripts[channelType]
	return exists
}

// reloadLoop runs the periodic reload check
func (sm *ScriptManager) reloadLoop() {
	ticker := time.NewTicker(sm.reloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := sm.checkForUpdates(); err != nil {
				logrus.WithError(err).Error("Failed to check for script updates")
			}
		case <-sm.stopChan:
			return
		}
	}
}

// checkForUpdates checks for script changes and reloads if necessary
func (sm *ScriptManager) checkForUpdates() error {
	scripts, err := sm.scriptService.GetEnabledScripts()
	if err != nil {
		return fmt.Errorf("failed to get enabled scripts: %w", err)
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Track which scripts are still enabled
	currentScripts := make(map[string]bool)

	for _, script := range scripts {
		currentScripts[script.ChannelType] = true
		scriptKey := fmt.Sprintf("%d", script.ID)

		// Calculate version hash (using updated_at + script hash)
		versionHash := fmt.Sprintf("%s-%s", script.UpdatedAt.Format(time.RFC3339), sm.calculateScriptHash(script.Script))

		// Check if script needs reloading
		if existingVersion, exists := sm.scriptVersions[scriptKey]; !exists || existingVersion != versionHash {
			logrus.WithFields(logrus.Fields{
				"script":      script.Name,
				"channel_type": script.ChannelType,
				"old_version": existingVersion,
				"new_version": versionHash,
			}).Info("Script update detected, reloading")

			if err := sm.loadScript(script); err != nil {
				logrus.WithError(err).WithField("script", script.Name).Error("Failed to reload script")
				continue
			}

			sm.scriptVersions[scriptKey] = versionHash
		}
	}

	// Remove scripts that are no longer enabled
	for channelType := range sm.activeScripts {
		if !currentScripts[channelType] {
			delete(sm.activeScripts, channelType)
			sm.channelFactory.UnregisterDynamicChannel(channelType)
			logrus.WithField("channel_type", channelType).Info("Script removed from active scripts")
		}
	}

	return nil
}

// loadAllEnabledScripts loads all enabled scripts
func (sm *ScriptManager) loadAllEnabledScripts() error {
	scripts, err := sm.scriptService.GetEnabledScripts()
	if err != nil {
		return fmt.Errorf("failed to get enabled scripts: %w", err)
	}

	for _, script := range scripts {
		if err := sm.loadScript(script); err != nil {
			logrus.WithError(err).WithField("script", script.Name).Error("Failed to load script")
			continue
		}

		scriptKey := fmt.Sprintf("%d", script.ID)
		versionHash := fmt.Sprintf("%s-%s", script.UpdatedAt.Format(time.RFC3339), sm.calculateScriptHash(script.Script))
		sm.scriptVersions[scriptKey] = versionHash
	}

	return nil
}

// loadScript loads a single script into the runtime
func (sm *ScriptManager) loadScript(script *models.ChannelScript) error {
	// Create script runtime to validate the script
	runtime, err := channel.NewScriptRuntime(script)
	if err != nil {
		return fmt.Errorf("failed to create script runtime: %w", err)
	}

	// Create a constructor function for this script
	scriptConstructor := func(f *channel.Factory, group *models.Group) (channel.ChannelProxy, error) {
		return channel.NewScriptChannel(f, group, script)
	}

	// Register with channel factory
	sm.channelFactory.RegisterDynamicChannel(script.ChannelType, scriptConstructor)

	// Store in active scripts
	sm.activeScripts[script.ChannelType] = runtime

	logrus.WithFields(logrus.Fields{
		"script":      script.Name,
		"channel_type": script.ChannelType,
		"version":     script.Version,
	}).Info("Script loaded successfully")

	return nil
}

// calculateScriptHash calculates a hash of the script content for version tracking
func (sm *ScriptManager) calculateScriptHash(script string) string {
	hash := sha256.Sum256([]byte(script))
	return fmt.Sprintf("%x", hash)[:16] // Use first 16 characters
}
