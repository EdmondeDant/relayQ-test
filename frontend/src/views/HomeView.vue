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
              Install-first landing hero
            </div>

            <h1 class="hero-title relative z-10 mb-5 pb-2 text-5xl font-black leading-[1.08] tracking-tight text-slate-950 dark:text-white sm:text-6xl lg:text-7xl">
              {{ siteName }}
            </h1>
            <p class="mx-auto mb-8 max-w-2xl text-base font-medium leading-8 text-slate-600 dark:text-slate-300 sm:text-xl lg:mx-0">
              {{ siteSubtitle }}
            </p>

            <div class="hero-highlight-card">
              <div class="hero-highlight-title">
                一条命令，把 OpenClaw / Hermes 和你的 RelayQ 网关一起装好
              </div>
              <div class="hero-highlight-desc">
                我们真心为刚刚了解ai技术的爱好者，共最方便最安全的服务，为初创ai公司提供种类众多的模型，并且稳定，安全，性价比高。
              </div>
              <div class="hero-highlight-marquee">
                <span class="hero-highlight-marquee-chip">Install Surface</span>
                <span class="hero-highlight-marquee-chip">Live Command</span>
                <span class="hero-highlight-marquee-chip">Gateway Ready</span>
                <span class="hero-highlight-marquee-chip">Product CTA</span>
              </div>
              <div class="hero-highlight-metrics">
                <div class="hero-highlight-metric">
                  <span class="hero-highlight-value">2</span>
                  <span class="hero-highlight-label">主推工具</span>
                </div>
                <div class="hero-highlight-metric">
                  <span class="hero-highlight-value">4</span>
                  <span class="hero-highlight-label">真实脚本</span>
                </div>
                <div class="hero-highlight-metric">
                  <span class="hero-highlight-value">/v1/models</span>
                  <span class="hero-highlight-label">模型联动</span>
                </div>
              </div>
            </div>

            <!-- CTA Button -->
            <div class="hero-cta-row">
              <a
                href="#install-lab"
                class="group hero-cta hero-cta-primary"
              >
                <span class="hero-cta-shine"></span>
                <span class="hero-cta-text">
                  <span class="hero-cta-title">立即生成安装命令</span>
                  <span class="hero-cta-subtitle">OpenClaw / Hermes setup center</span>
                </span>
                <Icon name="arrowRight" size="md" class="ml-2 transition-transform group-hover:translate-x-1" :stroke-width="2.5" />
              </a>
              <a
                href="#install-lab"
                class="group hero-cta hero-cta-secondary"
              >
                查看 OpenClaw / Hermes 面板
                <Icon name="sparkles" size="sm" class="ml-2 transition-transform group-hover:scale-110" />
              </a>
              <router-link
                :to="isAuthenticated ? dashboardPath : '/login'"
                class="group hero-cta hero-cta-tertiary"
              >
                {{ isAuthenticated ? '进入控制台补配置' : '登录后拿 token' }}
              </router-link>
            </div>

            <div class="hero-install-pill mt-5 inline-flex flex-wrap items-center justify-center gap-2 rounded-[1.3rem] px-4 py-3 text-left lg:justify-start">
              <span class="hero-install-label">主推能力</span>
              <span class="hero-install-chip">OpenClaw 一键安装</span>
              <span class="hero-install-chip">Hermes 真配置</span>
              <span class="hero-install-chip">自动拉取模型列表</span>
            </div>

            <div class="hero-proof-row">
              <div class="hero-proof-card">
                <div class="hero-proof-key">当前工具</div>
                <div class="hero-proof-value">{{ activeInstaller.title }}</div>
              </div>
              <div class="hero-proof-card">
                <div class="hero-proof-key">当前脚本</div>
                <div class="hero-proof-value">{{ currentInstallScriptLabel }}</div>
              </div>
              <div class="hero-proof-card">
                <div class="hero-proof-key">命令类型</div>
                <div class="hero-proof-value">{{ currentShellLabel }}</div>
              </div>
            </div>
          </div>

          <!-- Right: Anime Startup Console -->
          <div class="flex justify-center lg:justify-end">
            <div class="hero-card relative w-full max-w-[500px]">
              <div class="hero-card-glow hero-card-glow-a"></div>
              <div class="hero-card-glow hero-card-glow-b"></div>
              <div class="hero-card-grid"></div>
              <div class="hero-status-ribbon hero-status-ribbon-a">Install Mode</div>
              <div class="hero-status-ribbon hero-status-ribbon-b">{{ currentShellLabel }}</div>
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
                    <span class="terminal-title">install-command-center</span>
                  </div>
                  <div class="terminal-body">
                    <div class="code-line line-1">
                      <span class="code-prompt">$</span>
                      <span class="code-cmd">relayq install</span>
                      <span class="code-flag">--tool {{ activeInstaller.title.toLowerCase() }}</span>
                      <span class="code-url">--os {{ selectedInstallerOs === 'windows' ? 'windows' : 'linux-macos' }}</span>
                    </div>
                    <div class="code-line line-2">
                      <span class="code-comment"># script {{ currentInstallScriptLabel }} · model {{ currentPreviewModel }}</span>
                    </div>
                    <div class="code-line line-3">
                      <span class="code-success">READY</span>
                      <span class="code-response">{{ currentShellLabel }} command prepared</span>
                    </div>
                    <div class="code-line line-4">
                      <span class="code-prompt">$</span>
                      <span class="hero-command-inline">{{ heroStaticCommandPreview }}</span>
                    </div>
                  </div>
                </div>
              </div>
              <div class="hero-command-card">
                <div class="hero-command-card-top">
                  <span class="hero-command-pill">Raw Script</span>
                  <a :href="currentInstallScriptUrl" target="_blank" rel="noopener noreferrer" class="hero-command-link">
                    {{ currentInstallScriptUrl }}
                  </a>
                </div>
                <div class="hero-command-card-body">
                  <div class="hero-command-card-status">
                    <span class="hero-command-card-status-dot"></span>
                    {{ activeInstaller.title }} script is ready to run
                  </div>
                  <div class="hero-command-card-label">Command Preview</div>
                  <pre class="hero-command-card-pre"><code>{{ heroStaticCommandFull }}</code></pre>
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

        <section id="install-lab" class="install-lab mt-16 rounded-[2.25rem] p-6 sm:p-8">
          <div class="install-lab-orbit install-lab-orbit-a"></div>
          <div class="install-lab-orbit install-lab-orbit-b"></div>
          <div class="install-lab-grid"></div>
          <div class="mx-auto max-w-5xl text-center">
            <h2 class="text-3xl font-black tracking-tight text-slate-950 dark:text-white sm:text-4xl">
              OpenClaw / Hermes 真一键安装
            </h2>
          </div>

          <div class="mx-auto mt-10 grid max-w-6xl gap-6">
            <div class="grid gap-4">
              <div class="grid gap-4 md:grid-cols-2">
              <button
                v-for="tool in installerCards"
                :key="tool.id"
                type="button"
                class="install-tool-card text-left"
                :class="{ 'install-tool-card-active': selectedInstaller === tool.id }"
                @click="selectedInstaller = tool.id"
              >
                <div class="flex items-start justify-between gap-4">
                  <div class="flex items-start gap-3">
                    <div class="install-tool-avatar" :class="tool.avatarClass">
                      {{ tool.avatarText }}
                    </div>
                    <div>
                      <div class="text-lg font-black text-slate-950 dark:text-white">{{ tool.title }}</div>
                      <div class="mt-1 text-xs font-bold uppercase tracking-[0.2em] text-slate-400 dark:text-slate-500">
                        {{ tool.kicker }}
                      </div>
                    </div>
                  </div>
                  <span class="install-status-chip">
                    {{ selectedInstaller === tool.id ? '已选中' : '可预览' }}
                  </span>
                </div>

                <p class="mt-4 text-sm leading-7 text-slate-600 dark:text-slate-300">
                  {{ tool.description }}
                </p>

                <div class="mt-4 text-xs font-bold uppercase tracking-[0.18em] text-slate-400 dark:text-slate-500">
                  {{ tool.compatibility }}
                </div>

                <div class="mt-4 flex flex-wrap gap-2">
                  <span v-for="highlight in tool.highlights" :key="highlight" class="install-mini-chip">
                    {{ highlight }}
                  </span>
                </div>
              </button>
            </div>
            </div>

            <div class="install-studio mt-2">
              <div class="install-studio-glow install-studio-glow-a"></div>
              <div class="install-studio-glow install-studio-glow-b"></div>
              <div class="flex flex-col gap-4 border-b border-white/60 pb-5 dark:border-white/10 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <div class="text-xs font-black uppercase tracking-[0.22em] text-fuchsia-500">
                    {{ activeInstaller.kicker }}
                  </div>
                  <div class="mt-2 text-2xl font-black text-slate-950 dark:text-white">
                    {{ activeInstaller.title }} 一键安装
                  </div>
                  <p class="mt-2 text-sm leading-7 text-slate-600 dark:text-slate-300">
                    {{ activeInstaller.subtitle }}
                  </p>
                  <div class="mt-3 flex flex-wrap gap-2">
                    <span class="install-runtime-pill">{{ activeInstaller.runtime }}</span>
                    <span class="install-runtime-pill">{{ selectedInstallerOsLabel }}</span>
                    <span class="install-runtime-pill">{{ installFlowLabel }}</span>
                  </div>
                </div>

                <div class="install-os-tabs">
                  <button
                    type="button"
                    class="install-os-tab"
                    :class="{ 'install-os-tab-active': selectedInstallerOs === 'windows' }"
                    @click="selectedInstallerOs = 'windows'"
                  >
                    Windows
                  </button>
                  <button
                    type="button"
                    class="install-os-tab"
                    :class="{ 'install-os-tab-active': selectedInstallerOs === 'unix' }"
                    @click="selectedInstallerOs = 'unix'"
                  >
                    Linux / macOS
                  </button>
                </div>
              </div>

              <div class="mt-6 grid gap-4 md:grid-cols-2">
                <label class="install-field">
                  <span class="install-field-label">令牌/apikey</span>
                  <input
                    v-model="installToken"
                    type="text"
                    class="install-input"
                    :class="{ 'install-input-required': !installToken.trim() }"
                    placeholder="必填apikey"
                  />
                </label>

                <label class="install-field">
                  <span class="install-field-label">{{ modelFieldLabel }}</span>
                  <select
                    v-if="availableModels.length > 0"
                    v-model="installModel"
                    class="install-input"
                  >
                    <option value="" disabled>请选择模型</option>
                    <option v-for="model in availableModels" :key="model" :value="model">
                      {{ model }}
                    </option>
                  </select>
                  <input
                    v-else
                    v-model="installModel"
                    type="text"
                    class="install-input"
                    placeholder="请填写或选择模型"
                  />
                </label>
              </div>

              <div class="mt-4 grid gap-3 lg:grid-cols-[auto_1fr] lg:items-center">
                <button
                  type="button"
                  class="install-fetch-btn"
                  :disabled="modelsLoading"
                  @click="fetchAvailableModels()"
                >
                  {{ modelsLoading ? '正在拉取模型...' : '刷新模型列表' }}
                </button>

                <div class="install-fetch-meta">
                  <span v-if="modelsError" class="install-fetch-error">{{ modelsError }}</span>
                  <template v-else>
                    <span class="install-fetch-ok">{{ modelFetchStatus }}</span>
                    <span class="install-fetch-divider"></span>
                    <span>Base URL: {{ installBaseUrl }}</span>
                  </template>
                </div>
              </div>

              <div class="mt-5 grid gap-3 md:grid-cols-3">
                <div v-for="item in previewConfigItems" :key="item.label" class="install-config-card">
                  <div class="text-[11px] font-black uppercase tracking-[0.18em] text-slate-400 dark:text-slate-500">
                    {{ item.label }}
                  </div>
                  <div class="mt-2 break-all text-sm font-bold text-slate-700 dark:text-slate-200">
                    {{ item.value }}
                  </div>
                </div>
              </div>

              <div class="mt-5 grid gap-3 md:grid-cols-3">
                <div v-for="step in activeInstaller.steps" :key="step.title" class="install-step-card">
                  <div class="install-step-index">{{ step.index }}</div>
                  <div class="install-step-title">{{ step.title }}</div>
                  <div class="install-step-desc">{{ step.description }}</div>
                </div>
              </div>

              <div class="mt-6 overflow-hidden rounded-[1.6rem] border border-slate-900/10 bg-slate-950 shadow-2xl shadow-slate-900/15 dark:border-white/10">
                <div class="flex items-center justify-between border-b border-white/10 px-4 py-3">
                  <div class="flex items-center gap-2">
                    <span class="h-2.5 w-2.5 rounded-full bg-rose-400"></span>
                    <span class="h-2.5 w-2.5 rounded-full bg-amber-300"></span>
                    <span class="h-2.5 w-2.5 rounded-full bg-emerald-400"></span>
                    <span class="ml-2 text-xs font-black uppercase tracking-[0.2em] text-slate-400">
                      command preview
                    </span>
                  </div>
                  <div class="flex items-center gap-2">
                    <a
                      :href="currentInstallScriptUrl"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="install-copy-btn"
                    >
                      查看原始脚本
                    </a>
                    <button type="button" class="install-copy-btn" @click="copyInstallCommand">
                      复制命令
                    </button>
                  </div>
                </div>

                <div class="install-command-meta">
                  <div class="install-command-meta-card">
                    <div class="install-command-meta-label">Runtime</div>
                    <div class="install-command-meta-value">{{ currentShellLabel }}</div>
                  </div>
                  <div class="install-command-meta-card">
                    <div class="install-command-meta-label">Script Entry</div>
                    <div class="install-command-meta-value">{{ currentInstallScriptLabel }}</div>
                  </div>
                  <div class="install-command-meta-card">
                    <div class="install-command-meta-label">Script URL</div>
                    <div class="install-command-meta-value break-all">{{ currentInstallScriptUrl }}</div>
                  </div>
                </div>

                <pre class="install-command-pre"><code>{{ generatedInstallCommand }}</code></pre>

                <div class="install-command-footer">
                  <span class="install-command-footer-pill">{{ activeInstaller.title }}</span>
                  <span class="install-command-footer-pill">{{ selectedInstallerOsLabel }}</span>
                  <span class="install-command-footer-pill">{{ currentPreviewModel }}</span>
                </div>
              </div>

              <div class="mt-5 flex flex-col gap-3 rounded-[1.5rem] border border-white/60 bg-white/55 px-4 py-4 text-sm font-medium text-slate-600 backdrop-blur-xl dark:border-white/10 dark:bg-slate-950/35 dark:text-slate-300 md:flex-row md:items-center md:justify-between">
                <div class="leading-7">
                  {{ installBottomHint }}
                </div>
                <router-link
                  :to="isAuthenticated ? dashboardPath : '/login'"
                  class="inline-flex items-center justify-center rounded-full bg-slate-950 px-4 py-2 text-xs font-black text-white transition-all hover:-translate-y-0.5 hover:bg-violet-950 dark:bg-white dark:text-slate-950"
                >
                  {{ isAuthenticated ? '去控制台继续配置' : '登录后获取令牌' }}
                </router-link>
              </div>
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
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import { useClipboard } from '@/composables/useClipboard'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { buildServiceUrls } from '@/utils/toolConfigExport'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

