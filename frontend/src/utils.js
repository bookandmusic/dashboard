export const NETS = ['intranet', 'internet']

export function resolveAddress(site, mode) {
  const match = site.addresses.find(a => a.net === mode)
  return match || site.addresses[0]
}

export function hostOf(url) {
  return url.replace(/^https?:\/\//, '').replace(/\/$/, '')
}

export function fmtBytes(n) {
  if (!n) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const i = Math.min(units.length - 1, Math.floor(Math.log(n) / Math.log(1024)))
  const v = n / 1024 ** i
  return `${v >= 100 ? v.toFixed(0) : v.toFixed(1)} ${units[i]}`
}

export function fmtRate(bps) {
  return `${fmtBytes(bps)}/s`
}

export function fmtUptime(sec) {
  const d = Math.floor(sec / 86400)
  const h = String(Math.floor((sec % 86400) / 3600)).padStart(2, '0')
  const m = String(Math.floor((sec % 3600) / 60)).padStart(2, '0')
  const s = String(Math.floor(sec % 60)).padStart(2, '0')
  return d > 0 ? `${d}D ${h}:${m}:${s}` : `${h}:${m}:${s}`
}

export function fmtPct(n) {
  return `${Number(n || 0).toFixed(1)}%`
}

export function monogram(name) {
  return (name || '?').trim().charAt(0).toUpperCase()
}

const IMG_ICON_RE = /^((https?:)?\/\/|\/|\.\.\/|\.\/)|\.(png|jpe?g|svg|webp|gif|ico|avif)([?#].*)?$/i

export function isImageIcon(icon) {
  return typeof icon === 'string' && icon.trim() !== '' && IMG_ICON_RE.test(icon.trim())
}

/* 图标图片地址归一化：在线链接与 / 开头绝对路径原样返回，
   icons/x.svg、./icons/x.svg 统一为 /icons/x.svg（后端固定路由） */
export function iconSrc(icon) {
  const v = icon.trim()
  if (/^(https?:)?\/\//.test(v) || v.startsWith('/')) return v
  return '/' + v.replace(/^\.\//, '')
}

export function toPascal(name) {
  return name.trim().split(/[-_\s]+/).map(w => w.charAt(0).toUpperCase() + w.slice(1)).join('')
}
