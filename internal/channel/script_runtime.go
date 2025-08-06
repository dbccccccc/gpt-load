package channel

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gpt-load/internal/models"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/sirupsen/logrus"
)

// ScriptRuntime manages JavaScript execution for custom channels
type ScriptRuntime struct {
	vm              *goja.Runtime
	script          *models.ChannelScript
	metadata        *models.ChannelScriptMetadata
	mutex           sync.RWMutex
	httpRequestCount int
	lastRequestTime  time.Time
	logCount        int
	lastLogTime     time.Time
}

// NewScriptRuntime creates a new JavaScript runtime for a channel script
func NewScriptRuntime(script *models.ChannelScript) (*ScriptRuntime, error) {
	vm := goja.New()

	// Parse metadata
	var metadata models.ChannelScriptMetadata
	if err := json.Unmarshal(script.Metadata, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse script metadata: %w", err)
	}

	runtime := &ScriptRuntime{
		vm:       vm,
		script:   script,
		metadata: &metadata,
	}

	// Setup the JavaScript environment
	if err := runtime.setupEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to setup JavaScript environment: %w", err)
	}

	// Execute the script to load the channel implementation
	if err := runtime.loadScript(); err != nil {
		return nil, fmt.Errorf("failed to load script: %w", err)
	}

	return runtime, nil
}

// setupEnvironment configures the JavaScript runtime with utility functions
func (sr *ScriptRuntime) setupEnvironment() error {
	// Disable dangerous globals and functions
	sr.vm.Set("eval", goja.Undefined())
	sr.vm.Set("Function", goja.Undefined())
	sr.vm.Set("setTimeout", goja.Undefined())
	sr.vm.Set("setInterval", goja.Undefined())
	sr.vm.Set("setImmediate", goja.Undefined())
	sr.vm.Set("require", goja.Undefined())
	sr.vm.Set("process", goja.Undefined())
	sr.vm.Set("global", goja.Undefined())
	sr.vm.Set("globalThis", goja.Undefined())

	// Create utils object with helper functions
	utils := sr.vm.NewObject()

	// HTTP request function with rate limiting
	utils.Set("httpRequest", sr.createHTTPRequestFunc())

	// JSON utilities
	utils.Set("parseJSON", sr.createParseJSONFunc())

	// Base64 utilities
	utils.Set("base64Encode", sr.createBase64EncodeFunc())
	utils.Set("base64Decode", sr.createBase64DecodeFunc())

	// URL utilities
	utils.Set("parseURL", sr.createParseURLFunc())
	utils.Set("joinURL", sr.createJoinURLFunc())

	// Logging functions with rate limiting
	logObj := sr.vm.NewObject()
	logObj.Set("debug", sr.createLogFunc("debug"))
	logObj.Set("info", sr.createLogFunc("info"))
	logObj.Set("warn", sr.createLogFunc("warn"))
	logObj.Set("error", sr.createLogFunc("error"))
	utils.Set("log", logObj)

	// Set global utils (read-only)
	sr.vm.Set("utils", utils)

	// Set console for compatibility (limited)
	console := sr.vm.NewObject()
	console.Set("log", sr.createLogFunc("info"))
	console.Set("error", sr.createLogFunc("error"))
	console.Set("warn", sr.createLogFunc("warn"))
	console.Set("debug", sr.createLogFunc("debug"))
	sr.vm.Set("console", console)

	// Set execution limits
	sr.vm.SetMaxCallStackSize(100)

	return nil
}

// loadScript executes the JavaScript code and validates the export
func (sr *ScriptRuntime) loadScript() error {
	// Execute the script
	_, err := sr.vm.RunString(sr.script.Script)
	if err != nil {
		return fmt.Errorf("script execution failed: %w", err)
	}

	// Check if the script exports a function
	exportFunc := sr.vm.Get("exports")
	if exportFunc == nil {
		return fmt.Errorf("script must export a function")
	}

	// Check if it's a function
	if _, ok := goja.AssertFunction(exportFunc); !ok {
		return fmt.Errorf("script must export a function")
	}

	return nil
}

