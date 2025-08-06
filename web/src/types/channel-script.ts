/**
 * TypeScript interface definitions for channel scripts
 * These match the Go backend models
 */

export interface ChannelScript {
  id: number
  name: string
  display_name: string
  description: string
  author: string
  version: string
  channel_type: string
  script: string
  metadata: ChannelScriptMetadata
  config: Record<string, any>
  status: 'enabled' | 'disabled' | 'error'
  error_msg: string
  last_error?: string
  created_at: string
  updated_at: string
}

export interface ChannelScriptMetadata {
  name: string
  version: string
  description: string
  author: string
  channel_type: string
  supported_models: string[]
  default_test_model: string
  default_validation_endpoint: string
  required_config?: Record<string, string>
}

// For the script editor
export interface ScriptTemplate {
  name: string
  description: string
  template: string
  metadata: ChannelScriptMetadata
}

// Common script templates
export const SCRIPT_TEMPLATES: ScriptTemplate[] = [
  {
    name: 'OpenAI Compatible',
    description: 'Template for OpenAI-compatible API services',
    metadata: {
      name: 'OpenAI Compatible Channel',
      version: '1.0.0',
      description: 'Generic channel for OpenAI-compatible API services',
      author: 'GPT-Load Team',
      channel_type: 'openai_compatible',
      supported_models: ['*'],
      default_test_model: 'gpt-3.5-turbo',
      default_validation_endpoint: '/v1/chat/completions'
    },
    template: `function exports() {
  return {
    metadata: {
      name: "OpenAI Compatible Channel",
      version: "1.0.0",
      description: "Generic channel for OpenAI-compatible API services",
      author: "Your Name",
      channel_type: "openai_compatible",
      supported_models: ["*"],
      default_test_model: "gpt-3.5-turbo",
      default_validation_endpoint: "/v1/chat/completions"
    },

    buildUpstreamURL: function(originalUrl, group) {
      const url = utils.parseURL(originalUrl);
      const upstream = group.upstreams[0];
      const proxyPrefix = "/proxy/" + group.name;
      let path = url.pathname.replace(proxyPrefix, "");

      if (!path.startsWith("/v1/")) {
        path = "/v1" + path;
      }

      return utils.joinURL(upstream.url, path) + (url.search || "");
    },

    modifyRequest: function(request, apiKey, group) {
      request.headers["Authorization"] = "Bearer " + apiKey.key_value;
      request.headers["Content-Type"] = "application/json";
    },

    isStreamRequest: function(context) {
      if (context.request.headers["Accept"]?.includes("text/event-stream")) {
        return true;
      }

      try {
        const body = utils.parseJSON(new TextDecoder().decode(context.body_bytes));
        return body?.stream === true;
      } catch (e) {
        return false;
      }
    },

    extractModel: function(context) {
      try {
        const body = utils.parseJSON(new TextDecoder().decode(context.body_bytes));
        return body?.model || "";
      } catch (e) {
        return "";
      }
    },

    validateKey: function(apiKey, group) {
      return new Promise(function(resolve) {
        const upstream = group.upstreams[0];
        const testRequest = {
          method: "POST",
          url: utils.joinURL(upstream.url, "/v1/chat/completions"),
          headers: {
            "Authorization": "Bearer " + apiKey,
            "Content-Type": "application/json"
          },
          body: JSON.stringify({
            model: "gpt-3.5-turbo",
            messages: [{ role: "user", content: "test" }],
            max_tokens: 1
          })
        };

        utils.httpRequest(testRequest)
          .then(function(response) {
            resolve({ valid: response.status_code >= 200 && response.status_code < 300 });
          })
          .catch(function(error) {
            resolve({ valid: false, error: error.message });
          });
      });
    }
  };
}`
  },
  {
    name: 'Custom Service',
    description: 'Basic template for custom AI services',
    metadata: {
      name: 'Custom AI Service',
      version: '1.0.0',
      description: 'Custom AI service integration',
      author: 'Your Name',
      channel_type: 'custom_service',
      supported_models: ['model-1', 'model-2'],
      default_test_model: 'model-1',
      default_validation_endpoint: '/api/validate'
    },
    template: `function exports() {
  return {
    metadata: {
      name: "Custom AI Service",
      version: "1.0.0",
      description: "Custom AI service integration",
      author: "Your Name",
      channel_type: "custom_service",
      supported_models: ["model-1", "model-2"],
      default_test_model: "model-1",
      default_validation_endpoint: "/api/validate"
    },

    buildUpstreamURL: function(originalUrl, group) {
      // Implement your URL building logic here
      const url = utils.parseURL(originalUrl);
      const upstream = group.upstreams[0];
      // Add your custom path transformation
      return utils.joinURL(upstream.url, url.pathname);
    },

    modifyRequest: function(request, apiKey, group) {
      // Implement your authentication logic here
      request.headers["X-API-Key"] = apiKey.key_value;
    },

    isStreamRequest: function(context) {
      // Implement your streaming detection logic
      return false;
    },

    extractModel: function(context) {
      // Implement your model extraction logic
      return "";
    },

    validateKey: function(apiKey, group) {
      // Implement your key validation logic
      return new Promise(function(resolve) {
        resolve({ valid: true });
      });
    }
  };
}`
  }
]
