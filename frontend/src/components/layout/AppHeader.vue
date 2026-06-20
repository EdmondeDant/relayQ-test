<template>
  <header class="sticky top-0 z-30 px-3 pt-3 md:px-4 lg:px-6">
    <div class="header-shell flex min-h-[74px] items-center justify-between gap-4 px-4 md:px-6">
      <!-- Left: Mobile Menu Toggle + Page Title -->
      <div class="flex items-center gap-4">
        <button
          @click="toggleMobileSidebar"
          class="btn-ghost btn-icon lg:hidden"
          aria-label="Toggle Menu"
        >
          <Icon name="menu" size="md" />
        </button>

        <div class="hidden lg:block">
          <h1 class="header-page-title text-lg font-semibold">
            {{ pageTitle }}
          </h1>
          <p v-if="pageDescription" class="header-page-description text-xs">
            {{ pageDescription }}
          </p>
        </div>
      </div>

      <!-- Right: Announcements + Docs + Language + Subscriptions + Balance + User Dropdown -->
      <div class="flex items-center gap-3">
        <!-- Starter Install -->
        <router-link
          v-if="user"
          to="/starter-install"
          :title="t('nav.newUserMustInstall')"
          class="group header-pill-link header-pill-link-primary"
        >
          <Icon name="sparkles" size="sm" />
          <span>{{ t('nav.newUserMustInstall') }}</span>
        </router-link>

        <!-- Contact Us -->
        <router-link
          v-if="user"
          to="/contact-support"
          :title="contactInfo || t('common.contactSupport')"
          class="group header-pill-link header-pill-link-secondary"
        >
          <Icon name="chat" size="sm" />
          <span>{{ t('nav.contactUs') }}</span>
        </router-link>

        <!-- Announcement Bell -->
        <AnnouncementBell v-if="user" />

        <!-- Docs Link -->
        <a
          v-if="docUrl"
          :href="docUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-sm font-medium text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
        >
          <Icon name="book" size="sm" />
          <span class="hidden sm:inline">{{ t('nav.docs') }}</span>
        </a>

        <!-- Language Switcher -->
        <LocaleSwitcher />

        <!-- Subscription Progress (for users with active subscriptions) -->
        <SubscriptionProgressMini v-if="user" />

        <!-- Balance Display -->
        <div
          v-if="user"
          class="hidden items-center gap-2 rounded-xl bg-primary-50 px-3 py-1.5 dark:bg-primary-900/20 sm:flex"
        >
          <svg
            class="h-4 w-4 text-primary-600 dark:text-primary-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="1.5"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z"
            />
          </svg>
          <span class="text-sm font-semibold text-primary-700 dark:text-primary-300">
            ${{ user.balance?.toFixed(2) || '0.00' }}
          </span>
        </div>

        <!-- User Dropdown -->
        <div v-if="user" class="relative" ref="dropdownRef">
          <button
            @click="toggleDropdown"
            class="flex items-center gap-2 rounded-xl p-1.5 transition-colors hover:bg-gray-100 dark:hover:bg-dark-800"
            aria-label="User Menu"
          >
            <div class="flex h-8 w-8 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-primary-500 to-primary-600 text-sm font-medium text-white shadow-sm">
              <img
                v-if="avatarUrl"
                :src="avatarUrl"
                :alt="displayName"
                class="h-full w-full object-cover"
              >
              <span v-else>{{ userInitials }}</span>
            </div>
            <div class="hidden text-left md:block">
              <div class="text-sm font-medium text-gray-900 dark:text-white">
                {{ displayName }}
              </div>
              <div class="text-xs capitalize text-gray-500 dark:text-dark-400">
                {{ user.role }}
              </div>
            </div>
            <Icon name="chevronDown" size="sm" class="hidden text-gray-400 md:block" />
          </button>

          <!-- Dropdown Menu -->
          <transition name="dropdown">
            <div v-if="dropdownOpen" class="dropdown right-0 mt-2 w-56">
              <!-- User Info -->
              <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
                <div class="text-sm font-medium text-gray-900 dark:text-white">
                  {{ displayName }}
                </div>
                <div class="text-xs text-gray-500 dark:text-dark-400">{{ user.email }}</div>
              </div>

              <!-- Balance (mobile only) -->
              <div class="border-b border-gray-100 px-4 py-2 dark:border-dark-700 sm:hidden">
                <div class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('common.balance') }}
                </div>
                <div class="text-sm font-semibold text-primary-600 dark:text-primary-400">
                  ${{ user.balance?.toFixed(2) || '0.00' }}
                </div>
              </div>

              <div class="py-1">
                <router-link to="/profile" @click="closeDropdown" class="dropdown-item">
                  <Icon name="user" size="sm" />
                  {{ t('nav.profile') }}
                </router-link>

                <router-link to="/keys" @click="closeDropdown" class="dropdown-item">
                  <Icon name="key" size="sm" />
                  {{ t('nav.apiKeys') }}
                </router-link>

                <a
                  v-if="authStore.isAdmin"
                  href="https://github.com/Wei-Shaw/sub2api"
                  target="_blank"
                  rel="noopener noreferrer"
                  @click="closeDropdown"
                  class="dropdown-item"
                >
                  <svg class="h-4 w-4" fill="currentColor" viewBox="0 0 24 24">
                    <path
                      fill-rule="evenodd"
                      clip-rule="evenodd"
                      d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.17 6.839 9.49.5.092.682-.217.682-.482 0-.237-.008-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.167 22 16.418 22 12c0-5.523-4.477-10-10-10z"
                    />
                  </svg>
                  {{ t('nav.github') }}
                </a>

              </div>

              <!-- Contact Support (only show if configured) -->
              <div
                v-if="contactInfo"
                class="border-t border-gray-100 px-4 py-2.5 dark:border-dark-700"
              >
                <div class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                  <svg
                    class="h-3.5 w-3.5 flex-shrink-0"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    stroke-width="1.5"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="M20.25 8.511c.884.284 1.5 1.128 1.5 2.097v4.286c0 1.136-.847 2.1-1.98 2.193-.34.027-.68.052-1.02.072v3.091l-3-3c-1.354 0-2.694-.055-4.02-.163a2.115 2.115 0 01-.825-.242m9.345-8.334a2.126 2.126 0 00-.476-.095 48.64 48.64 0 00-8.048 0c-1.131.094-1.976 1.057-1.976 2.192v4.286c0 .837.46 1.58 1.155 1.951m9.345-8.334V6.637c0-1.621-1.152-3.026-2.76-3.235A48.455 48.455 0 0011.25 3c-2.115 0-4.198.137-6.24.402-1.608.209-2.76 1.614-2.76 3.235v6.226c0 1.621 1.152 3.026 2.76 3.235.577.075 1.157.14 1.74.194V21l4.155-4.155"
                    />
                  </svg>
                  <span>{{ t('common.contactSupport') }}:</span>
                  <span class="font-medium text-gray-700 dark:text-gray-300">{{
                    contactInfo
                  }}</span>
                </div>
              </div>

              <div v-if="showOnboardingButton" class="border-t border-gray-100 py-1 dark:border-dark-700">
                <button @click="handleReplayGuide" class="dropdown-item w-full">
                  <svg class="h-4 w-4" fill="currentColor" viewBox="0 0 24 24">
                    <path
                      d="M12 2a10 10 0 100 20 10 10 0 000-20zm0 14a1 1 0 110 2 1 1 0 010-2zm1.07-7.75c0-.6-.49-1.25-1.32-1.25-.7 0-1.22.4-1.43 1.02a1 1 0 11-1.9-.62A3.41 3.41 0 0111.8 5c2.02 0 3.25 1.4 3.25 2.9 0 2-1.83 2.55-2.43 3.12-.43.4-.47.75-.47 1.23a1 1 0 01-2 0c0-1 .16-1.82 1.1-2.7.69-.64 1.82-1.05 1.82-2.06z"
                    />
                  </svg>
                  {{ $t('onboarding.restartTour') }}
                </button>
              </div>

              <div class="border-t border-gray-100 py-1 dark:border-dark-700">
                <button
                  @click="handleLogout"
                  class="dropdown-item w-full text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
                >
                  <svg
                    class="h-4 w-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    stroke-width="1.5"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15M12 9l-3 3m0 0l3 3m-3-3h12.75"
                    />
                  </svg>
                  {{ t('nav.logout') }}
                </button>
              </div>
            </div>
          </transition>
        </div>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore, useAuthStore, useOnboardingStore } from '@/stores'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import SubscriptionProgressMini from '@/components/common/SubscriptionProgressMini.vue'
