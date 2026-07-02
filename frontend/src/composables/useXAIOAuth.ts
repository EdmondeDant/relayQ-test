import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import { extractApiErrorMessage, extractI18nErrorMessage } from '@/utils/apiError'

export interface XAITokenInfo {
  access_token?: string
  refresh_token?: string
  client_id?: string
  id_token?: string
  token_type?: string
  expires_in?: number
  expires_at?: number
  email?: string
  name?: string
  xai_user_id?: string
  subscription_plan?: string
  subscription_status?: string
  [key: string]: unknown
}

export function useXAIOAuth() {
  const appStore = useAppStore()
  const { t } = useI18n()
  const endpointPrefix = '/admin/xai'

  const authUrl = ref('')
  const sessionId = ref('')
  const oauthState = ref('')
  const loading = ref(false)
  const error = ref('')

  const resetState = () => {
    authUrl.value = ''
    sessionId.value = ''
    oauthState.value = ''
    loading.value = false
    error.value = ''
  }

  const generateAuthUrl = async (proxyId?: number | null, redirectUri?: string): Promise<boolean> => {
    loading.value = true
    authUrl.value = ''
    sessionId.value = ''
    oauthState.value = ''
    error.value = ''
    try {
      const payload: Record<string, unknown> = {}
      if (proxyId) payload.proxy_id = proxyId
      if (redirectUri) payload.redirect_uri = redirectUri
      const response = await adminAPI.accounts.generateAuthUrl(`${endpointPrefix}/generate-auth-url`, payload)
      authUrl.value = response.auth_url
      sessionId.value = response.session_id
      try {
        const parsed = new URL(response.auth_url)
        oauthState.value = parsed.searchParams.get('state') || ''
      } catch {
        oauthState.value = ''
      }
      return true
    } catch (err: any) {
      error.value = extractApiErrorMessage(err, '生成 Grok 授权链接失败')
      appStore.showError(error.value)
      return false
    } finally {
      loading.value = false
    }
  }

  const exchangeAuthCode = async (
    code: string,
    currentSessionId: string,
    state: string,
    proxyId?: number | null
  ): Promise<XAITokenInfo | null> => {
    if (!code.trim() || !currentSessionId || !state.trim()) {
      error.value = 'Missing auth code, session ID, or state'
      return null
    }
    loading.value = true
    error.value = ''
    try {
      const payload: { session_id: string; code: string; state: string; proxy_id?: number } = {
        session_id: currentSessionId,
        code: code.trim(),
        state: state.trim()
      }
      if (proxyId) payload.proxy_id = proxyId
      return await adminAPI.accounts.exchangeCode(`${endpointPrefix}/exchange-code`, payload) as XAITokenInfo
    } catch (err: any) {
      error.value = extractI18nErrorMessage(err, t, 'admin.accounts.oauth.xai.errors', 'Grok 授权码兑换失败')
      appStore.showError(error.value)
      return null
    } finally {
      loading.value = false
    }
  }

  const validateRefreshToken = async (refreshToken: string, proxyId?: number | null): Promise<XAITokenInfo | null> => {
    if (!refreshToken.trim()) {
      error.value = 'Missing refresh token'
      return null
    }
    loading.value = true
    error.value = ''
    try {
      return await adminAPI.accounts.refreshOpenAIToken(refreshToken.trim(), proxyId, `${endpointPrefix}/refresh-token`) as XAITokenInfo
    } catch (err: any) {
      error.value = extractI18nErrorMessage(err, t, 'admin.accounts.oauth.xai.errors', 'Grok Refresh Token 校验失败')
      appStore.showError(error.value)
      return null
    } finally {
      loading.value = false
    }
  }

  const buildCredentials = (tokenInfo: XAITokenInfo): Record<string, unknown> => {
    const creds: Record<string, unknown> = {
      access_token: tokenInfo.access_token,
      expires_at: tokenInfo.expires_at
    }
    if (tokenInfo.refresh_token) creds.refresh_token = tokenInfo.refresh_token
    if (tokenInfo.id_token) creds.id_token = tokenInfo.id_token
    if (tokenInfo.token_type) creds.token_type = tokenInfo.token_type
    if (tokenInfo.email) creds.email = tokenInfo.email
    if (tokenInfo.name) creds.name = tokenInfo.name
    if (tokenInfo.xai_user_id) creds.xai_user_id = tokenInfo.xai_user_id
    if (tokenInfo.subscription_plan) creds.subscription_plan = tokenInfo.subscription_plan
    if (tokenInfo.subscription_status) creds.subscription_status = tokenInfo.subscription_status
    if (tokenInfo.client_id) creds.client_id = tokenInfo.client_id
    return creds
  }

  const buildExtraInfo = (tokenInfo: XAITokenInfo): Record<string, string> | undefined => {
    const extra: Record<string, string> = {}
    if (tokenInfo.email) extra.email = tokenInfo.email
    if (tokenInfo.name) extra.name = tokenInfo.name
    if (tokenInfo.subscription_plan) extra.subscription_plan = tokenInfo.subscription_plan
    if (tokenInfo.subscription_status) extra.subscription_status = tokenInfo.subscription_status
    return Object.keys(extra).length > 0 ? extra : undefined
  }

  return { authUrl, sessionId, oauthState, loading, error, resetState, generateAuthUrl, exchangeAuthCode, validateRefreshToken, buildCredentials, buildExtraInfo }
}
