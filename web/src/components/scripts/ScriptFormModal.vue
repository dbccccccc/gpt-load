<template>
  <n-modal v-model:show="showModal" preset="dialog" title="Upload Script">
    <template #header>
      <div>Upload New Channel Script</div>
    </template>

    <n-form ref="formRef" :model="formData" :rules="rules" label-placement="top">
      <n-form-item label="Template" path="template">
        <n-select
          v-model:value="selectedTemplate"
          :options="templateOptions"
          placeholder="Choose a template to start with"
          @update:value="handleTemplateChange"
        />
      </n-form-item>

      <n-form-item label="Script Name" path="name">
        <n-input v-model:value="formData.name" placeholder="Enter script name" />
      </n-form-item>

      <n-form-item label="Display Name" path="display_name">
        <n-input v-model:value="formData.display_name" placeholder="Enter display name" />
      </n-form-item>

      <n-form-item label="Description" path="description">
        <n-input
          v-model:value="formData.description"
          type="textarea"
          placeholder="Enter description"
          :rows="3"
        />
      </n-form-item>

      <n-form-item label="Author" path="author">
        <n-input v-model:value="formData.author" placeholder="Enter author name" />
      </n-form-item>

      <n-form-item label="Channel Type" path="channel_type">
        <n-input v-model:value="formData.channel_type" placeholder="e.g., custom_ai" />
      </n-form-item>

      <n-form-item label="Version" path="version">
        <n-input v-model:value="formData.version" placeholder="e.g., 1.0.0" />
      </n-form-item>

      <n-form-item label="JavaScript Code" path="script">
        <n-input
          v-model:value="formData.script"
          type="textarea"
          placeholder="Paste your JavaScript code here..."
          :rows="15"
          style="font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;"
        />
      </n-form-item>
    </n-form>

    <template #action>
      <n-space>
        <n-button @click="showModal = false">Cancel</n-button>
        <n-button type="primary" @click="handleSubmit" :loading="submitting">
          Upload Script
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { scriptApi } from '@/api/scripts'
import { SCRIPT_TEMPLATES, type ChannelScript, type ChannelScriptMetadata } from '@/types/channel-script'
import {
    NButton,
    NForm,
    NFormItem,
    NInput,
    NModal,
    NSelect,
    NSpace,
    useMessage,
    type FormInst,
    type FormRules
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
const submitting = ref(false)
const selectedTemplate = ref<string>('')

const showModal = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value)
})

const formData = ref({
  name: '',
  display_name: '',
  description: '',
  author: '',
  version: '1.0.0',
  channel_type: '',
  script: ''
})

const templateOptions = SCRIPT_TEMPLATES.map(template => ({
  label: template.name,
  value: template.name,
  description: template.description
}))

const rules: FormRules = {
  name: { required: true, message: 'Script name is required' },
  channel_type: { required: true, message: 'Channel type is required' },
  version: { required: true, message: 'Version is required' },
  script: { required: true, message: 'Script code is required' }
}

const handleTemplateChange = (templateName: string) => {
  const template = SCRIPT_TEMPLATES.find(t => t.name === templateName)
  if (template) {
    formData.value = {
      name: template.metadata.name,
      display_name: template.metadata.name,
      description: template.metadata.description,
      author: template.metadata.author,
      version: template.metadata.version,
      channel_type: template.metadata.channel_type,
      script: template.template
    }
  }
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
    submitting.value = true

    // Extract metadata from form
    const metadata: ChannelScriptMetadata = {
      name: formData.value.name,
      version: formData.value.version,
      description: formData.value.description,
      author: formData.value.author,
      channel_type: formData.value.channel_type,
      supported_models: ['*'], // Default to all models
      default_test_model: 'gpt-3.5-turbo',
      default_validation_endpoint: '/v1/chat/completions'
    }

    await scriptApi.createScript({
      ...formData.value,
      metadata
    })

    message.success('Script uploaded successfully')
    emit('success')
    resetForm()
  } catch (error) {
    message.error('Failed to upload script')
    console.error('Upload script error:', error)
  } finally {
    submitting.value = false
  }
}

const resetForm = () => {
  formData.value = {
    name: '',
    display_name: '',
    description: '',
    author: '',
    version: '1.0.0',
    channel_type: '',
    script: ''
  }
  selectedTemplate.value = ''
}

watch(() => props.show, (show) => {
  if (!show) {
    resetForm()
  }
})
</script>
