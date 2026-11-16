import { expect, test } from '@playwright/test'

/** 通过真实后端登录并写入令牌，后续页面才能通过路由守卫。 */
async function signIn(page: import('@playwright/test').Page) {
  await page.goto('/signin')
  await page.getByRole('button', { name: '进入演示' }).click()
  await expect(page).toHaveURL(/\/dashboard$/)
}

test('loads a generated package route inside the SPA shell', async ({ page }) => {
  await signIn(page)
  await page.goto('/system/users')
  await expect(page.locator('h1')).toHaveText('用户管理')
  await expect(page.locator('.admin-brand strong')).toHaveText('UnoAdmin')
})

test('refreshes a deep link through the SPA fallback', async ({ page }) => {
  await signIn(page)
  await page.goto('/system/users')
  await expect(page.locator('h1')).toHaveText('用户管理')
  await page.reload()
  await expect(page.locator('h1')).toHaveText('用户管理')
})

// Verify the sign-in action rebuilds the session after logout.
test('re-enters the workspace after signing out', async ({ page }) => {
  await signIn(page)
  await page.getByRole('button', { name: '退出登录' }).click()
  await expect(page).toHaveURL(/\/signin$/)

  await page.getByRole('button', { name: '进入演示' }).click()
  await expect(page).toHaveURL(/\/dashboard$/)
  await expect(page.locator('h1')).toHaveText('工作台')
})
