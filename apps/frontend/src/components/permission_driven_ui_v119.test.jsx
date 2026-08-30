import { describe, expect, it } from 'vitest'
import fs from 'node:fs'

describe('permission-driven operational UI', () => {
  it('does not use legacy role-rank helpers in sensitive operational panels', () => {
    for (const file of ['Dashboard.jsx', 'DetectionRulesPanel.jsx', 'IncidentsPanel.jsx', 'AssetAlertsPanel.jsx']) {
      const src = fs.readFileSync(new URL(`./${file}`, import.meta.url), 'utf8')
      expect(src).not.toMatch(/\b(canWrite|canInvestigate|canAudit|isAdmin)\s*\(/)
      expect(src).toContain('hasPermission')
    }
  })

  it('maps write controls to backend capabilities', () => {
    const expected = {
      'Dashboard.jsx': ['assets.manage', 'orbital.manage', 'audit.read'],
      'DetectionRulesPanel.jsx': ['detections.manage'],
      'IncidentsPanel.jsx': ['incidents.manage'],
      'AssetAlertsPanel.jsx': ['alerts.manage'],
    }
    for (const [file, permissions] of Object.entries(expected)) {
      const src = fs.readFileSync(new URL(`./${file}`, import.meta.url), 'utf8')
      for (const permission of permissions) expect(src).toContain(`hasPermission('${permission}')`)
    }
  })
})
