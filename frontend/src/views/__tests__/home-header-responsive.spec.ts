import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('HomeView mobile nav menu guard', () => {
  it('uses a dropdown-style mobile menu for secondary actions instead of a custom expanded panel', () => {
    const source = readFileSync(resolve(__dirname, '../HomeView.vue'), 'utf-8')

    expect(source).toContain('sm:hidden')
    expect(source).toContain('hidden items-center gap-3 sm:flex')
    expect(source).toContain('mobileMenuOpen')
    expect(source).toContain('<transition name="dropdown">')
    expect(source).toContain('absolute right-0 z-50 mt-2')
    expect(source).toContain('closeMobileMenu')
    expect(source).not.toContain('mobile-menu-enter-active')
  })
})