// createHTTPRequestFunc creates the httpRequest utility function with security measures
func (sr *ScriptRuntime) createHTTPRequestFunc() func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(sr.vm.NewTypeError("httpRequest requires a request object"))
		}

		// Rate limiting: max 10 requests per minute
		sr.mutex.Lock()
		now := time.Now()
		if now.Sub(sr.lastRequestTime) < time.Minute {
			sr.httpRequestCount++
			if sr.httpRequestCount > 10 {
				sr.mutex.Unlock()
				panic(sr.vm.NewGoError(fmt.Errorf("rate limit exceeded: max 10 requests per minute")))
			}
		} else {
			sr.httpRequestCount = 1
			sr.lastRequestTime = now
		}
		sr.mutex.Unlock()

		// Convert JavaScript object to Go struct
		reqObj := call.Arguments[0].ToObject(sr.vm)
		method := reqObj.Get("method").String()
		urlStr := reqObj.Get("url").String()
		body := ""
		if bodyVal := reqObj.Get("body"); bodyVal != nil && !goja.IsUndefined(bodyVal) {
			body = bodyVal.String()
		}

		// Validate URL and prevent SSRF attacks
		if err := sr.validateURL(urlStr); err != nil {
			panic(sr.vm.NewGoError(err))
		}

		// Limit request body size (1MB max)
		if len(body) > 1024*1024 {
			panic(sr.vm.NewGoError(fmt.Errorf("request body too large: max 1MB allowed")))
		}

		// Create HTTP request with context timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var req *http.Request
		var err error
		if body != "" {
			req, err = http.NewRequestWithContext(ctx, method, urlStr, strings.NewReader(body))
		} else {
			req, err = http.NewRequestWithContext(ctx, method, urlStr, nil)
		}

		if err != nil {
			panic(sr.vm.NewGoError(err))
		}

		// Set headers with validation
		if headersVal := reqObj.Get("headers"); headersVal != nil && !goja.IsUndefined(headersVal) {
			headersObj := headersVal.ToObject(sr.vm)
			for _, key := range headersObj.Keys() {
				value := headersObj.Get(key).String()
				// Validate header values
				if sr.isValidHeaderValue(key, value) {
					req.Header.Set(key, value)
				}
			}
		}

		// Add security headers
		req.Header.Set("User-Agent", "GPT-Load-Script/1.0")

		// Create secure HTTP client
		client := &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 10 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout: 10 * time.Second,
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
			},
		}

		resp, err := client.Do(req)
		if err != nil {
			panic(sr.vm.NewGoError(err))
		}
		defer resp.Body.Close()

		// Create response object
		respObj := sr.vm.NewObject()
		respObj.Set("status_code", resp.StatusCode)

		// Set response headers
		headersObj := sr.vm.NewObject()
		for key, values := range resp.Header {
			if len(values) > 0 {
				headersObj.Set(key, values[0])
			}
		}
		respObj.Set("headers", headersObj)

		// Read response body with size limit (5MB max)
		bodyReader := io.LimitReader(resp.Body, 5*1024*1024)
		bodyBytes, err := io.ReadAll(bodyReader)
		if err != nil {
			panic(sr.vm.NewGoError(err))
		}
		respObj.Set("body", string(bodyBytes))

		return respObj
	}
}

// createParseJSONFunc creates the parseJSON utility function
func (sr *ScriptRuntime) createParseJSONFunc() func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Null()
		}

		jsonStr := call.Arguments[0].String()
		var result interface{}
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			return goja.Null()
		}

		return sr.vm.ToValue(result)
	}
}

// createBase64EncodeFunc creates the base64Encode utility function
func (sr *ScriptRuntime) createBase64EncodeFunc() func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return sr.vm.ToValue("")
		}

		data := call.Arguments[0].String()
		// Limit input size for base64 encoding (1MB max)
		if len(data) > 1024*1024 {
			panic(sr.vm.NewGoError(fmt.Errorf("base64 input too large: max 1MB allowed")))
		}

		encoded := base64.StdEncoding.EncodeToString([]byte(data))
		return sr.vm.ToValue(encoded)
	}
}

// createBase64DecodeFunc creates the base64Decode utility function
func (sr *ScriptRuntime) createBase64DecodeFunc() func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return sr.vm.ToValue("")
		}

		data := call.Arguments[0].String()
		// Limit input size for base64 decoding (1MB max)
		if len(data) > 1024*1024 {
			panic(sr.vm.NewGoError(fmt.Errorf("base64 input too large: max 1MB allowed")))
		}

		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			panic(sr.vm.NewGoError(fmt.Errorf("invalid base64 data: %w", err)))
		}

		return sr.vm.ToValue(string(decoded))
	}
}