type InstallerTool = 'openclaw' | 'hermes'
type InstallerOs = 'windows' | 'unix'

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const runtimeOrigin = ref('')
const selectedInstaller = ref<InstallerTool>('openclaw')
const selectedInstallerOs = ref<InstallerOs>('windows')
const installToken = ref('')
const installModel = ref('')
const availableModels = ref<string[]>([])
const modelsLoading = ref(false)
const modelsError = ref('')
let homeModelFetchRequestId = 0
let installTokenAutoFetchTimer: ReturnType<typeof setTimeout> | null = null

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

const installerCards = [
  {
    id: 'openclaw' as InstallerTool,
    title: 'OpenClaw',
    kicker: 'CLI Assistant',
    subtitle: '适合需要直接落地到本地 CLI 的用户，命令里自动带上 token、baseUrl 与所选模型。',
    description: '一键安装小龙虾，告别繁琐的设置，你的第一位ai员工即将诞生。',
    compatibility: 'Native Windows + Linux / macOS',
    runtime: 'CLI + official onboarding',
    highlights: ['官方 onboarding', '自动装 daemon', '装完直开控制台'],
    steps: [
      { index: '01', title: '装 CLI', description: '打开powerShell复制，粘贴，执行下面命令，即可安装。' },
      { index: '02', title: '走官方配置', description: '自动调用 OpenClaw 官方 onboarding，写入 baseUrl、apikey、所选模型。' },
      { index: '03', title: '即刻可用', description: '安装完成后自动启动 gateway，并直接打开控制台。' },
    ],
    avatarText: 'OC',
    avatarClass: 'install-avatar-openclaw',
  },
  {
    id: 'hermes' as InstallerTool,
    title: 'Hermes',
    kicker: 'Desktop Bridge',
    subtitle: '适合桌面桥接或本地代理场景，突出 baseUrl、token 与本地 profile 的快速导入。',
    description: '自动写入 Hermes 的新版 custom endpoint 配置，并在安装完成后直接进入可用会话。',
    compatibility: 'Native Windows + Linux / macOS',
    runtime: 'native installer + config.yaml + .env',
    highlights: ['原生 Windows', '新版 custom 配置', '装完直接开聊'],
    steps: [
      { index: '01', title: '装 Hermes', description: '调用 Hermes 官方安装器完成依赖和主程序安装。' },
      { index: '02', title: '落新版配置', description: '写入 config.yaml 与 .env，把 custom endpoint、apikey、所选模型一次配好。' },
      { index: '03', title: '打开即聊', description: '安装完成后直接启动 hermes，会话立刻可用。' },
    ],
    avatarText: 'HM',
    avatarClass: 'install-avatar-hermes',
  },
]

