<script setup lang="ts">
import { ref } from 'vue'

import type { FileStoragePort, StoredFile } from '@unoadmin/core'

interface Props {
  uploader: FileStoragePort
  accept?: string
  maxSizeMb?: number
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  accept: '*/*',
  maxSizeMb: 20,
  disabled: false,
})

const emit = defineEmits<{
  uploaded: [file: StoredFile]
  failed: [error: Error]
}>()

const input = ref<HTMLInputElement | null>(null)
const progress = ref(0)
const uploading = ref(false)
const errorMessage = ref('')
const uploadedFile = ref<StoredFile | null>(null)

/** Open the native file picker while keeping the visible control consistent. */
function openPicker() {
  if (!props.disabled && !uploading.value) input.value?.click()
}

/** Validate and upload one file through the injected storage port. */
async function handleChange(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  target.value = ''
  if (!file) return

  errorMessage.value = ''
  uploadedFile.value = null
  if (file.size > props.maxSizeMb * 1024 * 1024) {
    errorMessage.value = `文件不能超过 ${props.maxSizeMb}MB`
    return
  }

  uploading.value = true
  progress.value = 0
  try {
    const result = await props.uploader.upload({
      file,
      onProgress: (value) => {
        progress.value = value
      },
    })
    uploadedFile.value = result.file
    progress.value = 100
    emit('uploaded', result.file)
  } catch (error) {
    const normalized = error instanceof Error ? error : new Error('文件上传失败')
    errorMessage.value = normalized.message
    emit('failed', normalized)
  } finally {
    uploading.value = false
  }
}
</script>

<template>
  <div class="file-upload">
    <input
      ref="input"
      class="file-upload__input"
      type="file"
      :accept="accept"
      :disabled="disabled || uploading"
      @change="handleChange"
    >
    <button class="file-upload__button" type="button" :disabled="disabled || uploading" @click="openPicker">
      {{ uploading ? `上传中 ${progress}%` : '选择文件' }}
    </button>
    <div v-if="uploading" class="file-upload__track" aria-hidden="true">
      <span class="file-upload__bar" :style="{ width: `${progress}%` }" />
    </div>
    <p v-if="uploadedFile" class="file-upload__success">已上传：{{ uploadedFile.name }}</p>
    <p v-if="errorMessage" class="file-upload__error" role="alert">{{ errorMessage }}</p>
    <p class="file-upload__hint">单个文件不超过 {{ maxSizeMb }}MB</p>
  </div>
</template>
