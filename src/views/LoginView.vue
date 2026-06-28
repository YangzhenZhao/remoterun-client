<template>
  <section class="auth-layout">
    <article class="panel auth-panel">
      <div>
        <p class="section-label">Authentication</p>
        <h2>{{ isRegisterMode ? '注册账号' : '账号登录' }}</h2>
        <p class="subtitle">
          {{ isRegisterMode ? '注册成功后会自动登录，你就可以开始使用系统。' : '登录后才能查看服务器配置并执行预设命令。' }}
        </p>
      </div>

      <div class="auth-mode-switch" role="tablist" aria-label="认证方式切换">
        <button
          class="secondary-button auth-mode-button"
          type="button"
          :class="{ 'auth-mode-button-active': !isRegisterMode }"
          :disabled="submitting"
          @click="setMode('login')"
        >
          登录
        </button>
        <button
          class="secondary-button auth-mode-button"
          type="button"
          :class="{ 'auth-mode-button-active': isRegisterMode }"
          :disabled="submitting"
          @click="setMode('register')"
        >
          注册
        </button>
      </div>

      <form class="auth-form" @submit.prevent="submitAuth">
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

        <label v-if="isRegisterMode" class="field-group">
          <span>确认密码</span>
          <input
            v-model="confirmPassword"
            class="text-input"
            type="password"
            name="confirmPassword"
            autocomplete="new-password"
            required
          />
        </label>

        <p v-if="isRegisterMode" class="state-text">密码至少 8 位，用户名支持字母、数字、`_`、`-`、`.`。</p>
        <p v-if="successMessage" class="state-text success-text">{{ successMessage }}</p>
        <p v-if="errorMessage" class="state-text error-text">{{ errorMessage }}</p>

        <button class="primary-button auth-submit" type="submit" :disabled="submitting">
          {{ submitLabel }}
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

type AuthMode = 'login' | 'register'

const mode = ref<AuthMode>('login')
const username = ref('')
const password = ref('')
const confirmPassword = ref('')
const errorMessage = ref('')
const successMessage = ref('')
const submitting = computed(() => auth.state.loading)
const isRegisterMode = computed(() => mode.value === 'register')
const submitLabel = computed(() => {
  if (submitting.value) {
    return isRegisterMode.value ? '注册中...' : '登录中...'
  }

  return isRegisterMode.value ? '注册并登录' : '登录'
})

function setMode(nextMode: AuthMode): void {
  mode.value = nextMode
  errorMessage.value = ''
  successMessage.value = ''
  confirmPassword.value = ''
}

async function submitAuth(): Promise<void> {
  errorMessage.value = ''
  successMessage.value = ''

  if (isRegisterMode.value && password.value !== confirmPassword.value) {
    errorMessage.value = '两次输入的密码不一致'
    return
  }

  try {
    if (isRegisterMode.value) {
      await auth.register(username.value, password.value)
      successMessage.value = '注册成功，正在进入系统'
    } else {
      await auth.login(username.value, password.value)
    }

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

    errorMessage.value = isRegisterMode.value ? '注册失败，请稍后重试' : '登录失败，请稍后重试'
  }
}
</script>
