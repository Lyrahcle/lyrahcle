import { expect, test } from '@playwright/test'
import { loginAsAdmin } from './helpers'

test.describe('critical operator journeys', () => {
  test('login reaches mission workspace and Mission Control', async ({ page }) => {
    await loginAsAdmin(page)
    await expect(page.getByRole('button', { name: /mission control/i })).toBeVisible()
    await page.getByRole('button', { name: /mission control/i }).click()
    await expect(page).toHaveURL(/#\/mission$/)
    await expect(page.locator('main')).toBeVisible()
  })

  test('operator can move from RF Detection to Incidents', async ({ page }) => {
    await loginAsAdmin(page)
    await page.getByLabel('RF operations').getByRole('button', { name: 'RF detection' }).click()
    await expect(page).toHaveURL(/#\/signal$/)
    await expect(page.locator('main')).toBeVisible()
    await page.getByRole('button', { name: /^incidents$/i }).click()
    await expect(page).toHaveURL(/#\/incidents$/)
    await expect(page.locator('main')).toBeVisible()
  })

  test('admin can open Access Control', async ({ page }) => {
    await loginAsAdmin(page)
    await page.getByRole('button', { name: /access control/i }).click()
    await expect(page).toHaveURL(/#\/admin$/)
    await expect(page.locator('main')).toBeVisible()
  })
})
