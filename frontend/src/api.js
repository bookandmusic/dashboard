async function getJSON(url) {
  const res = await fetch(url, { headers: { Accept: 'application/json' } })
  if (!res.ok) throw new Error(`${url} → ${res.status}`)
  return res.json()
}

export const fetchConfig = () => getJSON('/api/config')
export const fetchStats = () => getJSON('/api/stats')
