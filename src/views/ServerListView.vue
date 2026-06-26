<template>
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

    <p v-if="loading" class="state-text">正在读取 `data/` 目录中的服务器配置...</p>
    <p v-else-if="errorMessage" class="state-text error-text">{{ errorMessage }}</p>
    <p v-else-if="servers.length === 0" class="state-text">
      目前没有可用服务器配置。请在 `data/` 目录中新增 `.json` 文件，`sample.json` 会被自动忽略。
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

        <p class="muted-text">配置来源：`data/{{ server.id }}.json`</p>

        <RouterLink class="primary-button" :to="`/servers/${server.id}`">
          进入服务器
        </RouterLink>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'

import { fetchServers } from '../services/api'
import type { ServerSummary } from '../types/remoterun'

const servers = ref<ServerSummary[]>([])
const loading = ref(false)
const errorMessage = ref('')

async function loadServers(): Promise<void> {
  loading.value = true
  errorMessage.value = ''

  try {
    servers.value = await fetchServers()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '读取服务器配置失败'
  } finally {
    loading.value = false
  }
}

onMounted(loadServers)
</script>
