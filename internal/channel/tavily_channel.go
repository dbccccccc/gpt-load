package channel

import (
	"context"
	"fmt"
	app_errors "gpt-load/internal/errors"
	"gpt-load/internal/models"
	"io"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

func init() {
	Register("tavily", newTavilyChannel)
}

type TavilyChannel struct {
	*BaseChannel
}

func newTavilyChannel(f *Factory, group *models.Group) (ChannelProxy, error) {
	base, err := f.newBaseChannel("tavily", group)
	if err != nil {
		return nil, err
	}

	return &TavilyChannel{
		BaseChannel: base,
	}, nil
}

// ModifyRequest sets the Authorization header for the Tavily service.
func (ch *TavilyChannel) ModifyRequest(req *http.Request, apiKey *models.APIKey, group *models.Group) {
	req.Header.Set("Authorization", "Bearer "+apiKey.KeyValue)
}

// IsStreamRequest checks if the request is for a streaming response.
// Tavily API doesn't support streaming, so this always returns false.
func (ch *TavilyChannel) IsStreamRequest(c *gin.Context, bodyBytes []byte) bool {
	// Tavily API doesn't support streaming responses
	return false
}

// ValidateKey checks if the given API key is valid by making a usage request.
func (ch *TavilyChannel) ValidateKey(ctx context.Context, key string) (bool, error) {
	upstreamURL := ch.getUpstreamURL()
	if upstreamURL == nil {
		return false, fmt.Errorf("no upstream URL configured for channel %s", ch.Name)
	}

	// Use the usage endpoint for validation as it's a simple GET request
	validationEndpoint := ch.ValidationEndpoint
	if validationEndpoint == "" {
		validationEndpoint = "/usage"
	}
	reqURL, err := url.JoinPath(upstreamURL.String(), validationEndpoint)
	if err != nil {
		return false, fmt.Errorf("failed to join upstream URL and validation endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create validation request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := ch.HTTPClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send validation request: %w", err)
	}
	defer resp.Body.Close()

	// A 200 OK status code indicates the key is valid and can make requests.
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	// For non-200 responses, parse the body to provide a more specific error reason.
	errorBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("key is invalid (status %d), but failed to read error body: %w", resp.StatusCode, err)
	}

	// Use the new parser to extract a clean error message.
	parsedError := app_errors.ParseUpstreamError(errorBody)

	return false, fmt.Errorf("[status %d] %s", resp.StatusCode, parsedError)
}
