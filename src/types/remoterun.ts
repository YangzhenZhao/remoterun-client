export interface CommandSummary {
  alias: string
}

export interface ServerSummary {
  id: string
  alias: string
  host: string
  port: number
  commands: CommandSummary[]
}

export interface RunResponse {
  success: boolean
  exit_code: number
  stdout: string
  stderr: string
  combined_log: string
}

export interface AuthUser {
  id: string
  username: string
}

export interface AuthSessionResponse {
  authenticated: boolean
  user?: AuthUser
  csrfToken?: string
}

export interface ApiError {
  error: string
}
