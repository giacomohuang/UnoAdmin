import { inject } from 'vue'

import { ADMIN_SERVICES_KEY } from '@unoadmin/core'
import type { DemoAppServices, DemoServices } from '../index'

/** Read only the demo business service ports from the application context. */
export function useDemoServices(): DemoServices {
  const services = inject(ADMIN_SERVICES_KEY) as DemoAppServices | undefined
  if (!services) throw new Error('Admin services are not provided')
  return services.demo
}
