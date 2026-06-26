import type { ApiError, RunResponse, ServerSummary } from '../types/remoterun'

async function requestJson<T>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
  const response = await fetch(input, init)
  const payload = (await response.json().catch(() => null)) as T | ApiError | null

  if (!response.ok) {
    if (payload && typeof payload === 'object' && 'error' in payload) {
      throw new Error(payload.error)
    }

    throw new Error(`请求失败: ${response.status}`)
  }

  if (payload === null) {
    throw new Error('服务返回了空响应')
  }

  return payload as T
}

export function fetchServers(): Promise<ServerSummary[]> {
  return requestJson<ServerSummary[]>('/api/servers')
}

export function fetchServerById(serverId: string): Promise<ServerSummary> {
  return requestJson<ServerSummary>(`/api/servers/${encodeURIComponent(serverId)}`)
}

export function runCommand(serverId: string, commandAlias: string): Promise<RunResponse> {
  return requestJson<RunResponse>('/api/run', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      serverId,
      commandAlias,
    }),
  })
}
