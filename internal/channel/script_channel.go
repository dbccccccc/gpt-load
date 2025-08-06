package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"gpt-load/internal/models"
	"net/http"
	"net/url"

	"github.com/dop251/goja"
	"github.com/gin-gonic/gin"
)

// ScriptChannel implements ChannelProxy interface for JavaScript-based channels
type ScriptChannel struct {
	*BaseChannel
	runtime *ScriptRuntime
	channel goja.Value // The JavaScript channel object
}

// NewScriptChannel creates a new script-based channel
func NewScriptChannel(f *Factory, group *models.Group, script *models.ChannelScript) (ChannelProxy, error) {
	base, err := f.newBaseChannel(script.ChannelType, group)
	if err != nil {
		return nil, err
	}

	// Create JavaScript runtime
	runtime, err := NewScriptRuntime(script)
	if err != nil {
		return nil, fmt.Errorf("failed to create script runtime: %w", err)
	}

	// Get the channel instance from the script
	exportFunc := runtime.vm.Get("exports")
	if exportFunc == nil {
		return nil, fmt.Errorf("script must export a function")
	}

	// Check if it's a function
	if callable, ok := goja.AssertFunction(exportFunc); !ok {
		return nil, fmt.Errorf("exports must be a function")
	} else {
		_ = callable // Use the callable if needed
	}

	// Call the export function to get the channel instance
	channelValue, err := runtime.vm.RunString("exports()")
	if err != nil {
		return nil, fmt.Errorf("failed to create channel instance: %w", err)
	}

	return &ScriptChannel{
		BaseChannel: base,
		runtime:     runtime,
		channel:     channelValue,
	}, nil
}

// BuildUpstreamURL delegates to the JavaScript implementation
func (sc *ScriptChannel) BuildUpstreamURL(originalURL *url.URL, group *models.Group) (string, error) {
	// Call the JavaScript buildUpstreamURL method
	channelObj := sc.channel.ToObject(sc.runtime.vm)
	buildURLFunc := channelObj.Get("buildUpstreamURL")

	if buildURLFunc == nil {
		// Fallback to base implementation
		return sc.BaseChannel.BuildUpstreamURL(originalURL, group)
	}

	// Check if it's a function
	callable, ok := goja.AssertFunction(buildURLFunc)
	if !ok {
		// Fallback to base implementation
		return sc.BaseChannel.BuildUpstreamURL(originalURL, group)
	}

	// Convert group to JavaScript object
	groupJSON, err := json.Marshal(group)
	if err != nil {
		return "", fmt.Errorf("failed to marshal group: %w", err)
	}

	var groupObj interface{}
	if err := json.Unmarshal(groupJSON, &groupObj); err != nil {
		return "", fmt.Errorf("failed to unmarshal group: %w", err)
	}

	// Call the JavaScript function
	result, err := callable(goja.Undefined(),
		sc.runtime.vm.ToValue(originalURL.String()),
		sc.runtime.vm.ToValue(groupObj))

	if err != nil {
		return "", fmt.Errorf("JavaScript buildUpstreamURL failed: %w", err)
	}

	return result.String(), nil
}

// ModifyRequest delegates to the JavaScript implementation
func (sc *ScriptChannel) ModifyRequest(req *http.Request, apiKey *models.APIKey, group *models.Group) {
	channelObj := sc.channel.ToObject(sc.runtime.vm)
	modifyFunc := channelObj.Get("modifyRequest")

	if modifyFunc == nil {
		return // No modification needed
	}

	// Check if it's a function
	modifyCallable, ok := goja.AssertFunction(modifyFunc)
	if !ok {
		return // No modification needed
	}

	// Convert request to JavaScript object
	reqObj := sc.runtime.vm.NewObject()
	reqObj.Set("method", req.Method)
	reqObj.Set("url", req.URL.String())

	// Set headers
	headersObj := sc.runtime.vm.NewObject()
	for key, values := range req.Header {
		if len(values) > 0 {
			headersObj.Set(key, values[0])
		}
	}
	reqObj.Set("headers", headersObj)

	// Set query parameters
	queryObj := sc.runtime.vm.NewObject()
	for key, values := range req.URL.Query() {
		if len(values) > 0 {
			queryObj.Set(key, values[0])
		}
	}
	reqObj.Set("query", queryObj)

	// Convert API key and group to JavaScript objects
	apiKeyJSON, _ := json.Marshal(apiKey)
	groupJSON, _ := json.Marshal(group)

	var apiKeyObj, groupObj interface{}
	json.Unmarshal(apiKeyJSON, &apiKeyObj)
	json.Unmarshal(groupJSON, &groupObj)

	// Call the JavaScript function
	modifyCallable(goja.Undefined(),
		reqObj,
		sc.runtime.vm.ToValue(apiKeyObj),
		sc.runtime.vm.ToValue(groupObj))

	// Apply modifications back to the request
	if modifiedHeaders := reqObj.Get("headers"); modifiedHeaders != nil {
		headersObj := modifiedHeaders.ToObject(sc.runtime.vm)
		for _, key := range headersObj.Keys() {
			value := headersObj.Get(key).String()
			req.Header.Set(key, value)
		}
	}
}

