import { readFile, readdir } from 'node:fs/promises'
import type { IncomingMessage, ServerResponse } from 'node:http'
import { basename, extname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import vue from '@vitejs/plugin-vue'
import { defineConfig, type Connect, type Plugin } from 'vite'

interface CommandConfigFile {
  alias: string
  command: string
}

interface ServerConfigFile {
  alias: string
  host: string
  port: number
  password: string
  commands: CommandConfigFile[]
}

interface LoadedServerConfig extends ServerConfigFile {
  id: string
}

interface PublicCommandConfig {
  alias: string
}

interface PublicServerConfig {
  id: string
  alias: string
  host: string
  port: number
  commands: PublicCommandConfig[]
}

interface RunCommandRequest {
  serverId: string
  commandAlias: string
}

const projectRoot = fileURLToPath(new URL('.', import.meta.url))
const dataDirectory = resolve(projectRoot, 'data')

function remoterunApiPlugin(): Plugin {
  const middleware = createApiMiddleware()

  return {
    name: 'remoterun-api',
    configureServer(server) {
      server.middlewares.use(middleware)
    },
    configurePreviewServer(server) {
      server.middlewares.use(middleware)
    },
  }
}

function createApiMiddleware(): Connect.NextHandleFunction {
  return async (
    req: IncomingMessage,
    res: ServerResponse,
    next: Connect.NextFunction,
  ) => {
    const requestUrl = new URL(req.url ?? '/', 'http://localhost')

    if (!requestUrl.pathname.startsWith('/api/')) {
      next()
      return
    }

    try {
      if (req.method === 'GET' && requestUrl.pathname === '/api/servers') {
        const servers = await loadServerConfigs()
        sendJson(res, 200, servers.map(toPublicServerConfig))
        return
      }

      if (req.method === 'GET' && requestUrl.pathname.startsWith('/api/servers/')) {
        const serverId = decodeURIComponent(requestUrl.pathname.replace('/api/servers/', ''))
        const server = (await loadServerConfigs()).find((item) => item.id === serverId)

        if (!server) {
          sendJson(res, 404, { error: `未找到服务器: ${serverId}` })
          return
        }

        sendJson(res, 200, toPublicServerConfig(server))
        return
      }

      if (req.method === 'POST' && requestUrl.pathname === '/api/run') {
        const payload = await readJsonBody(req)

        if (!isRunCommandRequest(payload)) {
          sendJson(res, 400, { error: '请求格式不正确，需要提供 serverId 和 commandAlias' })
          return
        }

        const server = (await loadServerConfigs()).find((item) => item.id === payload.serverId)
        if (!server) {
          sendJson(res, 404, { error: `未找到服务器: ${payload.serverId}` })
          return
        }

        const command = server.commands.find((item) => item.alias === payload.commandAlias)
        if (!command) {
          sendJson(res, 404, { error: `未找到命令: ${payload.commandAlias}` })
          return
        }

        const upstreamResponse = await fetch(`http://${server.host}:${server.port}/run`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            command: command.command,
            password: server.password,
          }),
          signal: AbortSignal.timeout(60_000),
        }).catch((error: unknown) => {
          throw new Error(
            error instanceof Error ? error.message : '连接远端 remoterun-server 失败',
          )
        })

        const upstreamPayload = await upstreamResponse.json().catch(() => null)
        if (upstreamPayload === null) {
          sendJson(res, 502, { error: '远端服务返回了无法解析的响应' })
          return
        }

        sendJson(res, upstreamResponse.status, upstreamPayload)
        return
      }

      sendJson(res, 404, { error: `未知接口: ${requestUrl.pathname}` })
    } catch (error) {
      sendJson(res, 500, {
        error: error instanceof Error ? error.message : '服务内部错误',
      })
    }
  }
}

async function loadServerConfigs(): Promise<LoadedServerConfig[]> {
  const entries = await readdir(dataDirectory, { withFileTypes: true }).catch((error: NodeJS.ErrnoException) => {
    if (error.code === 'ENOENT') {
      return []
    }

    throw error
  })

  const jsonFiles = entries
    .filter((entry) => entry.isFile() && extname(entry.name) === '.json' && entry.name !== 'sample.json')
    .sort((left, right) => left.name.localeCompare(right.name, 'zh-CN'))

  const servers = await Promise.all(
    jsonFiles.map(async (entry) => {
      const filePath = resolve(dataDirectory, entry.name)
      const rawContent = await readFile(filePath, 'utf-8')
      return parseServerConfig(entry.name, rawContent)
    }),
  )

  return servers.sort((left, right) => left.alias.localeCompare(right.alias, 'zh-CN'))
}

function parseServerConfig(filename: string, rawContent: string): LoadedServerConfig {
  const parsed = JSON.parse(rawContent) as unknown

  if (!isServerConfigFile(parsed)) {
    throw new Error(`配置文件格式不正确: ${filename}`)
  }

  return {
    id: basename(filename, '.json'),
    alias: parsed.alias.trim(),
    host: parsed.host.trim(),
    port: parsed.port,
    password: parsed.password,
    commands: parsed.commands.map((command) => ({
      alias: command.alias.trim(),
      command: command.command.trim(),
    })),
  }
}

function isServerConfigFile(value: unknown): value is ServerConfigFile {
  if (!isRecord(value)) {
    return false
  }

  return (
    isNonEmptyString(value.alias) &&
    isNonEmptyString(value.host) &&
    Number.isInteger(value.port) &&
    Number(value.port) > 0 &&
    Number(value.port) <= 65535 &&
    typeof value.password === 'string' &&
    Array.isArray(value.commands) &&
    value.commands.every(
      (command) =>
        isRecord(command) &&
        isNonEmptyString(command.alias) &&
        isNonEmptyString(command.command),
    )
  )
}

function isRunCommandRequest(value: unknown): value is RunCommandRequest {
  return (
    isRecord(value) &&
    isNonEmptyString(value.serverId) &&
    isNonEmptyString(value.commandAlias)
  )
}

function toPublicServerConfig(server: LoadedServerConfig): PublicServerConfig {
  return {
    id: server.id,
    alias: server.alias,
    host: server.host,
    port: server.port,
    commands: server.commands.map((command) => ({
      alias: command.alias,
    })),
  }
}

async function readJsonBody(req: IncomingMessage): Promise<unknown> {
  const chunks: Buffer[] = []

  for await (const chunk of req) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk))
  }

  if (chunks.length === 0) {
    return null
  }

  return JSON.parse(Buffer.concat(chunks).toString('utf-8')) as unknown
}

function sendJson(res: ServerResponse, statusCode: number, payload: unknown): void {
  if (!res.headersSent) {
    res.statusCode = statusCode
    res.setHeader('Content-Type', 'application/json; charset=utf-8')
  }

  res.end(JSON.stringify(payload))
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isNonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.trim().length > 0
}

export default defineConfig({
  plugins: [vue(), remoterunApiPlugin()],
})
