<template>
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <div
    v-else
    class="relative min-h-screen overflow-hidden bg-[radial-gradient(circle_at_top,rgba(136,77,255,0.14),transparent_30%),linear-gradient(180deg,#fcfbff_0%,#f5f4fb_48%,#efedf8_100%)] text-gray-900 dark:bg-[radial-gradient(circle_at_top,rgba(136,77,255,0.18),transparent_25%),linear-gradient(180deg,#10091d_0%,#140d24_46%,#0a0713_100%)] dark:text-white"
  >
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div class="absolute inset-0 bg-[linear-gradient(rgba(122,94,255,0.055)_1px,transparent_1px),linear-gradient(90deg,rgba(122,94,255,0.055)_1px,transparent_1px)] bg-[size:72px_72px] dark:bg-[linear-gradient(rgba(144,118,255,0.06)_1px,transparent_1px),linear-gradient(90deg,rgba(144,118,255,0.06)_1px,transparent_1px)]"></div>
      <div class="absolute left-[-8rem] top-24 h-72 w-72 rounded-full bg-primary-300/20 blur-3xl dark:bg-primary-500/20"></div>
      <div class="absolute right-[-6rem] top-12 h-80 w-80 rounded-full bg-fuchsia-300/20 blur-3xl dark:bg-fuchsia-500/15"></div>
      <div class="absolute bottom-[-8rem] left-1/2 h-96 w-96 -translate-x-1/2 rounded-full bg-primary-200/30 blur-3xl dark:bg-primary-700/20"></div>
    </div>

    <header class="relative z-20 px-6 py-5">
      <nav class="mx-auto flex max-w-7xl items-center justify-between">
        <div class="flex items-center gap-4">
          <div class="flex h-12 w-12 items-center justify-center overflow-hidden rounded-2xl border border-white/70 bg-white/80 shadow-[0_16px_40px_rgba(58,33,113,0.12)] backdrop-blur dark:border-white/10 dark:bg-white/5 dark:shadow-[0_16px_50px_rgba(0,0,0,0.35)]">
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-9 w-9 object-contain" />
          </div>
          <div>
            <p class="font-display text-lg font-semibold tracking-[-0.03em] text-gray-950 dark:text-white">
              {{ siteName }}
            </p>
            <p class="text-sm text-gray-500 dark:text-dark-300">{{ t('home.tags.subscriptionToApi') }}</p>
          </div>
        </div>

        <div class="flex items-center gap-3">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex h-11 w-11 items-center justify-center rounded-2xl border border-gray-200/70 bg-white/80 text-gray-600 shadow-sm backdrop-blur transition hover:border-primary-200 hover:text-primary-700 dark:border-white/10 dark:bg-white/5 dark:text-dark-300 dark:hover:border-primary-500/40 dark:hover:text-primary-200"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <button
            @click="toggleTheme"
            class="inline-flex h-11 w-11 items-center justify-center rounded-2xl border border-gray-200/70 bg-white/80 text-gray-600 shadow-sm backdrop-blur transition hover:border-primary-200 hover:text-primary-700 dark:border-white/10 dark:bg-white/5 dark:text-dark-300 dark:hover:border-primary-500/40 dark:hover:text-primary-200"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex items-center gap-2 rounded-full border border-primary-500/10 bg-gray-950 px-5 py-2.5 text-sm font-medium text-white shadow-[0_14px_28px_rgba(71,46,140,0.25)] transition hover:bg-primary-700 dark:border-primary-400/20 dark:bg-primary-500 dark:hover:bg-primary-400"
          >
            <span
              v-if="isAuthenticated"
              class="flex h-6 w-6 items-center justify-center rounded-full bg-white/15 text-[11px] font-semibold"
            >
              {{ userInitial }}
            </span>
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="relative z-10 px-6 pb-10 pt-6 md:pt-10">
      <section class="mx-auto flex min-h-[calc(100vh-6rem)] max-w-7xl items-center">
        <div class="grid w-full items-center gap-12 lg:grid-cols-[1.05fr_0.95fr]">
          <div class="max-w-3xl">
            <div class="inline-flex items-center gap-2 rounded-full border border-primary-200/70 bg-white/80 px-4 py-2 text-xs font-semibold uppercase tracking-[0.24em] text-primary-700 shadow-sm backdrop-blur dark:border-primary-500/20 dark:bg-white/5 dark:text-primary-200">
              <span class="h-2 w-2 rounded-full bg-primary-500"></span>
              {{ t('home.tags.subscriptionToApi') }}
            </div>

            <h1 class="font-display mt-8 text-5xl font-semibold leading-[0.98] tracking-[-0.05em] text-gray-950 dark:text-white md:text-6xl lg:text-7xl">
              {{ t('home.heroSubtitle') }}
            </h1>
            <p class="mt-6 max-w-2xl text-lg leading-8 text-gray-600 dark:text-dark-300 md:text-xl">
              {{ t('home.heroDescription') }}
            </p>

            <div class="mt-10 flex flex-col gap-4 sm:flex-row">
              <router-link
                :to="isAuthenticated ? dashboardPath : '/login'"
                class="inline-flex items-center justify-center rounded-full bg-gradient-to-r from-primary-600 via-primary-500 to-fuchsia-500 px-7 py-3.5 text-sm font-semibold text-white shadow-[0_22px_45px_rgba(101,62,196,0.32)] transition hover:translate-y-[-1px] hover:shadow-[0_28px_55px_rgba(101,62,196,0.38)]"
              >
                {{ isAuthenticated ? t('home.goToDashboard') : t('home.cta.button') }}
                <Icon name="arrowRight" size="md" class="ml-2" :stroke-width="2" />
              </router-link>
              <a
                v-if="docUrl"
                :href="docUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center justify-center rounded-full border border-gray-200/80 bg-white/80 px-7 py-3.5 text-sm font-semibold text-gray-700 shadow-sm backdrop-blur transition hover:border-primary-200 hover:text-primary-700 dark:border-white/10 dark:bg-white/5 dark:text-dark-200 dark:hover:border-primary-500/30 dark:hover:text-primary-200"
              >
                {{ t('home.viewDocs') }}
              </a>
            </div>

            <div class="mt-10 flex flex-wrap gap-3">
              <div
                v-for="tag in heroTags"
                :key="tag"
                class="rounded-full border border-white/70 bg-white/70 px-4 py-3 text-sm text-gray-600 shadow-sm backdrop-blur dark:border-white/10 dark:bg-white/[0.04] dark:text-dark-300"
              >
                <span class="font-semibold text-gray-950 dark:text-white">{{ tag }}</span>
              </div>
            </div>
          </div>

          <div class="relative">
            <div class="absolute left-6 top-6 h-32 w-32 rounded-full bg-primary-400/20 blur-3xl dark:bg-primary-500/20"></div>
            <div class="absolute bottom-8 right-6 h-32 w-32 rounded-full bg-fuchsia-400/20 blur-3xl dark:bg-fuchsia-500/15"></div>

            <div class="relative rounded-[2rem] border border-white/70 bg-white/82 p-6 shadow-[0_30px_90px_rgba(34,16,74,0.16)] backdrop-blur dark:border-white/10 dark:bg-[#161123]/85 dark:shadow-[0_30px_90px_rgba(0,0,0,0.45)]">
              <div class="flex items-start justify-between gap-4 border-b border-gray-200/80 pb-5 dark:border-white/10">
                <div>
                  <p class="text-xs font-semibold uppercase tracking-[0.22em] text-primary-700 dark:text-primary-300">
                    {{ t('home.providers.title') }}
                  </p>
                  <h2 class="mt-3 font-display text-2xl font-semibold tracking-[-0.03em] text-gray-950 dark:text-white">
                    {{ t('home.providers.description') }}
                  </h2>
                </div>
                <div class="rounded-full border border-primary-200 bg-primary-50 px-3 py-1 text-xs font-medium text-primary-700 dark:border-primary-500/20 dark:bg-primary-500/10 dark:text-primary-200">
                  {{ t('home.providers.supported') }}
                </div>
              </div>

              <div class="mt-5 space-y-4">
                <div
                  v-for="feature in heroFeatures"
                  :key="feature.title"
                  class="rounded-3xl border border-gray-200/80 bg-white/90 p-5 shadow-sm dark:border-white/10 dark:bg-white/[0.04]"
                >
                  <div class="flex items-start justify-between gap-4">
                    <div>
                      <p class="text-xs font-semibold uppercase tracking-[0.18em] text-gray-400 dark:text-dark-400">
                        {{ feature.kicker }}
                      </p>
                      <h3 class="mt-2 text-lg font-semibold text-gray-950 dark:text-white">{{ feature.title }}</h3>
                    </div>
                    <div class="rounded-2xl p-2.5" :class="feature.iconBg">
                      <Icon :name="feature.icon" size="sm" class="text-white" />
                    </div>
                  </div>
                  <p class="mt-3 text-sm leading-7 text-gray-500 dark:text-dark-300">{{ feature.desc }}</p>
                </div>
              </div>

              <div class="mt-5 flex flex-wrap gap-2">
                <span
                  v-for="provider in providerNames"
                  :key="provider"
                  class="rounded-full bg-[#f6f2ff] px-4 py-2 text-sm font-medium text-primary-700 dark:bg-white/[0.06] dark:text-primary-200"
                >
                  {{ provider }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isDark = ref(document.documentElement.classList.contains('dark'))

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

const heroTags = computed(() => ([
  t('home.tags.subscriptionToApi'),
  t('home.tags.stickySession'),
  t('home.tags.realtimeBilling')
]))

const heroFeatures = computed(() => ([
  {
    kicker: '01',
    title: t('home.features.unifiedGateway'),
    desc: t('home.features.unifiedGatewayDesc'),
    icon: 'server' as const,
    iconBg: 'bg-gradient-to-br from-primary-500 to-primary-700'
  },
  {
    kicker: '02',
    title: t('home.features.multiAccount'),
    desc: t('home.features.multiAccountDesc'),
    icon: 'shield' as const,
    iconBg: 'bg-gradient-to-br from-slate-600 to-slate-900'
  },
  {
    kicker: '03',
    title: t('home.features.balanceQuota'),
    desc: t('home.features.balanceQuotaDesc'),
    icon: 'creditCard' as const,
    iconBg: 'bg-gradient-to-br from-fuchsia-500 to-primary-500'
  }
]))

const providerNames = computed(() => ([
  t('home.providers.chatgpt'),
  t('home.providers.claude'),
  t('home.providers.gemini'),
  t('home.providers.antigravity'),
  t('home.providers.more')
]))

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>
