export async function fetchAssets() {
  const res = await fetch('/api/v1/assets')
  if (!res.ok) throw new Error('failed to fetch assets')
  return res.json()
}

export async function fetchPing() {
  const res = await fetch('/api/v1/ping')
  if (!res.ok) throw new Error('backend unreachable')
  return res.json()
}

export async function fetchRfReadings() {
  const res = await fetch('/api/v1/rf/readings')
  if (!res.ok) throw new Error('failed to fetch rf readings')
  return res.json()
}

export async function fetchSatellites() {
  const res = await fetch('/api/v1/satellites/')
  if (!res.ok) throw new Error('failed to fetch satellites')
  return res.json()
}
