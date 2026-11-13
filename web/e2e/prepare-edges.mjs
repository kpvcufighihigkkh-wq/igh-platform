// 为 e2e 准备 4 个独立边端实例的配置。
//
// 为什么不在仓库里放 4 份 configs-e2e-*：它们与 configs-winding 只差
// id/type/port/data_dir 几个字段，复制进仓库后每次改点表都要同步 4 份，
// 迟早漂移。这里在测试启动前从单一模板生成到 tmp/（已在 .gitignore 里）。
//
// 用法：node e2e/prepare-edges.mjs
import { cp, mkdir, readFile, rm, writeFile } from 'node:fs/promises'
import { existsSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const webDir = dirname(fileURLToPath(import.meta.url))
const rootDir = resolve(webDir, '..', '..')
const template = join(rootDir, 'configs-winding')
const outRoot = join(rootDir, 'tmp', 'e2e-edges')

// 端口避开 8080（开发机常驻的边端）与中心 8101。
const stations = [
  { type: 'winding', id: 'edge-e2e-winding', port: 8191 },
  { type: 'sorting', id: 'edge-e2e-sorting', port: 8192 },
  { type: 'packaging', id: 'edge-e2e-packaging', port: 8193 },
  { type: 'warehouse', id: 'edge-e2e-warehouse', port: 8194 },
]

// 包装工位需要一份垛型配置，否则页面只会显示"未配置垛型"的提示。
const palletDoc = {
  version: 1,
  updated_at: '2026-01-01T00:00:00+08:00',
  updated_by: 'e2e',
  remark: 'e2e 固定垛型：6 层 × 4 位 × 2 锭 = 48',
  pallets: {
    default: 'std48',
    layouts: [
      {
        code: 'std48',
        bobbins_per_position: 2,
        layers: Array.from({ length: 6 }, (_, i) => ({ layer: i + 1, positions: 4 })),
      },
    ],
  },
}

if (!existsSync(template)) {
  throw new Error(`模板配置不存在: ${template}`)
}

await rm(outRoot, { recursive: true, force: true })

for (const s of stations) {
  const dir = join(outRoot, s.type)
  await mkdir(dir, { recursive: true })
  await cp(template, dir, { recursive: true })

  const yamlPath = join(dir, 'edge.yaml')
  let yaml = await readFile(yamlPath, 'utf8')

  // 逐字段替换而不是重新生成整份 YAML：
  // 模板里有大量注释说明现场语义，重写会把它们丢掉。
  yaml = yaml
    .replace(/^(\s*id:\s*).*$/m, `$1"${s.id}"`)
    .replace(/^(\s*type:\s*).*$/m, `$1"${s.type}"`)
    .replace(/^(\s*data_dir:\s*).*$/m, `$1"./tmp/e2e-edges/data-${s.type}"`)
    .replace(/^(\s*port:\s*)8080\s*$/m, `$1${s.port}`)
    // 不连真实 PLC：连不上只会刷日志，不产生任何业务事件，也不污染中心。
    .replace(/^(\s*strict:\s*).*$/m, '$1false')
    // 开仿真协议有两个作用，缺一不可：
    //   1. app.go 只在检测到 simulation 设备时执行 SeedSimulation，
    //      不开就没有批次/等级/缺陷种子，页面空白、断言无从下手；
    //   2. 仿真器按真实点表产数据，覆盖完整解码路径。
    // 仿真产生的事件只写本地库，因为下面的 server.url 指向不可达地址。
    .replace(/^(\s*instance_files:)/m, '$1')
    .replace(/^(\s*)poll_interval_ms:/m, '$1protocol_override: "simulation"\n$1poll_interval_ms:')
    // 指向不可达地址：e2e 必须完全隔离，不能连开发机常驻的中心 8101。
    // 否则测试结果取决于中心库里当时有什么数据，本机跑和 CI 跑结论不同。
    .replace(/^(\s*)url:\s*"http:\/\/127\.0\.0\.1:8101"/m, '$1url: "http://127.0.0.1:9"')

  await writeFile(yamlPath, yaml, 'utf8')

  if (s.type === 'packaging') {
    // data_dir 是相对**进程工作目录**解析的（边端从仓库根启动），
    // 不是相对配置目录。写错位置的话边端读不到垛型，页面只会显示"未配置垛型"。
    const dataDir = join(outRoot, `data-${s.type}`)
    await mkdir(dataDir, { recursive: true })
    await writeFile(
      join(dataDir, 'site_config.json'),
      JSON.stringify(palletDoc, null, 2),
      'utf8',
    )
  }
}

console.log(`e2e 边端配置已生成: ${outRoot}`)
for (const s of stations) {
  console.log(`  ${s.type.padEnd(10)} http://127.0.0.1:${s.port}  (${s.id})`)
}
