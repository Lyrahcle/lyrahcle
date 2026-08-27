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
