/// <reference types="vite-plugin-pwa/client" />

declare global {
  interface Window {
    __WB_MANIFEST: any;
  }
}

export {};
