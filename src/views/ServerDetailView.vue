<template>
  <section class="detail-layout">
    <div class="detail-header">
      <RouterLink class="secondary-button" to="/servers">返回列表</RouterLink>

      <button class="secondary-button" type="button" @click="loadServer" :disabled="loadingServer">
        {{ loadingServer ? '刷新中...' : '刷新详情' }}
      </button>
    </div>

    <p v-if="loadingServer" class="state-text">正在加载服务器详情...</p>
    <p v-else-if="errorMessage" class="state-text error-text">{{ errorMessage }}</p>

    <template v-else-if="server">
      <section class="panel">
        <div class="panel-header">
          <div>
            <p class="section-label">Server</p>
            <h2>{{ server.alias }}</h2>
          </div>
          <span class="command-badge">{{ server.host }}:{{ server.port }}</span>
        </div>

        <p class="muted-text">
          当前页面只展示命令别名；密码和真实命令保存在后端数据库，不会发送到浏览器。
        </p>
      </section>

      <section class="panel">
        <div class="panel-header">
          <div>
            <p class="section-label">Commands</p>
            <h2>预设命令</h2>
          </div>
        </div>

        <p v-if="server.commands.length === 0" class="state-text">该服务器没有配置任何命令。</p>

        <div v-else class="command-list">
          <button
            v-for="command in server.commands"
            :key="command.alias"
            class="command-item"
            type="button"
            :disabled="runningAlias === command.alias"
            @click="executeCommand(command.alias)"
          >
            <span>{{ command.alias }}</span>
            <span>{{ runningAlias === command.alias ? '执行中...' : '运行命令' }}</span>
          </button>
        </div>
      </section>

      <section class="panel">
        <div class="panel-header">
          <div>
            <p class="section-label">Create</p>
            <h2>添加命令</h2>
          </div>
        </div>

        <p class="muted-text">点击按钮后填写表单，保存成功后会刷新页面并显示最新命令。</p>

        <button
          v-if="!showCreateCommandForm"
          class="primary-button"
          type="button"
          @click="showCreateCommandForm = true"
        >
          添加命令
        </button>

        <form v-else class="server-form" @submit.prevent="submitCreateCommand">
          <div class="form-grid">
            <label class="field-group">
              <span>命令别名</span>
              <input
                v-model.trim="createCommandForm.alias"
                class="text-input"
                type="text"
                maxlength="120"
                placeholder="例如：重启服务"
              />
            </label>

            <label class="field-group">
              <span>命令内容</span>
              <textarea
                v-model.trim="createCommandForm.command"
                class="text-input text-area-input"
                rows="3"
                placeholder="例如：systemctl restart my-app"
              />
            </label>
          </div>

          <p v-if="createCommandErrorMessage" class="state-text error-text">{{ createCommandErrorMessage }}</p>

          <div class="form-actions">
            <button class="primary-button" type="submit" :disabled="creatingCommand">
              {{ creatingCommand ? '保存中...' : '保存命令' }}
            </button>
            <button
              class="secondary-button"
              type="button"
              @click="cancelCreateCommand"
              :disabled="creatingCommand"
            >
              取消
            </button>
          </div>
        </form>
      </section>

      <section class="panel">
        <div class="panel-header">
          <div>
            <p class="section-label">Result</p>
            <h2>执行结果</h2>
          </div>
          <span v-if="lastCommandAlias" class="muted-text">最近命令：{{ lastCommandAlias }}</span>
        </div>

        <p v-if="runErrorMessage" class="state-text error-text">{{ runErrorMessage }}</p>
        <p v-else-if="!runResult" class="state-text">点击任意命令后，这里会显示执行结果。</p>

        <template v-else>
          <div class="result-summary">
            <span :class="runResult.success ? 'result-success' : 'result-failure'">
              {{ runResult.success ? '执行成功' : '执行失败' }}
            </span>
            <span class="muted-text">退出码：{{ runResult.exit_code }}</span>
          </div>

          <pre class="result-log">{{ runResult.combined_log || '(无输出)' }}</pre>
        </template>
      </section>
    </template>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'

import { RequestError, createCommand, fetchServerById, runCommand } from '../services/api'
import { clearAuthState, useAuthStore } from '../stores/auth'
import type { CommandInput, RunResponse, ServerSummary } from '../types/remoterun'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const server = ref<ServerSummary | null>(null)
const loadingServer = ref(false)
const errorMessage = ref('')
const runningAlias = ref('')
const lastCommandAlias = ref('')
const runErrorMessage = ref('')
const runResult = ref<RunResponse | null>(null)
const creatingCommand = ref(false)
const createCommandErrorMessage = ref('')
const showCreateCommandForm = ref(false)

const createCommandForm = reactive<CommandInput>({
  alias: '',
  command: '',
})

function resetCreateCommandForm(): void {
  createCommandForm.alias = ''
  createCommandForm.command = ''
  createCommandErrorMessage.value = ''
}

function cancelCreateCommand(): void {
  resetCreateCommandForm()
  showCreateCommandForm.value = false
}

async function handleUnauthorized(): Promise<void> {
  clearAuthState()
  await router.replace('/login')
}

async function loadServer(): Promise<void> {
  const serverId = String(route.params.id ?? '')

  if (!serverId) {
    errorMessage.value = '缺少服务器 ID'
    server.value = null
    return
  }

  loadingServer.value = true
  errorMessage.value = ''

  try {
    server.value = await fetchServerById(serverId)
  } catch (error) {
    if (error instanceof RequestError && error.status === 401) {
      await handleUnauthorized()
      return
    }

    server.value = null
    errorMessage.value = error instanceof Error ? error.message : '读取服务器详情失败'
  } finally {
    loadingServer.value = false
  }
}

async function submitCreateCommand(): Promise<void> {
  if (!server.value) {
    return
  }

  creatingCommand.value = true
  createCommandErrorMessage.value = ''

  try {
    await createCommand(
      server.value.id,
      {
        alias: createCommandForm.alias.trim(),
        command: createCommandForm.command.trim(),
      },
      auth.state.csrfToken,
    )

    if (typeof window !== 'undefined') {
      window.location.reload()
    }
  } catch (error) {
    if (error instanceof RequestError && error.status === 401) {
      await handleUnauthorized()
      return
    }

    createCommandErrorMessage.value = error instanceof Error ? error.message : '保存命令失败'
  } finally {
    creatingCommand.value = false
  }
}

async function executeCommand(commandAlias: string): Promise<void> {
  if (!server.value) {
    return
  }

  runningAlias.value = commandAlias
  lastCommandAlias.value = commandAlias
  runErrorMessage.value = ''
  runResult.value = null

  try {
    runResult.value = await runCommand(server.value.id, commandAlias, auth.state.csrfToken)
  } catch (error) {
    if (error instanceof RequestError && error.status === 401) {
      await handleUnauthorized()
      return
    }

    runErrorMessage.value = error instanceof Error ? error.message : '执行命令失败'
  } finally {
    runningAlias.value = ''
  }
}

watch(
  () => route.params.id,
  () => {
    cancelCreateCommand()
    runErrorMessage.value = ''
    runResult.value = null
    lastCommandAlias.value = ''
    void loadServer()
  },
)

onMounted(loadServer)
</script>