import AnnouncementBell from '@/components/common/AnnouncementBell.vue'
import Icon from '@/components/icons/Icon.vue'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const adminSettingsStore = useAdminSettingsStore()
const onboardingStore = useOnboardingStore()

const user = computed(() => authStore.user)
const dropdownOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)
const contactInfo = computed(() => appStore.contactInfo)
const docUrl = computed(() => appStore.docUrl)
const avatarUrl = computed(() => user.value?.avatar_url?.trim() || '')

// 只在标准模式的管理员下显示新手引导按钮
const showOnboardingButton = computed(() => {
  return !authStore.isSimpleMode && user.value?.role === 'admin'
})

const userInitials = computed(() => {
  if (!user.value) return ''
  // Prefer username, fallback to email
  if (user.value.username) {
    return user.value.username.substring(0, 2).toUpperCase()
  }
  if (user.value.email) {
    // Get the part before @ and take first 2 chars
    const localPart = user.value.email.split('@')[0]
    return localPart.substring(0, 2).toUpperCase()
  }
  return ''
})

const displayName = computed(() => {
  if (!user.value) return ''
  return user.value.username || user.value.email?.split('@')[0] || ''
})

const pageTitle = computed(() => {
  // For custom pages, use the menu item's label instead of generic "自定义页面"
  if (route.name === 'CustomPage') {
    const id = route.params.id as string
    const publicItems = appStore.cachedPublicSettings?.custom_menu_items ?? []
    const menuItem = publicItems.find((item) => item.id === id)
      ?? (authStore.isAdmin ? adminSettingsStore.customMenuItems.find((item) => item.id === id) : undefined)
    if (menuItem?.label) return menuItem.label
  }
  const titleKey = route.meta.titleKey as string
  if (titleKey) {
    return t(titleKey)
  }
  return (route.meta.title as string) || ''
})

