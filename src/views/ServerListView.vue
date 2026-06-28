<template>
  <section class="detail-layout">
    <section class="panel">
      <div class="panel-header">
        <div>
          <p class="section-label">Servers</p>
          <h2>服务器列表</h2>
        </div>
        <div class="header-actions">
          <RouterLink class="primary-button" to="/servers/new">添加服务器</RouterLink>
          <button class="secondary-button" type="button" @click="loadServers" :disabled="loading">
            {{ loading ? '刷新中...' : '刷新列表' }}
          </button>
        </div>
      </div>

      <p v-if="loading" class="state-text">正在从数据库加载服务器列表...</p>
      <p v-else-if="errorMessage" class="state-text error-text">{{ errorMessage }}</p>
      <p v-else-if="servers.length === 0" class="state-text">目前还没有服务器，点击右上角按钮去新增一台。</p>

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
import { onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'

import { RequestError, fetchServers } from '../services/api'
import { clearAuthState } from '../stores/auth'
import type { ServerSummary } from '../types/remoterun'

const router = useRouter()

const servers = ref<ServerSummary[]>([])
const loading = ref(false)
const errorMessage = ref('')

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

onMounted(loadServers)
</script>
