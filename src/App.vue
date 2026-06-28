<template>
  <div class="app-shell">
    <header class="app-header">
      <div>
        <p class="eyebrow">RemoteRun</p>
        <h1>服务器命令控制台</h1>
        <p class="subtitle">通过账号密码登录后，从后端读取服务器配置并安全执行预设命令。</p>
      </div>

      <div class="header-actions">
        <RouterLink v-if="auth.state.user" class="header-link" to="/servers">服务器列表</RouterLink>
        <p v-if="auth.state.user" class="user-pill">当前账号：{{ auth.state.user.username }}</p>
        <button
          v-if="auth.state.user"
          class="secondary-button"
          type="button"
          :disabled="auth.state.loading"
          @click="handleLogout"
        >
          {{ auth.state.loading ? '退出中...' : '退出登录' }}
        </button>
        <RouterLink v-else class="header-link" to="/login">去登录</RouterLink>
      </div>
    </header>

    <main class="app-content">
      <section v-if="auth.state.loading && !auth.state.initialized" class="panel">
        <p class="state-text">正在恢复登录状态...</p>
      </section>
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { RouterLink, RouterView } from 'vue-router'

import { useAuthStore } from './stores/auth'

const auth = useAuthStore()
const router = useRouter()

async function handleLogout(): Promise<void> {
  await auth.logout()
  await router.replace('/login')
}
</script>
