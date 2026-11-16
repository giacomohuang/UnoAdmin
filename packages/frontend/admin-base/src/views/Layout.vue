<script setup lang="ts">
import type { NavigationItem, TenantId } from '@unoadmin/core'
import { useAdminRuntime } from '@unoadmin/runtime'
import { computed, ref } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'

const runtime = useAdminRuntime()
const route = useRoute()
const router = useRouter()
const sidebarOpen = ref(false)
const navigation = computed(() => runtime.getNavigation())

/** Keep the active navigation state aligned with nested route paths. */
function isActive(item: NavigationItem) {
  return Boolean(item.to && (route.path === item.to || route.path.startsWith(`${item.to}/`)))
}

/** Switch the runtime tenant and close the mobile navigation affordance. */
async function handleTenantChange(event: Event) {
  const target = event.target as HTMLSelectElement
  await runtime.switchTenant(target.value as TenantId)
  sidebarOpen.value = false
}

/** End the local session before returning to the manually declared sign-in route. */
async function signOut() {
  await runtime.signOut()
  await router.push('/signin')
}
</script>

<template>
  <!-- The reusable shell owns navigation and tenant context, while pages own their data. -->
  <div class="admin-layout">
    <aside class="admin-sidebar" :class="{ 'admin-sidebar--open': sidebarOpen }">
      <div class="admin-brand">
        <span class="admin-brand__mark">{{ runtime.branding.productMark }}</span>
        <div>
          <strong>{{ runtime.branding.productName }}</strong>
          <span>管理工作台</span>
        </div>
      </div>

      <nav class="admin-nav" aria-label="主导航">
        <template v-for="item in navigation" :key="item.id">
          <RouterLink v-if="item.to" class="admin-nav__item" :class="{ 'admin-nav__item--active': isActive(item) }" :to="item.to" @click="sidebarOpen = false">
            <span>{{ item.label }}</span>
          </RouterLink>
          <div v-else class="admin-nav__group">
            <span class="admin-nav__label">{{ item.label }}</span>
            <RouterLink v-for="child in item.children" :key="child.id" class="admin-nav__item admin-nav__item--child" :class="{ 'admin-nav__item--active': isActive(child) }" :to="child.to || '/404'" @click="sidebarOpen = false">
              <span>{{ child.label }}</span>
            </RouterLink>
          </div>
        </template>
      </nav>

      <div class="admin-sidebar__footer">
        <span>模块 {{ runtime.modules.length }} 个</span>
        <span>SPA 模式</span>
      </div>
    </aside>

    <div v-if="sidebarOpen" class="admin-backdrop" aria-hidden="true" @click="sidebarOpen = false" />

    <section class="admin-main">
      <header class="admin-topbar">
        <button class="admin-menu-button" type="button" aria-label="打开导航" @click="sidebarOpen = !sidebarOpen">
          <span />
          <span />
          <span />
        </button>
        <div class="admin-topbar__context">
          <span class="admin-topbar__eyebrow">当前空间</span>
          <select class="admin-tenant-select" :value="runtime.state.currentTenant?.id" aria-label="切换租户" @change="handleTenantChange">
            <option v-for="tenant in runtime.state.tenants" :key="tenant.id" :value="tenant.id">
              {{ tenant.name }}
            </option>
          </select>
        </div>
        <div class="admin-topbar__account">
          <div class="admin-avatar">{{ runtime.state.user?.displayName.slice(0, 1) }}</div>
          <div class="admin-account-copy">
            <strong>{{ runtime.state.user?.displayName }}</strong>
            <span>{{ runtime.state.user?.email }}</span>
          </div>
          <button class="admin-signout" type="button" @click="signOut">退出登录</button>
        </div>
      </header>

      <main class="admin-content">
        <RouterView />
      </main>
    </section>
  </div>
</template>

<style scoped>
.admin-layout {
  display: flex;
  min-height: 100vh;
  color: #18323a;
  background: #f5f9f8;
}

.admin-sidebar {
  display: flex;
  position: fixed;
  z-index: 20;
  top: 0;
  bottom: 0;
  left: 0;
  flex-direction: column;
  width: 248px;
  padding: 24px 16px 18px;
  color: #d9eeeb;
  background: #143a3d;
}

.admin-brand {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 10px 28px;
}