const pageDescription = computed(() => {
  const descKey = route.meta.descriptionKey as string
  if (descKey) {
    return t(descKey)
  }
  return (route.meta.description as string) || ''
})

function toggleMobileSidebar() {
  appStore.toggleMobileSidebar()
}

function toggleDropdown() {
  dropdownOpen.value = !dropdownOpen.value
}

function closeDropdown() {
  dropdownOpen.value = false
}

async function handleLogout() {
  closeDropdown()
  try {
    await authStore.logout()
  } catch (error) {
    // Ignore logout errors - still redirect to login
    console.error('Logout error:', error)
  }
  await router.push('/login')
}

function handleReplayGuide() {
  closeDropdown()
  onboardingStore.replay()
}

function handleClickOutside(event: MouseEvent) {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) {
    closeDropdown()
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.header-shell {
  position: relative;
  isolation: isolate;
  overflow: visible;
  border: 1px solid rgba(255, 255, 255, 0.72);
  border-radius: 9999px;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.82), rgba(244, 247, 255, 0.66)),
    rgba(255, 255, 255, 0.72);
  box-shadow:
    0 24px 60px rgba(99, 102, 241, 0.14),
    0 14px 30px rgba(15, 23, 42, 0.08),
    inset 0 1px 0 rgba(255, 255, 255, 0.82),
    inset 0 -12px 24px rgba(255, 255, 255, 0.22);
  backdrop-filter: blur(22px) saturate(170%);
}

.header-shell::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background:
    radial-gradient(circle at 14% 50%, rgba(255, 255, 255, 0.56) 0%, rgba(255, 255, 255, 0) 28%),
    radial-gradient(circle at 86% 40%, rgba(191, 219, 254, 0.18) 0%, rgba(191, 219, 254, 0) 24%);
  opacity: 0.85;
  pointer-events: none;
  z-index: 0;
}

.header-shell::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background:
    linear-gradient(115deg, transparent 28%, rgba(255, 255, 255, 0.02) 38%, rgba(255, 255, 255, 0.28) 50%, rgba(255, 255, 255, 0.02) 62%, transparent 72%);
  background-size: 220% 100%;
  background-position: -140% 0;
  animation: header-shell-shimmer 9s ease-in-out infinite;
  pointer-events: none;
  z-index: 0;
}

.header-page-title {
  position: relative;
  z-index: 1;
  color: #4338ca;
  letter-spacing: -0.015em;
  text-shadow: 0 1px 0 rgba(255, 255, 255, 0.36);
}

.header-page-description {
  position: relative;
  z-index: 1;
  color: #7c3aed;
  opacity: 0.72;
}

.header-pill-link {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  overflow: hidden;
  border-radius: 9999px;
  border: 1px solid rgba(255, 255, 255, 0.72);
  padding: 0.45rem 0.95rem;
  font-size: 0.875rem;
  font-weight: 800;
  line-height: 1;
  box-shadow:
    0 10px 24px rgba(99, 102, 241, 0.12),
    inset 0 1px 0 rgba(255, 255, 255, 0.35);
  backdrop-filter: blur(16px);
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease,
    border-color 0.2s ease,
    color 0.2s ease;
}

.header-pill-link::before {
  content: '';
  position: absolute;
  inset: 0;
  opacity: 0.96;
  transition: opacity 0.2s ease;
}