const activeInstaller = computed(() =>
  installerCards.find((item) => item.id === selectedInstaller.value) ?? installerCards[0]
)

const selectedInstallerOsLabel = computed(() =>
  selectedInstallerOs.value === 'windows' ? 'Windows' : 'Linux / macOS'
)

const modelFieldLabel = computed(() => '所选模型')

const installFlowLabel = computed(() =>
  selectedInstaller.value === 'hermes' && selectedInstallerOs.value === 'windows'
    ? '原生 Windows 安装'
    : '直接一键安装'
)

const installRouteRoot = computed(() => {
  const configured = (appStore.cachedPublicSettings as Record<string, unknown> | undefined)?.api_base_url
  if (typeof configured === 'string' && configured.trim()) {
    return configured
  }
  const origin = runtimeOrigin.value || window.location.origin
  return origin.replace(/\/+$/, '')
})

const installBaseUrl = computed(() =>
  buildServiceUrls(installRouteRoot.value).openai
)

const installScriptOrigin = computed(() =>
  buildServiceUrls(installRouteRoot.value).root || (runtimeOrigin.value || window.location.origin).replace(/\/+$/, '')
)

const currentPreviewModel = computed(() =>
  installModel.value.trim() || '未选择模型'
)

const currentInstallScriptLabel = computed(() => {
  if (selectedInstaller.value === 'openclaw') {
    return selectedInstallerOs.value === 'windows' ? 'openclaw-windows' : 'openclaw-linux'
  }
  return selectedInstallerOs.value === 'windows' ? 'hermes-windows' : 'hermes-linux'
})