.admin-brand__mark {
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  border-radius: 10px;
  color: #143a3d;
  background: #f3c969;
  font-weight: 900;
}

.admin-brand strong,
.admin-brand span {
  display: block;
}

.admin-brand strong {
  color: #fff;
  font-size: 15px;
}

.admin-brand div span {
  margin-top: 3px;
  color: #9fc0bc;
  font-size: 11px;
}

.admin-nav {
  display: grid;
  gap: 6px;
}

.admin-nav__group {
  display: grid;
  gap: 4px;
  margin-top: 12px;
}

.admin-nav__label {
  padding: 8px 12px 5px;
  color: #7fa8a4;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.admin-nav__item {
  display: flex;
  min-height: 42px;
  align-items: center;
  padding: 0 12px;
  border-radius: 8px;
  color: #c9e2df;
  font-size: 14px;
  text-decoration: none;
}

.admin-nav__item:hover,
.admin-nav__item--active {
  color: #fff;
  background: #1e5354;
}

.admin-nav__item--active {
  box-shadow: inset 3px 0 #f3c969;
}

.admin-nav__item--child {
  margin-left: 8px;
  color: #a9cbc7;
}

.admin-sidebar__footer {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  margin-top: auto;
  padding: 16px 10px 0;
  border-top: 1px solid rgb(159 192 188 / 20%);
  color: #8db2ae;
  font-size: 11px;
}

.admin-main {
  display: flex;
  width: calc(100% - 248px);
  min-width: 0;
  min-height: 100vh;
  margin-left: 248px;
  flex-direction: column;
}

.admin-topbar {
  display: flex;
  position: sticky;
  z-index: 10;
  top: 0;
  min-height: 72px;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 0 34px;
  border-bottom: 1px solid #dfeaea;
  background: rgb(255 255 255 / 92%);
  backdrop-filter: blur(12px);
}

.admin-topbar__context,
.admin-topbar__account {
  display: flex;
  align-items: center;
  gap: 10px;
}

.admin-topbar__eyebrow,
.admin-account-copy span {
  color: #7d9296;
  font-size: 11px;
}

.admin-tenant-select {
  min-height: 34px;
  max-width: 200px;
  padding: 0 30px 0 10px;
  border: 0;
  color: #18323a;
  background: transparent;
  font: inherit;
  font-weight: 700;
}

.admin-avatar {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: 50%;
  color: #143a3d;
  background: #f3c969;
  font-weight: 800;
}

.admin-account-copy {
  display: grid;
  gap: 2px;
}

.admin-account-copy strong {
  font-size: 13px;
}

.admin-signout {
  margin-left: 10px;
  padding: 6px 0 6px 10px;
  border: 0;
  border-left: 1px solid #d9e5e5;
  color: #6d8086;
  background: transparent;
  cursor: pointer;
  font: inherit;
  font-size: 12px;
}

.admin-signout:hover {
  color: #b42318;
}

.admin-content {
  width: min(100%, 1280px);
  box-sizing: border-box;
  margin: 0 auto;
  padding: 34px;
}

.admin-menu-button,
.admin-backdrop {
  display: none;
}

@media (max-width: 860px) {
  .admin-sidebar {
    transform: translateX(-100%);
    transition: transform 160ms ease;
  }

  .admin-sidebar--open {
    transform: translateX(0);
  }

  .admin-backdrop {
    display: block;
    position: fixed;
    z-index: 15;
    inset: 0;
    background: rgb(20 58 61 / 36%);
  }

  .admin-main {
    width: 100%;
    margin-left: 0;
  }

  .admin-menu-button {
    display: grid;
    width: 36px;
    height: 36px;
    align-content: center;
    gap: 4px;
    padding: 8px;
    border: 1px solid #d9e5e5;
    border-radius: 8px;
    background: #fff;
    cursor: pointer;
  }

  .admin-menu-button span {
    display: block;
    height: 2px;
    border-radius: 2px;
    background: #18323a;
  }

  .admin-topbar {
    padding: 0 20px;
  }

  .admin-content {
    padding: 24px 20px;
  }
}

@media (max-width: 620px) {
  .admin-topbar__eyebrow,
  .admin-account-copy,
  .admin-signout {
    display: none;
  }

  .admin-topbar__context {
    margin-right: auto;
  }

  .admin-topbar {
    gap: 10px;
  }
}
</style>
