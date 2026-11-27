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
      // 监听所有网卡：现场平板、其他电脑用 http://<本机局域网IP>:3000 直接打开，
      // 不需要在每台机器上装 Node。接口仍由本机 Vite 代理到后端，
      // 所以后端只需要被本机访问得到即可。
      host: true,
      port: 3000,
      // 端口被占时直接报错，不要静默换到 3001——现场按固定地址发链接
      strictPort: true,
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
