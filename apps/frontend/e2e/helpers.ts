import type { Page, Route } from '@playwright/test'

const permissions = [
  'users.read', 'users.manage', 'audit.read', 'rbac.read', 'rbac.manage',
  'programs.read', 'assets.read', 'incidents.read', 'incidents.manage',
  'rf.read', 'alerts.read', 'agents.read', 'agents.manage', 'orbital.read',
]

function jwt(): string {
  const encode = (value: object) => Buffer.from(JSON.stringify(value)).toString('base64url')
  return `${encode({ alg: 'none', typ: 'JWT' })}.${encode({ exp: 4102444800, password_change_required: false })}.signature`
}

function fulfillJSON(route: Route, body: unknown) {
  return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) })
}

export async function installMockApi(page: Page) {
  await page.route('**/api/v1/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    if (path === '/api/v1/auth/login') return fulfillJSON(route, { token: jwt(), role: 'admin' })
    if (path === '/api/v1/auth/logout') return fulfillJSON(route, {})
    if (path === '/api/v1/rbac/me') return fulfillJSON(route, { permissions })
    if (path === '/api/v1/ping') return fulfillJSON(route, { status: 'ok', db: 'ok' })
    if (path.endsWith('/stream')) return route.fulfill({ status: 200, contentType: 'text/event-stream', body: '' })
    if (path === '/api/v1/incidents/' || path === '/api/v1/asset-alerts/' || path === '/api/v1/detection-rules/' || path === '/api/v1/audit-log') {
      return fulfillJSON(route, { items: [], total: 0, limit: 50, offset: 0 })
    }
    if (path.startsWith('/api/v1/threat-intel/space-cves')) return fulfillJSON(route, { items: [], total: 0, limit: 50, offset: 0 })
    if (path === '/api/v1/rbac/permissions' || path === '/api/v1/rbac/roles') return fulfillJSON(route, [])
    if (path === '/api/v1/organizations/me') return fulfillJSON(route, { platform_admin: false })
    return fulfillJSON(route, [])
  })
}

export async function loginAsAdmin(page: Page) {
  await installMockApi(page)
  await page.goto('/')
  await page.locator('#identifier').fill('admin@example.test')
  await page.locator('#password').fill('correct horse battery staple')
  await page.locator('form').filter({ has: page.locator('#identifier') }).locator('button[type="submit"]').click()
  await page.getByRole('navigation').waitFor()
}
