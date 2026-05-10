import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { viteSingleFile } from 'vite-plugin-singlefile'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), viteSingleFile()],
  build: {
    // 禁用代码分割
    codeSplitting: false,
    // 禁用 CSS 代码分割
    cssCodeSplit: false,
    // 资源内联阈值（100MB），确保所有资源都转为 base64 内联
    assetsInlineLimit: 100000000,
  },
})
