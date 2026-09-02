import { useCallback, useEffect, useState } from 'react'
import {
  DEFAULT_TAB,
  getTabFromLocation,
  setTabInLocation,
  subscribeTabChanges,
  type RouteTab,
} from '../lib/routeTabs'

export interface HashRouteState {
  activeTab: RouteTab
  changeTab: (tab: RouteTab) => void
}

export function useHashRoute(fallback: RouteTab = DEFAULT_TAB): HashRouteState {
  const [activeTab, setActiveTab] = useState<RouteTab>(() => getTabFromLocation(fallback))

  const changeTab = useCallback((tab: RouteTab) => {
    setActiveTab(tab)
    setTabInLocation(tab)
  }, [])

  useEffect(() => {
    setTabInLocation(activeTab, { replace: true })
    return subscribeTabChanges(setActiveTab)
  }, [])

  return { activeTab, changeTab }
}