// IsStreamRequest delegates to the JavaScript implementation
func (sc *ScriptChannel) IsStreamRequest(c *gin.Context, bodyBytes []byte) bool {
	channelObj := sc.channel.ToObject(sc.runtime.vm)
	isStreamFunc := channelObj.Get("isStreamRequest")

	if isStreamFunc == nil {
		return false // Default to non-streaming
	}

	// Check if it's a function
	streamCallable, ok := goja.AssertFunction(isStreamFunc)
	if !ok {
		return false // Default to non-streaming
	}

	// Create context object
	contextObj := sc.runtime.vm.NewObject()

	// Set request object
	reqObj := sc.runtime.vm.NewObject()
	reqObj.Set("method", c.Request.Method)
	reqObj.Set("url", c.Request.URL.String())

	headersObj := sc.runtime.vm.NewObject()
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headersObj.Set(key, values[0])
		}
	}
	reqObj.Set("headers", headersObj)

	queryObj := sc.runtime.vm.NewObject()
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryObj.Set(key, values[0])
		}
	}
	reqObj.Set("query", queryObj)

	contextObj.Set("request", reqObj)
	contextObj.Set("body_bytes", sc.runtime.vm.ToValue(bodyBytes))
	contextObj.Set("original_url", c.Request.URL.String())

	// Call the JavaScript function
	result, err := streamCallable(goja.Undefined(), contextObj)
	if err != nil {
		return false
	}

	return result.ToBoolean()
}

// ExtractModel delegates to the JavaScript implementation
func (sc *ScriptChannel) ExtractModel(c *gin.Context, bodyBytes []byte) string {
	channelObj := sc.channel.ToObject(sc.runtime.vm)
	extractFunc := channelObj.Get("extractModel")

	if extractFunc == nil {
		return "" // Default to empty model
	}

	// Check if it's a function
	extractCallable, ok := goja.AssertFunction(extractFunc)
	if !ok {
		return "" // Default to empty model
	}

	// Create context object
	contextObj := sc.runtime.vm.NewObject()

	// Set request object
	reqObj := sc.runtime.vm.NewObject()
	reqObj.Set("method", c.Request.Method)
	reqObj.Set("url", c.Request.URL.String())

	headersObj := sc.runtime.vm.NewObject()
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headersObj.Set(key, values[0])
		}
	}
	reqObj.Set("headers", headersObj)

	queryObj := sc.runtime.vm.NewObject()
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryObj.Set(key, values[0])
		}
	}
	reqObj.Set("query", queryObj)

	contextObj.Set("request", reqObj)
	contextObj.Set("body_bytes", sc.runtime.vm.ToValue(bodyBytes))
	contextObj.Set("original_url", c.Request.URL.String())

	// Call the JavaScript function
	result, err := extractCallable(goja.Undefined(), contextObj)
	if err != nil {
		return ""
	}

	return result.String()
}

// ValidateKey delegates to the JavaScript implementation
func (sc *ScriptChannel) ValidateKey(ctx context.Context, key string) (bool, error) {
	channelObj := sc.channel.ToObject(sc.runtime.vm)
	validateFunc := channelObj.Get("validateKey")

	if validateFunc == nil {
		return false, fmt.Errorf("validateKey method not implemented in script")
	}

	// Check if it's a function
	validateCallable, ok := goja.AssertFunction(validateFunc)
	if !ok {
		return false, fmt.Errorf("validateKey method not implemented in script")
	}

	// Convert group to JavaScript object
	groupJSON, err := json.Marshal(sc.BaseChannel.effectiveConfig)
	if err != nil {
		return false, fmt.Errorf("failed to marshal group config: %w", err)
	}

	var groupObj interface{}
	if err := json.Unmarshal(groupJSON, &groupObj); err != nil {
		return false, fmt.Errorf("failed to unmarshal group config: %w", err)
	}

	// Call the JavaScript function
	result, err := validateCallable(goja.Undefined(),
		sc.runtime.vm.ToValue(key),
		sc.runtime.vm.ToValue(groupObj))

	if err != nil {
		return false, fmt.Errorf("JavaScript validateKey failed: %w", err)
	}

	// The result should be a Promise or an object with {valid: boolean, error?: string}
	resultObj := result.ToObject(sc.runtime.vm)
	validValue := resultObj.Get("valid")
	if validValue == nil {
		return false, fmt.Errorf("validateKey must return an object with 'valid' property")
	}

	valid := validValue.ToBoolean()
	if !valid {
		if errorValue := resultObj.Get("error"); errorValue != nil && !goja.IsUndefined(errorValue) {
			return false, fmt.Errorf(errorValue.String())
		}
		return false, fmt.Errorf("key validation failed")
	}

	return true, nil
}
