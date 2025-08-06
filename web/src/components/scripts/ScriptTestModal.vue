<template>
  <n-modal v-model:show="showModal" preset="dialog" title="Test Script">
    <div v-if="script">
      <n-space vertical>
        <n-card title="Script Information">
          <n-descriptions :column="2" bordered>
            <n-descriptions-item label="Name">{{ script.display_name || script.name }}</n-descriptions-item>
            <n-descriptions-item label="Version">{{ script.version }}</n-descriptions-item>
            <n-descriptions-item label="Channel Type">{{ script.channel_type }}</n-descriptions-item>
            <n-descriptions-item label="Author">{{ script.author }}</n-descriptions-item>
          </n-descriptions>
        </n-card>

        <n-card title="Test Configuration">
          <n-form label-placement="top">
            <n-form-item label="Test Data (JSON)">
              <n-input
                v-model:value="testData"
                type="textarea"
                :rows="8"
                placeholder='{"test": "data"}'
                style="font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;"
              />
            </n-form-item>
          </n-form>
        </n-card>

        <n-button type="primary" @click="runTest" :loading="testing" block>
          Run Test
        </n-button>

        <n-card v-if="testResult" :title="testResult.valid ? 'Test Passed' : 'Test Failed'">
          <n-alert
            :type="testResult.valid ? 'success' : 'error'"
            :title="testResult.valid ? 'Script test successful' : 'Test failed'"
          >
            {{ testResult.message || testResult.error }}
          </n-alert>

          <div v-if="testResult.runtime" style="margin-top: 16px;">
            <n-text depth="3">Runtime: {{ testResult.runtime }}</n-text>
          </div>
        </n-card>
      </n-space>
    </div>

    <template #action>
      <n-button @click="showModal = false">Close</n-button>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { scriptApi, type ScriptTestResult } from '@/api/scripts'
import type { ChannelScript } from '@/types/channel-script'
import {
    NAlert,
    NButton,
    NCard,
    NDescriptions,
    NDescriptionsItem,
    NForm,
    NFormItem,
    NInput,
    NModal,
    NSpace,
    NText,
    useMessage
} from 'naive-ui'
import { computed, ref } from 'vue'

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
const testing = ref(false)
const testData = ref('{}')
const testResult = ref<ScriptTestResult | null>(null)

const showModal = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value)
})

const runTest = async () => {
  if (!props.script) return

  testing.value = true
  try {
    let parsedTestData = {}
    try {
      parsedTestData = JSON.parse(testData.value)
    } catch (e) {
      message.error('Invalid JSON in test data')
      return
    }

    const result = await scriptApi.testScript({
      script: props.script.script,
      metadata: props.script.metadata,
      test_data: parsedTestData
    })

    testResult.value = result.data
  } catch (error) {
    message.error('Failed to run test')
    console.error('Test error:', error)
  } finally {
    testing.value = false
  }
}
</script>
