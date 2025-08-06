import http from "@/utils/http";

export interface Setting {
  key: string;
  name: string;
  value: string | number;
  type: "int" | "string";
  min_value?: number;
  description: string;
}

export interface SettingCategory {
  category_name: string;
  settings: Setting[];
}

export type SettingsUpdatePayload = Record<string, string | number>;

export interface ChannelTypeInfo {
  type: string;
  is_script: boolean;
  display_name?: string;
  description?: string;
  default_test_model?: string;
  default_validation_endpoint?: string;
  default_upstream?: string;
  supported_models?: string[];
  required_config?: Record<string, string>;
}

export const settingsApi = {
  async getSettings(): Promise<SettingCategory[]> {
    const response = await http.get("/settings");
    return response.data || [];
  },
  updateSettings(data: SettingsUpdatePayload): Promise<void> {
    return http.put("/settings", data);
  },
  async getChannelTypes(): Promise<string[]> {
    const response = await http.get("/channel-types");
    return response.data || [];
  },
  async getChannelTypesWithMetadata(): Promise<ChannelTypeInfo[]> {
    const response = await http.get("/channel-types-with-metadata");
    return response.data || [];
  },
};
