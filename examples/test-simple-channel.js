function exports() {
  return {
    metadata: {
      name: "Test Channel",
      version: "1.0.0",
      description: "Simple test channel",
      author: "Test",
      channel_type: "test"
    },

    initialize: function (config) {
      utils.log.info("Test channel initialized");
    },

    buildUpstreamURL: function (originalUrl, group) {
      return "https://api.example.com/test";
    },

    modifyRequest: function (request, apiKey, group) {
      request.headers["Authorization"] = "Bearer " + apiKey.key_value;
    },

    isStreamRequest: function (context) {
      return false;
    },

    extractModel: function (context) {
      return "test-model";
    },

    validateKey: function (apiKey, group) {
      return new Promise(function (resolve) {
        resolve({ valid: true });
      });
    }
  };
}
