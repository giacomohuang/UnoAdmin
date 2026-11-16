import { inject } from 'vue'

import { ADMIN_SERVICES_KEY, type AppServices, type BaseServices } from '@unoadmin/core'

/** Read only the base service ports owned by admin-base pages. */
export function useAdminBaseServices(): BaseServices {
  const services = inject(ADMIN_SERVICES_KEY) as AppServices | undefined
  if (!services) throw new Error('Admin services are not provided')
  return services.base
}
