import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('AppHeader mobile responsive guard', () => {
  it('should keep the right action area from squeezing on mobile', () => {
    const source = readFileSync(resolve(__dirname, '../AppHeader.vue'), 'utf-8')

    expect(source).toMatch(/flex h-16[^\n]*justify-between[^\n]*gap-3[^\n]*px-4[^\n]*md:px-6/)
    expect(source).toMatch(/flex[^\n]*min-w-0[^\n]*flex-1[^\n]*gap-3[^\n]*sm:gap-4/)
    expect(source).toMatch(/ml-2[^\n]*flex[^\n]*min-w-0[^\n]*flex-shrink-0[^\n]*gap-2[^\n]*sm:gap-3/)
    expect(source).toContain('hidden md:flex items-center gap-1.5')
    expect(source).toContain('class="hidden md:block"')
  })
})
