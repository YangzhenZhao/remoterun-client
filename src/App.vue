<template>
  <div class="app-shell">
    <header class="app-header">
      <div>
        <p class="eyebrow">RemoteRun</p>
        <h1>服务器命令控制台</h1>
        <p class="subtitle">从本地 `data/` 目录读取服务器配置，安全地执行预设命令。</p>
      </div>

      <RouterLink class="header-link" to="/servers">服务器列表</RouterLink>
    </header>

    <main class="app-content">
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
</script>
