<template>
  <section class="auth-layout">
    <article class="panel auth-panel">
      <div>
        <p class="section-label">Authentication</p>
        <h2>账号登录</h2>
        <p class="subtitle">登录后才能查看服务器配置并执行预设命令。</p>
      </div>

      <form class="auth-form" @submit.prevent="submitLogin">
        <label class="field-group">
          <span>用户名</span>
          <input
            v-model.trim="username"
            class="text-input"
            type="text"
            name="username"
            autocomplete="username"
            maxlength="64"
            required
          />
        </label>

        <label class="field-group">
          <span>密码</span>
          <input
            v-model="password"
            class="text-input"
            type="password"
            name="password"
            autocomplete="current-password"
            required
          />
        </label>

        <p v-if="errorMessage" class="state-text error-text">{{ errorMessage }}</p>

        <button class="primary-button auth-submit" type="submit" :disabled="submitting">
          {{ submitting ? '登录中...' : '登录' }}
        </button>
      </form>
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { RequestError } from '../services/api'
import { useAuthStore } from '../stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const errorMessage = ref('')
const submitting = computed(() => auth.state.loading)

async function submitLogin(): Promise<void> {
  errorMessage.value = ''

  try {
    await auth.login(username.value, password.value)
    const redirectPath =
      typeof route.query.redirect === 'string' && route.query.redirect.startsWith('/')
        ? route.query.redirect
        : '/servers'

    await router.replace(redirectPath)
  } catch (error) {
    if (error instanceof RequestError) {
      errorMessage.value = error.message
      return
    }

    errorMessage.value = '登录失败，请稍后重试'
  }
}
</script>
