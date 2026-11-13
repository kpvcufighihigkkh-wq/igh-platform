import { test, expect, type Page } from '@playwright/test'

// 边端工位页面的端到端测试。
//
// 连的是真实边端二进制（端口 8191~8194），不是 Vite：
// 边端前端编在二进制里，由二进制按 edge.type 注入工位标记、
// 只注册本工位路由。用 Vite 测不出这条约束。
//
// 断言只依赖**稳定的结构与业务语义**，不依赖具体文案措辞——
// 措辞会随现场反馈调整，写死在测试里只会制造无谓的返工。

const EDGES = {
  winding: 'http://127.0.0.1:8191',
  sorting: 'http://127.0.0.1:8192',
  packaging: 'http://127.0.0.1:8193',
  warehouse: 'http://127.0.0.1:8194',
} as const

// 边端注入的工位标记：路由据此只注册本工位页面
async function openStation(page: Page, kind: keyof typeof EDGES) {
  await page.goto(EDGES[kind] + '/')
  await page.waitForLoadState('domcontentloaded')
}

test.describe('边端：只暴露本工位', () => {
  test('根路径直接进入本工位页面', async ({ page }) => {
    for (const [kind, base] of Object.entries(EDGES)) {
      await page.goto(base + '/')
      // 二进制把 window.__IGH_EDGE__ 注入 index.html，路由据此选页面
      const marker = await page.evaluate(
        () => (window as unknown as { __IGH_EDGE__?: { type?: string } }).__IGH_EDGE__,
      )
      expect(marker?.type, `${kind} 的工位标记`).toBe(kind)
      await expect(page).toHaveURL(new RegExp(`/station/${kind}$`))
    }
  })

  test('管理端路径不会暴露在边端', async ({ page }) => {
    // 边端没有 /api/v1/auth/login，也没有 admin 路由。
    // 打开 /admin 必须被重定向回本工位，而不是显示登录页或白屏。
    await page.goto(EDGES.winding + '/admin/dashboard')
    await expect(page).toHaveURL(/\/station\/winding$/)
  })
})

test.describe('卷绕工位', () => {
  test('批次与统计区渲染', async ({ page }) => {
    await openStation(page, 'winding')

    await expect(page.getByRole('heading', { name: '卷绕工位' })).toBeVisible()
    await expect(page.getByRole('heading', { name: '当前批次' })).toBeVisible()
    await expect(page.getByRole('heading', { name: '最近落纱' })).toBeVisible()

    // 边端种子会建一条批次，页面应显示它而不是"暂无活跃批次"
    await expect(page.getByText('SIM-LOT-001')).toBeVisible({ timeout: 10000 })

    // 今日统计三块
    for (const label of ['今日落纱', '今日丝饼', 'PLC 数据']) {
      await expect(page.getByText(label, { exact: true })).toBeVisible()
    }
  })

  test('未读到料车 RFID 时装车按钮禁用', async ({ page }) => {
    await openStation(page, 'winding')

    // 料车号来自 PLC；连不上 PLC 时页面必须禁用操作，
    // 而不是给一个点了没反应的按钮。
    const btn = page.getByRole('button', { name: /装车/ })
    await expect(btn).toBeVisible({ timeout: 10000 })
    await expect(btn).toBeDisabled()
    await expect(page.getByText(/当前料车/)).toBeVisible()
  })
})

