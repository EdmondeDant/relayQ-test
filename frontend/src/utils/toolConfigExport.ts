export type ConfigExportToolId =
  | 'claude-code'
  | 'opencode'
  | 'cursor'
  | 'curl'
  | 'python-sdk'
  | 'anthropic-sdk'

export type CcSwitchImportAppId =
  | 'codex'
  | 'claude'
  | 'gemini'
  | 'opencode'
  | 'openclaw'
  | 'hermes'

export interface ToolConfigContext {
  providerName: string
  homepage: string
  routeBaseUrl: string
  apiKey: string
  modelName: string
}

export interface ToolServiceUrls {
  root: string
  openai: string
  anthropic: string
  gemini: string
}

export function normalizeRouteRoot(url: string): string {
  const trimmed = url.trim().replace(/\/+$/, '')
  if (!trimmed) return ''
  return trimmed.replace(/\/(v1|v1beta)$/i, '')
}

export function buildServiceUrls(routeBaseUrl: string): ToolServiceUrls {
  const root = normalizeRouteRoot(routeBaseUrl)
  return {
    root,
    openai: `${root}/v1`,
    anthropic: root,
    gemini: `${root}/v1beta`,
  }
}

export function detectModelFamily(modelName: string): 'anthropic' | 'openai' | 'gemini' {
  const normalized = modelName.trim().toLowerCase()
  if (normalized.includes('claude')) return 'anthropic'
  if (normalized.includes('gemini')) return 'gemini'
  return 'openai'
}

function sanitizeProviderSlug(providerName: string): string {
  return (
    providerName
      .toLowerCase()
      .replace(/[^a-z0-9_]/g, '_')
      .replace(/^_+|_+$/g, '') || 'relayq'
  )
}

function encodeUtf8Base64(input: string): string {
  const bytes = new TextEncoder().encode(input)
  let binary = ''
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte)
  })
  return btoa(binary)
}

export function resolveToolBaseUrl(toolId: ConfigExportToolId, routeBaseUrl: string): string {
  const urls = buildServiceUrls(routeBaseUrl)
  switch (toolId) {
    case 'claude-code':
    case 'anthropic-sdk':
      return urls.anthropic
    default:
      return urls.openai
  }
}

function resolveOpenCodeProvider(modelName: string, routeBaseUrl: string) {
  const family = detectModelFamily(modelName)
  const urls = buildServiceUrls(routeBaseUrl)

  if (family === 'anthropic') {
    return {
      providerId: 'anthropic',
      baseUrl: urls.anthropic,
      modelRef: `anthropic/${modelName}`,
    }
  }

  return {
    providerId: 'openai',
    baseUrl: urls.openai,
    modelRef: `openai/${modelName}`,
  }
}

function buildCcSwitchProviderConfig(appId: CcSwitchImportAppId, context: ToolConfigContext) {
  const urls = buildServiceUrls(context.routeBaseUrl)

  switch (appId) {
    case 'claude':
      return {
        env: {
          ANTHROPIC_AUTH_TOKEN: context.apiKey,
          ANTHROPIC_BASE_URL: urls.anthropic,
          ANTHROPIC_MODEL: context.modelName,
          ANTHROPIC_DEFAULT_HAIKU_MODEL: context.modelName,
          ANTHROPIC_DEFAULT_SONNET_MODEL: context.modelName,
          ANTHROPIC_DEFAULT_OPUS_MODEL: context.modelName,
        },
      }
    case 'codex': {
      const providerSlug = sanitizeProviderSlug(context.providerName)
      return {
        auth: { OPENAI_API_KEY: context.apiKey },
        config: `model_provider = "${providerSlug}"
model = "${context.modelName}"
model_reasoning_effort = "high"
disable_response_storage = true
[model_providers.${providerSlug}]
name = "${providerSlug}"
base_url = "${urls.openai}"
wire_api = "responses"
requires_openai_auth = true
`,
      }
    }
    case 'gemini':
      return {
        GEMINI_API_KEY: context.apiKey,
        GOOGLE_GEMINI_BASE_URL: urls.gemini,
        GEMINI_MODEL: context.modelName,
      }
    case 'opencode':
      return {
        npm: '@ai-sdk/openai-compatible',
        options: {
          baseURL: urls.openai,
          apiKey: context.apiKey,
        },
        models: {
          [context.modelName]: {
            name: context.modelName,
            options: {
              store: false,
            },
          },
        },
      }
    case 'openclaw':
      return {
        baseUrl: urls.openai,
        apiKey: context.apiKey,
        api: 'openai-completions',
        models: [{ id: context.modelName, name: context.modelName }],
      }
    case 'hermes':
      return {
        name: context.providerName,
        base_url: urls.openai,
        api_key: context.apiKey,
        api_mode: 'chat_completions',
        models: [{ id: context.modelName, name: context.modelName }],
      }
    default:
      return null
  }
}

