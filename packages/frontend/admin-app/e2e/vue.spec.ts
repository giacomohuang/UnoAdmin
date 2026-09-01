import { test, expect } from '@playwright/test'

test('loads the generated demo package route inside the SPA shell', async ({ page }) => {
  await page.goto('/demo/overview')
  await expect(page.locator('h1')).toHaveText('示例业务')
  await expect(page.locator('.admin-brand strong')).toHaveText('UnoAdmin')
})

test('refreshes a deep link through the SPA fallback', async ({ page }) => {
  await page.goto('/system/users')
  await expect(page.locator('h1')).toHaveText('用户管理')
  await page.reload()
  await expect(page.locator('h1')).toHaveText('用户管理')
})

// Verify the demo sign-in action rebuilds the session after logout.
test('re-enters the demo after signing out', async ({ page }) => {
  await page.goto('/dashboard')
  await page.getByRole('button', { name: '退出登录' }).click()
  await expect(page).toHaveURL(/\/signin$/)

  await page.getByRole('button', { name: '进入演示' }).click()
  await expect(page).toHaveURL(/\/dashboard$/)
  await expect(page.locator('h1')).toHaveText('工作台')
})