test.describe('外检工位', () => {
  test('扫码框自动聚焦且等级按钮可用', async ({ page }) => {
    await openStation(page, 'sorting')

    const scan = page.getByPlaceholder('扫描丝锭标签')
    await expect(scan).toBeVisible({ timeout: 10000 })
    // 工人上手就是扫，不需要先找输入框
    await expect(scan).toBeFocused()

    for (const g of ['AAA', 'AA', 'AA1', 'AA2', 'A1', 'A']) {
      await expect(page.getByRole('button', { name: g, exact: true })).toBeVisible()
    }
  })

  test('缺陷代码来自接口，可多选', async ({ page }) => {
    await openStation(page, 'sorting')

    // 边端种子内置 7 个缺陷代码（毛丝/油污/成型不良/色差/断丝/尾丝过长/叠丝）
    const first = page.locator('.btn-defect').first()
    await expect(first).toBeVisible({ timeout: 10000 })
    await expect(page.locator('.btn-defect')).toHaveCount(7)

    // 点击后选中态生效（多选：再点一次取消）
    await first.click()
    await expect(first).toHaveClass(/active/)
    await first.click()
    await expect(first).not.toHaveClass(/active/)
  })

  test('扫到不存在的编码给出明确提示而不是静默失败', async ({ page }) => {
    await openStation(page, 'sorting')

    const scan = page.getByPlaceholder('扫描丝锭标签')
    await scan.fill('NOT-EXIST-CODE')
    await scan.press('Enter')

    await expect(page.getByText(/查无此丝锭/)).toBeVisible({ timeout: 10000 })
    // 提交按钮保持禁用：没有定位到丝锭就不该允许判级
    await expect(page.getByRole('button', { name: /提交判级/ })).toBeDisabled()
  })

  test('未定位丝锭前提交按钮禁用', async ({ page }) => {
    await openStation(page, 'sorting')
    await expect(page.getByPlaceholder('扫描丝锭标签')).toBeVisible({ timeout: 10000 })
    await expect(page.getByRole('button', { name: /提交判级/ })).toBeDisabled()
  })
})

test.describe('包装工位', () => {
  test('垛型网格按配置渲染', async ({ page }) => {
    await openStation(page, 'packaging')

    // e2e 配置注入了 6 层 × 4 位 × 2 锭 = 48 的垛型
    await expect(page.getByText('std48')).toBeVisible({ timeout: 10000 })
    await expect(page.getByText(/\/\s*48\s*锭/)).toBeVisible()

    // 层标签 L1..L6 与格子数
    for (const l of ['L1', 'L2', 'L3', 'L4', 'L5', 'L6']) {
      await expect(page.getByText(l, { exact: true })).toBeVisible()
    }
    await expect(page.locator('.cell')).toHaveCount(24)
  })

  test('PLC 未连接时明确显示而不是沿用旧值', async ({ page }) => {
    await openStation(page, 'packaging')

    // 不连 PLC 时，"PLC 已装锭数"不能继续显示一个冻结的数字
    await expect(page.getByText('PLC 已装锭数')).toBeVisible({ timeout: 10000 })
    await expect(page.getByText(/PLC 未连接|^—$/)).toBeVisible()
  })
})

test.describe('仓库工位', () => {
  test('库位区与操作区渲染，未选库位时按钮禁用', async ({ page }) => {
    await openStation(page, 'warehouse')

    await expect(page.getByRole('heading', { name: '仓库工位' })).toBeVisible()
    await expect(page.getByText('库位状态')).toBeVisible()

    // 手动入库/出库必须先点选库位；未选中时禁用而不是点了没反应
    await expect(page.getByText(/先在库位图上点选/)).toBeVisible({ timeout: 10000 })
    await expect(page.getByRole('button', { name: '手动入库' })).toBeDisabled()
    await expect(page.getByRole('button', { name: '手动出库' })).toBeDisabled()
  })
})

test.describe('通用约束', () => {
  test('主操作按钮不小于 64px，正文不小于 18px', async ({ page }) => {
    // 戴沾油手套操作、车间 1~1.5m 观看距离，这是硬约束而不是建议。
    await openStation(page, 'sorting')
    await expect(page.getByRole('button', { name: /提交判级/ })).toBeVisible({ timeout: 10000 })

    const submit = await page.getByRole('button', { name: /提交判级/ }).boundingBox()
    expect(submit!.height, '主操作按钮高度').toBeGreaterThanOrEqual(64)

    const grade = await page.locator('.btn-grade').first().boundingBox()
    expect(grade!.height, '等级按钮高度').toBeGreaterThanOrEqual(64)

    const fontSize = await page.locator('.btn-grade').first().evaluate(
      (el) => parseFloat(getComputedStyle(el).fontSize),
    )
    expect(fontSize, '按钮字号').toBeGreaterThanOrEqual(18)
  })

  test('页面不出现整页滚动条', async ({ page }) => {
    // 现场屏幕固定在支架上，整页滚动会让操作区滑出可视范围
    for (const kind of Object.keys(EDGES) as (keyof typeof EDGES)[]) {
      await openStation(page, kind)
      await page.waitForTimeout(500)
      const overflow = await page.evaluate(() => {
        const el = document.querySelector('.station')
        if (!el) return null
        return getComputedStyle(el).overflowY
      })
      expect(overflow, `${kind} 的 overflow-y`).toBe('hidden')
    }
  })
})
