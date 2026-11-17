import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

import type { AccountId, AuthPort, AuthSession, LegalEntityId, TenantId } from '@unoadmin/core'
import { getGeneratedRouteRecords } from '@unoadmin/vite-plugins/routes-generator'
import { createAdminSession } from '../app-session'
import { installRouterGuards } from '../router/guards'

/** 本地测试替身，覆盖集团租户（一个租户 + 两个经营主体）的会话形态。 */
const sessionFixture: AuthSession = {
  account: {
    id: 'acc-001' as AccountId,
    username: 'chen.yu',
    displayName: '陈宇',
    email: 'chen.yu@example.com',
    phone: '',
    status: 'active',
    tenantId: 'tenant-holding' as TenantId,
    dataScope: 'tenant',
    platformAdmin: false,
    legalEntityIds: ['le-hq' as LegalEntityId],
    orgIds: [],
    roleIds: [],
    roleNames: ['集团管理员'],
    legalEntityNames: ['集团总部', '华东公司']
  },
  permissions: ['*'],
  tenants: [{ id: 'tenant-holding' as TenantId, name: '汉唐控股集团', code: 'HANTANG', mode: 'group', plan: 'enterprise', contact: '', status: 'active' }],
  currentTenantId: 'tenant-holding' as TenantId,
  legalEntities: [
    { id: 'le-hq' as LegalEntityId, tenantId: 'tenant-holding' as TenantId, parentId: '', type: 'GROUP', name: '集团总部', code: 'HT-HQ', address: '', status: 'active' },
    { id: 'le-east' as LegalEntityId, tenantId: 'tenant-holding' as TenantId, parentId: 'le-hq', type: 'COMPANY', name: '华东公司', code: 'HT-EAST', address: '', status: 'active' }
  ],
  currentLegalEntityId: 'le-hq' as LegalEntityId,
  dataScope: 'tenant'
}
const testAuth: AuthPort = {
  async signIn() {
    return sessionFixture
  },
  async restore() {
    return sessionFixture
  },
  async signOut() {}
}

/** 与 vite.config.ts 中的排除规则保持一致。 */
const routeExcludePatterns = ['**/components', 'Layout.vue'] as const

describe('generated application route contract', () => {
  it('contains base package pages', () => {
    const routeRecords = getGeneratedRouteRecords({
      layoutDir: '/layout.vue',
      exclude: routeExcludePatterns,
      routeRoots: [{ directory: '../admin-base/src/views', basePath: '' }]
    })

    const routePaths: string[] = []
    const collectPaths = (routes: typeof routeRecords) => {
      for (const route of routes) {
        routePaths.push(route.path)
        if (route.children) collectPaths(route.children)
      }
    }
    collectPaths(routeRecords)

    expect(routePaths).toContain('/system/accounts')
    expect(routePaths).toContain('/system/entities')
    expect(routePaths).toContain('/system/organizations')
    expect(routePaths).not.toContain('/layout')
  })

  it('redirects unauthenticated access and rejects missing permissions', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/signin', name: 'signin', component: { template: '<div />' } },
        { path: '/403', name: 'forbidden', component: { template: '<div />' } },
        { path: '/private', name: 'private', component: { template: '<div />' }, meta: { permission: 'private:read' } }
      ]
    })
    const state = { account: null as { id: string } | null }
    const session = {
      state,
      can: () => false
    } as never
    installRouterGuards(router, session)

    await router.push('/private')
    expect(router.currentRoute.value.name).toBe('signin')

    state.account = { id: 'acc-1' }
    await router.push('/private')
    expect(router.currentRoute.value.name).toBe('forbidden')
  })

  it('restores the session after sign-out', async () => {
    const session = createAdminSession({ auth: testAuth })

    await session.initialize()
    await session.signOut()
    expect(session.state.account).toBeNull()

    await session.signIn()
    expect(session.state.account?.username).toBe('chen.yu')
    expect(session.state.currentLegalEntity?.name).toBe('集团总部')
  })

  it('limits entity access by data scope', async () => {
    const session = createAdminSession({ auth: testAuth })
    await session.initialize()

    // 租户级范围可访问全部主体。
    expect(session.canAccessEntity('le-east')).toBe(true)

    const entityScoped = createAdminSession({
      auth: {
        ...testAuth,
        async restore() {
          return { ...sessionFixture, dataScope: 'entity' as const }
        }
      }
    })
    await entityScoped.initialize()
    expect(entityScoped.canAccessEntity('le-hq')).toBe(true)
    expect(entityScoped.canAccessEntity('le-south')).toBe(false)
  })
})