const currentInstallScriptUrl = computed(() =>
  `${installScriptOrigin.value}/install/${currentInstallScriptLabel.value}`
)

const currentShellLabel = computed(() =>
  selectedInstallerOs.value === 'windows' ? 'PowerShell' : 'Bash'
)

const heroStaticCommandPreview = 'curl -fsSL https://relayq.ai/bootstrap | bash'

const heroStaticCommandFull =
  "curl -fsSL https://relayq.ai/bootstrap | bash -s -- --tool starter --profile demo"

const generatedInstallCommand = computed(() => {
  const token = installToken.value.trim()
  const model = installModel.value.trim()
  if (!token || !model) {
    return '需要去注册会员获得apikey，并正确选择模型时，全自动安装命令才会生成。'
  }
  if (selectedInstaller.value === 'openclaw') {
    return selectedInstallerOs.value === 'windows'
      ? `$key='${token}'; $model='${model}'; $base_url='${installBaseUrl.value}'; iwr -useb ${installScriptOrigin.value}/install/openclaw-windows | iex`
      : `key='${token}' model='${model}' base_url='${installBaseUrl.value}' bash <(curl -fsSL ${installScriptOrigin.value}/install/openclaw-linux)`
  }

  return selectedInstallerOs.value === 'windows'
    ? `$env:HERMES_API_KEY='${token}'; $env:HERMES_BASE_URL='${installBaseUrl.value}'; $env:HERMES_DEFAULT_MODEL='${model}'; irm ${installScriptOrigin.value}/install/hermes-windows | iex`
    : `OPENAI_API_KEY='${token}' OPENAI_BASE_URL='${installBaseUrl.value}' HERMES_DEFAULT_MODEL='${model}' bash <(curl -fsSL ${installScriptOrigin.value}/install/hermes-linux)`
})

const previewConfigItems = computed(() => {
  const token = installToken.value.trim() || 'sk-your-token'
  if (selectedInstaller.value === 'openclaw') {
    return [
      { label: 'Base URL', value: installBaseUrl.value },
      { label: 'Token', value: maskToken(token) },
      { label: 'Selected Model', value: installModel.value.trim() || '未选择模型' },
    ]
  }

  return [
    { label: 'Base URL', value: installBaseUrl.value },
    { label: 'Token', value: maskToken(token) },
    { label: 'Selected Model', value: installModel.value.trim() || '未选择模型' },
  ]
})

const modelFetchStatus = computed(() => {
  if (modelsLoading.value) return '正在连接 /models...'
  if (availableModels.value.length > 0) return `已读取 ${availableModels.value.length} 个模型`
  return '填入 apikey 后自动读取 /v1/models'
})

const installBottomHint = computed(() => {
  if (selectedInstaller.value === 'hermes' && selectedInstallerOs.value === 'windows') {
    return 'Hermes 当前已切到官方原生 Windows 安装器：自动安装、自动写入 key 与 baseUrl，并直接启动 Hermes。'
  }
  if (selectedInstaller.value === 'openclaw') {
    return 'OpenClaw 安装脚本会走官方 onboarding、自动装 daemon，并在完成后直接打开 Control UI。'
  }
  return 'Hermes 安装脚本会写入 ~/.hermes/config.yaml 与 ~/.hermes/.env，验证接口后直接启动 hermes。'
})

function maskToken(token: string) {
  if (!token) return '未填写'
  if (token.length <= 10) return token
  return `${token.slice(0, 6)}...${token.slice(-4)}`
}

async function copyInstallCommand() {
  await copyToClipboard(generatedInstallCommand.value, '安装命令已复制')
}

async function fetchAvailableModels(options: { silent?: boolean } = {}) {
  const { silent = false } = options
  const token = installToken.value.trim()
  if (!token) {
    modelsError.value = '请先填写 token'
    if (!silent) {
      appStore.showError('请先填写 token')
    }
    return
  }

  const requestId = ++homeModelFetchRequestId
  modelsLoading.value = true
  modelsError.value = ''
  try {
    const response = await fetch(`${installBaseUrl.value}/models`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }

    const payload = await response.json() as { data?: Array<{ id?: string }> }
    const models = Array.from(
      new Set((payload.data || []).map((item) => item.id?.trim()).filter((item): item is string => Boolean(item)))
    ).sort((a, b) => a.localeCompare(b))

    if (requestId !== homeModelFetchRequestId) {
      return
    }

    availableModels.value = models
    if (models.length > 0) {
      installModel.value = models.includes(installModel.value.trim()) ? installModel.value.trim() : ''
      if (!silent) {
        appStore.showSuccess(`已获取 ${models.length} 个模型`)
      }
      return
    }

    modelsError.value = '接口返回成功，但没有可选模型'
    if (!silent) {
      appStore.showError('没有读取到模型列表')
    }
  } catch (error) {
    if (requestId !== homeModelFetchRequestId) {
      return
    }

    modelsError.value = error instanceof Error ? error.message : '模型列表获取失败'
    if (!silent) {
      appStore.showError('模型列表获取失败')
    }
  } finally {
    if (requestId === homeModelFetchRequestId) {
      modelsLoading.value = false
    }
  }
}

function scheduleAutoFetchAvailableModels() {
  if (installTokenAutoFetchTimer) {
    clearTimeout(installTokenAutoFetchTimer)
    installTokenAutoFetchTimer = null
  }

  availableModels.value = []
  modelsError.value = ''

  if (!installToken.value.trim()) {
    return
  }

  installTokenAutoFetchTimer = setTimeout(() => {
    installTokenAutoFetchTimer = null
    void fetchAvailableModels({ silent: true })
  }, 400)
}

watch(selectedInstaller, (_, previousInstaller) => {
  if (!previousInstaller) {
    return
  }

  installModel.value = ''

  if (installToken.value.trim()) {
    scheduleAutoFetchAvailableModels()
  }
})

watch(selectedInstallerOs, () => {
  if (installToken.value.trim()) {
    scheduleAutoFetchAvailableModels()
  }
})

watch(installToken, () => {
  scheduleAutoFetchAvailableModels()
})

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
  runtimeOrigin.value = window.location.origin
  initTheme()

  // Check auth state
  authStore.checkAuth()

  // Always refresh public settings once on home mount so site_name/site_subtitle
  // stay in sync after admins change them in settings.
  void appStore.fetchPublicSettings(true)
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

