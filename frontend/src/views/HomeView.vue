<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Default Home Page -->
  <div v-else class="anime-home relative flex min-h-screen flex-col overflow-hidden text-slate-900 dark:text-white">
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div class="aurora aurora-a"></div>
      <div class="aurora aurora-b"></div>
      <div class="aurora aurora-c"></div>
      <div class="grid-glow"></div>
      <div class="float-capsule capsule-a">AI Startup</div>
      <div class="float-capsule capsule-b">API Gateway</div>
      <div class="float-capsule capsule-c">Realtime Billing</div>
      <div class="float-capsule capsule-d">Dream → Launch</div>
      <div class="star-dot star-a"></div>
      <div class="star-dot star-b"></div>
      <div class="star-dot star-c"></div>
    </div>

    <!-- Header -->
    <header class="relative z-20 px-4 py-5 sm:px-6">
      <nav class="mx-auto flex max-w-7xl items-center justify-between rounded-full border border-white/60 bg-white/55 px-4 py-3 shadow-2xl shadow-fuchsia-500/10 backdrop-blur-2xl dark:border-white/10 dark:bg-slate-950/35">
        <!-- Logo -->
        <div class="flex items-center gap-3">
          <div class="relative h-11 w-11 overflow-hidden rounded-2xl bg-white shadow-lg shadow-cyan-400/20 ring-2 ring-white/80 dark:bg-slate-900 dark:ring-white/10">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain p-1" />
          </div>
          <div class="hidden leading-tight sm:block">
            <div class="text-sm font-black tracking-wide text-slate-900 dark:text-white">{{ siteName }}</div>
            <div class="text-[11px] font-semibold uppercase tracking-[0.22em] text-fuchsia-500">Creator Cloud</div>
          </div>
        </div>

        <!-- Nav Actions -->
        <div class="flex items-center gap-2 sm:gap-3">
          <!-- Language Switcher -->
          <LocaleSwitcher />

          <!-- Doc Link -->
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="home-icon-btn"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>

          <!-- Theme Toggle -->
          <button
            @click="toggleTheme"
            class="home-icon-btn"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>

          <!-- Login / Dashboard Button -->
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="inline-flex items-center gap-1.5 rounded-full bg-slate-950 py-1 pl-1 pr-3 text-white shadow-lg shadow-violet-500/25 transition-all hover:-translate-y-0.5 hover:bg-violet-950 dark:bg-white dark:text-slate-950"
          >
            <span class="flex h-6 w-6 items-center justify-center rounded-full bg-gradient-to-br from-cyan-300 via-fuchsia-400 to-orange-300 text-[10px] font-black text-white shadow-inner">
              {{ userInitial }}
            </span>
            <span class="text-xs font-bold">{{ t('home.dashboard') }}</span>
            <svg class="h-3 w-3 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 19.5l15-15m0 0H8.25m11.25 0v11.25" />
            </svg>
          </router-link>
          <router-link
            v-else
            to="/login"
            class="rounded-full bg-slate-950 px-4 py-2 text-xs font-black text-white shadow-lg shadow-fuchsia-500/25 transition-all hover:-translate-y-0.5 hover:bg-violet-950 dark:bg-white dark:text-slate-950"
          >
            {{ t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <!-- Main Content -->
    <main class="relative z-10 flex-1 px-4 pb-16 pt-10 sm:px-6 lg:pt-16">
      <div class="mx-auto max-w-7xl">
        <!-- Hero Section -->
        <div class="grid items-center gap-12 lg:grid-cols-[1.05fr_0.95fr] lg:gap-16">
          <!-- Left: Text Content -->
          <div class="text-center lg:text-left">
            <div class="mb-6 inline-flex items-center gap-2 rounded-full border border-white/70 bg-white/70 px-4 py-2 text-xs font-black uppercase tracking-[0.22em] text-violet-600 shadow-xl shadow-violet-500/10 backdrop-blur-xl dark:border-white/10 dark:bg-white/10 dark:text-cyan-200">
              <span class="h-2 w-2 rounded-full bg-gradient-to-r from-cyan-400 to-fuchsia-400 shadow-lg shadow-cyan-400/60"></span>
              Young AI founder workspace
            </div>

            <h1 class="hero-title relative z-10 mb-5 pb-2 text-5xl font-black leading-[1.08] tracking-tight text-slate-950 dark:text-white sm:text-6xl lg:text-7xl">
              {{ siteName }}
            </h1>
            <p class="mx-auto mb-8 max-w-2xl text-base font-medium leading-8 text-slate-600 dark:text-slate-300 sm:text-xl lg:mx-0">
              {{ siteSubtitle }}
            </p>

            <!-- CTA Button -->
            <div class="flex flex-col items-center gap-4 sm:flex-row lg:justify-start">
              <router-link
                :to="isAuthenticated ? dashboardPath : '/login'"
                class="group inline-flex items-center rounded-full bg-gradient-to-r from-cyan-400 via-fuchsia-500 to-orange-300 px-7 py-3.5 text-base font-black text-white shadow-2xl shadow-fuchsia-500/30 transition-all hover:-translate-y-1 hover:shadow-cyan-400/30"
              >
                {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
                <Icon name="arrowRight" size="md" class="ml-2 transition-transform group-hover:translate-x-1" :stroke-width="2.5" />
              </router-link>
              <div class="rounded-full border border-white/70 bg-white/55 px-5 py-3 text-sm font-bold text-slate-600 shadow-lg backdrop-blur-xl dark:border-white/10 dark:bg-white/10 dark:text-slate-200">
                Build fast. Route smart. Launch cute.
              </div>
            </div>
          </div>

          <!-- Right: Anime Startup Console -->
          <div class="flex justify-center lg:justify-end">
            <div class="hero-card relative w-full max-w-[500px]">
              <div class="mascot-orb">
                <div class="mascot-face">
                  <span class="eye eye-left"></span>
                  <span class="eye eye-right"></span>
                  <span class="mouth"></span>
                </div>
              </div>
              <div class="terminal-container">
                <div class="terminal-window">
                  <div class="terminal-header">
                    <div class="terminal-buttons">
                      <span class="btn-close"></span>
                      <span class="btn-minimize"></span>
                      <span class="btn-maximize"></span>
                    </div>
                    <span class="terminal-title">startup-console</span>
                  </div>
                  <div class="terminal-body">
                    <div class="code-line line-1">
                      <span class="code-prompt">$</span>
                      <span class="code-cmd">launch</span>
                      <span class="code-flag">--team young</span>
                      <span class="code-url">/ai/gateway</span>
                    </div>
                    <div class="code-line line-2">
                      <span class="code-comment"># Floating capsules online...</span>
                    </div>
                    <div class="code-line line-3">
                      <span class="code-success">200 OK</span>
                      <span class="code-response">{ "vibe": "anime startup" }</span>
                    </div>
                    <div class="code-line line-4">
                      <span class="code-prompt">$</span>
                      <span class="cursor"></span>
                    </div>
                  </div>
                </div>
              </div>
              <div class="mini-card mini-card-a">MVP</div>
              <div class="mini-card mini-card-b">API</div>
              <div class="mini-card mini-card-c">Billing</div>
            </div>
          </div>
        </div>

        <!-- Feature Tags -->
        <div class="mt-16 flex flex-wrap items-center justify-center gap-4 md:gap-6">
          <div class="pill-card pill-cyan">
            <Icon name="swap" size="sm" />
            <span>{{ t('home.tags.subscriptionToApi') }}</span>
          </div>
          <div class="pill-card pill-violet">
            <Icon name="shield" size="sm" />
            <span>{{ t('home.tags.stickySession') }}</span>
          </div>
          <div class="pill-card pill-orange">
            <Icon name="chart" size="sm" />
            <span>{{ t('home.tags.realtimeBilling') }}</span>
          </div>
        </div>

        <!-- Features Grid -->
        <div class="mt-12 grid gap-6 md:grid-cols-3">
          <div class="feature-card feature-blue">
            <div class="feature-icon">
              <Icon name="server" size="lg" class="text-white" />
            </div>
            <h3>{{ t('home.features.unifiedGateway') }}</h3>
            <p>{{ t('home.features.unifiedGatewayDesc') }}</p>
          </div>

          <div class="feature-card feature-pink">
            <div class="feature-icon">
              <svg class="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z" />
              </svg>
            </div>
            <h3>{{ t('home.features.multiAccount') }}</h3>
            <p>{{ t('home.features.multiAccountDesc') }}</p>
          </div>

          <div class="feature-card feature-yellow">
            <div class="feature-icon">
              <svg class="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z" />
              </svg>
            </div>
            <h3>{{ t('home.features.balanceQuota') }}</h3>
            <p>{{ t('home.features.balanceQuotaDesc') }}</p>
          </div>
        </div>

        <!-- Supported Providers -->
        <section class="mt-16 rounded-[2rem] border border-white/60 bg-white/45 p-6 shadow-2xl shadow-cyan-500/10 backdrop-blur-2xl dark:border-white/10 dark:bg-white/5 sm:p-8">
          <div class="mb-8 text-center">
            <h2 class="text-3xl font-black tracking-tight text-slate-950 dark:text-white">
              {{ t('home.providers.title') }}
            </h2>
            <p class="mt-3 text-sm font-medium text-slate-600 dark:text-slate-300">
              {{ t('home.providers.description') }}
            </p>
          </div>

          <div class="flex flex-wrap items-center justify-center gap-4">
            <div class="provider-capsule">
              <div class="provider-avatar from-orange-300 to-rose-400">C</div>
              <span>{{ t('home.providers.claude') }}</span>
              <em>{{ t('home.providers.supported') }}</em>
            </div>
            <div class="provider-capsule">
              <div class="provider-avatar from-emerald-300 to-cyan-400">G</div>
              <span>GPT</span>
              <em>{{ t('home.providers.supported') }}</em>
            </div>
            <div class="provider-capsule">
              <div class="provider-avatar from-blue-300 to-violet-400">G</div>
              <span>{{ t('home.providers.gemini') }}</span>
              <em>{{ t('home.providers.supported') }}</em>
            </div>
            <div class="provider-capsule">
              <div class="provider-avatar from-fuchsia-400 to-pink-500">A</div>
              <span>{{ t('home.providers.antigravity') }}</span>
              <em>{{ t('home.providers.supported') }}</em>
            </div>
            <div class="provider-capsule opacity-70">
              <div class="provider-avatar from-slate-400 to-slate-600">+</div>
              <span>{{ t('home.providers.more') }}</span>
              <em>{{ t('home.providers.soon') }}</em>
            </div>
          </div>
        </section>
      </div>
    </main>

    <!-- Footer -->
    <footer class="relative z-10 px-6 py-8">
      <div class="mx-auto flex max-w-7xl flex-col items-center justify-center gap-4 rounded-full border border-white/50 bg-white/40 px-6 py-4 text-center shadow-lg backdrop-blur-xl dark:border-white/10 dark:bg-white/5 sm:flex-row sm:justify-between sm:text-left">
        <p class="text-sm font-semibold text-slate-500 dark:text-slate-400">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
        <div class="flex items-center gap-4">
          <span class="footer-link">联系 VX：Mictimeles</span>
        </div>
      </div>
    </footer>
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

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

// Theme
const isDark = ref(document.documentElement.classList.contains('dark'))

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

// Toggle theme
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

// Initialize theme
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

  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
.anime-home {
  background:
    radial-gradient(circle at 12% 12%, rgba(34, 211, 238, 0.35), transparent 30%),
    radial-gradient(circle at 85% 18%, rgba(217, 70, 239, 0.32), transparent 32%),
    radial-gradient(circle at 70% 78%, rgba(251, 146, 60, 0.25), transparent 34%),
    linear-gradient(135deg, #f8fbff 0%, #fff7fd 45%, #eff6ff 100%);
}

:deep(.dark) .anime-home,
.dark .anime-home {
  background:
    radial-gradient(circle at 16% 12%, rgba(34, 211, 238, 0.22), transparent 30%),
    radial-gradient(circle at 85% 16%, rgba(217, 70, 239, 0.22), transparent 32%),
    radial-gradient(circle at 70% 78%, rgba(251, 146, 60, 0.16), transparent 34%),
    linear-gradient(135deg, #06111f 0%, #160b2e 48%, #06111f 100%);
}

.aurora {
  position: absolute;
  border-radius: 9999px;
  filter: blur(28px);
  opacity: 0.7;
  animation: drift 9s ease-in-out infinite;
}

.aurora-a {
  left: -7rem;
  top: 10rem;
  width: 22rem;
  height: 22rem;
  background: rgba(34, 211, 238, 0.42);
}

.aurora-b {
  right: -8rem;
  top: 4rem;
  width: 26rem;
  height: 26rem;
  background: rgba(217, 70, 239, 0.36);
  animation-delay: -2s;
}

.aurora-c {
  bottom: -8rem;
  left: 42%;
  width: 24rem;
  height: 24rem;
  background: rgba(251, 146, 60, 0.3);
  animation-delay: -4s;
}

.grid-glow {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(124, 58, 237, 0.08) 1px, transparent 1px),
    linear-gradient(90deg, rgba(14, 165, 233, 0.08) 1px, transparent 1px);
  background-size: 44px 44px;
  mask-image: linear-gradient(to bottom, black, transparent 80%);
}

.float-capsule,
.mini-card,
.pill-card,
.provider-capsule {
  border: 1px solid rgba(255, 255, 255, 0.68);
  background: rgba(255, 255, 255, 0.58);
  box-shadow: 0 18px 50px rgba(99, 102, 241, 0.14);
  backdrop-filter: blur(18px);
}

.float-capsule {
  position: absolute;
  border-radius: 9999px;
  padding: 0.75rem 1rem;
  font-size: 0.72rem;
  font-weight: 900;
  color: rgb(76, 29, 149);
  animation: floaty 5.5s ease-in-out infinite;
}

.capsule-a { left: 8%; top: 22%; color: #0891b2; }
.capsule-b { right: 10%; top: 30%; color: #c026d3; animation-delay: -1.1s; }
.capsule-c { left: 14%; bottom: 26%; color: #ea580c; animation-delay: -2.2s; }
.capsule-d { right: 16%; bottom: 18%; color: #7c3aed; animation-delay: -3.2s; }

.star-dot {
  position: absolute;
  width: 0.55rem;
  height: 0.55rem;
  border-radius: 9999px;
  background: white;
  box-shadow: 0 0 24px rgba(255, 255, 255, 0.9);
  animation: twinkle 2.4s ease-in-out infinite;
}

.star-a { left: 28%; top: 16%; }
.star-b { right: 32%; top: 52%; animation-delay: -0.8s; }
.star-c { left: 54%; bottom: 24%; animation-delay: -1.4s; }

.home-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  padding: 0.55rem;
  color: rgb(100, 116, 139);
  background: rgba(255, 255, 255, 0.52);
  border: 1px solid rgba(255, 255, 255, 0.7);
  transition: all 0.2s ease;
}

.home-icon-btn:hover {
  transform: translateY(-2px);
  color: rgb(124, 58, 237);
  box-shadow: 0 12px 30px rgba(124, 58, 237, 0.14);
}

.hero-title {
  background: linear-gradient(90deg, #0f172a, #7c3aed 38%, #db2777 68%, #0891b2);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  text-shadow: 0 18px 50px rgba(217, 70, 239, 0.18);
}

:deep(.dark) .hero-title,
.dark .hero-title {
  background: linear-gradient(90deg, #ffffff, #a5f3fc 34%, #f0abfc 65%, #fed7aa);
  -webkit-background-clip: text;
  background-clip: text;
}

.hero-card {
  min-height: 410px;
}

.mascot-orb {
  position: absolute;
  right: 1rem;
  top: -1.5rem;
  z-index: 3;
  display: flex;
  width: 6.5rem;
  height: 6.5rem;
  align-items: center;
  justify-content: center;
  border-radius: 2rem;
  background: linear-gradient(135deg, rgba(34, 211, 238, 0.95), rgba(217, 70, 239, 0.92), rgba(251, 146, 60, 0.88));
  box-shadow: 0 24px 60px rgba(217, 70, 239, 0.35);
  transform: rotate(10deg);
  animation: floaty 4.8s ease-in-out infinite;
}

.mascot-face {
  position: relative;
  width: 4.3rem;
  height: 3.6rem;
  border-radius: 1.4rem;
  background: rgba(255, 255, 255, 0.92);
}

.eye {
  position: absolute;
  top: 1.18rem;
  width: 0.55rem;
  height: 0.8rem;
  border-radius: 9999px;
  background: #111827;
}

.eye-left { left: 1.05rem; }
.eye-right { right: 1.05rem; }

.mouth {
  position: absolute;
  bottom: 0.9rem;
  left: 50%;
  width: 1.1rem;
  height: 0.45rem;
  border-bottom: 3px solid #111827;
  border-radius: 9999px;
  transform: translateX(-50%);
}

.terminal-container {
  position: relative;
  display: inline-block;
  width: 100%;
  padding-top: 3rem;
}

.terminal-window {
  width: min(100%, 460px);
  margin-inline: auto;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.22);
  border-radius: 2rem;
  background: linear-gradient(145deg, rgba(15, 23, 42, 0.95), rgba(49, 46, 129, 0.9));
  box-shadow:
    0 35px 90px rgba(79, 70, 229, 0.28),
    0 0 0 8px rgba(255, 255, 255, 0.18),
    inset 0 1px 0 rgba(255, 255, 255, 0.16);
  transform: perspective(1000px) rotateX(3deg) rotateY(-5deg);
  transition: transform 0.3s ease;
}

.terminal-window:hover {
  transform: perspective(1000px) rotateX(0deg) rotateY(0deg) translateY(-5px);
}

.terminal-header {
  display: flex;
  align-items: center;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(255, 255, 255, 0.08);
  padding: 14px 18px;
}

.terminal-buttons {
  display: flex;
  gap: 8px;
}

.terminal-buttons span {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.btn-close { background: #fb7185; }
.btn-minimize { background: #facc15; }
.btn-maximize { background: #34d399; }

.terminal-title {
  flex: 1;
  margin-right: 54px;
  text-align: center;
  font-family: ui-monospace, monospace;
  font-size: 12px;
  font-weight: 800;
  color: rgba(255, 255, 255, 0.55);
}

.terminal-body {
  padding: 26px 28px 30px;
  font-family: ui-monospace, 'Fira Code', monospace;
  font-size: 14px;
  line-height: 2.1;
}

.code-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  opacity: 0;
  animation: line-appear 0.5s ease forwards;
}

.line-1 { animation-delay: 0.3s; }
.line-2 { animation-delay: 1s; }
.line-3 { animation-delay: 1.8s; }
.line-4 { animation-delay: 2.5s; }

.code-prompt { color: #67e8f9; font-weight: 900; }
.code-cmd { color: #f0abfc; }
.code-flag { color: #fdba74; }
.code-url { color: #86efac; }
.code-comment { color: #94a3b8; font-style: italic; }
.code-success {
  border-radius: 9999px;
  background: rgba(52, 211, 153, 0.18);
  padding: 2px 10px;
  color: #86efac;
  font-weight: 900;
}
.code-response { color: #fde68a; }

.cursor {
  display: inline-block;
  width: 8px;
  height: 16px;
  background: #67e8f9;
  animation: blink 1s step-end infinite;
}

.mini-card {
  position: absolute;
  z-index: 4;
  border-radius: 9999px;
  padding: 0.65rem 0.95rem;
  font-size: 0.75rem;
  font-weight: 1000;
  color: #4c1d95;
  animation: floaty 5s ease-in-out infinite;
}

.mini-card-a { left: 0.2rem; top: 7rem; color: #0891b2; }
.mini-card-b { right: 0.4rem; bottom: 5rem; color: #c026d3; animation-delay: -1.4s; }
.mini-card-c { left: 3rem; bottom: 1.5rem; color: #ea580c; animation-delay: -2.4s; }

.pill-card {
  display: inline-flex;
  align-items: center;
  gap: 0.65rem;
  border-radius: 9999px;
  padding: 0.85rem 1.25rem;
  font-size: 0.9rem;
  font-weight: 900;
  color: #334155;
  transition: all 0.25s ease;
}

.pill-card:hover,
.feature-card:hover,
.provider-capsule:hover {
  transform: translateY(-5px);
}

.pill-cyan { color: #0891b2; }
.pill-violet { color: #7c3aed; }
.pill-orange { color: #ea580c; }

.feature-card {
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.62);
  border-radius: 2rem;
  background: rgba(255, 255, 255, 0.58);
  padding: 1.6rem;
  box-shadow: 0 24px 70px rgba(99, 102, 241, 0.12);
  backdrop-filter: blur(18px);
  transition: all 0.25s ease;
}

.feature-card::before {
  position: absolute;
  inset: -40% -20% auto auto;
  width: 12rem;
  height: 12rem;
  content: '';
  border-radius: 9999px;
  opacity: 0.2;
  filter: blur(2px);
}

.feature-blue::before { background: #22d3ee; }
.feature-pink::before { background: #e879f9; }
.feature-yellow::before { background: #fb923c; }

.feature-icon {
  display: flex;
  width: 3.2rem;
  height: 3.2rem;
  align-items: center;
  justify-content: center;
  border-radius: 1.25rem;
  background: linear-gradient(135deg, #22d3ee, #a855f7, #fb7185);
  box-shadow: 0 18px 35px rgba(168, 85, 247, 0.25);
}

.feature-card h3 {
  margin-top: 1rem;
  font-size: 1.1rem;
  font-weight: 1000;
  color: #0f172a;
}

.feature-card p {
  margin-top: 0.55rem;
  color: #64748b;
  font-size: 0.9rem;
  line-height: 1.75;
}

.provider-capsule {
  display: inline-flex;
  align-items: center;
  gap: 0.7rem;
  border-radius: 9999px;
  padding: 0.75rem 1rem;
  font-size: 0.9rem;
  font-weight: 900;
  color: #334155;
  transition: all 0.25s ease;
}

.provider-avatar {
  display: flex;
  width: 2.15rem;
  height: 2.15rem;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  background-image: linear-gradient(135deg, var(--tw-gradient-stops));
  color: white;
  font-size: 0.8rem;
  font-weight: 1000;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.3);
}

.provider-capsule em {
  border-radius: 9999px;
  background: rgba(34, 211, 238, 0.14);
  padding: 0.2rem 0.5rem;
  color: #0891b2;
  font-size: 0.65rem;
  font-style: normal;
  font-weight: 1000;
}

.footer-link {
  font-size: 0.9rem;
  font-weight: 800;
  color: #64748b;
  transition: color 0.2s ease;
}

.footer-link:hover {
  color: #7c3aed;
}

:deep(.dark) .float-capsule,
:deep(.dark) .mini-card,
:deep(.dark) .pill-card,
:deep(.dark) .provider-capsule,
:deep(.dark) .feature-card,
.dark .float-capsule,
.dark .mini-card,
.dark .pill-card,
.dark .provider-capsule,
.dark .feature-card {
  border-color: rgba(255, 255, 255, 0.1);
  background: rgba(15, 23, 42, 0.58);
  color: #e2e8f0;
}

:deep(.dark) .feature-card h3,
.dark .feature-card h3 {
  color: #ffffff;
}

:deep(.dark) .feature-card p,
.dark .feature-card p {
  color: #cbd5e1;
}

@keyframes drift {
  0%, 100% { transform: translate3d(0, 0, 0) scale(1); }
  50% { transform: translate3d(18px, -16px, 0) scale(1.05); }
}

@keyframes floaty {
  0%, 100% { transform: translateY(0) rotate(-2deg); }
  50% { transform: translateY(-14px) rotate(2deg); }
}

@keyframes twinkle {
  0%, 100% { opacity: 0.35; transform: scale(0.8); }
  50% { opacity: 1; transform: scale(1.2); }
}

@keyframes line-appear {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes blink {
  0%, 50% { opacity: 1; }
  51%, 100% { opacity: 0; }
}

@media (max-width: 640px) {
  .float-capsule,
  .mini-card {
    display: none;
  }

  .terminal-body {
    padding: 22px 18px 24px;
    font-size: 12px;
  }

  .mascot-orb {
    width: 5rem;
    height: 5rem;
    border-radius: 1.5rem;
  }
}
</style>