.header-pill-link::after {
  content: '';
  position: absolute;
  inset: -60% auto -60% -25%;
  width: 5rem;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.42), transparent);
  transform: rotate(20deg) translateX(-160%);
  transition: transform 0.55s ease;
}

.header-pill-link:hover {
  transform: translateY(-2px);
  box-shadow:
    0 16px 30px rgba(99, 102, 241, 0.18),
    inset 0 1px 0 rgba(255, 255, 255, 0.4);
}

.header-pill-link:hover::after {
  transform: rotate(20deg) translateX(280%);
}

.header-pill-link :deep(svg),
.header-pill-link span {
  position: relative;
  z-index: 1;
}

.header-pill-link-primary {
  color: #5b21b6;
}

.header-pill-link-primary::before {
  background:
    linear-gradient(135deg, rgba(34, 211, 238, 0.24), rgba(168, 85, 247, 0.18) 55%, rgba(251, 146, 60, 0.2)),
    rgba(255, 255, 255, 0.78);
}

.header-pill-link-primary:hover {
  border-color: rgba(168, 85, 247, 0.34);
  color: #4c1d95;
}

.header-pill-link-secondary {
  color: #1e293b;
}

.header-pill-link-secondary::before {
  background:
    linear-gradient(135deg, rgba(59, 130, 246, 0.16), rgba(217, 70, 239, 0.12)),
    rgba(255, 255, 255, 0.76);
}

.header-pill-link-secondary:hover {
  border-color: rgba(59, 130, 246, 0.28);
  color: #0f172a;
}

:deep(.dark) .header-pill-link,
.dark .header-pill-link {
  border-color: rgba(255, 255, 255, 0.1);
  box-shadow:
    0 12px 26px rgba(15, 23, 42, 0.35),
    inset 0 1px 0 rgba(255, 255, 255, 0.05);
}

:deep(.dark) .header-pill-link-primary,
.dark .header-pill-link-primary {
  color: #e9d5ff;
}

:deep(.dark) .header-pill-link-primary::before,
.dark .header-pill-link-primary::before {
  background:
    linear-gradient(135deg, rgba(34, 211, 238, 0.2), rgba(168, 85, 247, 0.24) 55%, rgba(251, 146, 60, 0.16)),
    rgba(15, 23, 42, 0.82);
}

:deep(.dark) .header-pill-link-secondary,
.dark .header-pill-link-secondary {
  color: #e2e8f0;
}

:deep(.dark) .header-pill-link-secondary::before,
.dark .header-pill-link-secondary::before {
  background:
    linear-gradient(135deg, rgba(59, 130, 246, 0.16), rgba(217, 70, 239, 0.14)),
    rgba(15, 23, 42, 0.78);
}

:deep(.dark) .header-shell,
.dark .header-shell {
  border-color: rgba(255, 255, 255, 0.08);
  background:
    linear-gradient(135deg, rgba(15, 23, 42, 0.88), rgba(30, 41, 59, 0.72)),
    rgba(15, 23, 42, 0.82);
  box-shadow:
    0 28px 64px rgba(2, 6, 23, 0.52),
    inset 0 1px 0 rgba(255, 255, 255, 0.08),
    inset 0 -12px 26px rgba(59, 130, 246, 0.08);
}

:deep(.dark) .header-shell::before,
.dark .header-shell::before {
  background:
    radial-gradient(circle at 14% 50%, rgba(96, 165, 250, 0.18) 0%, rgba(96, 165, 250, 0) 28%),
    radial-gradient(circle at 86% 38%, rgba(167, 139, 250, 0.16) 0%, rgba(167, 139, 250, 0) 24%);
  opacity: 0.92;
}

:deep(.dark) .header-shell::after,
.dark .header-shell::after {
  background:
    linear-gradient(115deg, transparent 28%, rgba(255, 255, 255, 0.01) 38%, rgba(255, 255, 255, 0.14) 50%, rgba(255, 255, 255, 0.01) 62%, transparent 72%);
  background-size: 220% 100%;
  background-position: -140% 0;
}

:deep(.dark) .header-page-title,
.dark .header-page-title {
  color: #d8b4fe;
}

:deep(.dark) .header-page-description,
.dark .header-page-description {
  color: #c4b5fd;
  opacity: 0.78;
}

@keyframes header-shell-shimmer {
  0% {
    opacity: 0;
    background-position: -140% 0;
  }
  12% {
    opacity: 0.78;
  }
  24% {
    opacity: 0;
    background-position: 140% 0;
  }
  100% {
    opacity: 0;
    background-position: 140% 0;
  }
}

.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.2s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: scale(0.95) translateY(-4px);
}
</style>
