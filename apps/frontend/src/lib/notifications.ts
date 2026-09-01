const STORAGE_KEY = 'lyra_notifications_enabled'

export function getNotificationsEnabled(): boolean {
  return localStorage.getItem(STORAGE_KEY) === 'true'
}

export function isNotificationSupported(): boolean {
  return typeof window !== 'undefined' && 'Notification' in window
}

export async function setNotificationsEnabled(enabled: boolean): Promise<boolean> {
  if (!enabled) {
    localStorage.setItem(STORAGE_KEY, 'false')
    return true
  }

  if (!isNotificationSupported()) {
    localStorage.setItem(STORAGE_KEY, 'false')
    return false
  }

  const permission = await Notification.requestPermission()
  const granted = permission === 'granted'
  localStorage.setItem(STORAGE_KEY, granted ? 'true' : 'false')
  return granted
}
