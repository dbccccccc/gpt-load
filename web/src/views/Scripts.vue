<template>
  <div class="scripts-page">
    <n-space vertical size="large">
      <!-- Header -->
      <n-card>
        <template #header>
          <n-space justify="space-between" align="center">
            <div>
              <h2>Dynamic Channel Scripts</h2>
              <p>Manage custom channel scripts for AI service integrations</p>
            </div>
            <n-space>
              <n-button @click="reloadAllScripts" :loading="reloadingAll">
                <template #icon>
                  <n-icon><RefreshIcon /></n-icon>
                </template>
                Reload All
              </n-button>
              <n-button type="primary" @click="showCreateModal = true">
                <template #icon>
                  <n-icon><AddIcon /></n-icon>
                </template>
                Upload Script
              </n-button>
            </n-space>
          </n-space>
        </template>
      </n-card>

      <!-- Scripts List -->
      <n-card>
        <ScriptList
          :scripts="scripts"
          :loading="loading"
          @edit="handleEdit"
          @delete="handleDelete"
          @enable="handleEnable"
          @disable="handleDisable"
          @view-logs="handleViewLogs"
          @test="handleTest"
          @reload="handleReload"
        />
      </n-card>
    </n-space>

    <!-- Create/Edit Script Modal -->
    <ScriptFormModal
      v-model:show="showCreateModal"
      :script="editingScript"
      @success="handleScriptSaved"
    />

    <!-- Script Editor Modal -->
    <ScriptEditorModal
      v-model:show="showEditorModal"
      :script="editingScript"
      @success="handleScriptSaved"
    />

    <!-- Test Script Modal -->
    <ScriptTestModal
      v-model:show="showTestModal"
      :script="testingScript"
    />

    <!-- Script Logs Modal -->
    <ScriptLogsModal
      v-model:show="showLogsModal"
      :script="logsScript"
    />
  </div>
</template>

<script setup lang="ts">
import { scriptApi } from '@/api/scripts'
import ScriptEditorModal from '@/components/scripts/ScriptEditorModal.vue'
import ScriptFormModal from '@/components/scripts/ScriptFormModal.vue'
import ScriptList from '@/components/scripts/ScriptList.vue'
import ScriptLogsModal from '@/components/scripts/ScriptLogsModal.vue'
import ScriptTestModal from '@/components/scripts/ScriptTestModal.vue'
import type { ChannelScript } from '@/types/channel-script'
import { Add as AddIcon, Refresh as RefreshIcon } from '@vicons/ionicons5'
import { NButton, NCard, NIcon, NSpace, useMessage } from 'naive-ui'
import { onMounted, ref } from 'vue'

const message = useMessage()

// State
const scripts = ref<ChannelScript[]>([])
const loading = ref(false)
const reloadingAll = ref(false)
const showCreateModal = ref(false)
const showEditorModal = ref(false)
const showTestModal = ref(false)
const showLogsModal = ref(false)
const editingScript = ref<ChannelScript | null>(null)
const testingScript = ref<ChannelScript | null>(null)
const logsScript = ref<ChannelScript | null>(null)

// Methods
const loadScripts = async () => {
  loading.value = true
  try {
    const response = await scriptApi.getScripts()
    scripts.value = response.data
  } catch (error) {
    message.error('Failed to load scripts')
    console.error('Load scripts error:', error)
  } finally {
    loading.value = false
  }
}

const handleEdit = (script: ChannelScript) => {
  editingScript.value = script
  showEditorModal.value = true
}

const handleDelete = async (script: ChannelScript) => {
  try {
    await scriptApi.deleteScript(script.id)
    message.success('Script deleted successfully')
    await loadScripts()
  } catch (error) {
    message.error('Failed to delete script')
    console.error('Delete script error:', error)
  }
}

const handleEnable = async (script: ChannelScript) => {
  try {
    await scriptApi.enableScript(script.id)
    message.success('Script enabled successfully')
    await loadScripts()
  } catch (error) {
    message.error('Failed to enable script')
    console.error('Enable script error:', error)
  }
}

const handleDisable = async (script: ChannelScript) => {
  try {
    await scriptApi.disableScript(script.id)
    message.success('Script disabled successfully')
    await loadScripts()
  } catch (error) {
    message.error('Failed to disable script')
    console.error('Disable script error:', error)
  }
}

const handleTest = (script: ChannelScript) => {
  testingScript.value = script
  showTestModal.value = true
}

const handleViewLogs = (script: ChannelScript) => {
  logsScript.value = script
  showLogsModal.value = true
}

const handleReload = async (script: ChannelScript) => {
  try {
    await scriptApi.reloadScript(script.id)
    message.success('Script reloaded successfully')
    await loadScripts()
  } catch (error) {
    message.error('Failed to reload script')
    console.error('Reload script error:', error)
  }
}

const reloadAllScripts = async () => {
  reloadingAll.value = true
  try {
    await scriptApi.reloadAllScripts()
    message.success('All scripts reloaded successfully')
    await loadScripts()
  } catch (error) {
    message.error('Failed to reload scripts')
    console.error('Reload all scripts error:', error)
  } finally {
    reloadingAll.value = false
  }
}

const handleScriptSaved = () => {
  showCreateModal.value = false
  showEditorModal.value = false
  editingScript.value = null
  loadScripts()
}

// Lifecycle
onMounted(() => {
  loadScripts()
})
</script>

<style scoped>
.scripts-page {
  padding: 24px;
}
</style>
