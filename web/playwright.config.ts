import { defineConfig } from '@playwright/test'

// 边端页面测试要连真实的边端进程。
//
// 为什么不直接用 Vite：边端前端是**编进二进制**的，由二进制按 edge.type
// 注入工位标记并只注册本工位路由。Vite 开发服务器没有这个标记，
// 打开 /station/sorting 也会渲染出来——那是管理端行为，
// 测不出"边端只暴露本工位"这条真实约束，也就测不到我们最关心的东西。
//
// 4 个工位各起一个实例（8191~8194），配置由 e2e/prepare-edges.mjs
// 从 configs-winding 生成到 tmp/，仓库里只维护一份模板。
const edges = [
  { config: './tmp/e2e-edges/winding', port: 8191 },
  { config: './tmp/e2e-edges/sorting', port: 8192 },
  { config: './tmp/e2e-edges/packaging', port: 8193 },
  { config: './tmp/e2e-edges/warehouse', port: 8194 },
]

export default defineConfig({
  testDir: './e2e',
  timeout: 30000,
  // 串行：4 个实例共用同一份样板配置生成的本地库，
  // 并行跑时断言会互相看到对方产生的数据。
  fullyParallel: false,
  workers: 1,
  reporter: [['list']],
  use: { headless: true },
  projects: [
    { name: 'edge', testMatch: /edge-.*\.spec\.ts/ },
    // 管理端：走 Vite + 中心服务。
    // 与边端分开是因为依赖完全不同——边端要真实二进制，
    // 管理端要 8101 的中心与 3000 的开发服务器。
    {
      name: 'admin',
      testMatch: /app\.spec\.ts/,
      use: { baseURL: 'http://localhost:3000' },
    },
  ],
  webServer: [
    ...edges.map((e) => ({
      // 用 cwd + 根相对路径，不要写 "../bin/..."：
      // Windows 下 Playwright 通过 cmd 启动，以 ".." 开头的可执行路径会被当成
      // 不存在的命令（cmd 只认得盘符或绝对路径）。
      command: `bin\\igh-edge-e2e.exe --config ${e.config} --log-level error`,
      cwd: '..',
      url: `http://127.0.0.1:${e.port}/health`,
      reuseExistingServer: false,
      timeout: 60000,
      stdout: 'ignore' as const,
      stderr: 'pipe' as const,
    })),
  ],
})
