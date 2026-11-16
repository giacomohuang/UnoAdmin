import { createAdminBaseModule } from '@unoadmin/admin-base'

/** The composition root decides which reusable modules are enabled in this app. */
export const modules = [
  createAdminBaseModule(),
]
