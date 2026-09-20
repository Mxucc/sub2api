/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
  // 定制版（fork）构建时注入：发布/更新来源仓库与镜像名（默认指向官方正式版）
  readonly VITE_UPDATE_REPO?: string
  readonly VITE_DOCKER_IMAGE?: string
  readonly BASE_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

declare module '*.md?raw' {
  const content: string
  export default content
}
