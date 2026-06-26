import { createRouter, createWebHistory } from 'vue-router'

import ServerDetailView from '../views/ServerDetailView.vue'
import ServerListView from '../views/ServerListView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/servers',
    },
    {
      path: '/servers',
      name: 'servers',
      component: ServerListView,
    },
    {
      path: '/servers/:id',
      name: 'server-detail',
      component: ServerDetailView,
      props: true,
    },
  ],
})

export default router
