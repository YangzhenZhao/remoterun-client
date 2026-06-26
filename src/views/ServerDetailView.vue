<template>
  <section class="detail-layout">
    <div class="detail-header">
      <RouterLink class="secondary-button" to="/servers">返回列表</RouterLink>

      <button class="secondary-button" type="button" @click="loadServer" :disabled="loadingServer">
        {{ loadingServer ? '刷新中...' : '刷新配置' }}
      </button>
    </div>

    <p v-if="loadingServer" class="state-text">正在加载服务器配置...</p>
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
          当前页面只展示命令别名；密码保存在本地 `data/{{ server.id }}.json`，不会发送到浏览器。
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
import { onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

import { fetchServerById, runCommand } from '../services/api'
import type { RunResponse, ServerSummary } from '../types/remoterun'

const route = useRoute()

const server = ref<ServerSummary | null>(null)
const loadingServer = ref(false)
const errorMessage = ref('')
const runningAlias = ref('')
const lastCommandAlias = ref('')
const runErrorMessage = ref('')
const runResult = ref<RunResponse | null>(null)

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
    server.value = null
    errorMessage.value = error instanceof Error ? error.message : '读取服务器配置失败'
  } finally {
    loadingServer.value = false
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
    runResult.value = await runCommand(server.value.id, commandAlias)
  } catch (error) {
    runErrorMessage.value = error instanceof Error ? error.message : '执行命令失败'
  } finally {
    runningAlias.value = ''
  }
}

watch(
  () => route.params.id,
  () => {
    runErrorMessage.value = ''
    runResult.value = null
    lastCommandAlias.value = ''
    void loadServer()
  },
)

onMounted(loadServer)
</script>
