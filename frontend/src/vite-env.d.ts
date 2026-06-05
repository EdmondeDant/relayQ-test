/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
  readonly BASE_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module 'mammoth/mammoth.browser' {
  interface ExtractRawTextInput {
    arrayBuffer: ArrayBuffer
  }

  interface ExtractRawTextResult {
    value: string
    messages: unknown[]
  }

  const mammoth: {
    extractRawText(input: ExtractRawTextInput): Promise<ExtractRawTextResult>
  }

  export default mammoth
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}
