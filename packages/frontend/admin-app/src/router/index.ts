import { createRouter, createWebHistory } from 'vue-router'

import generatedRoutes from 'virtual:router'

/** Manual routes are limited to auth/error pages; package business routes are generated. */
const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/dashboard',
    },
    {
      path: '/signin',
      name: 'signin',
      component: () => import('../views/Signin.vue'),
      meta: { title: '登录' },
    },
    ...generatedRoutes,
    {
      path: '/403',
      name: 'forbidden',
      component: () => import('../views/403.vue'),
      meta: { title: '无权访问' },
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('../views/404.vue'),
      meta: { title: '页面不存在' },
    },
  ],
})

export default router
