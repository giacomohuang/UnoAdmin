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
