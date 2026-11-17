import { expect, test, type Page } from '@playwright/test'

/** 种子账号的默认密码，可通过环境变量覆盖。 */
const password = process.env.E2E_PASSWORD ?? 'admin123'

/** 用账号密码登录，并等待进入工作台。 */
async function signIn(page: Page, accountName = 'chen.yu') {
  await page.goto('/signin')
  await page.getByPlaceholder('请输入账号').fill(accountName)
  await page.getByPlaceholder('请输入密码').fill(password)
  await page.getByRole('button', { name: '登录' }).click()
  await expect(page).toHaveURL(/\/dashboard$/)
}

test('登录后进入工作台并渲染侧边导航', async ({ page }) => {
  await signIn(page)
  await expect(page.locator('h1')).toHaveText('工作台')
  // 一级导航常驻，二级菜单在切到对应分组后才出现。
  await expect(page.locator('[data-navigation-key="dashboard"]')).toBeVisible()
  await page.locator('[data-navigation-key="system"]').click()
  await expect(page.getByRole('link', { name: '经营主体' })).toBeVisible()
  await expect(page.getByRole('link', { name: '账户' })).toBeVisible()
})

test('深链刷新后仍保持会话', async ({ page }) => {
  await signIn(page)
  await page.goto('/system/permissions')
  await expect(page.locator('h1')).toHaveText('权限')
  // 权限目录由后端下发，首列固定是「工作台」与「系统」两个根节点。
  await expect(page.locator('li[data-ui-miller-columns-row]').first()).toContainText('工作台')

  await page.reload()
  await expect(page.locator('h1')).toHaveText('权限')
})

test('没有编辑权限时列表页不渲染新建入口', async ({ page }) => {
  // 集团管理员只有 tenant:view，没有 tenant:edit。
  await signIn(page)
  await page.goto('/system/tenants')
  await expect(page.locator('h1')).toHaveText('租户')
  await expect(page.locator('tbody tr').first()).toBeVisible()
  await expect(page.getByRole('button', { name: '新建租户' })).toHaveCount(0)
})

test('退出登录后受保护的路由重新跳回登录页', async ({ page }) => {
  await signIn(page)
  await page.getByRole('button', { name: '账号菜单' }).click()
  await page.getByText('退出登录').click()
  await expect(page).toHaveURL(/\/signin$/)

  await page.goto('/system/accounts')
  await expect(page).toHaveURL(/\/signin/)
})
