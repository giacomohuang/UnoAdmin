import type { AdminModule, AppServices, ListQuery, PageResult, PermissionDefinition } from '@unoadmin/core'

export interface DemoOverview {
  activeProjects: number
  pendingTasks: number
  monthlyGrowth: number
}

export interface DemoRecord {
  id: string
  name: string
  owner: string
  status: 'active' | 'paused'
  updatedAt: string
}

/** Service contract owned by this business package rather than by core. */
export interface DemoServices {
  getOverview(): Promise<DemoOverview>
  listRecords(query: ListQuery): Promise<PageResult<DemoRecord>>
}

export type DemoAppServices = AppServices & { demo: DemoServices }

const demoPermissions: PermissionDefinition[] = [
  { code: 'demo:read', label: '查看示例业务' },
  { code: 'demo:write', label: '编辑示例业务' },
]

/** Create the sample domain module used to prove package-level route generation. */
export function createDemoModule(): AdminModule {
  return {
    id: 'demo',
    routeNamespace: 'demo',
    navigation: [
      {
        id: 'demo',
        label: '示例业务',
        permission: 'demo:read',
        children: [
          { id: 'demo-overview', label: '业务概览', to: '/demo/overview', permission: 'demo:read' },
          { id: 'demo-records', label: '项目记录', to: '/demo/records', permission: 'demo:read' },
        ],
      },
    ],
    permissions: demoPermissions,
  }
}
