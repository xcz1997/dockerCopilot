import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  base: '/',
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  build: {
    outDir: '../front',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        // 使用 chunk 前缀避免下划线开头的文件名（go-zero 路由不兼容）
        entryFileNames: 'assets/e[hash].js',
        chunkFileNames: 'assets/c[hash].js',
        assetFileNames: 'assets/a[hash].[ext]',
        // 禁用代码分割，合并为单个文件
        manualChunks: undefined
      }
    }
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:12712',
        changeOrigin: true
      }
    }
  }
})