:deep(.dark) .hero-highlight-card,
.dark .hero-highlight-card,
:deep(.dark) .hero-proof-card,
.dark .hero-proof-card,
:deep(.dark) .hero-command-card,
.dark .hero-command-card,
:deep(.dark) .hero-highlight-marquee-chip,
.dark .hero-highlight-marquee-chip,
:deep(.dark) .hero-cta-secondary,
.dark .hero-cta-secondary,
:deep(.dark) .hero-cta-tertiary,
.dark .hero-cta-tertiary {
  border-color: rgba(255, 255, 255, 0.1);
  background: rgba(15, 23, 42, 0.62);
  color: #e2e8f0;
}

:deep(.dark) .hero-title,
.dark .hero-title {
  background: linear-gradient(90deg, #ffffff, #a5f3fc 34%, #f0abfc 65%, #fed7aa);
  -webkit-background-clip: text;
  background-clip: text;
}

.hero-highlight-card {
  margin-bottom: 1.6rem;
  max-width: 44rem;
  border: 1px solid rgba(255, 255, 255, 0.72);
  border-radius: 1.8rem;
  background:
    radial-gradient(circle at top right, rgba(34, 211, 238, 0.16), transparent 24%),
    radial-gradient(circle at left center, rgba(217, 70, 239, 0.14), transparent 24%),
    rgba(255, 255, 255, 0.62);
  padding: 1.25rem 1.25rem 1.15rem;
  box-shadow: 0 24px 60px rgba(99, 102, 241, 0.12);
  backdrop-filter: blur(18px);
}

.hero-highlight-kicker {
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  background: linear-gradient(90deg, #22d3ee, #a855f7);
  padding: 0.42rem 0.78rem;
  font-size: 0.7rem;
  font-weight: 1000;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: white;
}

.hero-highlight-title {
  margin-top: 0.95rem;
  font-size: 1.4rem;
  font-weight: 1000;
  line-height: 1.35;
  color: #0f172a;
}

.hero-highlight-desc {
  margin-top: 0.7rem;
  font-size: 0.95rem;
  line-height: 1.8;
  color: #64748b;
}

.hero-highlight-marquee {
  display: flex;
  flex-wrap: wrap;
  gap: 0.55rem;
  margin-top: 0.95rem;
}

.hero-highlight-marquee-chip {
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  border: 1px solid rgba(255, 255, 255, 0.72);
  background: rgba(255, 255, 255, 0.72);
  padding: 0.34rem 0.72rem;
  font-size: 0.7rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  color: #6d28d9;
}

.hero-highlight-metrics,
.hero-proof-row {
  display: grid;
  gap: 0.8rem;
}

.hero-highlight-metrics {
  margin-top: 1rem;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.hero-highlight-metric,
.hero-proof-card {
  border: 1px solid rgba(255, 255, 255, 0.7);
  border-radius: 1.2rem;
  background: rgba(255, 255, 255, 0.76);
  padding: 0.9rem 1rem;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.35);
}

.hero-highlight-value,
.hero-proof-value {
  display: block;
  font-size: 1rem;
  font-weight: 1000;
  color: #0f172a;
}

.hero-highlight-label,
.hero-proof-key {
  display: block;
  margin-top: 0.2rem;
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: #64748b;
}

.hero-proof-row {
  margin-top: 1rem;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.hero-cta-row {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}

.hero-cta {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: 1.5rem;
  transition: transform 0.25s ease, box-shadow 0.25s ease, border-color 0.25s ease, color 0.25s ease;
}

.hero-cta:hover {
  transform: translateY(-4px);
}

.hero-cta-primary {
  gap: 0.85rem;
  min-width: min(100%, 22rem);
  padding: 1rem 1.25rem;
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: linear-gradient(90deg, #22d3ee, #a855f7 48%, #fb923c);
  box-shadow:
    0 24px 60px rgba(168, 85, 247, 0.35),
    0 8px 24px rgba(34, 211, 238, 0.18);
  color: white;
}

.hero-cta-primary:hover {
  box-shadow:
    0 30px 80px rgba(168, 85, 247, 0.4),
    0 12px 32px rgba(34, 211, 238, 0.24);
}

.hero-cta-secondary,
.hero-cta-tertiary {
  padding: 0.95rem 1.15rem;
  border: 1px solid rgba(255, 255, 255, 0.8);
  background: rgba(255, 255, 255, 0.7);
  box-shadow: 0 20px 50px rgba(99, 102, 241, 0.12);
  backdrop-filter: blur(18px);
}

.hero-cta-secondary {
  font-size: 0.9rem;
  font-weight: 1000;
  color: #0f172a;
}

.hero-cta-secondary:hover {
  border-color: rgba(217, 70, 239, 0.28);
  color: #7c3aed;
}

.hero-cta-tertiary {
  font-size: 0.86rem;
  font-weight: 900;
  color: #475569;
}

.hero-cta-tertiary:hover {
  border-color: rgba(34, 211, 238, 0.32);
  color: #0f172a;
}

.hero-cta-shine {
  position: absolute;
  inset: -40% auto -40% -20%;
  width: 9rem;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.38), transparent);
  transform: rotate(18deg);
  animation: heroShine 4.4s ease-in-out infinite;
}

.hero-cta-text {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}

.hero-cta-title {
  font-size: 1rem;
  font-weight: 1000;
  line-height: 1.1;
}

.hero-cta-subtitle {
  margin-top: 0.2rem;
  font-size: 0.68rem;
  font-weight: 800;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.82);
}

.hero-card {
  position: relative;
  min-height: 540px;
}

.hero-card-glow,
.install-lab-orbit,
.install-studio-glow {
  position: absolute;
  border-radius: 9999px;
  filter: blur(24px);
}

.hero-card-glow {
  z-index: 0;
  opacity: 0.72;
}

.hero-card-glow-a {
  left: 0.8rem;
  top: 4rem;
  width: 10rem;
  height: 10rem;
  background: rgba(34, 211, 238, 0.3);
}

.hero-card-glow-b {
  right: 0.5rem;
  top: 9rem;
  width: 12rem;
  height: 12rem;
  background: rgba(217, 70, 239, 0.3);
}

.hero-card-grid {
  position: absolute;
  inset: 2rem 0.5rem 4rem;
  z-index: 0;
  border-radius: 2.2rem;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.09) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.09) 1px, transparent 1px);
  background-size: 28px 28px;
  mask-image: radial-gradient(circle at center, black, transparent 80%);
  opacity: 0.5;
}

.hero-status-ribbon {
  position: absolute;
  z-index: 4;
  display: inline-flex;
  align-items: center;
  border: 1px solid rgba(255, 255, 255, 0.78);
  border-radius: 9999px;
  background: rgba(255, 255, 255, 0.72);
  padding: 0.48rem 0.82rem;
  font-size: 0.68rem;
  font-weight: 1000;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  box-shadow: 0 16px 32px rgba(99, 102, 241, 0.14);
  backdrop-filter: blur(16px);
}

