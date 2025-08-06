/**
 * Grok (X.AI) Channel Script for GPT-Load
 *
 * This script provides integration with X.AI's Grok API, which offers
 * advanced AI capabilities with OpenAI-compatible endpoints.
 *
 * Features:
 * - Chat completions with Grok models
 * - Image understanding and generation
 * - Function calling and structured outputs
 * - Streaming responses
 * - Anthropic-compatible endpoints
 *
 * API Documentation: https://docs.x.ai/docs/overview
 * Base URL: https://api.x.ai
 *
 * @author GPT-Load Community
 * @version 1.0.0
 */

function exports() {
  return {
    // Channel metadata
    metadata: {
      name: "Grok Channel",
      version: "1.1.0",
      description: "X.AI Grok API integration with OpenAI-compatible endpoints",
      author: "GPT-Load Community",
      channel_type: "grok",
      supported_models: [
        "grok-4-0709",
        "grok-4",
        "grok-3.5",
        "grok-3",
        "grok-2",
        "grok-1.5v",
        "grok-1.5",
        "grok-1",
      ],
      default_test_model: "grok-4",
      default_validation_endpoint: "/v1/models",
      default_upstream: "https://api.x.ai",
      required_config: {
        base_url: "X.AI API base URL (default: https://api.x.ai)",
        region: "Regional endpoint (optional: us-east-1, eu-west-1)",
      },
    },

    // Initialize the channel
    initialize: function (config) {
      utils.log.info("Grok channel initialized");
      var baseUrl = config.base_url || "https://api.x.ai";
      var region = config.region;

      if (region) {
        // Use regional endpoint if specified
        baseUrl = "https://" + region + ".api.x.ai";
        utils.log.info("Using regional endpoint: " + baseUrl);
      } else {
        utils.log.info("Using default X.AI API base URL: " + baseUrl);
      }
    },

    // Build upstream URL
    buildUpstreamURL: function (originalUrl, group) {
      var parsed = utils.parseURL(originalUrl);
      if (!parsed) {
        throw new Error("Invalid URL: " + originalUrl);
      }

      // Get the base URL from config or use default
      var baseUrl =
        (group.config && group.config.base_url) || "https://api.x.ai";
      var region = group.config && group.config.region;

      if (region) {
        baseUrl = "https://" + region + ".api.x.ai";
      }

      var proxyPrefix = "/proxy/" + group.name;
      var requestRoute = parsed.pathname;

      // Remove proxy prefix
      if (requestRoute.startsWith(proxyPrefix)) {
        requestRoute = requestRoute.substring(proxyPrefix.length);
      }

      // Ensure path starts with /v1 for X.AI API
      if (!requestRoute.startsWith("/v1")) {
        if (requestRoute.startsWith("/")) {
          requestRoute = "/v1" + requestRoute;
        } else {
          requestRoute = "/v1/" + requestRoute;
        }
      }

      // Map common endpoints
      var endpointMap = {
        "/v1/chat/completions": "/v1/chat/completions",
        "/v1/completions": "/v1/completions",
        "/v1/messages": "/v1/messages", // Anthropic compatible
        "/v1/complete": "/v1/complete", // Anthropic compatible legacy
        "/v1/images/generations": "/v1/images/generations",
        "/v1/models": "/v1/models",
        "/v1/language-models": "/v1/language-models",
        "/v1/image-generation-models": "/v1/image-generation-models",
        "/v1/tokenize-text": "/v1/tokenize-text",
        "/v1/api-key": "/v1/api-key",
      };

      // Use mapped endpoint or pass through
      var mappedRoute = endpointMap[requestRoute] || requestRoute;
      var targetUrl = utils.joinURL(baseUrl, mappedRoute);

      return targetUrl + (parsed.search ? "?" + parsed.search : "");
    },

    // Modify request for X.AI API
    modifyRequest: function (request, apiKey, group) {
      // Set authorization header
      request.headers["Authorization"] = "Bearer " + apiKey.key_value;

      // Set content type for POST requests
      if (request.method === "POST" && !request.headers["Content-Type"]) {
        request.headers["Content-Type"] = "application/json";
      }

      // Add User-Agent
      request.headers["User-Agent"] = "GPT-Load-Proxy/1.0";

      // Handle regional endpoints
      var region = group.config && group.config.region;
      if (region) {
        request.headers["X-Region"] = region;
      }
    },

    // Check if request should be streamed
    isStreamRequest: function (context) {
      if (context.request.body) {
        try {
          var body = utils.parseJSON(context.request.body);
          return body && body.stream === true;
        } catch (e) {
          return false;
        }
      }
      return false;
    },

    // Extract "model" from request
    extractModel: function (context) {
      var parsed = utils.parseURL(context.request.url);
      if (parsed && parsed.pathname) {
        var proxyPrefix = "/proxy/";
        var urlRoute = parsed.pathname;

        // Remove proxy prefix if present
        var proxyIndex = urlRoute.indexOf(proxyPrefix);
        if (proxyIndex !== -1) {
          var groupEnd = urlRoute.indexOf("/", proxyIndex + proxyPrefix.length);
          if (groupEnd !== -1) {
            urlRoute = urlRoute.substring(groupEnd);
          }
        }

        // Extract model from request body for chat completions
        if (
          urlRoute.includes("/chat/completions") ||
          urlRoute.includes("/completions") ||
          urlRoute.includes("/messages")
        ) {
          if (context.request.body) {
            try {
              var body = utils.parseJSON(context.request.body);
              if (body && body.model) {
                return body.model;
              }
            } catch (e) {
              // Ignore JSON parsing errors
            }
          }
        }

        // Extract endpoint type as fallback
        if (urlRoute.includes("/chat/completions")) return "chat";
        if (urlRoute.includes("/completions")) return "completions";
        if (urlRoute.includes("/messages")) return "messages";
        if (urlRoute.includes("/images/generations")) return "image-generation";
        if (urlRoute.includes("/models")) return "models";
      }

      return "grok-4"; // Default model
    },

    // Validate API key with X.AI
    validateKey: function (apiKey, group) {
      return new Promise(function (resolve) {
        try {
          var baseUrl =
            (group.config && group.config.base_url) || "https://api.x.ai";
          var region = group.config && group.config.region;

          if (region) {
            baseUrl = "https://" + region + ".api.x.ai";
          }

          var testUrl = utils.joinURL(baseUrl, "/v1/api-key");

          // Create test request to validate API key
          var testRequest = {
            method: "GET",
            url: testUrl,
            headers: {
              Authorization: "Bearer " + apiKey.key_value,
              "User-Agent": "GPT-Load-Proxy/1.0",
            },
            timeout: 10000,
          };

          utils.log.debug("Validating X.AI API key...");

          utils.httpClient
            .request(testRequest)
            .then(function (response) {
              if (response.status_code >= 200 && response.status_code < 300) {
                utils.log.info("X.AI API key validation successful");
                resolve({ valid: true });
              } else if (response.status_code === 401) {
                utils.log.warn(
                  "X.AI API key validation failed: Invalid API key"
                );
                resolve({ valid: false, error: "Invalid API key" });
              } else {
                var errorMsg =
                  "Validation failed with status " + response.status_code;
                utils.log.warn(errorMsg);
                resolve({ valid: false, error: errorMsg });
              }
            })
            .catch(function (error) {
              utils.log.error(
                "X.AI API key validation error: " + error.message
              );
              resolve({ valid: false, error: error.message });
            });
        } catch (error) {
          utils.log.error(
            "X.AI API key validation exception: " + error.message
          );
          resolve({ valid: false, error: error.message });
        }
      });
    },

    // Modify response from X.AI
    modifyResponse: function (response, context) {
      // Add X.AI specific headers
      response.headers["X-Provider"] = "X.AI";
      response.headers["X-Channel"] = "grok";

      // Add model information if available in response
      if (response.body) {
        try {
          var body = utils.parseJSON(response.body);
          if (body && body.model) {
            response.headers["X-Model"] = body.model;
          }
          if (body && body.usage) {
            response.headers["X-Tokens-Used"] =
              body.usage.total_tokens || "unknown";
          }
        } catch (e) {
          // Ignore JSON parsing errors
        }
      }
    },

    // Health check for X.AI service
    healthCheck: function (group) {
      return new Promise(function (resolve) {
        var baseUrl =
          (group.config && group.config.base_url) || "https://api.x.ai";
        var region = group.config && group.config.region;

        if (region) {
          baseUrl = "https://" + region + ".api.x.ai";
        }

        var healthUrl = utils.joinURL(baseUrl, "/v1/models");

        utils.httpClient
          .request({
            method: "GET",
            url: healthUrl,
            headers: {
              "User-Agent": "GPT-Load-Proxy/1.0",
            },
            timeout: 5000,
          })
          .then(function (response) {
            // X.AI should return some response (even if it's an error about missing auth)
            // A 2xx, 4xx response indicates the service is up
            var healthy =
              response.status_code >= 200 && response.status_code < 500;
            utils.log.debug(
              "X.AI health check result: " + (healthy ? "healthy" : "unhealthy")
            );
            resolve(healthy);
          })
          .catch(function (error) {
            utils.log.debug("X.AI health check failed: " + error.message);
            resolve(false);
          });
      });
    },

    // Handle Grok-specific errors
    handleError: function (error, context) {
      utils.log.error("Grok service error: " + error);

      // Provide user-friendly error messages for common X.AI errors
      if (error.includes("401")) {
        return {
          error:
            "Invalid X.AI API key. Please check your API key configuration.",
          code: "INVALID_API_KEY",
        };
      } else if (error.includes("429")) {
        return {
          error: "Rate limit exceeded. Please try again later.",
          code: "RATE_LIMIT_EXCEEDED",
        };
      } else if (error.includes("503")) {
        return {
          error:
            "X.AI service temporarily unavailable. Please try again later.",
          code: "SERVICE_UNAVAILABLE",
        };
      }

      return {
        error: "Grok service error: " + error,
        code: "UNKNOWN_ERROR",
      };
    },
  };
}
