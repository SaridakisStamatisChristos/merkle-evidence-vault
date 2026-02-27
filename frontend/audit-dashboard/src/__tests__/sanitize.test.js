import { describe, it, expect } from 'vitest'
import { JSDOM } from 'jsdom'
import sanitizeHTML from '../sanitize'

describe('sanitizeHTML', () => {
  it('removes script tags and dangerous attributes', () => {
    const dom = new JSDOM(`<!doctype html><html><body></body></html>`)
    const win = dom.window
    const dangerous = `<div onclick="alert('xss')">click</div><script>alert(1)</script>`
    const cleaned = sanitizeHTML(win, dangerous)
    expect(cleaned).not.toContain('<script>')
    expect(cleaned).not.toContain('onclick')
    expect(cleaned).toContain('click')
  })
})
