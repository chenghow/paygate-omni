import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('../views/Login.vue')
    },
    {
      path: '/',
      name: 'Layout',
      component: () => import('../layout/index.vue'),
      redirect: '/dashboard',
      children: [
        {
          path: 'dashboard',
          name: 'Dashboard',
          component: () => import('../views/Dashboard.vue')
        },
        {
          path: 'merchants',
          name: 'Merchants',
          component: () => import('../views/Merchants.vue')
        },
        {
          path: 'channels',
          name: 'Channels',
          component: () => import('../views/Channels.vue')
        },
        {
          path: 'orders',
          name: 'Orders',
          component: () => import('../views/Orders.vue')
        },
        {
          path: 'test',
          name: 'ApiTest',
          component: () => import('../views/ApiTest.vue')
        }
      ]
    }
  ]
})

router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('admin_token')
  if (to.path !== '/login' && !token) {
    next('/login')
  } else {
    next()
  }
})

export default router
