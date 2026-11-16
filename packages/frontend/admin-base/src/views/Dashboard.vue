<script setup lang="ts">
import { computed } from 'vue'

import { useAdminRuntime } from '../runtime'

const runtime = useAdminRuntime()
const moduleCount = computed(() => runtime.modules.length)
const tenantCount = computed(() => runtime.state.tenants.length)
</script>

<template>
  <section class="base-page">
    <header class="base-page__heading">
      <div>
        <h1>工作台</h1>
        <p>从这里进入基础管理能力和已安装的业务模块。</p>
      </div>
      <span class="base-status">运行正常</span>
    </header>

    <div class="dashboard-grid">
      <article class="dashboard-card dashboard-card--gold">
        <span>当前租户</span>
        <strong>{{ runtime.state.currentTenant?.name || '未选择' }}</strong>
        <small>{{ runtime.state.currentTenant?.code || '请先配置租户上下文' }}</small>
      </article>
      <article class="dashboard-card dashboard-card--teal">
        <span>可用租户</span>
        <strong>{{ tenantCount }}</strong>
        <small>由认证会话提供</small>
      </article>
      <article class="dashboard-card dashboard-card--blue">
        <span>已安装模块</span>
        <strong>{{ moduleCount }}</strong>
        <small>路由由 routes-generator 生成</small>
      </article>
    </div>

    <div class="base-panel dashboard-welcome">
      <div>
        <span class="dashboard-welcome__kicker">模块化骨架</span>
        <h2>基础管理能力已经作为独立 package 接入。</h2>
        <p>左侧系统管理包含用户、角色、权限和租户页面；示例业务模块用于验证跨 package 的自动路由。</p>
      </div>
      <RouterLink class="base-button" to="/demo/overview">查看示例业务</RouterLink>
    </div>
  </section>
</template>

<style scoped>
.dashboard-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.dashboard-card {
  display: grid;
  min-height: 142px;
  align-content: space-between;
  gap: 10px;
  padding: 20px;
  border-radius: 12px;
  color: #18323a;
}

.dashboard-card span,
.dashboard-card small {
  font-size: 12px;
}

.dashboard-card strong {
  font-size: 24px;
}

.dashboard-card small {
  color: rgb(24 50 58 / 65%);
}

.dashboard-card--gold {
  background: #f3c969;
}

.dashboard-card--teal {
  background: #bde5de;
}

.dashboard-card--blue {
  background: #c6ddec;
}

.dashboard-welcome {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 30px;
  padding: 26px;
}

.dashboard-welcome__kicker {
  color: #0f766e;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.dashboard-welcome h2 {
  margin: 8px 0 0;
  font-size: 21px;
}

.dashboard-welcome p {
  max-width: 620px;
  margin: 8px 0 0;
  color: #6d8086;
  line-height: 1.6;
}

.dashboard-welcome .base-button {
  flex: 0 0 auto;
  color: #fff;
  text-decoration: none;
}

@media (max-width: 760px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }

  .dashboard-welcome {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
