import { describe, it, expect } from 'vitest'
import { formatEventType } from '../formatEventType'

describe('formatEventType', () => {
  it('replaces underscores with spaces', () => {
    expect(formatEventType('port_scan')).toBe('port scan')
    expect(formatEventType('login_attempt')).toBe('login attempt')
  })

  it('passes plain strings through', () => {
    expect(formatEventType('critical')).toBe('critical')
  })

  it('handles empty input', () => {
    expect(formatEventType('')).toBe('')
  })
})
