import type { Translate } from '../../lib/I18nContext'
import type { RouteTab } from '../../lib/routeTabs'

export type DashboardNavigationSection = 'overview' | 'mission' | 'security' | 'rf' | 'platform' | 'preferences'

export interface DashboardNavigationItem {
  id: RouteTab
  label: string
  section: DashboardNavigationSection
}

export interface DashboardNavigationAccess {
  connectivity: boolean
  administration: boolean
  platform: boolean
}

export function buildDashboardNavigation(t: Translate, access: DashboardNavigationAccess): DashboardNavigationItem[] {
  return [
    { id: 'orbital', label: t('navigation.items.overview'), section: 'overview' },
    { id: 'graph', label: t('navigation.items.correlationGraph'), section: 'overview' },
    ...(access.connectivity ? [{ id: 'stations' as const, label: t('tabs.stations'), section: 'mission' as const }] : []),
    { id: 'assets', label: t('tabs.assets'), section: 'mission' },
    { id: 'incidents', label: t('tabs.incidents'), section: 'security' },
    { id: 'intel', label: t('tabs.intel'), section: 'security' },
    { id: 'signal', label: t('navigation.items.rfDetection'), section: 'rf' },
    { id: 'decoder', label: t('tabs.decoder'), section: 'rf' },
    { id: 'connectivity', label: t('navigation.items.sdrFleet'), section: 'platform' },
    ...(access.administration ? [{ id: 'admin' as const, label: t('navigation.items.accessControl'), section: 'platform' as const }] : []),
    ...(access.platform ? [{ id: 'platform' as const, label: t('tabs.platform'), section: 'platform' as const }] : []),
    { id: 'settings', label: t('tabs.settings'), section: 'preferences' },
  ]
}