.hero-status-ribbon-a {
  left: -0.6rem;
  top: 2.4rem;
  color: #0369a1;
}

.hero-status-ribbon-b {
  right: -0.4rem;
  top: 19rem;
  color: #b45309;
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
  padding-top: 2.4rem;
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
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

.terminal-window:hover {
  transform: perspective(1000px) rotateX(0deg) rotateY(0deg) translateY(-8px);
  box-shadow:
    0 45px 110px rgba(79, 70, 229, 0.34),
    0 0 0 8px rgba(255, 255, 255, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.16);
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

.hero-command-inline {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  color: #d8f3ff;
  font-size: 0.75rem;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.hero-command-card {
  position: relative;
  margin: 1rem auto 0;
  width: min(100%, 460px);
  border: 1px solid rgba(255, 255, 255, 0.72);
  border-radius: 1.5rem;
  background: rgba(255, 255, 255, 0.66);
  padding: 1rem;
  box-shadow: 0 24px 60px rgba(99, 102, 241, 0.14);
  backdrop-filter: blur(18px);
}

.hero-command-card::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  padding: 1px;
  background: linear-gradient(135deg, rgba(34, 211, 238, 0.38), rgba(217, 70, 239, 0.4), rgba(251, 146, 60, 0.35));
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  pointer-events: none;
}

.hero-command-card-top {
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem;
  align-items: center;
  justify-content: space-between;
}

.hero-command-pill {
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  background: rgba(14, 165, 233, 0.1);
  padding: 0.35rem 0.7rem;
  font-size: 0.68rem;
  font-weight: 1000;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: #0369a1;
}

.hero-command-link {
  max-width: 16rem;
  overflow: hidden;
  font-size: 0.72rem;
  font-weight: 800;
  color: #7c3aed;
  text-decoration: none;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hero-command-link:hover {
  color: #c026d3;
}

.hero-command-card-body {
  margin-top: 0.85rem;
  border-radius: 1.1rem;
  background: rgba(15, 23, 42, 0.95);
  padding: 0.9rem 1rem;
}

.hero-command-card-status {
  display: inline-flex;
  align-items: center;
  gap: 0.42rem;
  margin-bottom: 0.75rem;
  font-size: 0.72rem;
  font-weight: 900;
  color: #86efac;
}

.hero-command-card-status-dot {
  width: 0.45rem;
  height: 0.45rem;
  border-radius: 9999px;
  background: #34d399;
  box-shadow: 0 0 0 6px rgba(52, 211, 153, 0.16);
}

.hero-command-card-label {
  font-size: 0.68rem;
  font-weight: 1000;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: #94a3b8;
}

.hero-command-card-pre {
  margin: 0.7rem 0 0;
  overflow-x: auto;
  color: #d8f3ff;
  font-family: ui-monospace, 'Fira Code', monospace;
  font-size: 0.77rem;
  line-height: 1.8;
  white-space: pre-wrap;
  word-break: break-word;
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

.hero-install-pill {
  border: 1px solid rgba(255, 255, 255, 0.7);
  background: rgba(255, 255, 255, 0.58);
  box-shadow: 0 18px 45px rgba(99, 102, 241, 0.12);
  backdrop-filter: blur(18px);
}

.hero-install-label,
.hero-install-chip {
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  font-weight: 900;
}

.hero-install-label {
  background: linear-gradient(90deg, #22d3ee, #a855f7);
  padding: 0.45rem 0.8rem;
  font-size: 0.72rem;
  color: white;
}

.hero-install-chip {
  background: rgba(124, 58, 237, 0.08);
  padding: 0.42rem 0.75rem;
  font-size: 0.75rem;
  color: #6d28d9;
}

.install-lab {
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.65);
  background:
    radial-gradient(circle at top left, rgba(34, 211, 238, 0.18), transparent 26%),
    radial-gradient(circle at top right, rgba(217, 70, 239, 0.16), transparent 24%),
    rgba(255, 255, 255, 0.46);
  box-shadow: 0 28px 90px rgba(99, 102, 241, 0.14);
  backdrop-filter: blur(22px);
}

.install-lab-orbit {
  z-index: 0;
  opacity: 0.55;
}

.install-lab-orbit-a {
  left: -5rem;
  top: 4rem;
  width: 14rem;
  height: 14rem;
  background: rgba(34, 211, 238, 0.22);
}

.install-lab-orbit-b {
  right: -5rem;
  top: 0;
  width: 16rem;
  height: 16rem;
  background: rgba(217, 70, 239, 0.2);
}

.install-lab-grid {
  position: absolute;
  inset: 0;
  z-index: 0;
  background-image:
    linear-gradient(rgba(124, 58, 237, 0.06) 1px, transparent 1px),
    linear-gradient(90deg, rgba(34, 211, 238, 0.06) 1px, transparent 1px);
  background-size: 38px 38px;
  mask-image: linear-gradient(to bottom, black, transparent 92%);
}

.install-top-rail {
  position: relative;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 1rem;
  border-radius: 9999px;
  border: 1px solid rgba(255, 255, 255, 0.72);
  background: rgba(255, 255, 255, 0.76);
  padding: 0.45rem 0.9rem;
  box-shadow: 0 15px 34px rgba(99, 102, 241, 0.1);
  backdrop-filter: blur(18px);
}

.install-top-rail-dot {
  width: 0.42rem;
  height: 0.42rem;
  border-radius: 9999px;
  background: linear-gradient(135deg, #22d3ee, #a855f7);
}

.install-top-rail-text {
  font-size: 0.68rem;
  font-weight: 1000;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: #7c3aed;
}

.install-badge {
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  border: 1px solid rgba(255, 255, 255, 0.72);
  background: rgba(255, 255, 255, 0.72);
  padding: 0.7rem 1rem;
  font-size: 0.75rem;
  font-weight: 1000;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: #7c3aed;
  box-shadow: 0 15px 40px rgba(124, 58, 237, 0.14);
}

.install-summary-card,
.install-tool-card,
.install-studio,
.install-config-card,
.install-metric-card,
.install-step-card {
  border: 1px solid rgba(255, 255, 255, 0.68);
  background: rgba(255, 255, 255, 0.58);
  box-shadow: 0 18px 50px rgba(99, 102, 241, 0.12);
  backdrop-filter: blur(18px);
}

.install-summary-card {
  position: relative;
  border-radius: 1.8rem;
  padding: 1.35rem;
}

.install-summary-glow {
  position: absolute;
  inset: auto 1.2rem -1.4rem auto;
  width: 8rem;
  height: 8rem;
  border-radius: 9999px;
  background: rgba(251, 146, 60, 0.18);
  filter: blur(28px);
  pointer-events: none;
}

.install-kicker {
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  background: rgba(251, 191, 36, 0.18);
  padding: 0.45rem 0.8rem;
  font-size: 0.72rem;
  font-weight: 1000;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: #b45309;
}

.install-summary-chip,
.install-runtime-pill {
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  padding: 0.42rem 0.76rem;
  font-size: 0.72rem;
  font-weight: 900;
}

.install-summary-chip {
  background: rgba(14, 165, 233, 0.1);
  color: #0369a1;
}

.install-metric-card {
  border-radius: 1.25rem;
  padding: 1rem;
}

.install-metric-value {
  font-size: 1.6rem;
  font-weight: 1000;
  color: #0f172a;
}

.install-metric-label {
  margin-top: 0.25rem;
  font-size: 0.78rem;
  font-weight: 800;
  color: #64748b;
}

.install-tool-card {
  position: relative;
  overflow: hidden;
  border-radius: 1.7rem;
  padding: 1.35rem;
  transition: all 0.25s ease;
}

.install-tool-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.14), transparent 45%, rgba(168, 85, 247, 0.06));
  opacity: 0;
  transition: opacity 0.25s ease;
}

.install-tool-card:hover,
.install-studio:hover {
  transform: translateY(-4px);
}

.install-tool-card:hover::before {
  opacity: 1;
}

.install-tool-card-active {
  border-color: rgba(168, 85, 247, 0.38);
  box-shadow:
    0 24px 60px rgba(168, 85, 247, 0.16),
    inset 0 0 0 1px rgba(34, 211, 238, 0.16);
}

.install-tool-avatar {
  display: flex;
  width: 3rem;
  height: 3rem;
  align-items: center;
  justify-content: center;
  border-radius: 1rem;
  color: white;
  font-size: 0.85rem;
  font-weight: 1000;
  box-shadow: 0 18px 35px rgba(168, 85, 247, 0.25);
}

.install-avatar-openclaw {
  background: linear-gradient(135deg, #22d3ee, #a855f7, #fb7185);
}

.install-avatar-hermes {
  background: linear-gradient(135deg, #38bdf8, #6366f1, #f97316);
}

.install-status-chip,
.install-mini-chip {
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  font-weight: 900;
}

.install-status-chip {
  background: rgba(34, 211, 238, 0.14);
  padding: 0.36rem 0.7rem;
  font-size: 0.7rem;
  color: #0891b2;
}

.install-mini-chip {
  background: rgba(124, 58, 237, 0.1);
  padding: 0.42rem 0.7rem;
  font-size: 0.72rem;
  color: #7c3aed;
}

.install-runtime-pill {
  background: rgba(251, 146, 60, 0.12);
  color: #c2410c;
}

.install-studio {
  position: relative;
  overflow: hidden;
  border-radius: 1.9rem;
  padding: 1.5rem;
  transition: all 0.25s ease;
}

.install-studio-glow {
  z-index: 0;
  opacity: 0.75;
  pointer-events: none;
}

.install-studio-glow-a {
  right: -2rem;
  top: -2rem;
  width: 9rem;
  height: 9rem;
  background: rgba(34, 211, 238, 0.16);
}

.install-studio-glow-b {
  left: -2rem;
  bottom: 2rem;
  width: 10rem;
  height: 10rem;
  background: rgba(217, 70, 239, 0.14);
}

.install-os-tabs {
  display: inline-flex;
  border-radius: 9999px;
  border: 1px solid rgba(255, 255, 255, 0.7);
  background: rgba(255, 255, 255, 0.75);
  padding: 0.25rem;
}

.install-os-tab {
  border-radius: 9999px;
  padding: 0.65rem 1rem;
  font-size: 0.78rem;
  font-weight: 900;
  color: #64748b;
  transition: all 0.2s ease;
}

.install-os-tab-active {
  background: linear-gradient(90deg, #22d3ee, #a855f7, #fb7185);
  color: white;
  box-shadow: 0 14px 28px rgba(168, 85, 247, 0.22);
}

.install-field {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.install-field-label {
  font-size: 0.72rem;
  font-weight: 1000;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: #64748b;
}

.install-input,
.install-static-value {
  border-radius: 1rem;
  border: 1px solid rgba(255, 255, 255, 0.8);
  background: rgba(255, 255, 255, 0.85);
  padding: 0.95rem 1rem;
  font-size: 0.95rem;
  font-weight: 700;
  color: #0f172a;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.5);
}

.install-input::placeholder {
  color: #94a3b8;
}

.install-input-required {
  border: 2px solid rgba(239, 68, 68, 0.98);
  box-shadow:
    0 0 0 4px rgba(239, 68, 68, 0.18),
    inset 0 1px 0 rgba(255, 255, 255, 0.5);
}

.install-input-required::placeholder {
  color: #dc2626;
}

.install-input:focus {
  outline: none;
  border-color: rgba(168, 85, 247, 0.38);
  box-shadow:
    0 0 0 4px rgba(168, 85, 247, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.5);
}

.install-config-card {
  border-radius: 1.25rem;
  padding: 1rem;
}

.install-step-card {
  border-radius: 1.25rem;
  padding: 1rem;
}

.install-step-index {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  background: linear-gradient(135deg, #22d3ee, #a855f7);
  min-width: 2.25rem;
  padding: 0.35rem 0.7rem;
  font-size: 0.72rem;
  font-weight: 1000;
  color: white;
}

.install-step-title {
  margin-top: 0.9rem;
  font-size: 0.95rem;
  font-weight: 1000;
  color: #0f172a;
}

.install-step-desc {
  margin-top: 0.45rem;
  font-size: 0.84rem;
  line-height: 1.75;
  color: #64748b;
}

.install-fetch-btn {
  border-radius: 9999px;
  background: linear-gradient(90deg, #22d3ee, #a855f7, #fb7185);
  padding: 0.85rem 1.2rem;
  font-size: 0.82rem;
  font-weight: 1000;
  color: white;
  box-shadow: 0 18px 35px rgba(168, 85, 247, 0.22);
  transition: transform 0.2s ease, box-shadow 0.2s ease, opacity 0.2s ease;
}

.install-fetch-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 24px 45px rgba(168, 85, 247, 0.28);
}

.install-fetch-btn:disabled {
  opacity: 0.75;
  cursor: not-allowed;
}

.install-fetch-meta {
  display: flex;
  min-height: 3rem;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.7rem;
  border-radius: 1rem;
  border: 1px solid rgba(255, 255, 255, 0.75);
  background: rgba(255, 255, 255, 0.7);
  padding: 0.85rem 1rem;
  font-size: 0.82rem;
  font-weight: 700;
  color: #64748b;
}

.install-fetch-ok {
  color: #0f766e;
}

.install-fetch-error {
  color: #dc2626;
}

.install-fetch-divider {
  width: 0.28rem;
  height: 0.28rem;
  border-radius: 9999px;
  background: #cbd5e1;
}

.install-copy-btn {
  border-radius: 9999px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(255, 255, 255, 0.08);
  padding: 0.55rem 0.95rem;
  font-size: 0.75rem;
  font-weight: 1000;
  color: white;
  transition: all 0.2s ease;
}

.install-copy-btn:hover {
  background: rgba(255, 255, 255, 0.14);
  transform: translateY(-1px);
}

.install-command-pre {
  margin: 0;
  overflow-x: auto;
  padding: 1.35rem 1.4rem 1.5rem;
  color: #d8f3ff;
  font-family: ui-monospace, 'Fira Code', monospace;
  font-size: 0.9rem;
  line-height: 1.9;
  white-space: pre-wrap;
  word-break: break-word;
}

.install-command-meta {
  display: grid;
  gap: 0.75rem;
  padding: 1rem 1rem 0;
}

.install-command-meta-card {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 1rem;
  background: rgba(255, 255, 255, 0.04);
  padding: 0.85rem 0.95rem;
}

.install-command-meta-label {
  font-size: 0.68rem;
  font-weight: 1000;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: #94a3b8;
}

.install-command-meta-value {
  margin-top: 0.35rem;
  color: #f8fafc;
  font-size: 0.8rem;
  font-weight: 800;
  line-height: 1.6;
}

.install-command-footer {
  display: flex;
  flex-wrap: wrap;
  gap: 0.55rem;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  padding: 0 1rem 1rem;
}

.install-command-footer-pill {
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  background: rgba(255, 255, 255, 0.08);
  padding: 0.38rem 0.72rem;
  font-size: 0.72rem;
  font-weight: 900;
  color: #d8b4fe;
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
.dark .hero-install-pill,
.dark .install-lab,
.dark .install-summary-card,
.dark .install-tool-card,
.dark .install-studio,
.dark .install-config-card,
.dark .install-metric-card,
.dark .install-step-card,
.dark .float-capsule,
.dark .mini-card,
.dark .pill-card,
.dark .provider-capsule,
.dark .feature-card {
  border-color: rgba(255, 255, 255, 0.1);
  background: rgba(15, 23, 42, 0.58);
  color: #e2e8f0;
}

.dark .install-input-required {
  border-color: rgba(248, 113, 113, 0.98);
  box-shadow:
    0 0 0 4px rgba(248, 113, 113, 0.18),
    inset 0 1px 0 rgba(255, 255, 255, 0.08);
}

.dark .install-input-required::placeholder {
  color: #fca5a5;
}

:deep(.dark) .feature-card h3,
.dark .feature-card h3 {
  color: #ffffff;
}

:deep(.dark) .feature-card p,
.dark .feature-card p {
  color: #cbd5e1;
}

:deep(.dark) .install-badge,
.dark .install-badge,
:deep(.dark) .install-os-tabs,
.dark .install-os-tabs,
:deep(.dark) .install-fetch-meta,
.dark .install-fetch-meta,
:deep(.dark) .install-input,
.dark .install-input,
:deep(.dark) .install-static-value,
.dark .install-static-value {
  border-color: rgba(255, 255, 255, 0.1);
  background: rgba(15, 23, 42, 0.72);
  color: #e2e8f0;
}

:deep(.dark) .install-input::placeholder,
.dark .install-input::placeholder,
:deep(.dark) .install-field-label,
.dark .install-field-label,
:deep(.dark) .install-metric-label,
.dark .install-metric-label,
:deep(.dark) .install-step-desc,
.dark .install-step-desc {
  color: #94a3b8;
}

:deep(.dark) .install-step-title,
.dark .install-step-title,
:deep(.dark) .install-metric-value,
.dark .install-metric-value {
  color: #ffffff;
}

:deep(.dark) .hero-install-chip,
.dark .hero-install-chip,
:deep(.dark) .install-summary-chip,
.dark .install-summary-chip {
  background: rgba(255, 255, 255, 0.08);
  color: #d8b4fe;
}

:deep(.dark) .hero-highlight-title,
.dark .hero-highlight-title,
:deep(.dark) .hero-highlight-value,
.dark .hero-highlight-value,
:deep(.dark) .hero-proof-value,
.dark .hero-proof-value {
  color: #fff;
}

:deep(.dark) .hero-highlight-desc,
.dark .hero-highlight-desc,
:deep(.dark) .hero-highlight-label,
.dark .hero-highlight-label,
:deep(.dark) .hero-proof-key,
.dark .hero-proof-key,
:deep(.dark) .install-top-rail-text,
.dark .install-top-rail-text {
  color: #94a3b8;
}

:deep(.dark) .hero-status-ribbon,
.dark .hero-status-ribbon,
:deep(.dark) .install-top-rail,
.dark .install-top-rail {
  border-color: rgba(255, 255, 255, 0.1);
  background: rgba(15, 23, 42, 0.76);
}

:deep(.dark) .install-runtime-pill,
.dark .install-runtime-pill {
  background: rgba(251, 146, 60, 0.14);
  color: #fdba74;
}

:deep(.dark) .install-fetch-ok,
.dark .install-fetch-ok {
  color: #5eead4;
}

:deep(.dark) .install-fetch-divider,
.dark .install-fetch-divider {
  background: rgba(255, 255, 255, 0.16);
}

@media (min-width: 768px) {
  .install-command-meta {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
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

@keyframes heroShine {
  0%, 100% { transform: translateX(-140%) rotate(18deg); opacity: 0; }
  18% { opacity: 0.85; }
  48% { transform: translateX(260%) rotate(18deg); opacity: 0.3; }
  60% { opacity: 0; }
}

@media (max-width: 640px) {
  .float-capsule,
  .mini-card {
    display: none;
  }

  .hero-highlight-metrics,
  .hero-proof-row,
  .install-command-meta {
    grid-template-columns: 1fr;
  }

  .hero-cta-row {
    align-items: stretch;
  }

  .hero-cta {
    width: 100%;
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

  .hero-highlight-title {
    font-size: 1.15rem;
  }

  .hero-status-ribbon {
    display: none;
  }
}

@media (min-width: 640px) {
  .hero-cta-row {
    flex-direction: row;
    justify-content: flex-start;
  }
}
</style>
