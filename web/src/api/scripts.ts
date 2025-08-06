import type { ChannelScript, ChannelScriptMetadata } from "@/types/channel-script";
import http from "@/utils/http";

export interface CreateScriptRequest {
  name: string;
  display_name?: string;
  description?: string;
  author?: string;
  version: string;
  channel_type: string;
  script: string;
  metadata: ChannelScriptMetadata;
  config?: Record<string, any>;
}

export interface UpdateScriptRequest {
  name?: string;
  display_name?: string;
  description?: string;
  author?: string;
  version?: string;
  script?: string;
  metadata?: ChannelScriptMetadata;
  config?: Record<string, any>;
}

export interface ValidateScriptRequest {
  script: string;
  metadata: ChannelScriptMetadata;
}

export interface TestScriptRequest {
  script: string;
  metadata: ChannelScriptMetadata;
  test_data?: Record<string, any>;
}

export interface ScriptValidationResult {
  valid: boolean;
  error?: string;
  message?: string;
}

export interface ScriptTestResult {
  valid: boolean;
  error?: string;
  message?: string;
  runtime?: string;
  test_data_processed?: boolean;
}

export interface ScriptLogEntry {
  timestamp: string;
  level: string;
  message: string;
}

export const scriptApi = {
  // Get all scripts
  async getScripts() {
    return http.get<ChannelScript[]>("/scripts");
  },

  // Get script by ID
  async getScript(id: number) {
    return http.get<ChannelScript>(`/scripts/${id}`);
  },

  // Create new script
  async createScript(data: CreateScriptRequest) {
    return http.post<ChannelScript>("/scripts", data);
  },

  // Update script
  async updateScript(id: number, data: UpdateScriptRequest) {
    return http.put<ChannelScript>(`/scripts/${id}`, data);
  },

  // Delete script
  async deleteScript(id: number) {
    return http.delete(`/scripts/${id}`);
  },

  // Enable script
  async enableScript(id: number) {
    return http.post(`/scripts/${id}/enable`);
  },

  // Disable script
  async disableScript(id: number) {
    return http.post(`/scripts/${id}/disable`);
  },

  // Validate script
  async validateScript(data: ValidateScriptRequest) {
    return http.post<ScriptValidationResult>("/scripts/validate", data);
  },

  // Test script
  async testScript(data: TestScriptRequest) {
    return http.post<ScriptTestResult>("/scripts/test", data);
  },

  // Get script logs
  async getScriptLogs(id: number) {
    return http.get<ScriptLogEntry[]>(`/scripts/${id}/logs`);
  },

  // Hot-reload specific script
  async reloadScript(id: number) {
    return http.post(`/scripts/${id}/reload`);
  },

  // Hot-reload all scripts
  async reloadAllScripts() {
    return http.post("/scripts/reload-all");
  },

  // Get active scripts
  async getActiveScripts() {
    return http.get<{ active_scripts: string[]; count: number }>("/scripts/active");
  },
};
