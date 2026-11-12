import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { NaiveUiResolver } from 'unplugin-vue-components/resolvers'

// 开发代理目标。
//
// 不写死端口的原因：8099 在本机上是原 Jiatong 项目的静态服务，
// 而新中心的默认端口是 8101。写死 8099 时 `npm run dev` 会把所有
// 接口打到老项目上，页面全是 404/数据不对，很难看出问题在哪。
//
// 覆盖方式（任选其一）：
//   VITE_API_TARGET=http://192.168.1.10:8099 npm run dev
//   web/.env.local 里写 VITE_API_TARGET=http://192.168.1.10:8099
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const target = env.VITE_API_TARGET || 'http://127.0.0.1:8101'
  const wsTarget = target.replace(/^http/, 'ws')

  return {
    plugins: [
      vue(),
      AutoImport({
        imports: ['vue', 'vue-router'],
      }),
      Components({
        resolvers: [NaiveUiResolver()],
      }),
    ],
    server: {
      port: 3000,
      proxy: {
        '/api': {
          target,
          changeOrigin: true,
        },
        '/ws': {
          target: wsTarget,
          ws: true,
        },
      },
    },
  }
})
