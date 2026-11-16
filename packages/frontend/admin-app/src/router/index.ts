import generatedRoutes from 'virtual:router'
import { createRouter, createWebHistory } from 'vue-router'

/** Manual routes are limited to auth/error pages; package business routes are generated. */
const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    ...generatedRoutes,
    {
      path: '/',
      redirect: '/dashboard'
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('../views/404.vue'),
      meta: { title: '页面不存在' }
    }
  ]
})

export default router
