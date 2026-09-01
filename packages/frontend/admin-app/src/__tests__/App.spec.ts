import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

import { getGeneratedRouteRecords } from '@unoadmin/vite-plugins/routes-generator'
import { installRouterGuards } from '../router/guards'

describe('generated application route contract', () => {
  it('contains base and demo package pages', () => {
    const routeRecords = getGeneratedRouteRecords({
      layoutDir: '/layout.vue',
      routeRoots: [
        // The application package resolves its generated route roots from sibling packages.
        { moduleId: 'admin-base', directory: '../admin-base/src/views', basePath: '' },
        { moduleId: 'demo', directory: '../domain-demo/src/views', basePath: '/demo' },
      ],
    })

    const routePaths: string[] = []
    const collectPaths = (routes: typeof routeRecords) => {
      for (const route of routes) {
        routePaths.push(route.path)
        if (route.children) collectPaths(route.children)
      }
    }
    collectPaths(routeRecords)

    expect(routePaths).toContain('/system/users')
    expect(routePaths).toContain('/demo/overview')
    expect(routePaths).toContain('/demo/records')
  })

  it('redirects unauthenticated access and rejects missing permissions', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/signin', name: 'signin', component: { template: '<div />' } },
        { path: '/403', name: 'forbidden', component: { template: '<div />' } },
        { path: '/private', name: 'private', component: { template: '<div />' }, meta: { permission: 'private:read' } },
      ],
    })
    const state = { user: null as { id: string } | null }
    const runtime = {
      state,
      can: () => false,
    } as never
    installRouterGuards(router, runtime)

    await router.push('/private')
    expect(router.currentRoute.value.name).toBe('signin')

    state.user = { id: 'user-1' }
    await router.push('/private')
    expect(router.currentRoute.value.name).toBe('forbidden')
  })
})
