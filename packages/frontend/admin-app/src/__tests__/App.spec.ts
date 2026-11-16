import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

import { getGeneratedRouteRecords } from '@unoadmin/vite-plugins/routes-generator'
import { authAdapter } from '../adapters/mock'
import { createAdminSession } from '../app-session'
import { installRouterGuards } from '../router/guards'

/** Keep the route contract aligned with the exclusions in vite.config.ts. */
const routeExcludePatterns = ['**/components', 'Layout.vue'] as const

describe('generated application route contract', () => {
  it('contains base and demo package pages', () => {
    const routeRecords = getGeneratedRouteRecords({
      layoutDir: '/layout.vue',
      // Reuse the same Gitignore-style filters as the Vite plugin configuration.
      exclude: routeExcludePatterns,
      routeRoots: [
        // The application package resolves its generated route roots from sibling packages.
        { directory: '../admin-base/src/views', basePath: '' },
        { namePrefix: 'demo', directory: '../domain-demo/src/views', basePath: '/demo' },
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
    expect(routePaths).not.toContain('/layout')
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
    const session = {
      state,
      can: () => false,
    } as never
    installRouterGuards(router, session)

    await router.push('/private')
    expect(router.currentRoute.value.name).toBe('signin')

    state.user = { id: 'user-1' }
    await router.push('/private')
    expect(router.currentRoute.value.name).toBe('forbidden')
  })

  // Re-entering the demo after logout must restore the session before routing.
  it('restores the demo session after sign-out', async () => {
    const session = createAdminSession({
      auth: authAdapter,
    })

    await session.initialize()
    await session.signOut()
    expect(session.state.user).toBeNull()

    await session.signIn()
    expect(session.state.user?.username).toBe('lin.wei')
  })
})
