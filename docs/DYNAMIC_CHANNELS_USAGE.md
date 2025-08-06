# Dynamic Channels Usage Guide

This guide shows you how to use GPT-Load's Dynamic Channel System to add custom AI service integrations through JavaScript scripts.

## Quick Start

### 1. Upload a Custom Channel Script

**Via API:**

```bash
curl -X POST http://localhost:3001/api/scripts \
  -H "Authorization: Bearer your-admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Custom Channel",
    "display_name": "Custom AI Service",
    "description": "Integration for my custom AI service",
    "author": "Your Name",
    "version": "1.0.0",
    "channel_type": "my_custom_ai",
    "script": "function exports() { return { /* your implementation */ }; }",
    "metadata": {
      "name": "My Custom Channel",
      "version": "1.0.0",
      "description": "Custom AI service integration",
      "author": "Your Name",
      "channel_type": "my_custom_ai",
      "supported_models": ["model-1", "model-2"],
      "default_test_model": "model-1",
      "default_validation_endpoint": "/v1/chat/completions"
    }
  }'
```

**Via Web UI (Future):**

1. Navigate to Admin → Scripts
2. Click "Upload Script"
3. Paste your JavaScript code
4. Fill in metadata
5. Click "Save"

### 2. Enable the Script

```bash
curl -X POST http://localhost:3001/api/scripts/1/enable \
  -H "Authorization: Bearer your-admin-token"
```

### 3. Create a Group Using the Custom Channel

```bash
curl -X POST http://localhost:3001/api/groups \
  -H "Authorization: Bearer your-admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-custom-group",
    "display_name": "My Custom AI Group",
    "channel_type": "my_custom_ai",
    "upstreams": [{"url": "https://api.myservice.com", "weight": 1}],
    "test_model": "model-1"
  }'
```

### 4. Add API Keys

```bash
curl -X POST http://localhost:3001/api/keys/add-multiple \
  -H "Authorization: Bearer your-admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "group_id": 1,
    "keys": ["your-api-key-1", "your-api-key-2"]
  }'
```

### 5. Use the Proxy

```bash
curl -X POST http://localhost:3001/proxy/my-custom-group/v1/chat/completions \
  -H "Authorization: Bearer your-proxy-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "model-1",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## Real-World Examples

### Example 1: Ollama Integration

```javascript
function exports() {
  return {
    metadata: {
      name: "Ollama Channel",
      version: "1.0.0",
      description: "Integration for Ollama local LLM server",
      author: "Community",
      channel_type: "ollama",
      supported_models: ["llama2", "codellama", "mistral"],
      default_test_model: "llama2",
      default_validation_endpoint: "/api/generate",
    },

    buildUpstreamURL: function (originalUrl, group) {
      const url = utils.parseURL(originalUrl);
      const upstream = group.upstreams[0];
      const proxyPrefix = "/proxy/" + group.name;
      let path = url.pathname.replace(proxyPrefix, "");

      // Convert OpenAI format to Ollama format
      if (path.includes("/chat/completions")) {
        path = "/api/chat";
      } else if (path.includes("/completions")) {
        path = "/api/generate";
      }

      return utils.joinURL(upstream.url, path);
    },

    modifyRequest: function (request, apiKey, group) {
      // Ollama doesn't require API keys for local usage
      // Remove authorization header if present
      delete request.headers["Authorization"];

      // Transform OpenAI request format to Ollama format
      if (request.body) {
        try {
          const body = utils.parseJSON(request.body);
          if (body.messages) {
            // Convert to Ollama chat format
            const ollamaBody = {
              model: body.model,
              messages: body.messages,
              stream: body.stream || false,
            };
            request.body = JSON.stringify(ollamaBody);
          }
        } catch (e) {
          utils.log.warn("Could not transform request body");
        }
      }
    },

    isStreamRequest: function (context) {
      try {
        const body = utils.parseJSON(
          new TextDecoder().decode(context.body_bytes)
        );
        return body && body.stream === true;
      } catch (e) {
        return false;
      }
    },

    extractModel: function (context) {
      try {
        const body = utils.parseJSON(
          new TextDecoder().decode(context.body_bytes)
        );
        return body ? body.model : "";
      } catch (e) {
        return "";
      }
    },

    validateKey: function (apiKey, group) {
      // Ollama validation - just check if server is reachable
      return new Promise(function (resolve) {
        const upstream = group.upstreams[0];
        const testUrl = utils.joinURL(upstream.url, "/api/tags");

        utils
          .httpRequest({
            method: "GET",
            url: testUrl,
            headers: {},
          })
          .then(function (response) {
            resolve({ valid: response.status_code === 200 });
          })
          .catch(function (error) {
            resolve({ valid: false, error: error.message });
          });
      });
    },
  };
}
```

### Example 2: Azure OpenAI Integration

```javascript
function exports() {
  return {
    metadata: {
      name: "Azure OpenAI Channel",
      version: "1.0.0",
      description: "Integration for Azure OpenAI Service",
      author: "Community",
      channel_type: "azure_openai",
      supported_models: ["gpt-35-turbo", "gpt-4"],
      default_test_model: "gpt-35-turbo",
      default_validation_endpoint:
        "/openai/deployments/{deployment}/chat/completions",
    },

    buildUpstreamURL: function (originalUrl, group) {
      const url = utils.parseURL(originalUrl);
      const upstream = group.upstreams[0];
      const proxyPrefix = "/proxy/" + group.name;
      let path = url.pathname.replace(proxyPrefix, "");

      // Extract model from path and convert to Azure deployment format
      const model = this.extractModel({
        request: { url: originalUrl },
        body_bytes: new Uint8Array(),
      });
      if (model && path.includes("/chat/completions")) {
        path = `/openai/deployments/${model}/chat/completions?api-version=2023-12-01-preview`;
      }

      return utils.joinURL(upstream.url, path);
    },

    modifyRequest: function (request, apiKey, group) {
      // Azure uses api-key header instead of Authorization
      delete request.headers["Authorization"];
      request.headers["api-key"] = apiKey.key_value;

      // Remove model from request body (it's in the URL for Azure)
      if (request.body) {
        try {
          const body = utils.parseJSON(request.body);
          if (body.model) {
            delete body.model;
            request.body = JSON.stringify(body);
          }
        } catch (e) {
          utils.log.warn("Could not modify request body");
        }
      }
    },

    // ... other methods similar to OpenAI compatible
  };
}
```

## API Reference

### Script Management Endpoints

#### List Scripts

```
GET /api/scripts
```

#### Get Script

```
GET /api/scripts/:id
```

#### Create Script

```
POST /api/scripts
Content-Type: application/json