// createParseURLFunc creates the parseURL utility function
func (sr *ScriptRuntime) createParseURLFunc() func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Null()
		}

		urlStr := call.Arguments[0].String()
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			return goja.Null()
		}

		urlObj := sr.vm.NewObject()
		urlObj.Set("protocol", parsedURL.Scheme)
		urlObj.Set("host", parsedURL.Host)
		urlObj.Set("pathname", parsedURL.Path)
		urlObj.Set("search", parsedURL.RawQuery)
		urlObj.Set("hash", parsedURL.Fragment)

		return urlObj
	}
}

// createJoinURLFunc creates the joinURL utility function
func (sr *ScriptRuntime) createJoinURLFunc() func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return sr.vm.ToValue("")
		}

		base := call.Arguments[0].String()
		path := call.Arguments[1].String()

		baseURL, err := url.Parse(base)
		if err != nil {
			return sr.vm.ToValue("")
		}

		joined, err := url.JoinPath(baseURL.String(), path)
		if err != nil {
			return sr.vm.ToValue("")
		}

		return sr.vm.ToValue(joined)
	}
}

// createLogFunc creates logging functions with rate limiting
func (sr *ScriptRuntime) createLogFunc(level string) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}

		// Rate limiting: max 50 log messages per minute
		sr.mutex.Lock()
		now := time.Now()
		if now.Sub(sr.lastLogTime) < time.Minute {
			sr.logCount++
			if sr.logCount > 50 {
				sr.mutex.Unlock()
				return goja.Undefined() // Silently drop excessive logs
			}
		} else {
			sr.logCount = 1
			sr.lastLogTime = now
		}
		sr.mutex.Unlock()

		message := call.Arguments[0].String()
		// Limit log message length
		if len(message) > 1000 {
			message = message[:1000] + "... (truncated)"
		}

		logEntry := logrus.WithFields(logrus.Fields{
			"script":      sr.script.Name,
			"script_type": sr.script.ChannelType,
		})

		switch level {
		case "debug":
			logEntry.Debug(message)
		case "info":
			logEntry.Info(message)
		case "warn":
			logEntry.Warn(message)
		case "error":
			logEntry.Error(message)
		}

		return goja.Undefined()
	}
}

// validateURL validates URLs and prevents SSRF attacks
func (sr *ScriptRuntime) validateURL(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTP and HTTPS schemes are allowed")
	}

	// Prevent access to private networks
	host := parsedURL.Hostname()
	if host == "" {
		return fmt.Errorf("URL must have a valid hostname")
	}

	// Check for localhost and private IP ranges
	if sr.isPrivateOrLocalhost(host) {
		return fmt.Errorf("access to private networks is not allowed")
	}

	// Validate hostname format
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9.-]+$`, host); !matched {
		return fmt.Errorf("invalid hostname format")
	}

	return nil
}

// isPrivateOrLocalhost checks if a hostname is localhost or private IP
func (sr *ScriptRuntime) isPrivateOrLocalhost(host string) bool {
	// Check for localhost
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	// Parse as IP address
	ip := net.ParseIP(host)
	if ip == nil {
		// Not an IP, check for localhost-like hostnames
		return strings.Contains(host, "localhost") || strings.Contains(host, "local")
	}

	// Check private IP ranges
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

// isValidHeaderValue validates HTTP header values
func (sr *ScriptRuntime) isValidHeaderValue(key, value string) bool {
	// Limit header value length
	if len(value) > 8192 {
		return false
	}

	// Block dangerous headers
	dangerousHeaders := []string{
		"host", "connection", "upgrade", "proxy-connection",
		"proxy-authenticate", "proxy-authorization", "te", "trailers",
		"transfer-encoding", "content-length",
	}

	keyLower := strings.ToLower(key)
	for _, dangerous := range dangerousHeaders {
		if keyLower == dangerous {
			return false
		}
	}

	// Validate header value format (basic check)
	for _, char := range value {
		if char < 32 || char > 126 {
			if char != '\t' { // Allow tab character
				return false
			}
		}
	}

	return true
}
