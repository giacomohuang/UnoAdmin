import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'

import { getGeneratedRouteRecords } from '@unoadmin/vite-plugins/routes-generator'
import { afterEach, describe, expect, it } from 'vitest'

const temporaryRoots = new Set<string>()

/** Create an isolated route root from a map of relative file paths to contents. */
function createRouteRoot(files: Record<string, string>): string {
  const root = mkdtempSync(join(tmpdir(), 'unoadmin-routes-'))
  temporaryRoots.add(root)

  for (const [relativePath, content] of Object.entries(files)) {
    const fullPath = join(root, relativePath)
    mkdirSync(dirname(fullPath), { recursive: true })
    writeFileSync(fullPath, content)
  }

  return root
}

/** Flatten generated routes so assertions can focus on public paths. */
function collectRoutePaths(routeRecords: ReturnType<typeof getGeneratedRouteRecords>): string[] {
  return routeRecords.flatMap((route) => [route.path, ...collectRoutePaths(route.children ?? [])])
}

/** Build a page route configuration block with a stable serialized shape. */
function routerConfig(config: Record<string, unknown>): string {
  return `<router lang="json">${JSON.stringify(config)}</router>`
}

// Remove every temporary route root even when an assertion fails.
afterEach(() => {
  for (const root of temporaryRoots) rmSync(root, { recursive: true, force: true })
  temporaryRoots.clear()
})

describe('routes-generator exclusions', () => {
  it('does not apply built-in page or directory exclusions', () => {
    const directory = createRouteRoot({
      'Layout.vue': '',
      'Signin.vue': '',
      '403.vue': '',
      '404.vue': '',
      'components/Widget.vue': '',
      'Visible.vue': ''
    })

    const paths = collectRoutePaths(
      getGeneratedRouteRecords({
        routeRoots: [{ moduleId: 'test', directory }]
      })
    )

    expect(paths).toEqual(expect.arrayContaining(['/layout', '/signin', '/403', '/404', '/components/widget', '/visible']))
  })

  it('supports wildcard, negation, and root-specific Gitignore rules', () => {
    const directory = createRouteRoot({
      'Visible.vue': '',
      'System/Users.vue': '',
      'System/Roles.vue': '',
      'System/routeMeta.json': '{"title":',
      'components/Hidden.vue': '',
      'Components/HiddenCase.vue': '',
      'legacy/Deep/Hidden.vue': ''
    })
    const routes = getGeneratedRouteRecords({
      exclude: ['components/', 'System/*.vue'],
      routeRoots: [
        {
          moduleId: 'test',
          directory,
          exclude: ['!System/Users.vue', 'legacy/**', 'System/routeMeta.json']
        }
      ]
    })
    const paths = collectRoutePaths(routes)

    expect(paths).toContain('/visible')
    expect(paths).toContain('/system/users')
    expect(paths).not.toContain('/system/roles')
    expect(paths).not.toContain('/components/hidden')
    expect(paths).not.toContain('/components/hiddencase')
    expect(paths).not.toContain('/legacy/deep/hidden')
    expect(routes.find((route) => route.path === '/system')?.meta.title).toBe('system')
  })
})

describe('routes-generator layout modes', () => {
  it('creates standalone pages without a Layout parent', () => {
    const directory = createRouteRoot({
      'Dashboard.vue': '',
      'Preview.vue': routerConfig({ layout: false, routePath: '/preview', title: '预览' }),
    })
    const routes = getGeneratedRouteRecords({
      routeRoots: [{ moduleId: 'test', directory }],
    })
    const preview = routes.find((route) => route.path === '/preview')
    const dashboard = routes.find((route) => route.path === '/dashboard')

    expect(preview?.component).toContain('Preview.vue')
    expect(preview?.children).toBeUndefined()
    expect(preview?.meta).toEqual(expect.objectContaining({ title: '预览' }))
    expect(preview?.meta).not.toHaveProperty('layout')
    expect(dashboard?.component).toBe('Layout')
    expect(dashboard?.children?.[0]?.component).toContain('Dashboard.vue')
  })

  it('allows the root Index page with an explicit root routePath', () => {
    const directory = createRouteRoot({
      'Index.vue': routerConfig({ layout: false, routePath: '/' }),
    })
    const routes = getGeneratedRouteRecords({
      routeRoots: [{ moduleId: 'test', directory }],
    })

    expect(routes).toHaveLength(1)
    expect(routes[0]).toEqual(expect.objectContaining({
      path: '/',
      name: 'test-index',
      props: true,
    }))
    expect(routes[0]?.component).toContain('Index.vue')
    expect(routes[0]?.children).toBeUndefined()
  })

  it('keeps standalone nested routes independent from Layout containers', () => {
    const directory = createRouteRoot({
      'Reports/ReportPreview.vue': routerConfig({ layout: false, param: ':id' }),
    })
    const routes = getGeneratedRouteRecords({
      routeRoots: [{ moduleId: 'test', directory }],
    })

    expect(routes.map((route) => route.path)).toEqual(['/reports/reportpreview/:id'])
    expect(routes[0]?.alias).toEqual(['/reports/report-preview/:id'])
    expect(routes[0]?.component).toContain('ReportPreview.vue')
  })

  it('allows a standalone page beside Layout siblings with different paths', () => {
    const directory = createRouteRoot({
      'Reports/Preview.vue': routerConfig({ layout: false }),
      'Reports/Users.vue': '',
    })
    const routes = getGeneratedRouteRecords({
      routeRoots: [{ moduleId: 'test', directory }],
    })

    expect(new Set(routes.map((route) => route.path))).toEqual(new Set(['/reports', '/reports/preview']))
    expect(routes.find((route) => route.path === '/reports')?.component).toBe('Layout')
    expect(routes.find((route) => route.path === '/reports/preview')?.children).toBeUndefined()
  })

  it('rejects a root Index page without an explicit routePath', () => {
    const directory = createRouteRoot({
      'Index.vue': routerConfig({ layout: false }),
    })

    expect(() => getGeneratedRouteRecords({
      routeRoots: [{ moduleId: 'test', directory }],
    })).toThrow('must resolve to a non-empty path')
  })

  it('rejects Layout and standalone routes that claim the same path', () => {
    const directory = createRouteRoot({
      'Reports/Index.vue': routerConfig({ layout: false }),
      'Reports/Users.vue': '',
    })

    expect(() => getGeneratedRouteRecords({
      routeRoots: [{ moduleId: 'test', directory }],
    })).toThrow('Route path /reports is declared by both test (standalone) and test (layout)')

    const layoutFirstDirectory = createRouteRoot({
      'Reports/Alpha.vue': '',
      'Reports/Preview.vue': routerConfig({ layout: false, routePath: '/reports' }),
    })
    expect(() => getGeneratedRouteRecords({
      routeRoots: [{ moduleId: 'test', directory: layoutFirstDirectory }],
    })).toThrow('Route path /reports is declared by both test (layout) and test (standalone)')
  })

  it('rejects a non-boolean layout value', () => {
    const directory = createRouteRoot({
      'Preview.vue': routerConfig({ layout: 'standalone' }),
    })

    expect(() => getGeneratedRouteRecords({
      routeRoots: [{ moduleId: 'test', directory }],
    })).toThrow('field "layout" must be a boolean')
  })
})
