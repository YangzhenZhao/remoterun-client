import { createRouter, createWebHistory } from 'vue-router'

import { useAuthStore } from '../stores/auth'
import LoginView from '../views/LoginView.vue'
import ServerCreateView from '../views/ServerCreateView.vue'
import ServerDetailView from '../views/ServerDetailView.vue'
import ServerListView from '../views/ServerListView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/login',
    },
    {
      path: '/login',
      name: 'login',
      component: LoginView,
      meta: {
        guestOnly: true,
      },
    },
    {
      path: '/servers',
      name: 'servers',
      component: ServerListView,
      meta: {
        requiresAuth: true,
      },
    },
    {
      path: '/servers/new',
      name: 'server-create',
      component: ServerCreateView,
      meta: {
        requiresAuth: true,
      },
    },
    {
      path: '/servers/:id',
      name: 'server-detail',
      component: ServerDetailView,
      props: true,
      meta: {
        requiresAuth: true,
      },
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()

  if (!auth.state.initialized) {
    await auth.initializeAuth()
  }

  if (to.meta.requiresAuth && !auth.state.user) {
    return {
      name: 'login',
      query: {
        redirect: to.fullPath,
      },
    }
  }

  if (to.meta.guestOnly && auth.state.user) {
    return {
      name: 'servers',
    }
  }

  return true
})

export default router
