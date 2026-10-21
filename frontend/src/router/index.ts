import { createRouter, createWebHistory } from 'vue-router'

import HomeView from '../views/HomeView.vue'
import AccountView from '../views/AccountView.vue'
import RegisterView from '../views/RegisterView.vue'
import VideoView from '../views/VideoView.vue'
import VideoDetailView from '../views/VideoDetailView.vue'
import { useAuthStore } from '../stores/auth'
import RelationView from '../views/RelationView.vue'
import SettingsView from '@/views/SettingsView.vue'
import UserProfileView from '@/views/UserProfileView.vue'
import ChangePasswordView from '@/views/ChangePasswordView.vue'

const routes = [
    { path: '/', name: 'home', component: HomeView },
    { path: '/account', name: 'account', component: AccountView },
    { path: '/account/register', name: 'account-register', component: RegisterView },
    { path: '/video', name: 'video', component: VideoView, meta: { requiresAuth: true } },
    { path: '/video/:id', name: 'video-detail', component: VideoDetailView, props: true },
    { path: '/account/change-password', name: 'account-change-password', component: ChangePasswordView },
    { path: '/settings', name: 'settings', component: SettingsView, meta: { requiresAuth: true } },
    { path: '/u/:id', name: 'user-profile', component: UserProfileView },
    { path: '/relations/:id', name: 'relations', component: RelationView, props: true, meta: { requiresAuth: true } },
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