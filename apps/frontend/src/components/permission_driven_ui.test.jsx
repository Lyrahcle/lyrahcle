import { describe, expect, it } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'

function readComponent(file) {
  return fs.readFileSync(path.join(process.cwd(), 'src', 'components', file), 'utf8')
}

describe('permission-driven operational UI', () => {
  it('does not use legacy role-rank helpers in sensitive operational panels', () => {
    for (const file of ['Dashboard.tsx', 'ConnectivityPanel.jsx', 'DetectionRulesPanel.jsx', 'IncidentsPanel.tsx', 'AssetAlertsPanel.jsx']) {
      const src = readComponent(file)
      expect(src).not.toMatch(/\b(canWrite|canInvestigate|canAudit|isAdmin)\s*\(/)
      expect(src).toContain('hasPermission')
    }
  })

  it('maps write controls to backend capabilities', () => {
    const expected = {
      'Dashboard.tsx': ['assets.manage', 'orbital.manage', 'audit.read', 'agents.read', 'agents.manage'],
      'ConnectivityPanel.jsx': ['agents.read', 'agents.manage'],
      'DetectionRulesPanel.jsx': ['detections.manage'],
      'IncidentsPanel.tsx': ['incidents.manage'],
      'AssetAlertsPanel.jsx': ['alerts.manage'],
    }
    for (const [file, permissions] of Object.entries(expected)) {
      const src = readComponent(file)
      for (const permission of permissions) expect(src).toContain(`hasPermission('${permission}')`)
    }
  })

  it('does not gate connectivity or incident assignment on users.manage', () => {
    const connectivity = readComponent('ConnectivityPanel.jsx')
    const incidents = readComponent('IncidentsPanel.tsx')
    expect(connectivity).not.toContain("hasPermission('users.manage')")
    expect(connectivity).not.toMatch(/\badmin\s*[},]/)
    expect(incidents).not.toContain("hasPermission('users.manage')")
    expect(incidents).toContain("hasPermission('users.read')")
  })

})
