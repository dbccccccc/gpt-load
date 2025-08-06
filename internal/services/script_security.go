package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dop251/goja"
)

// ScriptSecurityValidator provides security validation for JavaScript scripts
type ScriptSecurityValidator struct {
	// Patterns that are not allowed in scripts
	forbiddenPatterns []string
	// Maximum script size in bytes
	maxScriptSize int
}

// NewScriptSecurityValidator creates a new script security validator
func NewScriptSecurityValidator() *ScriptSecurityValidator {
	return &ScriptSecurityValidator{
		forbiddenPatterns: []string{
			// Dangerous JavaScript patterns
			`eval\s*\(`,
			`Function\s*\(`,
			`setTimeout\s*\(`,
			`setInterval\s*\(`,
			`setImmediate\s*\(`,
			`require\s*\(`,
			`import\s+.*from`,
			`import\s*\(`,
			`process\.`,
			`global\.`,
			`globalThis\.`,
			`window\.`,
			`document\.`,
			`location\.`,
			`navigator\.`,
			`XMLHttpRequest`,
			`fetch\s*\(`,
			`WebSocket`,
			`Worker\s*\(`,
			`SharedWorker\s*\(`,
			`ServiceWorker`,
			`localStorage`,
			`sessionStorage`,
			`indexedDB`,
			`crypto\.subtle`,
			`performance\.`,
			`console\.trace`,
			`console\.profile`,
			`debugger`,
			`__proto__`,
			`constructor\.constructor`,
			`\.call\s*\(\s*null`,
			`\.apply\s*\(\s*null`,
			`\.bind\s*\(\s*null`,
			// File system and network access attempts
			`fs\.`,
			`path\.`,
			`os\.`,
			`child_process`,
			`cluster\.`,
			`net\.`,
			`http\.`,
			`https\.`,
			`url\.`,
			`querystring\.`,
			// Potential code injection
			`new\s+Function`,
			`\.constructor\s*\(`,
			`String\.fromCharCode`,
			`String\.fromCodePoint`,
			`unescape\s*\(`,
			`decodeURI\s*\(`,
			`decodeURIComponent\s*\(`,
			// Infinite loops and resource exhaustion
			`while\s*\(\s*true\s*\)`,
			`for\s*\(\s*;\s*;\s*\)`,
			`setInterval\s*\(\s*.*,\s*0\s*\)`,
		},
		maxScriptSize: 1024 * 1024, // 1MB max
	}
}

// ValidateScript performs comprehensive security validation on a JavaScript script
func (v *ScriptSecurityValidator) ValidateScript(script string) error {
	// Check script size
	if len(script) > v.maxScriptSize {
		return fmt.Errorf("script too large: %d bytes (max %d bytes)", len(script), v.maxScriptSize)
	}

	// Check for forbidden patterns
	for _, pattern := range v.forbiddenPatterns {
		matched, err := regexp.MatchString(pattern, script)
		if err != nil {
			continue // Skip invalid regex patterns
		}
		if matched {
			return fmt.Errorf("script contains forbidden pattern: %s", pattern)
		}
	}

	// Check for excessive complexity
	if err := v.validateComplexity(script); err != nil {
		return err
	}

	// Validate syntax by attempting to parse
	if err := v.validateSyntax(script); err != nil {
		return err
	}

	// Check for required structure
	if err := v.validateStructure(script); err != nil {
		return err
	}

	return nil
}

// validateComplexity checks for potentially problematic code complexity
func (v *ScriptSecurityValidator) validateComplexity(script string) error {
	lines := strings.Split(script, "\n")

	// Check line count
	if len(lines) > 10000 {
		return fmt.Errorf("script too complex: %d lines (max 10000)", len(lines))
	}

	// Check for excessive nesting
	maxNesting := 0
	currentNesting := 0
	for _, line := range lines {
		openBraces := strings.Count(line, "{")
		closeBraces := strings.Count(line, "}")
		currentNesting += openBraces - closeBraces
		if currentNesting > maxNesting {
			maxNesting = currentNesting
		}
		if maxNesting > 20 {
			return fmt.Errorf("script too complex: excessive nesting depth (%d levels)", maxNesting)
		}
	}

	// Check for excessive function definitions
	functionCount := len(regexp.MustCompile(`function\s+\w+`).FindAllString(script, -1))
	if functionCount > 100 {
		return fmt.Errorf("script too complex: too many functions (%d, max 100)", functionCount)
	}

	return nil
}

// validateSyntax checks if the script has valid JavaScript syntax
func (v *ScriptSecurityValidator) validateSyntax(script string) error {
	vm := goja.New()

	// Try to parse the script
	_, err := vm.RunString(script)
	if err != nil {
		return fmt.Errorf("syntax error: %w", err)
	}

	return nil
}

// validateStructure ensures the script follows the required channel structure
func (v *ScriptSecurityValidator) validateStructure(script string) error {
	vm := goja.New()

	// Execute the script
	_, err := vm.RunString(script)
	if err != nil {
		return fmt.Errorf("script execution failed: %w", err)
	}

	// Check for exports function
	exportFunc := vm.Get("exports")
	if exportFunc == nil {
		return fmt.Errorf("script must define an 'exports' function")
	}

	if _, ok := goja.AssertFunction(exportFunc); !ok {
		return fmt.Errorf("'exports' must be a function")
	}

	// Try to call exports and validate the returned object
	channelValue, err := vm.RunString("exports()")
	if err != nil {
		return fmt.Errorf("exports() function failed: %w", err)
	}

	channelObj := channelValue.ToObject(vm)
	if channelObj == nil {
		return fmt.Errorf("exports() must return an object")
	}

	// Check for required methods
	requiredMethods := []string{
		"buildUpstreamURL",
		"modifyRequest",
		"isStreamRequest",
		"extractModel",
		"validateKey",
	}

	for _, method := range requiredMethods {
		methodValue := channelObj.Get(method)
		if methodValue == nil {
			return fmt.Errorf("missing required method: %s", method)
		}
		if _, ok := goja.AssertFunction(methodValue); !ok {
			return fmt.Errorf("'%s' must be a function", method)
		}
	}

	// Check for metadata
	metadataValue := channelObj.Get("metadata")
	if metadataValue == nil {
		return fmt.Errorf("missing required 'metadata' property")
	}

	metadataObj := metadataValue.ToObject(vm)
	if metadataObj == nil {
		return fmt.Errorf("'metadata' must be an object")
	}

	// Check required metadata fields
	requiredMetadataFields := []string{
		"name", "version", "description", "author", "channel_type",
	}

	for _, field := range requiredMetadataFields {
		fieldValue := metadataObj.Get(field)
		if fieldValue == nil || goja.IsUndefined(fieldValue) {
			return fmt.Errorf("missing required metadata field: %s", field)
		}
	}

	return nil
}
