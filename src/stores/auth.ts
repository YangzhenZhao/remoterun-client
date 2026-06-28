import { reactive, readonly } from 'vue'

import {
  AUTH_UNAUTHORIZED_EVENT,
  fetchSession,
  login as loginRequest,
  register as registerRequest,
  logout as logoutRequest,
} from '../services/api'
import type { AuthSessionResponse, AuthUser } from '../types/remoterun'

interface AuthState {
  csrfToken: string
  initialized: boolean
  loading: boolean
  user: AuthUser | null
}

const state = reactive<AuthState>({
  csrfToken: '',
  initialized: false,
  loading: false,
  user: null,
})

let initializePromise: Promise<void> | null = null

function applySession(payload: AuthSessionResponse): void {
  state.user = payload.authenticated ? (payload.user ?? null) : null
  state.csrfToken = payload.authenticated ? (payload.csrfToken ?? '') : ''
  state.initialized = true
}

export function clearAuthState(): void {
  state.user = null
  state.csrfToken = ''
  state.initialized = true
}

export async function initializeAuth(force = false): Promise<void> {
  if (state.loading && initializePromise) {
    return initializePromise
  }

  if (!force && state.initialized) {
    return
  }

  state.loading = true
  initializePromise = (async () => {
    try {
      const payload = await fetchSession()
      applySession(payload)
    } catch {
      clearAuthState()
    } finally {
      state.loading = false
      initializePromise = null
    }
  })()

  return initializePromise
}

export async function login(username: string, password: string): Promise<void> {
  state.loading = true
  try {
    const payload = await loginRequest(username, password)
    applySession(payload)
  } finally {
    state.loading = false
  }
}

export async function register(username: string, password: string): Promise<void> {
  state.loading = true
  try {
    const payload = await registerRequest(username, password)
    applySession(payload)
  } finally {
    state.loading = false
  }
}

export async function logout(): Promise<void> {
  state.loading = true
  try {
    if (state.csrfToken) {
      await logoutRequest(state.csrfToken)
    }
  } finally {
    clearAuthState()
    state.loading = false
  }
}

if (typeof window !== 'undefined') {
  window.addEventListener(AUTH_UNAUTHORIZED_EVENT, () => {
    clearAuthState()
  })
}

export function useAuthStore() {
  return {
    state: readonly(state),
    clearAuthState,
    initializeAuth,
    login,
    register,
    logout,
  }
}
