import type {
  ApiError,
  AuthSessionResponse,
  RunResponse,
  ServerSummary,
} from '../types/remoterun'

export const AUTH_UNAUTHORIZED_EVENT = 'auth:unauthorized'

export class RequestError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'RequestError'
    this.status = status
  }
}

interface RequestJsonOptions {
  csrfToken?: string
}

async function requestJson<T>(
  input: RequestInfo | URL,
  init?: RequestInit,
  options: RequestJsonOptions = {},
): Promise<T> {
  const headers = new Headers(init?.headers)
  headers.set('Accept', 'application/json')

  if (options.csrfToken) {
    headers.set('X-CSRF-Token', options.csrfToken)
  }

  const response = await fetch(input, {
    ...init,
    credentials: 'include',
    headers,
  })

  const payload = (await response.json().catch(() => null)) as T | ApiError | null

  if (!response.ok) {
    if (response.status === 401 && typeof window !== 'undefined') {
      window.dispatchEvent(new Event(AUTH_UNAUTHORIZED_EVENT))
    }

    if (payload && typeof payload === 'object' && 'error' in payload) {
      throw new RequestError(payload.error, response.status)
    }

    throw new RequestError(`请求失败: ${response.status}`, response.status)
  }

  if (payload === null) {
    throw new RequestError('服务返回了空响应', response.status)
  }

  return payload as T
}

export function fetchSession(): Promise<AuthSessionResponse> {
  return requestJson<AuthSessionResponse>('/api/auth/session')
}

export function login(username: string, password: string): Promise<AuthSessionResponse> {
  return requestJson<AuthSessionResponse>('/api/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      username,
      password,
    }),
  })
}

export function logout(csrfToken: string): Promise<{ ok: boolean }> {
  return requestJson<{ ok: boolean }>(
    '/api/auth/logout',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    },
    { csrfToken },
  )
}

export function fetchServers(): Promise<ServerSummary[]> {
  return requestJson<ServerSummary[]>('/api/servers')
}

export function fetchServerById(serverId: string): Promise<ServerSummary> {
  return requestJson<ServerSummary>(`/api/servers/${encodeURIComponent(serverId)}`)
}

export function runCommand(
  serverId: string,
  commandAlias: string,
  csrfToken: string,
): Promise<RunResponse> {
  return requestJson<RunResponse>(
    '/api/run',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        serverId,
        commandAlias,
      }),
    },
    { csrfToken },
  )
}
