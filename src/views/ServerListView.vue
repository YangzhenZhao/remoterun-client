<template>
  <section class="detail-layout">
    <section class="panel">
      <div class="panel-header">
        <div>
          <p class="section-label">Create</p>
          <h2>新增服务器</h2>
        </div>
      </div>

      <p class="muted-text">服务器和命令会直接保存到数据库，新增后立即出现在列表中。</p>

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
            <h3>命令配置</h3>
          </div>
          <button class="secondary-button" type="button" @click="addCommand">
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

        <p v-if="createSuccessMessage" class="state-text success-text">{{ createSuccessMessage }}</p>
        <p v-if="createErrorMessage" class="state-text error-text">{{ createErrorMessage }}</p>

        <div class="form-actions">
          <button class="primary-button" type="submit" :disabled="submitting">
            {{ submitting ? '保存中...' : '保存到数据库' }}
          </button>
          <button class="secondary-button" type="button" @click="resetCreateForm" :disabled="submitting">
            重置表单
          </button>
        </div>
      </form>
    </section>

    <section class="panel">
      <div class="panel-header">
        <div>
          <p class="section-label">Servers</p>
          <h2>服务器列表</h2>
        </div>
        <button class="secondary-button" type="button" @click="loadServers" :disabled="loading">
          {{ loading ? '刷新中...' : '刷新列表' }}
        </button>
      </div>

      <p v-if="loading" class="state-text">正在从数据库加载服务器列表...</p>
      <p v-else-if="errorMessage" class="state-text error-text">{{ errorMessage }}</p>
      <p v-else-if="servers.length === 0" class="state-text">
        目前还没有服务器。可以先在上方表单手动新增一条。
      </p>

      <div v-else class="server-grid">
        <article v-for="server in servers" :key="server.id" class="server-card">
          <div class="server-card-header">
            <div>
              <h3>{{ server.alias }}</h3>
              <p>{{ server.host }}:{{ server.port }}</p>
            </div>
            <span class="command-badge">{{ server.commands.length }} 条命令</span>
          </div>

          <p class="muted-text">数据来源：数据库记录 #{{ server.id }}</p>

          <RouterLink class="primary-button" :to="`/servers/${server.id}`">
            进入服务器
          </RouterLink>
        </article>
      </div>
    </section>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'

import { RequestError, createServer, fetchServers } from '../services/api'
import { clearAuthState, useAuthStore } from '../stores/auth'
import type { CommandInput, CreateServerInput, ServerSummary } from '../types/remoterun'

const router = useRouter()
const auth = useAuthStore()

const servers = ref<ServerSummary[]>([])
const loading = ref(false)
const submitting = ref(false)
const errorMessage = ref('')
const createErrorMessage = ref('')
const createSuccessMessage = ref('')

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
  createErrorMessage.value = ''
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

async function loadServers(): Promise<void> {
  loading.value = true
  errorMessage.value = ''

  try {
    servers.value = await fetchServers()
  } catch (error) {
    if (error instanceof RequestError && error.status === 401) {
      await handleUnauthorized()
      return
    }

    errorMessage.value = error instanceof Error ? error.message : '读取服务器列表失败'
  } finally {
    loading.value = false
  }
}

async function submitCreateServer(): Promise<void> {
  submitting.value = true
  createErrorMessage.value = ''
  createSuccessMessage.value = ''

  try {
    const server = await createServer(buildCreatePayload(), auth.state.csrfToken)
    createSuccessMessage.value = `已新增服务器：${server.alias}`
    resetCreateForm()
    await loadServers()
  } catch (error) {
    if (error instanceof RequestError && error.status === 401) {
      await handleUnauthorized()
      return
    }

    createErrorMessage.value = error instanceof Error ? error.message : '保存服务器失败'
  } finally {
    submitting.value = false
  }
}

onMounted(loadServers)
</script>
