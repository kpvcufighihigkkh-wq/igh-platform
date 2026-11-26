import { defineConfig } from '@playwright/test'

// 两组 e2e 的依赖完全不同，用 E2E_TARGET 选择启动哪一侧：
//
//   E2E_TARGET=edge  —— 只起 4 个边端实例（8191~8194）
//   E2E_TARGET=admin —— 只起 Vite（3000），中心服务默认外部已启动，E2E_CENTER=1 时由这里代管
//   不设              —— 两侧都起（本地全量跑）
//
// 边端必须连真实二进制而不是 Vite：边端前端是**编进二进制**的，由二进制按
// edge.type 注入工位标记并只注册本工位路由。Vite 没有这个标记，
// 打开 /station/sorting 也能渲染——那是管理端行为，
// 测不出"边端只暴露本工位"这条真实约束。
const target = process.env.E2E_TARGET || 'all'
const wantEdges = target !== 'admin'
const wantVite = target !== 'edge'

// 中心服务是否由 Playwright 拉起。
//
// 默认不拉：本地开发时中心通常已经在 8101 跑着（可能连的是有数据的开发库），
// 抢它的端口只会把事情搞乱。CI 里没有常驻进程，用 E2E_CENTER=1 让这里代管，
// 好处是启动/就绪/退出生命周期都和 Playwright 绑定，不需要在流水线里
// 手动 nohup + 轮询 /health，也不会留下僵尸进程拖住 runner。
//
// 端口由 E2E_CENTER_PORT 决定（默认 8101）。本地想验证这条路径又不撞
// 常驻实例时，配合 `npm run e2e:center-config` 换成 8199 之类即可——
// 配置文件的端口和这里的健康检查地址必须同时变，所以只用这一个变量。
const withCenter = process.env.E2E_CENTER === '1'
const centerPort = Number(process.env.E2E_CENTER_PORT || 8101)

// 4 个工位各起一个实例，配置由 e2e/prepare-edges.mjs 从 configs-winding
// 生成到 tmp/，仓库里只维护一份模板。
const edges = [
  { config: './tmp/e2e-edges/winding', port: 8191 },
  { config: './tmp/e2e-edges/sorting', port: 8192 },
  { config: './tmp/e2e-edges/packaging', port: 8193 },
  { config: './tmp/e2e-edges/warehouse', port: 8194 },
]

// 路径分隔符随平台变化：Windows 的 cmd 不认 "/"，Linux 不认 "\"。
// 扩展名统一保留 .exe——Go 产物用这个名字在 Linux 上也能正常执行，
// 统一命名省掉一处平台分支。
const sep = process.platform === 'win32' ? '\\' : '/'
const edgeBin = `bin${sep}igh-edge-e2e.exe`
const serverBin = `bin${sep}igh-server.exe`
// 配置目录走相对路径即可：webServer 的 cwd 已设为仓库根，
// 与边端那四条 --config 的解析方式一致。
//
// 默认用 e2e/prepare-center.mjs 生成的临时目录（端口已被改过），
// 而不是 configs-dev3：直接用后者的话，中心的端口是 8101，
// 与本机常驻实例冲突，E2E_CENTER_PORT 也就形同虚设。
const centerConfig = process.env.E2E_CENTER_CONFIG || './tmp/e2e-center'

export default defineConfig({
  testDir: './e2e',
  timeout: 30000,
  // 串行：4 个边端实例共用同一份样板配置生成的本地库，
  // 并行跑时断言会互相看到对方产生的数据。
  fullyParallel: false,
  workers: 1,
  reporter: [['list']],
  use: { headless: true },
  projects: [
    { name: 'edge', testMatch: /edge-.*\.spec\.ts/ },
    // 管理端走 Vite + 中心服务，与边端的依赖完全不同，因此分成两个项目。
    {
      name: 'admin',
      testMatch: /app\.spec\.ts/,
      use: { baseURL: 'http://localhost:3000' },
    },
  ],
  webServer: [
    ...(wantEdges
      ? edges.map((e) => ({
          // 用 cwd + 根相对路径，不要写 "../bin/..."：
          // Windows 下 Playwright 通过 cmd 启动，以 ".." 开头的可执行路径
          // 会被当成不存在的命令（cmd 只认得盘符或绝对路径）。
          command: `${edgeBin} --config ${e.config} --log-level error`,
          cwd: '..',
          url: `http://127.0.0.1:${e.port}/health`,
          reuseExistingServer: false,
          timeout: 60000,
          stdout: 'ignore' as const,
          stderr: 'pipe' as const,
        }))
      : []),
    ...(withCenter
      ? [
          {
            // --config 指向 prepare-center.mjs 生成的临时配置：
            // 端口在那里改过，这里只管把进程拉起来。
            command: `${serverBin} --config ${centerConfig} --log-level warn`,
            cwd: '..',
            // 中心连库失败会直接 os.Exit，Playwright 报的是"进程已退出"，
            // 比起测试里一个个超时，这个错误信息更容易定位。
            url: `http://127.0.0.1:${centerPort}/health`,
            reuseExistingServer: !process.env.CI,
            timeout: 180000,
            stdout: 'ignore' as const,
            stderr: 'pipe' as const,
          },
        ]
      : []),
    ...(wantVite
      ? [
          {
            // --strictPort：端口被占时直接失败，而不是悄悄换到 3001。
            // 换端口会让 baseURL 指错地方，测试连到别的进程上却依然"通过"。
            command: 'npx vite --port 3000 --strictPort',
            url: 'http://localhost:3000/',
            reuseExistingServer: !process.env.CI,
            timeout: 120000,
            // 代管中心时，Vite 必须代理到那个端口，而不是默认的 8101：
            // 否则页面请求打到常驻实例上，测的就不是刚拉起来的这个中心。
            // vite.config.ts 读的是 VITE_API_TARGET（见 loadEnv）。
            ...(withCenter ? { env: { VITE_API_TARGET: `http://127.0.0.1:${centerPort}` } } : {}),
          },
        ]
      : []),
  ],
})
