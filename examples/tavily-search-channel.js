/**
 * Tavily Search Service Channel Script
 *
 * This script creates a channel for Tavily's AI-powered search API.
 * Tavily provides intelligent web search with summarized answers,
 * making it perfect for AI applications that need real-time information.
 *
 * Features:
 * - Web search with AI-generated answers
 * - Content extraction from URLs
 * - Site crawling and mapping
 * - Usage tracking
 *
 * API Documentation: https://docs.tavily.com/
 */

function exports() {
  return {
    // Channel metadata
    metadata: {
      name: "Tavily Search Channel",
      version: "1.1.0",
      description: "AI-powered web search and content extraction service",
      author: "GPT-Load Community",
      channel_type: "tavily_search",
      supported_models: ["search", "extract", "crawl", "map"], // Tavily endpoints
      default_test_model: "search",
      default_validation_endpoint: "/search",
      default_upstream: "https://api.tavily.com",
      required_config: {
        base_url: "Tavily API base URL (default: https://api.tavily.com)",
      },
    },

    // Initialize the channel
    initialize: function (config) {
      utils.log.info("Tavily Search channel initialized");
      var baseUrl = config.base_url || "https://api.tavily.com";
      utils.log.info("Using Tavily API base URL: " + baseUrl);
    },

    // Build upstream URL
    buildUpstreamURL: function (originalUrl, group) {
      var parsed = utils.parseURL(originalUrl);
      if (!parsed) {
        throw new Error("Invalid URL: " + originalUrl);
      }

      // Get the base URL from config or use default
      var baseUrl =
        (group.config && group.config.base_url) || "https://api.tavily.com";
      var proxyPrefix = "/proxy/" + group.name;
      var requestRoute = parsed.pathname;

      // Remove proxy prefix
      if (requestRoute.startsWith(proxyPrefix)) {
        requestRoute = requestRoute.substring(proxyPrefix.length);
      }

      // Ensure path starts with / for Tavily API
      if (!requestRoute.startsWith("/")) {
        requestRoute = "/" + requestRoute;
      }

      // Map common endpoints
      var endpointMap = {
        "/search": "/search",
        "/extract": "/extract",
        "/crawl": "/crawl",
        "/map": "/map",
        "/usage": "/usage",
      };

      // Use mapped endpoint or pass through
      var mappedRoute = endpointMap[requestRoute] || requestRoute;
      var targetUrl = utils.joinURL(baseUrl, mappedRoute);

      return targetUrl + (parsed.search ? "?" + parsed.search : "");
    },

    // Modify request for Tavily authentication
    modifyRequest: function (request, apiKey, group) {
      // Add Tavily-style Bearer authorization
      request.headers["Authorization"] = "Bearer " + apiKey.key_value;

      // Ensure content type for POST requests
      if (request.method === "POST" && !request.headers["Content-Type"]) {
        request.headers["Content-Type"] = "application/json";
      }

      // Add user agent
      request.headers["User-Agent"] = "GPT-Load-Proxy/1.0 (Tavily-Channel)";

      // For Tavily, we need to ensure the API key is also in the request body for some endpoints
      if (request.method === "POST" && request.body) {
        try {
          var body = utils.parseJSON(request.body);
          if (body && !body.api_key) {
            body.api_key = apiKey.key_value;
            request.body = JSON.stringify(body);
          }
        } catch (e) {
          utils.log.debug("Could not parse request body for API key injection");
        }
      }

      utils.log.debug("Added Tavily authentication headers and API key");
    },

    // Tavily doesn't use streaming, so always return false
    isStreamRequest: function (context) {
      return false;
    },

    // Extract "model" (endpoint type) from request
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

        // Extract endpoint type
        if (urlRoute.startsWith("/search")) return "search";
        if (urlRoute.startsWith("/extract")) return "extract";
        if (urlRoute.startsWith("/crawl")) return "crawl";
        if (urlRoute.startsWith("/map")) return "map";
        if (urlRoute.startsWith("/usage")) return "usage";
      }

      return "search"; // Default to search
    },

    // Validate API key with Tavily
    validateKey: function (apiKey, group) {
      return new Promise(function (resolve) {
        try {
          var baseUrl =
            (group.config && group.config.base_url) || "https://api.tavily.com";
          var testUrl = utils.joinURL(baseUrl, "/search");

          // Create minimal test search request
          var testRequest = {
            method: "POST",
            url: testUrl,
            headers: {
              Authorization: "Bearer " + apiKey,
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              query: "test",
              api_key: apiKey,
              max_results: 1,
              include_answer: false,
            }),
          };

          utils.log.debug("Validating Tavily API key with test search");

          utils
            .httpRequest(testRequest)
            .then(function (response) {
              if (response.status_code >= 200 && response.status_code < 300) {
                utils.log.info("Tavily API key validation successful");
                resolve({ valid: true });
              } else if (response.status_code === 401) {
                utils.log.warn(
                  "Tavily API key validation failed: Unauthorized"
                );
                resolve({ valid: false, error: "Invalid API key" });
              } else if (response.status_code === 429) {
                utils.log.warn(
                  "Tavily API key validation failed: Rate limited"
                );
                resolve({ valid: false, error: "Rate limited" });
              } else if (response.status_code === 432) {
                utils.log.warn(
                  "Tavily API key validation failed: Insufficient credits"
                );
                resolve({ valid: false, error: "Insufficient credits" });
              } else {
                var errorMsg =
                  "Validation failed with status " + response.status_code;
                utils.log.warn(errorMsg);
                resolve({ valid: false, error: errorMsg });
              }
            })
            .catch(function (error) {
              utils.log.error(
                "Tavily key validation request failed: " + error.message
              );
              resolve({
                valid: false,
                error: "Network error: " + error.message,
              });
            });
        } catch (error) {
          utils.log.error("Tavily key validation error: " + error.message);
          resolve({ valid: false, error: error.message });
        }
      });
    },

    // Transform response to add metadata
    transformResponse: function (response, context) {
      // Add custom headers
      response.headers["X-Processed-By"] = "GPT-Load-Tavily-Channel";
      response.headers["X-Channel-Type"] = "tavily_search";
      response.headers["X-Service-Provider"] = "Tavily";

      // Add response time if available in Tavily response
      if (response.body) {
        try {
          var body = utils.parseJSON(response.body);
          if (body && body.response_time) {
            response.headers["X-Tavily-Response-Time"] = body.response_time;
          }
        } catch (e) {
          // Ignore JSON parsing errors
        }
      }

      return response;
    },

    // Handle Tavily-specific errors
    handleError: function (error, context) {
      utils.log.error("Tavily service error: " + error);

      // Provide user-friendly error messages for common Tavily errors
      if (error.includes("401")) {
        return "Authentication failed. Please check your Tavily API key.";
      } else if (error.includes("429")) {
        return "Rate limit exceeded. Please try again later.";
      } else if (error.includes("432")) {
        return "Insufficient credits. Please check your Tavily account balance.";
      } else if (error.includes("433")) {
        return "Usage limit exceeded for your plan.";
      } else if (error.includes("timeout")) {
        return "Request timeout. Tavily service may be overloaded.";
      } else if (error.includes("400")) {
        return "Bad request. Please check your search parameters.";
      }

      return error; // Return original error for other cases
    },

    // Health check for Tavily service
    healthCheck: function (group) {
      return new Promise(function (resolve) {
        var baseUrl =
          (group.config && group.config.base_url) || "https://api.tavily.com";

        // Use a simple GET request to check if the service is available
        // Note: Tavily doesn't have a dedicated health endpoint, so we'll check the base URL
        utils
          .httpRequest({
            method: "GET",
            url: baseUrl,
            headers: {
              "User-Agent": "GPT-Load-Health-Check/1.0",
            },
          })
          .then(function (response) {
            // Tavily should return some response (even if it's an error about missing auth)
            // A 2xx, 4xx response indicates the service is up
            var healthy =
              response.status_code >= 200 && response.status_code < 500;
            utils.log.debug(
              "Tavily health check result: " +
                (healthy ? "healthy" : "unhealthy")
            );
            resolve(healthy);
          })
          .catch(function (error) {
            utils.log.warn("Tavily health check failed: " + error.message);
            resolve(false);
          });
      });
    },

    // Handle Tavily-specific errors
    handleError: function (error, context) {
      utils.log.error("Tavily service error: " + error);

      // Provide user-friendly error messages for common Tavily errors
      if (error.includes("401")) {
        return {
          error:
            "Invalid Tavily API key. Please check your API key configuration.",
          code: "INVALID_API_KEY",
        };
      } else if (error.includes("429")) {
        return {
          error: "Rate limit exceeded. Please try again later.",
          code: "RATE_LIMIT_EXCEEDED",
        };
      } else if (error.includes("400")) {
        return {
          error: "Invalid search query. Please check your request parameters.",
          code: "INVALID_QUERY",
        };
      } else if (error.includes("503")) {
        return {
          error:
            "Tavily service temporarily unavailable. Please try again later.",
          code: "SERVICE_UNAVAILABLE",
        };
      }

      return {
        error: "Tavily service error: " + error,
        code: "UNKNOWN_ERROR",
      };
    },
  };
}
