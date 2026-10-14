import { describe, it, expect } from 'vitest'
import { getUIDefault } from '../getUIDefault'

describe('getUIDefault', () => {
  it('returns empty string for missing defaults', () => {
    expect(getUIDefault(undefined)).toBe('')
    expect(getUIDefault(null)).toBe('')
  })

  it('passes plain values through', () => {
    expect(getUIDefault('8080')).toBe('8080')
    expect(getUIDefault(8080)).toBe('8080')
    expect(getUIDefault('info')).toBe('info')
  })

  it('resolves else branches', () => {
    expect(getUIDefault('{{ if .X }}a{{ else }}fallback{{ end }}')).toBe('fallback')
  })

  it('resolves template function defaults', () => {
    expect(getUIDefault('{{ defaultPort 8080 }}')).toBe('8080')
  })

  it('returns empty string for unresolvable templates', () => {
    expect(getUIDefault('{{ .Something }}')).toBe('')
  })
})
