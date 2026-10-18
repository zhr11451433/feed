import { createRouter, createWebHistory } from 'vue-router'

import HomeView from '../views/HomeView.vue'
import AccountView from '../views/AccountView.vue'
import RegisterView from '../views/RegisterView.vue'
import VideoView from '../views/VideoView.vue'
import VideoDetailView from '../views/VideoDetailView.vue'
import { useAuthStore } from '../stores/auth'

const routes = [
    { path: '/', name: 'home', component: HomeView },
    { path: '/account', name: 'account', component: AccountView },
    { path: '/account/register', name: 'account-register', component: RegisterView },
    { path: '/video', name: 'video', component: VideoView, meta: { requiresAuth: true } },
    { path: '/video/:id', name: 'video-detail', component: VideoDetailView, props: true },
  ]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, _from, next) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.isLoggedIn) {
    next({ path: '/account', query: { redirect: to.fullPath } })
    return
  }
  next()
})

export default router