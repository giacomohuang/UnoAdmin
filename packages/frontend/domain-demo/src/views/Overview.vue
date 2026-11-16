<script setup lang="ts">
import { onMounted, ref } from 'vue'

import { useAdminRuntime } from '@unoadmin/admin-base/runtime'
import type { DemoAppServices, DemoOverview } from '@unoadmin/domain-demo'
import { FileUpload } from '@unoadmin/ui'

const runtime = useAdminRuntime<DemoAppServices>()
const overview = ref<DemoOverview | null>(null)
const loading = ref(false)
const uploadMessage = ref('')

/** Load domain data through the app-provided demo service. */
async function loadOverview() {
  loading.value = true
  overview.value = await runtime.services.demo.getOverview()
  loading.value = false
}

/** Keep the example page visibly connected to the shared upload component. */
function handleUploaded(file: { name: string }) {
  uploadMessage.value = `文件 ${file.name} 已完成上传`
}

onMounted(loadOverview)
</script>

<template>
  <section class="base-page">
    <header class="base-page__heading">
      <div>
        <h1>示例业务</h1>
        <p>这个页面来自独立的 domain-demo package，并通过自动路由接入应用。</p>
      </div>
      <span class="base-status">domain-demo</span>
    </header>

    <div v-if="loading" class="base-panel base-empty">正在加载业务概览...</div>
    <div v-else class="demo-stats">
      <article class="demo-stat"><span>进行中的项目</span><strong>{{ overview?.activeProjects }}</strong><small>当前租户</small></article>
      <article class="demo-stat"><span>待处理任务</span><strong>{{ overview?.pendingTasks }}</strong><small>需要关注</small></article>
      <article class="demo-stat"><span>本月增长</span><strong>{{ overview?.monthlyGrowth }}%</strong><small>较上月</small></article>
    </div>

    <div class="base-panel demo-upload-panel">
      <div>
        <h2>共享文件上传</h2>
        <p>组件来自 <code>@unoadmin/ui</code>，存储实现由应用 adapter 注入。</p>
      </div>
      <FileUpload :uploader="runtime.services.storage" accept=".pdf,.png,.jpg" @uploaded="handleUploaded" />
      <span v-if="uploadMessage" class="base-status">{{ uploadMessage }}</span>
    </div>
  </section>
</template>

<style scoped>
.demo-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.demo-stat {
  display: grid;
  gap: 12px;
  padding: 22px;
  border: 1px solid #d9e5e5;
  border-radius: 12px;
  background: #fff;
}

.demo-stat span,
.demo-stat small {
  color: #6d8086;
  font-size: 12px;
}

.demo-stat strong {
  color: #18323a;
  font-size: 30px;
}

.demo-upload-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 24px;
}

.demo-upload-panel h2 {
  margin: 0;
  font-size: 17px;
}

.demo-upload-panel p {
  margin: 6px 0 0;
  color: #6d8086;
  font-size: 13px;
}

code {
  color: #0f766e;
  font-size: 12px;
}

@media (max-width: 760px) {
  .demo-stats {
    grid-template-columns: 1fr;
  }

  .demo-upload-panel {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>

<router lang="json">
{
  "title": "示例业务概览",
  "permission": "demo:read"
}
</router>