{
  "name": "string",
  "display_name": "string",
  "description": "string",
  "author": "string",
  "version": "string",
  "channel_type": "string",
  "script": "string",
  "metadata": { ... },
  "config": { ... }
}
```

#### Update Script

```
PUT /api/scripts/:id
Content-Type: application/json

{
  "script": "string",
  "metadata": { ... }
}
```

#### Delete Script

```
DELETE /api/scripts/:id
```

#### Enable Script

```
POST /api/scripts/:id/enable
```

#### Disable Script

```
POST /api/scripts/:id/disable
```

#### Validate Script

```
POST /api/scripts/validate
Content-Type: application/json

{
  "script": "string",
  "metadata": { ... }
}
```

#### Test Script

```
POST /api/scripts/test
Content-Type: application/json

{
  "script": "string",
  "metadata": { ... },
  "test_data": { ... }
}
```

## Best Practices

### 1. Error Handling

Always wrap your code in try-catch blocks:

```javascript
validateKey: function(apiKey, group) {
  return new Promise(function(resolve) {
    try {
      // Your validation logic
    } catch (error) {
      utils.log.error("Validation error: " + error.message);
      resolve({ valid: false, error: error.message });
    }
  });
}
```

### 2. Logging

Use appropriate log levels:

```javascript
utils.log.debug("Detailed debugging info");
utils.log.info("General information");
utils.log.warn("Warning messages");
utils.log.error("Error messages");
```

### 3. Request Transformation

Be careful when modifying request bodies:

```javascript
modifyRequest: function(request, apiKey, group) {
  if (request.body) {
    try {
      const body = utils.parseJSON(request.body);
      // Modify body
      request.body = JSON.stringify(body);
    } catch (e) {
      utils.log.warn("Could not parse request body");
    }
  }
}
```

### 4. Validation

Always validate inputs:

```javascript
buildUpstreamURL: function(originalUrl, group) {
  if (!group.upstreams || group.upstreams.length === 0) {
    throw new Error("No upstreams configured");
  }

  const url = utils.parseURL(originalUrl);
  if (!url) {
    throw new Error("Invalid URL: " + originalUrl);
  }

  // Continue with logic
}
```

## Troubleshooting

### Common Issues

1. **Script not loading**: Check syntax and required methods
2. **Authentication failing**: Verify API key handling in `modifyRequest`
3. **URL building errors**: Check `buildUpstreamURL` implementation
4. **Model extraction failing**: Verify `extractModel` logic

### Debug Mode

Enable debug logging to see detailed information:

```javascript
utils.log.debug("Request URL: " + context.request.url);
utils.log.debug("Request headers: " + JSON.stringify(context.request.headers));
```

### Testing Scripts

Use the validation endpoint to test your scripts before enabling:

```bash
curl -X POST http://localhost:3001/api/scripts/validate \
  -H "Authorization: Bearer your-admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "script": "your-script-code",
    "metadata": { ... }
  }'
```

This comprehensive system makes GPT-Load truly universal for any AI service integration!
