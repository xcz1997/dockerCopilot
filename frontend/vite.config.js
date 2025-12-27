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
        // 使用纯哈希文件名，避免任何特殊字符
        entryFileNames: 'assets/[hash].js',
        chunkFileNames: 'assets/[hash].js',
        assetFileNames: 'assets/[hash].[ext]',
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
