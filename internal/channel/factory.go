package channel

import (
	"encoding/json"
	"fmt"
	"gpt-load/internal/config"
	"gpt-load/internal/httpclient"
	"gpt-load/internal/models"
	"net/url"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// channelConstructor defines the function signature for creating a new channel proxy.
type channelConstructor func(f *Factory, group *models.Group) (ChannelProxy, error)

var (
	// channelRegistry holds the mapping from channel type string to its constructor.
	channelRegistry = make(map[string]channelConstructor)
)

// Register adds a new channel constructor to the registry.
func Register(channelType string, constructor channelConstructor) {
	if _, exists := channelRegistry[channelType]; exists {
		panic(fmt.Sprintf("channel type '%s' is already registered", channelType))
	}
	channelRegistry[channelType] = constructor
}

// GetChannels returns a slice of all registered channel type names.
func GetChannels() []string {
	supportedTypes := make([]string, 0, len(channelRegistry))
	for t := range channelRegistry {
		supportedTypes = append(supportedTypes, t)
	}
	return supportedTypes
}

// Factory is responsible for creating channel proxies.
type Factory struct {
	settingsManager  *config.SystemSettingsManager
	clientManager    *httpclient.HTTPClientManager
	db               *gorm.DB
	channelCache     map[uint]ChannelProxy
	cacheLock        sync.Mutex
	staticChannels   map[string]channelConstructor
	dynamicChannels  map[string]channelConstructor
	mutex            sync.RWMutex
}

// NewFactory creates a new channel factory.
func NewFactory(settingsManager *config.SystemSettingsManager, clientManager *httpclient.HTTPClientManager, db *gorm.DB) *Factory {
	return &Factory{
		settingsManager: settingsManager,
		clientManager:   clientManager,
		db:              db,
		channelCache:    make(map[uint]ChannelProxy),
		staticChannels:  channelRegistry,
		dynamicChannels: make(map[string]channelConstructor),
	}
}

// GetChannel returns a channel proxy based on the group's channel type.
func (f *Factory) GetChannel(group *models.Group) (ChannelProxy, error) {
	f.cacheLock.Lock()
	defer f.cacheLock.Unlock()

	if channel, ok := f.channelCache[group.ID]; ok {
		if !channel.IsConfigStale(group) {
			return channel, nil
		}
	}

	logrus.Debugf("Creating new channel for group %d with type '%s'", group.ID, group.ChannelType)

	// First check if this is a dynamic channel type (hot-reloaded scripts)
	f.mutex.RLock()
	if dynamicConstructor, exists := f.dynamicChannels[group.ChannelType]; exists {
		f.mutex.RUnlock()
		channel, err := dynamicConstructor(f, group)
		if err != nil {
			return nil, err
		}
		f.channelCache[group.ID] = channel
		return channel, nil
	}
	f.mutex.RUnlock()

	// Then check if this is a static channel type
	constructor, ok := channelRegistry[group.ChannelType]
	if ok {
		channel, err := constructor(f, group)
		if err != nil {
			return nil, err
		}
		f.channelCache[group.ID] = channel
		return channel, nil
	}

	// If not a static or dynamic channel, check for script channel
	scriptChannel, err := f.createScriptChannel(group)
	if err != nil {
		return nil, fmt.Errorf("unsupported channel type '%s' and no script found: %w", group.ChannelType, err)
	}

	f.channelCache[group.ID] = scriptChannel
	return scriptChannel, nil
}

// createScriptChannel creates a script-based channel for the given group
func (f *Factory) createScriptChannel(group *models.Group) (ChannelProxy, error) {
	// Look for an enabled script with the matching channel type
	var script models.ChannelScript
	err := f.db.Where("channel_type = ? AND status = ?", group.ChannelType, "enabled").First(&script).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no enabled script found for channel type: %s", group.ChannelType)
		}
		return nil, fmt.Errorf("failed to query script for channel type %s: %w", group.ChannelType, err)
	}

	// Create the script channel
	scriptChannel, err := NewScriptChannel(f, group, &script)
	if err != nil {
		// Update script status to error
		now := time.Now()
		f.db.Model(&script).Updates(map[string]interface{}{
			"status":     "error",
			"error_msg":  err.Error(),
			"last_error": &now,
		})
		return nil, fmt.Errorf("failed to create script channel: %w", err)
	}

	logrus.Infof("Created script channel '%s' for group %d", script.Name, group.ID)
	return scriptChannel, nil
}