function resolveCcSwitchEndpoint(appId: CcSwitchImportAppId, routeBaseUrl: string): string {
  const urls = buildServiceUrls(routeBaseUrl)
  switch (appId) {
    case 'claude':
      return urls.anthropic
    case 'gemini':
      return urls.gemini
    default:
      return urls.openai
  }
}

const CCSWITCH_PROVIDER_NAME = 'Relayq'

export function buildCcSwitchImportLink(appId: CcSwitchImportAppId, context: ToolConfigContext): string {
  const endpoint = resolveCcSwitchEndpoint(appId, context.routeBaseUrl)
  const ccswitchProviderName = CCSWITCH_PROVIDER_NAME
  const params = new URLSearchParams({
    resource: 'provider',
    app: appId,
    name: ccswitchProviderName,
    homepage: context.homepage,
    endpoint,
    apiKey: context.apiKey,
    model: context.modelName,
    enabled: 'true',
    notes: `${ccswitchProviderName} - ${context.modelName}`,
  })

  const providerConfig = buildCcSwitchProviderConfig(appId, context)
  if (providerConfig) {
    params.set('configFormat', 'json')
    params.set('config', encodeUtf8Base64(JSON.stringify(providerConfig)))
  }

  return `ccswitch://v1/import?${params.toString()}`
}

export function buildToolConfigExport(toolId: ConfigExportToolId, context: ToolConfigContext): string {
  const urls = buildServiceUrls(context.routeBaseUrl)

  switch (toolId) {
    case 'claude-code':
      return `{
  "env": {
    "ANTHROPIC_API_KEY": "${context.apiKey}",
    "ANTHROPIC_BASE_URL": "${urls.anthropic}",
    "ANTHROPIC_MODEL": "${context.modelName}"
  }
}`
    case 'opencode': {
      const provider = resolveOpenCodeProvider(context.modelName, context.routeBaseUrl)
      return `{
  "$schema": "https://opencode.ai/config.json",
  "provider": {
    "${provider.providerId}": {
      "options": {
        "baseURL": "${provider.baseUrl}",
        "apiKey": "${context.apiKey}"
      },
      "models": {
        "${context.modelName}": {
          "name": "${context.modelName}",
          "options": {
            "store": false
          }
        }
      }
    }
  },
  "model": "${provider.modelRef}"
}`
    }
    case 'cursor':
      return `API Key: ${context.apiKey}
Base URL: ${urls.openai}
Model: ${context.modelName}`
    case 'curl':
      return `curl ${urls.openai}/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${context.apiKey}" \\
  -d '{
    "model": "${context.modelName}",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'`
    case 'python-sdk':
      return `from openai import OpenAI
client = OpenAI(
    api_key="${context.apiKey}",
    base_url="${urls.openai}"
)
response = client.chat.completions.create(
    model="${context.modelName}",
    messages=[
        {"role": "user", "content": "Hello!"}
    ]
)
print(response.choices[0].message.content)`
    case 'anthropic-sdk':
      return `import anthropic
client = anthropic.Anthropic(
    api_key="${context.apiKey}",
    base_url="${urls.anthropic}"
)
message = client.messages.create(
    model="${context.modelName}",
    max_tokens=1024,
    messages=[
        {"role": "user", "content": "Hello!"}
    ]
)
print(message.content[0].text)`
    default:
      return ''
  }
}

export function getToolExportFileName(toolId: ConfigExportToolId): string {
  switch (toolId) {
    case 'curl':
      return 'api-call.sh'
    case 'python-sdk':
    case 'anthropic-sdk':
      return 'main.py'
    case 'cursor':
      return 'cursor-config.txt'
    case 'claude-code':
      return 'claude-settings.json'
    case 'opencode':
      return 'opencode.json'
    default:
      return 'config.txt'
  }
}
