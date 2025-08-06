<template>
  <n-modal v-model:show="showModal" preset="dialog" title="Edit Script" style="width: 90%; max-width: 1200px;">
    <div v-if="script">
      <n-tabs type="line" animated>
        <n-tab-pane name="editor" tab="Script Editor">
          <n-form ref="formRef" :model="formData" label-placement="top">
            <n-grid :cols="2" :x-gap="16">
              <n-grid-item>
                <n-form-item label="Script Name">
                  <n-input v-model:value="formData.name" />
                </n-form-item>
              </n-grid-item>
              <n-grid-item>
                <n-form-item label="Display Name">
                  <n-input v-model:value="formData.display_name" />
                </n-form-item>
              </n-grid-item>
              <n-grid-item>
                <n-form-item label="Author">
                  <n-input v-model:value="formData.author" />
                </n-form-item>
              </n-grid-item>
              <n-grid-item>
                <n-form-item label="Version">
                  <n-input v-model:value="formData.version" />
                </n-form-item>
              </n-grid-item>
            </n-grid>

            <n-form-item label="Description">
              <n-input v-model:value="formData.description" type="textarea" :rows="2" />
            </n-form-item>

            <n-form-item label="JavaScript Code">
              <n-input
                v-model:value="formData.script"
                type="textarea"
                :rows="20"
                style="font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace; font-size: 14px;"
              />
            </n-form-item>
          </n-form>
        </n-tab-pane>

        <n-tab-pane name="validate" tab="Validate">
          <n-space vertical>
            <n-button type="primary" @click="validateScript" :loading="validating">
              Validate Script
            </n-button>
            <n-card v-if="validationResult" :title="validationResult.valid ? 'Validation Passed' : 'Validation Failed'">
              <n-alert
                :type="validationResult.valid ? 'success' : 'error'"
                :title="validationResult.valid ? 'Script is valid' : 'Validation errors found'"
              >
                {{ validationResult.message || validationResult.error }}
              </n-alert>
            </n-card>
          </n-space>
        </n-tab-pane>
      </n-tabs>
    </div>

    <template #action>
      <n-space>
        <n-button @click="showModal = false">Cancel</n-button>
        <n-button type="primary" @click="handleSave" :loading="saving">
          Save Changes
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { scriptApi, type ScriptValidationResult } from '@/api/scripts'
import type { ChannelScript } from '@/types/channel-script'
import {
    NAlert,
    NButton,
    NCard,
    NForm,
    NFormItem,
    NGrid,
    NGridItem,
    NInput,
    NModal,
    NSpace,
    NTabPane,
    NTabs,
    useMessage,
    type FormInst
} from 'naive-ui'
import { computed, ref, watch } from 'vue'

interface Props {
  show: boolean
  script?: ChannelScript | null
}

interface Emits {
  (e: 'update:show', value: boolean): void
  (e: 'success'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const message = useMessage()
const formRef = ref<FormInst>()
const saving = ref(false)
const validating = ref(false)
const validationResult = ref<ScriptValidationResult | null>(null)

const showModal = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value)
})

const formData = ref({
  name: '',
  display_name: '',
  description: '',
  author: '',
  version: '',
  script: ''
})

const validateScript = async () => {
  if (!props.script) return

  validating.value = true
  try {
    const result = await scriptApi.validateScript({
      script: formData.value.script,
      metadata: props.script.metadata
    })
    validationResult.value = result.data
  } catch (error) {
    message.error('Failed to validate script')
    console.error('Validation error:', error)
  } finally {
    validating.value = false
  }
}

const handleSave = async () => {
  if (!props.script) return

  saving.value = true
  try {
    await scriptApi.updateScript(props.script.id, formData.value)
    message.success('Script updated successfully')
    emit('success')
  } catch (error) {
    message.error('Failed to update script')
    console.error('Update script error:', error)
  } finally {
    saving.value = false
  }
}

watch(() => props.script, (script) => {
  if (script) {
    formData.value = {
      name: script.name,
      display_name: script.display_name,
      description: script.description,
      author: script.author,
      version: script.version,
      script: script.script
    }
    validationResult.value = null
  }
}, { immediate: true })
</script>
