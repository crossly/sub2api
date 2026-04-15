import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('HomeView mobile nav menu guard', () => {
  it('moves secondary header actions into a mobile dropdown menu instead of wrapping them into another row', () => {
    const source = readFileSync(resolve(__dirname, '../HomeView.vue'), 'utf-8')

    expect(source).toContain('sm:hidden')
    expect(source).toContain('hidden items-center gap-3 sm:flex')
    expect(source).toContain('mobileMenuOpen')
    expect(source).toContain("<transition name=\"mobile-menu\">")
    expect(source).toContain("v-if=\"mobileMenuOpen\"")
    expect(source).toContain("@click=\"closeMobileMenu\"")
    expect(source).toContain("t('home.viewDocs')")
  })
})
