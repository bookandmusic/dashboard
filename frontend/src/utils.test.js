import { describe, it, expect } from 'vitest'
import { resolveAddress, hostOf, fmtBytes, fmtRate, fmtUptime, fmtPct, monogram, isImageIcon, iconSrc } from './utils.js'

const dual = {
  name: '代码仓库',
  addresses: [
    { net: 'intranet', label: '内网', url: 'http://git.intra.example' },
    { net: 'internet', label: '外网', url: 'https://git.example.com' }
  ]
}
const lanOnly = {
  name: '制品仓库',
  addresses: [{ net: 'intranet', label: '内网', url: 'http://artifacts.intra.example' }]
}

describe('resolveAddress', () => {
  it('returns the address matching current mode', () => {
    expect(resolveAddress(dual, 'internet').url).toBe('https://git.example.com')
    expect(resolveAddress(dual, 'intranet').url).toBe('http://git.intra.example')
  })
  it('falls back to first address when mode missing', () => {
    expect(resolveAddress(lanOnly, 'internet').url).toBe('http://artifacts.intra.example')
  })
})

describe('hostOf', () => {
  it('strips protocol and trailing slash', () => {
    expect(hostOf('https://git.example.com/')).toBe('git.example.com')
    expect(hostOf('http://192.168.1.69:8096')).toBe('192.168.1.69:8096')
  })
})

describe('fmtBytes', () => {
  it('formats magnitudes', () => {
    expect(fmtBytes(0)).toBe('0 B')
    expect(fmtBytes(1024)).toBe('1.0 KB')
    expect(fmtBytes(64 * 1024 ** 4)).toBe('64.0 TB')
    expect(fmtBytes(512 * 1024 ** 3)).toBe('512 GB')
  })
})

describe('fmtRate', () => {
  it('appends per-second', () => {
    expect(fmtRate(1024 * 1024)).toBe('1.0 MB/s')
  })
})

describe('fmtUptime', () => {
  it('includes days when over 24h', () => {
    expect(fmtUptime(128 * 86400 + 7 * 3600 + 42 * 60 + 10)).toBe('128D 07:42:10')
  })
  it('omits days under 24h', () => {
    expect(fmtUptime(7 * 3600 + 2 * 60 + 3)).toBe('07:02:03')
  })
})

describe('fmtPct', () => {
  it('one decimal with percent sign', () => {
    expect(fmtPct(23.456)).toBe('23.5%')
    expect(fmtPct(null)).toBe('0.0%')
  })
})

describe('monogram', () => {
  it('uppercases first char', () => {
    expect(monogram('gitea')).toBe('G')
    expect(monogram('  思源笔记')).toBe('思')
    expect(monogram('')).toBe('?')
  })
})

describe('isImageIcon', () => {
  it('accepts urls, absolute and relative paths', () => {
    expect(isImageIcon('https://cdn.example.com/a.png')).toBe(true)
    expect(isImageIcon('//cdn.example.com/a.svg')).toBe(true)
    expect(isImageIcon('/icons/gitea.svg')).toBe(true)
    expect(isImageIcon('icons/gitea.svg')).toBe(true)
    expect(isImageIcon('./icons/gitea.svg')).toBe(true)
    expect(isImageIcon('logo')).toBe(false)
  })
  it('accepts bare filenames by extension', () => {
    expect(isImageIcon('gitea.svg')).toBe(true)
    expect(isImageIcon('a.PNG?v=1')).toBe(true)
  })
  it('rejects lucide names and empties', () => {
    expect(isImageIcon('git-branch')).toBe(false)
    expect(isImageIcon('arrow-left-right')).toBe(false)
    expect(isImageIcon('')).toBe(false)
    expect(isImageIcon(null)).toBe(false)
  })
})

describe('iconSrc', () => {
  it('keeps online links and absolute paths', () => {
    expect(iconSrc('https://cdn.example.com/a.png')).toBe('https://cdn.example.com/a.png')
    expect(iconSrc('//cdn.example.com/a.png')).toBe('//cdn.example.com/a.png')
    expect(iconSrc('/icons/gitea.svg')).toBe('/icons/gitea.svg')
  })
  it('normalizes relative paths to /icons route', () => {
    expect(iconSrc('icons/gitea.svg')).toBe('/icons/gitea.svg')
    expect(iconSrc('./icons/gitea.svg')).toBe('/icons/gitea.svg')
    expect(iconSrc('gitea.svg')).toBe('/gitea.svg')
  })
})
