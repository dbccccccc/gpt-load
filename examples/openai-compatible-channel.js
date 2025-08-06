/**
 * OpenAI-Compatible Channel Script
 *
 * This script creates a channel for any OpenAI-compatible API service.
 * It can be used for services like:
 * - OpenAI API
 * - Azure OpenAI
 * - Local LLM servers (Ollama, LM Studio, etc.)
 * - Other OpenAI-compatible services
 */

function exports() {
  return {
    // Channel metadata
    metadata: {
      name: "OpenAI Compatible Channel",
      version: "1.0.0",
      description: "Generic channel for OpenAI-compatible API services",
      author: "GPT-Load Team",
      channel_type: "openai_compatible",
      supported_models: ["*"], // Supports any model
      default_test_model: "gpt-3.5-turbo",
      default_validation_endpoint: "/v1/chat/completions",
      required_config: {
        base_url: "Base URL of the OpenAI-compatible service",
      },
    },

    // Initialize the channel
    initialize: function (config) {
      utils.log.info("OpenAI-compatible channel initialized");
      if (config.base_url) {
        utils.log.info("Using base URL: " + config.base_url);
      }
    },

    // Build upstream URL
    buildUpstreamURL: function (originalUrl, group) {
      const url = utils.parseURL(originalUrl);
      if (!url) {
        throw new Error("Invalid URL: " + originalUrl);
      }

      // Get the first upstream
      if (!group.upstreams || group.upstreams.length === 0) {
        throw new Error("No upstreams configured");
      }

      const upstream = group.upstreams[0];
      const proxyPrefix = "/proxy/" + group.name;
      let requestPath = url.pathname;

      // Remove proxy prefix
      if (requestPath.startsWith(proxyPrefix)) {
        requestPath = requestPath.substring(proxyPrefix.length);
      }

      // Ensure path starts with /v1 for OpenAI compatibility
      if (!requestPath.startsWith("/v1/")) {
        if (requestPath.startsWith("/")) {
          requestPath = "/v1" + requestPath;
        } else {
          requestPath = "/v1/" + requestPath;
        }
      }

      const targetUrl = utils.joinURL(upstream.url, requestPath);
      return targetUrl + (url.search ? "?" + url.search : "");
    },

    // Modify request for authentication
    modifyRequest: function (request, apiKey, group) {
      // Add OpenAI-style authorization
      request.headers["Authorization"] = "Bearer " + apiKey.key_value;

      // Ensure content type
      if (request.method === "POST" && !request.headers["Content-Type"]) {
        request.headers["Content-Type"] = "application/json";
      }

      // Add user agent
      request.headers["User-Agent"] = "GPT-Load-Proxy/1.0";

      utils.log.debug("Added OpenAI-compatible authentication headers");
    },

    // Detect streaming requests
    isStreamRequest: function (context) {
      // Check Accept header for SSE
      const acceptHeader = context.request.headers["Accept"];
      if (acceptHeader && acceptHeader.includes("text/event-stream")) {
        return true;
      }

      // Check query parameter
      if (context.request.query["stream"] === "true") {
        return true;
      }

      // Check request body
      if (context.body_bytes && context.body_bytes.length > 0) {
        try {
          const bodyStr = new TextDecoder().decode(context.body_bytes);
          const body = utils.parseJSON(bodyStr);
          if (body && body.stream === true) {
            return true;
          }
        } catch (e) {
          utils.log.debug("Could not parse request body for stream detection");
        }
      }

      return false;
    },

    // Extract model from request
    extractModel: function (context) {
      // Try to get model from request body
      if (context.body_bytes && context.body_bytes.length > 0) {
        try {
          const bodyStr = new TextDecoder().decode(context.body_bytes);
          const body = utils.parseJSON(bodyStr);
          if (body && body.model) {
            return body.model;
          }
        } catch (e) {
          utils.log.debug("Could not parse request body for model extraction");
        }
      }

      // Try to get model from URL path (for some endpoints)
      const url = utils.parseURL(context.request.url);
      if (url && url.pathname) {
        const pathParts = url.pathname.split("/");

        // Look for /models/{model_id} pattern
        const modelIndex = pathParts.indexOf("models");
        if (modelIndex !== -1 && modelIndex + 1 < pathParts.length) {
          return pathParts[modelIndex + 1];
        }

        // Look for /engines/{engine_id} pattern (legacy OpenAI)
        const engineIndex = pathParts.indexOf("engines");
        if (engineIndex !== -1 && engineIndex + 1 < pathParts.length) {
          return pathParts[engineIndex + 1];
        }
      }

      return "";
    },

    // Validate API key
    validateKey: function (apiKey, group) {
      return new Promise(function (resolve) {
        try {
          // Get upstream URL
          if (!group.upstreams || group.upstreams.length === 0) {
            resolve({ valid: false, error: "No upstreams configured" });
            return;
          }

          const upstream = group.upstreams[0];
          const testUrl = utils.joinURL(upstream.url, "/v1/chat/completions");

          // Create minimal test request
          const testRequest = {
            method: "POST",
            url: testUrl,
            headers: {
              Authorization: "Bearer " + apiKey,
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              model: "gpt-3.5-turbo",
              messages: [{ role: "user", content: "Hi" }],
              max_tokens: 1,
              temperature: 0,
            }),
          };

          utils.log.debug("Validating key with test request to: " + testUrl);

          utils
            .httpRequest(testRequest)
            .then(function (response) {
              if (response.status_code >= 200 && response.status_code < 300) {
                utils.log.info("API key validation successful");
                resolve({ valid: true });
              } else if (response.status_code === 401) {
                utils.log.warn("API key validation failed: Unauthorized");
                resolve({ valid: false, error: "Invalid API key" });
              } else if (response.status_code === 429) {
                utils.log.warn("API key validation failed: Rate limited");
                resolve({ valid: false, error: "Rate limited" });
              } else {
                const errorMsg =
                  "Validation failed with status " + response.status_code;
                utils.log.warn(errorMsg);
                resolve({ valid: false, error: errorMsg });
              }
            })
            .catch(function (error) {
              utils.log.error(
                "Key validation request failed: " + error.message
              );
              resolve({
                valid: false,
                error: "Network error: " + error.message,
              });
            });
        } catch (error) {
          utils.log.error("Key validation error: " + error.message);
          resolve({ valid: false, error: error.message });
        }
      });
    },

    // Optional: Transform response
    transformResponse: function (response, context) {
      // Add custom headers
      response.headers["X-Processed-By"] = "GPT-Load-OpenAI-Compatible";
      response.headers["X-Channel-Type"] = "openai_compatible";

      return response;
    },

    // Optional: Handle errors
    handleError: function (error, context) {
      utils.log.error("OpenAI-compatible service error: " + error);

      // Provide user-friendly error messages
      if (error.includes("401")) {
        return "Authentication failed. Please check your API key.";
      } else if (error.includes("429")) {
        return "Rate limit exceeded. Please try again later.";
      } else if (error.includes("timeout")) {
        return "Request timeout. The service may be overloaded.";
      }

      return error; // Return original error for other cases
    },

    // Optional: Health check
    healthCheck: function (group) {
      return new Promise(function (resolve) {
        if (!group.upstreams || group.upstreams.length === 0) {
          resolve(false);
          return;
        }

        const upstream = group.upstreams[0];
        const healthUrl = utils.joinURL(upstream.url, "/v1/models");

        utils
          .httpRequest({
            method: "GET",
            url: healthUrl,
            headers: {
              "User-Agent": "GPT-Load-Health-Check/1.0",
            },
          })
          .then(function (response) {
            const healthy =
              response.status_code >= 200 && response.status_code < 400;
            utils.log.debug(
              "Health check result: " + (healthy ? "healthy" : "unhealthy")
            );
            resolve(healthy);
          })
          .catch(function (error) {
            utils.log.warn("Health check failed: " + error.message);
            resolve(false);
          });
      });
    },
  };
}
