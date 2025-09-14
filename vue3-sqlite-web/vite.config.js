import { fileURLToPath, URL } from 'node:url';

import { copyFileSync, mkdirSync } from 'fs';
import { resolve } from 'path';

import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import vueDevTools from 'vite-plugin-vue-devtools';
import AutoImport from 'unplugin-auto-import/vite';
import { NaiveUiResolver } from 'unplugin-vue-components/resolvers';
import Components from 'unplugin-vue-components/vite';

function copyToPublicPlugin({ sources }) {
  return {
    name: 'copy-files-before-build',
    // 在配置解析完成后、构建开始前执行
    configResolved() {
      // 执行复制
      sources.forEach(({ src, dest, rename }) => {
        const srcPath = resolve(src)
        const destDir = resolve(dest || '.').split('/').slice(0, -1).join('/')
        mkdirSync(destDir, { recursive: true })

        const destPath = resolve(destDir, rename || src.split('/').pop())
        copyFileSync(srcPath, destPath)
        console.log(`📄 Copied: ${src} → ${dest || ''}${rename ? rename : src.split('/').pop()}`)
      })
    }
  };
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    vueDevTools(),
    AutoImport({
      imports: [
        'vue',
        {
          'naive-ui': [
            'useDialog',
            'useMessage',
            'useNotification',
            'useLoadingBar'
          ]
        }
      ]
    }),
    Components({
      resolvers: [NaiveUiResolver()]
    }),
    copyToPublicPlugin({
      sources: [
        { src: 'node_modules/ace-builds/src-noconflict/snippets/sql.js', dest: 'public/assets/ace-builds/src-noconflict/snippets/sql.js' },
      ]
    })
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    rollupOptions: {
      output: {
        manualChunks: {
          vue: ["vue", "vue-router"],
        }
      }
    }
  },
  server: {
    port: 5174,
    proxy: {
      '/api': {
        target: 'http://localhost:12249',
        // target: 'https://arb.cardm.top',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, '')
      },
    },
  },
});
