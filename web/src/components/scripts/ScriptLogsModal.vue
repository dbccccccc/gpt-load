<template>
  <n-modal v-model:show="showModal" preset="dialog" title="Script Logs" style="width: 80%; max-width: 1000px;">
    <div v-if="script">
      <n-space vertical>
        <n-card :title="`Logs for ${script.display_name || script.name}`">
          <template #header-extra>
            <n-button @click="loadLogs" :loading="loading" size="small">
              Refresh
            </n-button>
          </template>

          <n-data-table
            :columns="columns"
            :data="logs"
            :loading="loading"
            :pagination="pagination"
            :row-key="(row: ScriptLogEntry) => `${row.timestamp}-${row.message}`"
            size="small"
          />
        </n-card>
      </n-space>
    </div>

    <template #action>
      <n-button @click="showModal = false">Close</n-button>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { scriptApi, type ScriptLogEntry } from '@/api/scripts'
import type { ChannelScript } from '@/types/channel-script'
import {
    NButton,
    NCard,
    NDataTable,
    NModal,
    NSpace,
    NTag,
    useMessage,
    type DataTableColumns
} from 'naive-ui'
import { computed, h, ref, watch } from 'vue'

interface Props {
  show: boolean
  script?: ChannelScript | null
}

interface Emits {
  (e: 'update:show', value: boolean): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const message = useMessage()
const loading = ref(false)
const logs = ref<ScriptLogEntry[]>([])

const showModal = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value)
})

const pagination = {
  pageSize: 20
}

const getLogLevelTag = (level: string) => {
  const levelMap = {
    debug: { type: 'default', text: 'DEBUG' },
    info: { type: 'info', text: 'INFO' },
    warn: { type: 'warning', text: 'WARN' },
    error: { type: 'error', text: 'ERROR' }
  }
  return levelMap[level as keyof typeof levelMap] || { type: 'default', text: level.toUpperCase() }
}

const columns: DataTableColumns<ScriptLogEntry> = [
  {
    title: 'Timestamp',
    key: 'timestamp',
    width: 180,
    render: (row) => new Date(row.timestamp).toLocaleString()
  },
  {
    title: 'Level',
    key: 'level',
    width: 80,
    render: (row) => {
      const tag = getLogLevelTag(row.level)
      return h(NTag, { type: tag.type as any, size: 'small' }, { default: () => tag.text })
    }
  },
  {
    title: 'Message',
    key: 'message',
    ellipsis: {
      tooltip: true
    }
  }
]

const loadLogs = async () => {
  if (!props.script) return

  loading.value = true
  try {
    const response = await scriptApi.getScriptLogs(props.script.id)
    logs.value = response.data
  } catch (error) {
    message.error('Failed to load script logs')
    console.error('Load logs error:', error)
  } finally {
    loading.value = false
  }
}

watch(() => props.show, (show) => {
  if (show && props.script) {
    loadLogs()
  }
})
</script>