// InvalidateCache removes a channel from the cache, forcing recreation on next access
func (f *Factory) InvalidateCache(groupID uint) {
	f.cacheLock.Lock()
	defer f.cacheLock.Unlock()
	delete(f.channelCache, groupID)
	logrus.Debugf("Invalidated channel cache for group %d", groupID)
}

// InvalidateAllCache clears the entire channel cache
func (f *Factory) InvalidateAllCache() {
	f.cacheLock.Lock()
	defer f.cacheLock.Unlock()
	f.channelCache = make(map[uint]ChannelProxy)
	logrus.Debug("Invalidated all channel cache")
}

// newBaseChannel is a helper function to create and configure a BaseChannel.
func (f *Factory) newBaseChannel(name string, group *models.Group) (*BaseChannel, error) {
	type upstreamDef struct {
		URL    string `json:"url"`
		Weight int    `json:"weight"`
	}

	var defs []upstreamDef
	if err := json.Unmarshal(group.Upstreams, &defs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal upstreams for %s channel: %w", name, err)
	}

	if len(defs) == 0 {
		return nil, fmt.Errorf("at least one upstream is required for %s channel", name)
	}

	var upstreamInfos []UpstreamInfo
	for _, def := range defs {
		u, err := url.Parse(def.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse upstream url '%s' for %s channel: %w", def.URL, name, err)
		}
		weight := def.Weight
		if weight <= 0 {
			weight = 1
		}
		upstreamInfos = append(upstreamInfos, UpstreamInfo{URL: u, Weight: weight})
	}

	// Base configuration for regular requests, derived from the group's effective settings.
	clientConfig := &httpclient.Config{
		ConnectTimeout:        time.Duration(group.EffectiveConfig.ConnectTimeout) * time.Second,
		RequestTimeout:        time.Duration(group.EffectiveConfig.RequestTimeout) * time.Second,
		IdleConnTimeout:       time.Duration(group.EffectiveConfig.IdleConnTimeout) * time.Second,
		MaxIdleConns:          group.EffectiveConfig.MaxIdleConns,
		MaxIdleConnsPerHost:   group.EffectiveConfig.MaxIdleConnsPerHost,
		ResponseHeaderTimeout: time.Duration(group.EffectiveConfig.ResponseHeaderTimeout) * time.Second,
		DisableCompression:    false,
		WriteBufferSize:       32 * 1024,
		ReadBufferSize:        32 * 1024,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Create a dedicated configuration for streaming requests.
	streamConfig := *clientConfig
	streamConfig.RequestTimeout = 0
	streamConfig.DisableCompression = true
	streamConfig.WriteBufferSize = 0
	streamConfig.ReadBufferSize = 0
	// Use a larger, independent connection pool for streaming clients to avoid exhaustion.
	streamConfig.MaxIdleConns = max(group.EffectiveConfig.MaxIdleConns*2, 50)
	streamConfig.MaxIdleConnsPerHost = max(group.EffectiveConfig.MaxIdleConnsPerHost*2, 20)

	// Get both clients from the manager using their respective configurations.
	httpClient := f.clientManager.GetClient(clientConfig)
	streamClient := f.clientManager.GetClient(&streamConfig)

	return &BaseChannel{
		Name:               name,
		Upstreams:          upstreamInfos,
		HTTPClient:         httpClient,
		StreamClient:       streamClient,
		TestModel:          group.TestModel,
		ValidationEndpoint: group.ValidationEndpoint,
		channelType:        group.ChannelType,
		groupUpstreams:     group.Upstreams,
		effectiveConfig:    &group.EffectiveConfig,
	}, nil
}

// RegisterDynamicChannel registers a dynamic script-based channel constructor
func (f *Factory) RegisterDynamicChannel(channelType string, constructor channelConstructor) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.dynamicChannels[channelType] = constructor
	logrus.WithField("channel_type", channelType).Info("Dynamic channel registered")
}

// UnregisterDynamicChannel removes a dynamic channel
func (f *Factory) UnregisterDynamicChannel(channelType string) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	delete(f.dynamicChannels, channelType)
	logrus.WithField("channel_type", channelType).Info("Dynamic channel unregistered")
}

// GetRegisteredChannelTypes returns all registered channel types
func (f *Factory) GetRegisteredChannelTypes() []string {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	var types []string

	// Add static channel types
	for channelType := range f.staticChannels {
		types = append(types, channelType)
	}

	// Add dynamic channel types
	for channelType := range f.dynamicChannels {
		types = append(types, channelType)
	}

	return types
}

// IsDynamicChannel checks if a channel type is dynamic (script-based)
func (f *Factory) IsDynamicChannel(channelType string) bool {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	_, exists := f.dynamicChannels[channelType]
	return exists
}
