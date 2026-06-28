<template>
  <section class="detail-layout">
    <div class="detail-header">
      <RouterLink class="secondary-button" to="/servers">返回列表</RouterLink>
    </div>

    <section class="panel">
      <div class="panel-header">
        <div>
          <p class="section-label">Create</p>
          <h2>新增服务器</h2>
        </div>
      </div>

      <p class="muted-text">服务器保存成功后会返回主页，命令会一起写入数据库。</p>

      <form class="server-form" @submit.prevent="submitCreateServer">
        <div class="form-grid">
          <label class="field-group">
            <span>服务器名称</span>
            <input v-model.trim="createForm.alias" class="text-input" type="text" maxlength="120" placeholder="例如：生产环境" />
          </label>

          <label class="field-group">
            <span>主机地址</span>
            <input v-model.trim="createForm.host" class="text-input" type="text" maxlength="255" placeholder="例如：10.0.0.8" />
          </label>

          <label class="field-group">
            <span>端口</span>
            <input v-model.number="createForm.port" class="text-input" type="number" min="1" max="65535" placeholder="8080" />
          </label>

          <label class="field-group">
            <span>远端密码</span>
            <input v-model.trim="createForm.password" class="text-input" type="password" maxlength="255" placeholder="用于转发执行命令" />
          </label>
        </div>

        <div class="panel-header">
          <div>
            <p class="section-label">Commands</p>
            <h3>初始命令</h3>
          </div>
          <button class="secondary-button" type="button" @click="addCommand" :disabled="submitting">
            添加命令
          </button>
        </div>

        <div class="command-editor-list">
          <div v-for="(command, index) in createForm.commands" :key="index" class="command-editor">
            <label class="field-group">
              <span>命令别名</span>
              <input
                v-model.trim="command.alias"
                class="text-input"
                type="text"
                maxlength="120"
                :placeholder="`例如：命令 ${index + 1}`"
              />
            </label>

            <label class="field-group command-editor-main">
              <span>命令内容</span>
              <textarea
                v-model.trim="command.command"
                class="text-input text-area-input"
                rows="3"
                placeholder="例如：systemctl restart my-app"
              />
            </label>

            <button
              class="secondary-button command-remove-button"
              type="button"
              @click="removeCommand(index)"
              :disabled="submitting"
            >
              删除
            </button>
          </div>
        </div>

        <p v-if="errorMessage" class="state-text error-text">{{ errorMessage }}</p>

        <div class="form-actions">
          <button class="primary-button" type="submit" :disabled="submitting">
            {{ submitting ? '保存中...' : '保存并返回主页' }}
          </button>
          <button class="secondary-button" type="button" @click="resetCreateForm" :disabled="submitting">
            重置表单
          </button>
        </div>
      </form>
    </section>
  </section>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'

import { RequestError, createServer } from '../services/api'
import { clearAuthState, useAuthStore } from '../stores/auth'
import type { CommandInput, CreateServerInput } from '../types/remoterun'

const router = useRouter()
const auth = useAuthStore()

const submitting = ref(false)
const errorMessage = ref('')

const createForm = reactive<CreateServerInput>({
  alias: '',
  host: '',
  port: 8080,
  password: '',
  commands: [createEmptyCommand()],
})

function createEmptyCommand(): CommandInput {
  return {
    alias: '',
    command: '',
  }
}

function resetCreateForm(): void {
  createForm.alias = ''
  createForm.host = ''
  createForm.port = 8080
  createForm.password = ''
  createForm.commands = [createEmptyCommand()]
  errorMessage.value = ''
}

function addCommand(): void {
  createForm.commands.push(createEmptyCommand())
}

function removeCommand(index: number): void {
  createForm.commands.splice(index, 1)
  if (createForm.commands.length === 0) {
    createForm.commands.push(createEmptyCommand())
  }
}

function buildCreatePayload(): CreateServerInput {
  return {
    alias: createForm.alias.trim(),
    host: createForm.host.trim(),
    port: Number.isFinite(createForm.port) ? createForm.port : 0,
    password: createForm.password.trim(),
    commands: createForm.commands.map((command) => ({
      alias: command.alias.trim(),
      command: command.command.trim(),
    })),
  }
}

async function handleUnauthorized(): Promise<void> {
  clearAuthState()
  await router.replace('/login')
}

async function submitCreateServer(): Promise<void> {
  submitting.value = true
  errorMessage.value = ''

  try {
    await createServer(buildCreatePayload(), auth.state.csrfToken)
    await router.push('/servers')
  } catch (error) {
    if (error instanceof RequestError && error.status === 401) {
      await handleUnauthorized()
      return
    }

    errorMessage.value = error instanceof Error ? error.message : '保存服务器失败'
  } finally {
    submitting.value = false
  }
}
</script>
