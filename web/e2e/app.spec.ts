import { test, expect, Page } from '@playwright/test'

const BASE = 'http://localhost:3000'

async function login(page: Page) {
  await page.goto(BASE + '/login')
  await page.locator('input').nth(0).waitFor({ timeout: 10000 })
  await page.locator('input').nth(0).fill('admin')
  await page.locator('input').nth(1).fill('admin123')
  await page.locator('button:has-text("登录")').click()
  await page.waitForURL('**/admin/dashboard', { timeout: 10000 })
}

test.describe('IGH Platform v2', () => {
  test('root redirects to board (no station match)', async ({ page }) => {
    await page.goto(BASE + '/')
    await page.waitForURL('**/board/overview', { timeout: 10000 })
    await expect(page.locator('text=IGH 智能制造平台')).toBeVisible()
  })

  test('board overview shows KPI and charts', async ({ page }) => {
    await page.goto(BASE + '/board/overview')
    await expect(page.locator('text=今日落纱')).toBeVisible({ timeout: 5000 })
    await expect(page.locator('canvas').first()).toBeVisible({ timeout: 8000 })
  })

  test('board warehouse shows grid', async ({ page }) => {
    await page.goto(BASE + '/board/warehouse')
    await expect(page.locator('text=仓库实时状态')).toBeVisible()
    await expect(page.locator('.iso-svg').first()).toBeVisible({ timeout: 5000 })
  })

  test('board production shows lines', async ({ page }) => {
    await page.goto(BASE + '/board/production')
    await expect(page.locator('text=产线实时状态')).toBeVisible()
    await expect(page.locator('text=卷绕线01')).toBeVisible({ timeout: 5000 })
  })

  test('station winding page', async ({ page }) => {
    await page.goto(BASE + '/station/winding')
    await expect(page.locator('text=卷绕工位')).toBeVisible()
    await expect(page.locator('text=当前批次')).toBeVisible({ timeout: 5000 })
  })

  test('station sorting page', async ({ page }) => {
    await page.goto(BASE + '/station/sorting')
    await expect(page.locator('text=分拣工位')).toBeVisible()
    await expect(page.locator('text=录入分拣结果')).toBeVisible({ timeout: 5000 })
  })

  test('station packaging page', async ({ page }) => {
    await page.goto(BASE + '/station/packaging')
    await expect(page.locator('text=包装工位')).toBeVisible()
    await expect(page.locator('text=订单队列')).toBeVisible({ timeout: 5000 })
  })

  test('station warehouse page', async ({ page }) => {
    await page.goto(BASE + '/station/warehouse')
    await expect(page.locator('text=仓库工位')).toBeVisible()
    await expect(page.locator('text=库位状态')).toBeVisible({ timeout: 5000 })
  })

  test('login and admin dashboard', async ({ page }) => {
    await login(page)
    await expect(page.locator('text=今日落纱')).toBeVisible()
  })

  test('admin settings page', async ({ page }) => {
    await login(page)
    await page.click('text=系统设置')
    await expect(page.locator('text=工位绑定')).toBeVisible()
    await expect(page.locator('text=192.168.1.101')).toBeVisible({ timeout: 5000 })
  })

  test('admin statistics with charts', async ({ page }) => {
    await login(page)
    await page.click('text=统计分析')
    await expect(page.locator('canvas').first()).toBeVisible({ timeout: 8000 })
  })
})
