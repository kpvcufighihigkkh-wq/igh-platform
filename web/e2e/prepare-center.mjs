// 为 e2e 生成一份中心配置：复制 configs-dev3，只改端口。
//
// 为什么要复制而不是直接让 --port 覆盖：中心没有端口命令行参数，
// 端口只从 project.yaml 的 server.http_port / ws_port 读。
//
// 为什么要改端口：本机通常已有中心常驻在 8101，e2e 再起一个会抢端口。
// 换到 8199 后两者能并存，测试也不必先停掉开发用的那个。
//
// 用法：E2E_CENTER_PORT=8199 node e2e/prepare-center.mjs
import { cp, mkdir, readFile, rm, writeFile } from 'node:fs/promises'
import { existsSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const webDir = dirname(fileURLToPath(import.meta.url))
const rootDir = resolve(webDir, '..', '..')
const template = join(rootDir, 'configs-dev3')
const outDir = join(rootDir, 'tmp', 'e2e-center')

const httpPort = Number(process.env.E2E_CENTER_PORT || 8101)
// ws 端口取 http+1：两处必须一起改，否则中心会去监听一个已被
// 常驻实例占用的端口，启动时报 bind 失败而看起来像"数据库连不上"。
const wsPort = httpPort + 1

if (!existsSync(template)) {
  throw new Error(`中心模板配置不存在: ${template}`)
}

await rm(outDir, { recursive: true, force: true })
await mkdir(outDir, { recursive: true })
await cp(template, outDir, { recursive: true })

const yamlPath = join(outDir, 'project.yaml')
let yaml = await readFile(yamlPath, 'utf8')

// 逐行替换而不是重新序列化：模板里大量注释解释了现场语义
// （HA 模式、归档策略、DSN 多地址），重写会把它们全丢掉。
yaml = yaml
  .replace(/^(\s*http_port:\s*)\d+\s*$/m, `$1${httpPort}`)
  .replace(/^(\s*ws_port:\s*)\d+\s*$/m, `$1${wsPort}`)

await writeFile(yamlPath, yaml, 'utf8')

console.log(`中心 e2e 配置已生成: ${outDir}`)
console.log(`  http http://127.0.0.1:${httpPort}  ws :${wsPort}`)
