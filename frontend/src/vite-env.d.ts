/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL: string
  readonly VITE_ESCROW_ADDRESS: string
  readonly VITE_USDC_ADDRESS: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
