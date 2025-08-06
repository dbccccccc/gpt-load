<template>
  <div class="script-list">
    <n-data-table
      :columns="columns"
      :data="scripts"
      :loading="loading"
      :pagination="pagination"
      :row-key="(row: ChannelScript) => row.id"
    />
  </div>
</template>

<script setup lang="ts">
import type { ChannelScript } from "@/types/channel-script";
import {
  Code as CodeIcon,
  TrashOutline as DeleteIcon,
  DocumentTextOutline as LogsIcon,
  Play as PlayIcon,
  Refresh as RefreshIcon,
  Stop as StopIcon,
  FlaskOutline as TestIcon,
} from "@vicons/ionicons5";
import {
  NButton,
  NDataTable,
  NIcon,
  NSpace,
  NTag,
  NTooltip,
  type DataTableColumns,
} from "naive-ui";
import { h } from "vue";

interface Props {
  scripts: ChannelScript[];
  loading?: boolean;
}

interface Emits {
  (e: "edit", script: ChannelScript): void;
  (e: "delete", script: ChannelScript): void;
  (e: "enable", script: ChannelScript): void;
  (e: "disable", script: ChannelScript): void;
  (e: "test", script: ChannelScript): void;
  (e: "view-logs", script: ChannelScript): void;
  (e: "reload", script: ChannelScript): void;
}

withDefaults(defineProps<Props>(), {
  loading: false,
});

const emit = defineEmits<Emits>();

const pagination = {
  pageSize: 10,
};

const getStatusTag = (status: string) => {
  const statusMap = {
    enabled: { type: "success", text: "Enabled" },
    disabled: { type: "default", text: "Disabled" },
    error: { type: "error", text: "Error" },
  };
  return statusMap[status as keyof typeof statusMap] || { type: "default", text: status };
};

const columns: DataTableColumns<ChannelScript> = [
  {
    title: "Name",
    key: "name",
    render: row =>
      h("div", [
        h("div", { style: "font-weight: 500" }, row.display_name || row.name),
        h("div", { style: "font-size: 12px; color: #999" }, row.channel_type),
      ]),
  },
  {
    title: "Description",
    key: "description",
    ellipsis: {
      tooltip: true,
    },
  },
  {
    title: "Author",
    key: "author",
    width: 120,
  },
  {
    title: "Version",
    key: "version",
    width: 80,
  },
  {
    title: "Status",
    key: "status",
    width: 100,
    render: row => {
      const tag = getStatusTag(row.status);
      return h(NTag, { type: tag.type as any }, { default: () => tag.text });
    },
  },
  {
    title: "Updated",
    key: "updated_at",
    width: 120,
    render: row => new Date(row.updated_at).toLocaleDateString(),
  },
  {
    title: "Actions",
    key: "actions",
    width: 200,
    render: row =>
      h(
        NSpace,
        { size: "small" },
        {
          default: () => [
            h(
              NTooltip,
              { trigger: "hover" },
              {
                trigger: () =>
                  h(
                    NButton,
                    {
                      size: "small",
                      type: "primary",
                      ghost: true,
                      onClick: () => emit("edit", row),
                    },
                    {
                      icon: () => h(NIcon, null, { default: () => h(CodeIcon) }),
                    }
                  ),
                default: () => "Edit Script",
              }
            ),

            h(
              NTooltip,
              { trigger: "hover" },
              {
                trigger: () =>
                  h(
                    NButton,
                    {
                      size: "small",
                      type: row.status === "enabled" ? "warning" : "success",
                      ghost: true,
                      onClick: () =>
                        row.status === "enabled" ? emit("disable", row) : emit("enable", row),
                    },
                    {
                      icon: () =>
                        h(NIcon, null, {
                          default: () => (row.status === "enabled" ? h(StopIcon) : h(PlayIcon)),
                        }),
                    }
                  ),
                default: () => (row.status === "enabled" ? "Disable" : "Enable"),
              }
            ),

            h(
              NTooltip,
              { trigger: "hover" },
              {
                trigger: () =>
                  h(
                    NButton,
                    {
                      size: "small",
                      type: "info",
                      ghost: true,
                      onClick: () => emit("test", row),
                    },
                    {
                      icon: () => h(NIcon, null, { default: () => h(TestIcon) }),
                    }
                  ),
                default: () => "Test Script",
              }
            ),

            h(
              NTooltip,
              { trigger: "hover" },
              {
                trigger: () =>
                  h(
                    NButton,
                    {
                      size: "small",
                      type: "primary",
                      ghost: true,
                      onClick: () => emit("reload", row),
                    },
                    {
                      icon: () => h(NIcon, null, { default: () => h(RefreshIcon) }),
                    }
                  ),
                default: () => "Reload Script",
              }
            ),

            h(
              NTooltip,
              { trigger: "hover" },
              {
                trigger: () =>
                  h(
                    NButton,
                    {
                      size: "small",
                      type: "default",
                      ghost: true,
                      onClick: () => emit("view-logs", row),
                    },
                    {
                      icon: () => h(NIcon, null, { default: () => h(LogsIcon) }),
                    }
                  ),
                default: () => "View Logs",
              }
            ),

            h(
              NTooltip,
              { trigger: "hover" },
              {
                trigger: () =>
                  h(
                    NButton,
                    {
                      size: "small",
                      type: "error",
                      ghost: true,
                      onClick: () => emit("delete", row),
                    },
                    {
                      icon: () => h(NIcon, null, { default: () => h(DeleteIcon) }),
                    }
                  ),
                default: () => "Delete",
              }
            ),
          ],
        }
      ),
  },
];
</script>

<style scoped>
.script-list {
  width: 100%;
}
</style>
