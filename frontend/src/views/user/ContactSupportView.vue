<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="relative overflow-hidden rounded-[2rem] border border-amber-200/80 bg-[linear-gradient(135deg,rgba(255,251,235,0.96),rgba(254,242,242,0.96),rgba(240,249,255,0.96))] p-6 shadow-[0_28px_90px_-44px_rgba(217,119,6,0.45)] dark:border-amber-900/60 dark:bg-[linear-gradient(135deg,rgba(68,64,60,0.95),rgba(69,39,53,0.94),rgba(30,41,59,0.96))] sm:p-7">
        <div class="pointer-events-none absolute -right-10 top-6 h-28 w-28 rounded-full bg-pink-200/40 blur-3xl dark:bg-pink-400/10"></div>
        <div class="pointer-events-none absolute bottom-0 left-0 h-32 w-32 -translate-x-10 translate-y-10 rounded-full bg-sky-200/50 blur-3xl dark:bg-sky-400/10"></div>

        <div class="relative flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <div class="max-w-3xl">
            <div class="inline-flex rotate-[-2deg] items-center rounded-full border border-amber-300/80 bg-white/80 px-4 py-1.5 text-xs font-bold tracking-[0.22em] text-amber-700 shadow-sm dark:border-amber-500/30 dark:bg-slate-900/80 dark:text-amber-200">
              CONTACT ZONE
            </div>
            <h2 class="mt-4 text-3xl font-black tracking-tight text-slate-950 dark:text-white sm:text-4xl">
              来呀，聊聊呀
            </h2>
            <p class="mt-3 max-w-2xl text-sm leading-7 text-slate-700 dark:text-slate-200">
              有想法、想吐槽、想进群、想问点细节，都可以来敲站长。这里不是官腔接待处，更像一张随手留下的联系小纸条。
            </p>
            <div class="mt-4 flex flex-wrap gap-3 text-sm text-slate-600 dark:text-slate-300">
              <span class="rounded-full border border-white/80 bg-white/75 px-3 py-1 font-medium shadow-sm dark:border-slate-700 dark:bg-slate-900/80">
                下午到半夜，回复概率更高
              </span>
              <span class="rounded-full border border-pink-200/80 bg-pink-50/80 px-3 py-1 font-medium text-pink-700 shadow-sm dark:border-pink-500/20 dark:bg-pink-500/10 dark:text-pink-200">
                手慢无，但消息会看
              </span>
            </div>
          </div>

          <div class="grid gap-3 sm:grid-cols-2 lg:w-[440px]">
            <div class="rotate-[-2deg] rounded-[1.75rem] border border-emerald-200/80 bg-white/85 p-4 shadow-[0_18px_50px_-32px_rgba(16,185,129,0.55)] transition hover:-translate-y-0.5 dark:border-emerald-500/20 dark:bg-slate-900/85">
              <div class="inline-flex rounded-full bg-emerald-100 px-2.5 py-1 text-[11px] font-bold uppercase tracking-[0.24em] text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-200">
                WeChat
              </div>
              <div class="mt-3 text-lg font-black text-slate-950 dark:text-white">{{ wechatId }}</div>
              <p class="mt-1 text-sm leading-6 text-slate-600 dark:text-slate-300">适合随手戳一下，看到会回。</p>
              <button
                type="button"
                class="mt-3 rounded-full border border-emerald-300 bg-emerald-500 px-4 py-2 text-sm font-semibold text-white transition hover:scale-[1.02] hover:bg-emerald-600 dark:border-emerald-400/20 dark:bg-emerald-400 dark:text-slate-950 dark:hover:bg-emerald-300"
                @click="copyContact(wechatId, '微信号')"
              >
                复制微信号
              </button>
            </div>
            <div class="rotate-[1.5deg] rounded-[1.75rem] border border-sky-200/80 bg-white/85 p-4 shadow-[0_18px_50px_-32px_rgba(14,165,233,0.5)] transition hover:-translate-y-0.5 dark:border-sky-500/20 dark:bg-slate-900/85">
              <div class="inline-flex rounded-full bg-sky-100 px-2.5 py-1 text-[11px] font-bold uppercase tracking-[0.24em] text-sky-700 dark:bg-sky-500/10 dark:text-sky-200">
                QQ
              </div>
              <div class="mt-3 text-lg font-black text-slate-950 dark:text-white">{{ qqId }}</div>
              <p class="mt-1 text-sm leading-6 text-slate-600 dark:text-slate-300">老派但稳，适合直接留言。</p>
              <button
                type="button"
                class="mt-3 rounded-full border border-sky-300 bg-sky-500 px-4 py-2 text-sm font-semibold text-white transition hover:scale-[1.02] hover:bg-sky-600 dark:border-sky-400/20 dark:bg-sky-400 dark:text-slate-950 dark:hover:bg-sky-300"
                @click="copyContact(qqId, 'QQ 号')"
              >
                复制 QQ 号
              </button>
            </div>
          </div>
        </div>
      </section>

      <section class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <article class="rounded-[2rem] border border-amber-200/70 bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(255,251,235,0.95))] p-6 shadow-[0_24px_80px_-38px_rgba(245,158,11,0.35)] dark:border-amber-900/40 dark:bg-[linear-gradient(180deg,rgba(15,23,42,0.98),rgba(51,65,85,0.96))]">
          <div class="flex flex-wrap items-start justify-between gap-4">
            <div>
              <div class="inline-flex rotate-[-2deg] rounded-full border border-violet-200 bg-violet-50 px-3 py-1 text-[11px] font-bold uppercase tracking-[0.24em] text-violet-700 dark:border-violet-500/20 dark:bg-violet-500/10 dark:text-violet-200">
                群聊小门牌
              </div>
              <h3 class="mt-3 text-2xl font-black tracking-tight text-slate-950 dark:text-white">
                {{ displayedQr ? '扫一扫，进群唠嗑' : '群门口还在整理中' }}
              </h3>
              <p class="mt-2 max-w-xl text-sm leading-7 text-slate-600 dark:text-slate-300">
                {{ displayedQr ? '二维码已经贴好了，想进群直接扫就行。' : '二维码暂时还没贴上来，先走微信或 QQ 这条线也完全没问题。' }}
              </p>
            </div>
            <div
              class="rounded-full border px-3 py-1 text-xs font-bold tracking-[0.22em] shadow-sm"
              :class="displayedQr
                ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-200'
                : 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-200'"
            >
              {{ displayedQr ? '已贴好' : '待贴上' }}
            </div>
          </div>

          <div class="mt-6 rounded-[1.75rem] border border-dashed border-amber-300/80 bg-white/75 p-3 shadow-inner dark:border-amber-500/20 dark:bg-slate-900/70">
            <div class="flex min-h-[340px] items-center justify-center rounded-[1.4rem] bg-[radial-gradient(circle_at_top,_rgba(254,243,199,0.45),_transparent_40%),linear-gradient(180deg,rgba(255,255,255,0.92),rgba(255,247,237,0.92))] p-6 dark:bg-[radial-gradient(circle_at_top,_rgba(250,204,21,0.08),_transparent_35%),linear-gradient(180deg,rgba(15,23,42,0.9),rgba(30,41,59,0.92))]">
              <img
                v-if="displayedQr"
                :src="displayedQr"
                alt="官方群二维码"
                class="max-h-[420px] w-full max-w-[420px] rounded-[1.4rem] border-4 border-white object-contain shadow-[0_20px_60px_-30px_rgba(15,23,42,0.55)] dark:border-slate-800"
              />
              <div v-else class="max-w-sm rounded-[1.4rem] border border-white/80 bg-white/75 px-5 py-6 text-center text-sm leading-7 text-slate-600 shadow-sm dark:border-slate-700 dark:bg-slate-900/80 dark:text-slate-300">
                站长还没把群二维码贴出来，别慌。
                <br />
                你可以先加微信 {{ wechatId }}，或者 QQ {{ qqId }}，先搭上话再说。
              </div>
            </div>
          </div>

          <div v-if="isAdmin" class="mt-5 rounded-[1.4rem] border border-amber-200/80 bg-amber-50/80 px-4 py-3 text-sm leading-7 text-amber-900 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-100">
            选好图以后点一下“发布群二维码”，用户端会立刻刷到最新那张，不用等很久。
          </div>
        </article>

        <aside class="rounded-[2rem] border border-rose-200/70 bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(253,242,248,0.96))] p-6 shadow-[0_24px_80px_-40px_rgba(244,114,182,0.35)] dark:border-rose-900/40 dark:bg-[linear-gradient(180deg,rgba(15,23,42,0.98),rgba(58,28,43,0.96))]">
          <div class="inline-flex rotate-[-2deg] rounded-full border border-rose-200 bg-white/90 px-3 py-1 text-[11px] font-bold uppercase tracking-[0.24em] text-rose-700 shadow-sm dark:border-rose-500/20 dark:bg-slate-900/80 dark:text-rose-200">
            小纸条联系方式
          </div>
          <div class="mt-4 space-y-4">
            <div class="rounded-[1.5rem] border border-rose-200/80 bg-white/80 p-4 shadow-sm dark:border-rose-500/20 dark:bg-slate-900/80">
              <div class="text-sm font-medium text-slate-500 dark:text-slate-300">微信号</div>
              <div class="mt-1 text-lg font-black text-slate-950 dark:text-white">{{ wechatId }}</div>
            </div>
            <div class="rounded-[1.5rem] border border-sky-200/80 bg-white/80 p-4 shadow-sm dark:border-sky-500/20 dark:bg-slate-900/80">
              <div class="text-sm font-medium text-slate-500 dark:text-slate-300">QQ 号</div>
              <div class="mt-1 text-lg font-black text-slate-950 dark:text-white">{{ qqId }}</div>
            </div>
            <div class="rounded-[1.6rem] border border-amber-200/80 bg-[linear-gradient(135deg,rgba(255,251,235,0.96),rgba(255,255,255,0.92))] p-4 shadow-sm dark:border-amber-500/20 dark:bg-[linear-gradient(135deg,rgba(120,53,15,0.18),rgba(15,23,42,0.9))]">
              <div class="flex items-start justify-between gap-3">
                <div>
                  <div class="text-sm font-medium text-slate-500 dark:text-slate-300">联系电话</div>
                  <div class="mt-1 text-lg font-black text-slate-950 dark:text-white">+8617333910159</div>
                </div>
                <span class="rounded-full bg-amber-100 px-3 py-1 text-[11px] font-bold tracking-[0.18em] text-amber-700 dark:bg-amber-500/10 dark:text-amber-200">
                  下午在线
                </span>
              </div>
              <p class="mt-3 text-sm leading-7 text-slate-600 dark:text-slate-300">
                上午不要找他，你也找不到。
                <br />
                他是个技术宅，通常从下午到半夜处理业务。
              </p>
              <button
                type="button"
                class="mt-3 rounded-full border border-amber-300 bg-amber-500 px-4 py-2 text-sm font-semibold text-white transition hover:scale-[1.02] hover:bg-amber-600 dark:border-amber-400/20 dark:bg-amber-400 dark:text-slate-950 dark:hover:bg-amber-300"
                @click="copyContact('+8617333910159', '联系电话')"
              >
                复制联系电话
              </button>
            </div>
            <div v-if="contactInfo" class="rounded-[1.5rem] border border-slate-200/80 bg-white/80 p-4 text-sm leading-7 text-slate-600 shadow-sm dark:border-slate-700 dark:bg-slate-900/80 dark:text-slate-300">
              {{ contactInfo }}
            </div>
          </div>

          <div v-if="isAdmin" class="mt-6 border-t border-rose-200/80 pt-6 dark:border-rose-500/20">
            <div class="inline-flex rounded-full bg-white/80 px-3 py-1 text-[11px] font-bold uppercase tracking-[0.24em] text-rose-600 dark:bg-slate-900/80 dark:text-rose-200">
              管理员贴纸区
            </div>
            <p class="mt-3 text-sm leading-7 text-slate-600 dark:text-slate-300">
              选一张清晰点的群二维码，预览没问题就直接发，前台会马上看到新版本。
            </p>

            <label class="mt-4 flex cursor-pointer flex-col items-center justify-center rounded-[1.6rem] border border-dashed border-rose-300 bg-white/80 px-4 py-6 text-center transition hover:border-rose-400 hover:bg-rose-50/80 dark:border-rose-500/20 dark:bg-slate-900/70 dark:hover:border-rose-400/40 dark:hover:bg-slate-900">
              <span class="rounded-full bg-rose-100 px-3 py-1 text-xs font-bold text-rose-700 dark:bg-rose-500/10 dark:text-rose-200">
                选择群二维码图片
              </span>
              <span class="mt-3 text-xs leading-6 text-slate-500 dark:text-slate-400">{{ selectedFileName || '支持常见图片格式，最好用清晰的正方形二维码' }}</span>
              <input class="hidden" type="file" accept="image/*" @change="handleQrFileChange" />
            </label>

            <div v-if="saveError" class="mt-4 rounded-[1.4rem] border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700 dark:border-rose-900/70 dark:bg-rose-950/30 dark:text-rose-300">
              {{ saveError }}
            </div>

            <div class="mt-4 flex flex-wrap gap-3">
              <button
                type="button"
                class="rounded-full bg-slate-950 px-4 py-2.5 text-sm font-semibold text-white transition hover:scale-[1.02] hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-white dark:text-slate-950 dark:hover:bg-slate-200"
                :disabled="saving || !hasDraftChanges"
                @click="publishQrCode"
              >
                {{ saving ? '正在发布...' : '发布群二维码' }}
              </button>
              <button
                type="button"
                class="rounded-full border border-slate-300 bg-white/80 px-4 py-2.5 text-sm font-semibold text-slate-700 transition hover:scale-[1.02] hover:border-slate-400 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700 dark:bg-slate-900/80 dark:text-slate-200 dark:hover:border-slate-500 dark:hover:bg-slate-900"
                :disabled="saving || !displayedQr"
                @click="clearQrCode"
              >
                清空已发布二维码
              </button>
            </div>
          </div>
        </aside>
      </section>

      <section class="relative overflow-hidden rounded-[2rem] border border-emerald-200/70 bg-[linear-gradient(135deg,rgba(236,253,245,0.98),rgba(255,251,235,0.95),rgba(255,255,255,0.96))] p-6 shadow-[0_24px_90px_-45px_rgba(16,185,129,0.35)] dark:border-emerald-900/40 dark:bg-[linear-gradient(135deg,rgba(6,78,59,0.35),rgba(30,41,59,0.96),rgba(51,65,85,0.95))] sm:p-7">
        <div class="pointer-events-none absolute -right-16 top-8 h-36 w-36 rounded-full bg-emerald-200/40 blur-3xl dark:bg-emerald-400/10"></div>
        <div class="pointer-events-none absolute -left-10 bottom-0 h-32 w-32 rounded-full bg-violet-200/40 blur-3xl dark:bg-violet-400/10"></div>

        <div class="relative">
          <div class="flex flex-wrap items-start justify-between gap-4">
            <div class="max-w-3xl">
              <div class="inline-flex rotate-[-2deg] rounded-full border border-emerald-200 bg-white/90 px-3 py-1 text-[11px] font-bold uppercase tracking-[0.24em] text-emerald-700 shadow-sm dark:border-emerald-500/20 dark:bg-slate-900/80 dark:text-emerald-200">
                AI IDEA BOARD
              </div>
              <h3 class="mt-4 text-2xl font-black tracking-tight text-slate-950 dark:text-white sm:text-3xl">
                AI 创业想法留言板
              </h3>
              <p class="mt-3 max-w-2xl text-sm leading-7 text-slate-600 dark:text-slate-300">
                有脑洞就贴上来，方向、产品、工具、流量点子都行。这里像一面公开灵感墙，大家能看，管理员也会偶尔在下面贴一张官方便签。
              </p>
            </div>
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="rotate-[-2deg] rounded-[1.5rem] border border-emerald-200/80 bg-white/85 px-4 py-3 shadow-sm dark:border-emerald-500/20 dark:bg-slate-900/80">
                <div class="text-xs font-semibold uppercase tracking-[0.2em] text-emerald-600 dark:text-emerald-200">灵感总数</div>
                <div class="mt-2 text-2xl font-black text-slate-950 dark:text-white">{{ ideaPagination.total }}</div>
              </div>
              <div class="rotate-[1.5deg] rounded-[1.5rem] border border-violet-200/80 bg-white/85 px-4 py-3 shadow-sm dark:border-violet-500/20 dark:bg-slate-900/80">
                <div class="text-xs font-semibold uppercase tracking-[0.2em] text-violet-600 dark:text-violet-200">当前视角</div>
                <div class="mt-2 text-lg font-black text-slate-950 dark:text-white">{{ ideaMineOnly ? '我的留言' : '全部灵感' }}</div>
              </div>
            </div>
          </div>

          <div class="mt-8 grid gap-6 xl:grid-cols-[minmax(0,420px)_minmax(0,1fr)]">
            <div class="rounded-[1.8rem] border border-emerald-200/80 bg-white/85 p-5 shadow-[0_20px_60px_-36px_rgba(16,185,129,0.35)] dark:border-emerald-500/20 dark:bg-slate-900/82">
              <div class="flex items-center justify-between gap-3">
                <div>
                  <div class="text-sm font-black text-slate-950 dark:text-white">贴一张灵感小纸条</div>
                  <p class="mt-1 text-xs leading-6 text-slate-500 dark:text-slate-400">
                    标题建议说人话，正文尽量具体，别人才能接得住。
                  </p>
                </div>
              </div>

              <div class="mt-5 space-y-4">
                <label class="block">
                  <div class="mb-2 text-sm font-semibold text-slate-700 dark:text-slate-200">灵感标题</div>
                  <input
                    v-model.trim="ideaForm.title"
                    type="text"
                    maxlength="120"
                    placeholder="比如：做一个帮小团队验证 AI 产品需求的小工具"
                    class="w-full rounded-[1.2rem] border border-emerald-200/80 bg-white/90 px-4 py-3 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-emerald-400 focus:ring-2 focus:ring-emerald-200 dark:border-emerald-500/20 dark:bg-slate-950/80 dark:text-white dark:placeholder:text-slate-500 dark:focus:border-emerald-400 dark:focus:ring-emerald-500/20"
                  />
                  <div class="mt-2 text-right text-xs text-slate-400 dark:text-slate-500">{{ ideaTitleLength }}/120</div>
                </label>

                <label class="block">
                  <div class="mb-2 text-sm font-semibold text-slate-700 dark:text-slate-200">展开说说</div>
                  <textarea
                    v-model.trim="ideaForm.content"
                    rows="7"
                    maxlength="2000"
                    placeholder="写清楚目标用户、核心场景、为什么值得做，越具体越容易收到像样的回复。"
                    class="w-full rounded-[1.2rem] border border-emerald-200/80 bg-white/90 px-4 py-3 text-sm leading-7 text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-emerald-400 focus:ring-2 focus:ring-emerald-200 dark:border-emerald-500/20 dark:bg-slate-950/80 dark:text-white dark:placeholder:text-slate-500 dark:focus:border-emerald-400 dark:focus:ring-emerald-500/20"
                  ></textarea>
                  <div class="mt-2 flex items-center justify-between gap-3 text-xs text-slate-400 dark:text-slate-500">
                    <span>别写太空，写到能让别人接着想。</span>
                    <span>{{ ideaContentLength }}/2000</span>
                  </div>
                </label>

                <div v-if="ideaError" class="rounded-[1.2rem] border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700 dark:border-rose-900/70 dark:bg-rose-950/30 dark:text-rose-300">
                  {{ ideaError }}
                </div>

                <div class="flex flex-wrap items-center gap-3">
                  <button
                    type="button"
                    class="rounded-full bg-emerald-500 px-5 py-2.5 text-sm font-semibold text-white transition hover:scale-[1.02] hover:bg-emerald-600 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-emerald-400 dark:text-slate-950 dark:hover:bg-emerald-300"
                    :disabled="ideaSubmitting || !canSubmitIdea"
                    @click="submitIdeaMessage"
                  >
                    {{ ideaSubmitting ? '正在贴上去...' : '发布想法' }}
                  </button>
                  <button
                    type="button"
                    class="rounded-full border border-slate-300 bg-white/80 px-4 py-2.5 text-sm font-semibold text-slate-700 transition hover:scale-[1.02] hover:border-slate-400 hover:bg-slate-100 dark:border-slate-700 dark:bg-slate-900/80 dark:text-slate-200 dark:hover:border-slate-500 dark:hover:bg-slate-900"
                    :disabled="ideaSubmitting || (!ideaForm.title && !ideaForm.content)"
                    @click="resetIdeaForm"
                  >
                    清空草稿
                  </button>
                </div>
              </div>
            </div>

            <div class="rounded-[1.8rem] border border-violet-200/80 bg-white/85 p-5 shadow-[0_20px_60px_-36px_rgba(139,92,246,0.25)] dark:border-violet-500/20 dark:bg-slate-900/82">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <div class="text-sm font-black text-slate-950 dark:text-white">灵感墙</div>
                  <p class="mt-1 text-xs leading-6 text-slate-500 dark:text-slate-400">
                    {{ ideaMineOnly ? '这里只看你自己的留言。' : '这里会显示所有登录用户还没删除的留言。' }}
                  </p>
                </div>
                <div class="inline-flex rounded-full border border-violet-200 bg-violet-50 p-1 dark:border-violet-500/20 dark:bg-violet-500/10">
                  <button
                    type="button"
                    class="rounded-full px-4 py-2 text-sm font-semibold transition"
                    :class="ideaMineOnly ? 'text-slate-500 dark:text-slate-300' : 'bg-white text-violet-700 shadow-sm dark:bg-slate-900 dark:text-violet-200'"
                    @click="showAllIdeas"
                  >
                    全部灵感
                  </button>
                  <button
                    type="button"
                    class="rounded-full px-4 py-2 text-sm font-semibold transition"
                    :class="ideaMineOnly ? 'bg-white text-violet-700 shadow-sm dark:bg-slate-900 dark:text-violet-200' : 'text-slate-500 dark:text-slate-300'"
                    @click="showMineIdeas"
                  >
                    我的留言
                  </button>
                </div>
              </div>

              <div class="mt-5 space-y-4">
                <div
                  v-for="message in ideaList"
                  :key="message.id"
                  class="rounded-[1.5rem] border border-slate-200/80 bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(248,250,252,0.96))] p-4 shadow-sm dark:border-slate-700 dark:bg-[linear-gradient(180deg,rgba(15,23,42,0.96),rgba(30,41,59,0.94))]"
                >
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div class="min-w-0 flex-1">
                      <div class="flex flex-wrap items-center gap-2">
                        <h4 class="text-lg font-black text-slate-950 dark:text-white">{{ message.title }}</h4>
                        <span
                          v-if="message.is_mine"
                          class="rounded-full bg-amber-100 px-2.5 py-1 text-[11px] font-bold text-amber-700 dark:bg-amber-500/10 dark:text-amber-200"
                        >
                          我的
                        </span>
                        <span
                          v-if="message.admin_reply"
                          class="rounded-full bg-emerald-100 px-2.5 py-1 text-[11px] font-bold text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-200"
                        >
                          已有官方便签
                        </span>
                      </div>
                      <div class="mt-2 flex flex-wrap items-center gap-3 text-xs text-slate-500 dark:text-slate-400">
                        <span>{{ message.author_name }}</span>
                        <span>·</span>
                        <span>{{ formatIdeaTime(message.created_at) }}</span>
                      </div>
                    </div>

                    <div class="flex flex-wrap items-center gap-2">
                      <button
                        v-if="isAdmin && message.can_reply"
                        type="button"
                        class="rounded-full border border-violet-200 bg-violet-50 px-3 py-2 text-xs font-semibold text-violet-700 transition hover:scale-[1.02] hover:bg-violet-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-violet-500/20 dark:bg-violet-500/10 dark:text-violet-200 dark:hover:bg-violet-500/20"
                        :disabled="isIdeaActionLoading(message.id)"
                        @click="startReplyEdit(message)"
                      >
                        {{ replyEditingId === message.id ? '正在编辑回复' : (message.admin_reply ? '修改回复' : '回复') }}
                      </button>
                      <button
                        v-if="message.can_delete"
                        type="button"
                        class="rounded-full border border-rose-200 bg-rose-50 px-3 py-2 text-xs font-semibold text-rose-700 transition hover:scale-[1.02] hover:bg-rose-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-rose-500/20 dark:bg-rose-500/10 dark:text-rose-200 dark:hover:bg-rose-500/20"
                        :disabled="isIdeaActionLoading(message.id)"
                        @click="deleteIdeaMessage(message)"
                      >
                        {{ isIdeaActionLoading(message.id) ? '处理中...' : '删除' }}
                      </button>
                    </div>
                  </div>

                  <p class="mt-4 whitespace-pre-wrap text-sm leading-7 text-slate-700 dark:text-slate-200">
                    {{ message.content }}
                  </p>

                  <div
                    v-if="message.admin_reply"
                    class="mt-4 rounded-[1.3rem] border border-emerald-200/80 bg-[linear-gradient(135deg,rgba(236,253,245,0.92),rgba(255,255,255,0.96))] px-4 py-3 shadow-sm dark:border-emerald-500/20 dark:bg-[linear-gradient(135deg,rgba(6,95,70,0.2),rgba(15,23,42,0.92))]"
                  >
                    <div class="flex flex-wrap items-center justify-between gap-2">
                      <div class="text-xs font-bold uppercase tracking-[0.22em] text-emerald-700 dark:text-emerald-200">官方便签</div>
                      <div class="text-xs text-emerald-700/80 dark:text-emerald-200/80">
                        {{ message.admin_reply_at ? formatIdeaTime(message.admin_reply_at) : '' }}
                      </div>
                    </div>
                    <p class="mt-2 whitespace-pre-wrap text-sm leading-7 text-slate-700 dark:text-slate-100">
                      {{ message.admin_reply }}
                    </p>
                  </div>

                  <div
                    v-if="isAdmin && replyEditingId === message.id"
                    class="mt-4 rounded-[1.3rem] border border-violet-200/80 bg-violet-50/80 p-4 dark:border-violet-500/20 dark:bg-violet-500/10"
                  >
                    <div class="flex items-center justify-between gap-3">
                      <div class="text-sm font-black text-slate-950 dark:text-white">给这张纸条回个便签</div>
                      <div class="text-xs text-slate-400 dark:text-slate-500">{{ getReplyLength(message.id) }}/1000</div>
                    </div>
                    <textarea
                      v-model.trim="replyDrafts[message.id]"
                      rows="4"
                      maxlength="1000"
                      placeholder="比如：这个方向值得做，但建议先收窄到一个具体使用场景。"
                      class="mt-3 w-full rounded-[1.1rem] border border-violet-200/80 bg-white/90 px-4 py-3 text-sm leading-7 text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-violet-400 focus:ring-2 focus:ring-violet-200 dark:border-violet-500/20 dark:bg-slate-950/80 dark:text-white dark:placeholder:text-slate-500 dark:focus:border-violet-400 dark:focus:ring-violet-500/20"
                    ></textarea>
                    <div class="mt-3 flex flex-wrap items-center gap-3">
                      <button
                        type="button"
                        class="rounded-full bg-violet-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:scale-[1.02] hover:bg-violet-600 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-violet-400 dark:text-slate-950 dark:hover:bg-violet-300"
                        :disabled="isIdeaActionLoading(message.id) || !canSaveReply(message.id)"
                        @click="saveAdminReply(message)"
                      >
                        {{ isIdeaActionLoading(message.id) ? '保存中...' : '保存回复' }}
                      </button>
                      <button
                        type="button"
                        class="rounded-full border border-slate-300 bg-white/80 px-4 py-2.5 text-sm font-semibold text-slate-700 transition hover:scale-[1.02] hover:border-slate-400 hover:bg-slate-100 dark:border-slate-700 dark:bg-slate-900/80 dark:text-slate-200 dark:hover:border-slate-500 dark:hover:bg-slate-900"
                        :disabled="isIdeaActionLoading(message.id)"
                        @click="cancelReplyEdit"
                      >
                        先收起来
                      </button>
                    </div>
                  </div>
                </div>

                <div
                  v-if="!ideaLoading && ideaList.length === 0"
                  class="rounded-[1.5rem] border border-dashed border-slate-300/80 bg-white/70 px-5 py-8 text-center text-sm leading-7 text-slate-500 dark:border-slate-700 dark:bg-slate-900/70 dark:text-slate-400"
                >
                  {{ ideaMineOnly ? '你还没贴过灵感小纸条，先来一条试试。' : '这面墙现在还空着，来贴第一张灵感纸条吧。' }}
                </div>

                <div v-if="ideaLoading && ideaList.length === 0" class="rounded-[1.5rem] border border-slate-200/80 bg-white/70 px-5 py-8 text-center text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900/70 dark:text-slate-400">
                  正在把灵感墙挂出来...
                </div>

                <div v-if="ideaHasMore" class="pt-2 text-center">
                  <button
                    type="button"
                    class="rounded-full border border-slate-300 bg-white/90 px-5 py-2.5 text-sm font-semibold text-slate-700 transition hover:scale-[1.02] hover:border-slate-400 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700 dark:bg-slate-900/85 dark:text-slate-200 dark:hover:border-slate-500 dark:hover:bg-slate-900"
                    :disabled="ideaLoading"
                    @click="loadMoreIdeaMessages"
                  >
                    {{ ideaLoading ? '正在多挂几张...' : '加载更多' }}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ideaMessagesAPI } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import type { IdeaMessage } from '@/types'

const appStore = useAppStore()
const authStore = useAuthStore()

const wechatId = 'Mictimeles'
const qqId = '363164954'

const loading = ref(false)
const saving = ref(false)
const saveError = ref('')
const selectedFileName = ref('')
const qrDraft = ref('')
const ideaLoading = ref(false)
const ideaSubmitting = ref(false)
const ideaError = ref('')
const ideaList = ref<IdeaMessage[]>([])
const ideaActionLoadingIds = ref<number[]>([])
const replyEditingId = ref<number | null>(null)
const ideaPagination = reactive({
  page: 1,
  page_size: 10,
  pages: 1,
  total: 0
})
const ideaForm = reactive({
  title: '',
  content: ''
})
const replyDrafts = reactive<Record<number, string>>({})
const ideaMineOnly = ref(false)
const ideaPageSize = 10
const ideaTimeFormatter = new Intl.DateTimeFormat('zh-CN', {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit'
})

const isAdmin = computed(() => authStore.isAdmin)
const contactInfo = computed(() => appStore.cachedPublicSettings?.contact_info?.trim() || '')
const publishedQr = computed(() => appStore.cachedPublicSettings?.contact_group_qr?.trim() || '')
const displayedQr = computed(() => (isAdmin.value ? (qrDraft.value.trim() || publishedQr.value) : publishedQr.value))
const hasDraftChanges = computed(() => isAdmin.value && qrDraft.value.trim() !== publishedQr.value)
const ideaTitleLength = computed(() => ideaForm.title.length)
const ideaContentLength = computed(() => ideaForm.content.length)
const ideaHasMore = computed(() => ideaPagination.page < ideaPagination.pages)
const canSubmitIdea = computed(() => {
  const titleLength = ideaForm.title.trim().length
  const contentLength = ideaForm.content.trim().length
  return titleLength > 0 && titleLength <= 120 && contentLength > 0 && contentLength <= 2000
})

async function loadContactPage() {
  loading.value = true
  saveError.value = ''

  try {
    const settings = await appStore.fetchPublicSettings(true)
    qrDraft.value = settings?.contact_group_qr || ''
  } catch (error) {
    console.error('Failed to load contact support page:', error)
    appStore.showError('联系页初始化失败')
  } finally {
    loading.value = false
  }
}

function readFileAsDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()

    reader.onload = () => {
      if (typeof reader.result === 'string') {
        resolve(reader.result)
        return
      }
      reject(new Error('二维码读取失败'))
    }

    reader.onerror = () => reject(new Error('二维码读取失败'))
    reader.readAsDataURL(file)
  })
}

async function handleQrFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]

  if (!file) {
    return
  }

  if (!file.type.startsWith('image/')) {
    saveError.value = '请上传图片文件'
    target.value = ''
    return
  }

  try {
    qrDraft.value = await readFileAsDataUrl(file)
    selectedFileName.value = file.name
    saveError.value = ''
  } catch (error) {
    console.error('Failed to read qr image:', error)
    saveError.value = '二维码读取失败，请重新选择图片'
  } finally {
    target.value = ''
  }
}

async function publishQrCode() {
  if (!isAdmin.value) {
    return
  }

  saving.value = true
  saveError.value = ''

  try {
    await adminAPI.settings.updateSettings({
      contact_group_qr: qrDraft.value.trim()
    })
    await loadContactPage()
    appStore.showSuccess('群二维码已发布')
  } catch (error) {
    console.error('Failed to publish group qr:', error)
    saveError.value = '发布群二维码失败'
    appStore.showError('发布群二维码失败')
  } finally {
    saving.value = false
  }
}

async function clearQrCode() {
  qrDraft.value = ''
  selectedFileName.value = ''
  await publishQrCode()
}

async function copyContact(value: string, label: string) {
  try {
    await navigator.clipboard.writeText(value)
    appStore.showSuccess(`${label}已复制`)
  } catch (error) {
    console.error(`Failed to copy ${label}:`, error)
    appStore.showError(`${label}复制失败`)
  }
}

async function loadIdeaMessages(options: { reset?: boolean } = {}) {
  if (ideaLoading.value) {
    return
  }

  const reset = options.reset ?? false
  ideaLoading.value = true
  ideaError.value = ''
  if (reset) {
    ideaList.value = []
  }

  try {
    const nextPage = reset ? 1 : ideaPagination.page + 1
    const response = await ideaMessagesAPI.list(nextPage, ideaPageSize, {
      mine_only: ideaMineOnly.value
    })

    ideaPagination.page = response.page
    ideaPagination.page_size = response.page_size
    ideaPagination.pages = response.pages
    ideaPagination.total = response.total
    ideaList.value = reset ? response.items : mergeIdeaMessages(ideaList.value, response.items)
  } catch (error) {
    console.error('Failed to load idea messages:', error)
    ideaError.value = '灵感墙加载失败，稍后再试试'
  } finally {
    ideaLoading.value = false
  }
}

function resetIdeaForm() {
  ideaForm.title = ''
  ideaForm.content = ''
  ideaError.value = ''
}

async function submitIdeaMessage() {
  if (!canSubmitIdea.value || ideaSubmitting.value) {
    return
  }

  ideaSubmitting.value = true
  ideaError.value = ''

  try {
    const created = await ideaMessagesAPI.create({
      title: ideaForm.title.trim(),
      content: ideaForm.content.trim()
    })

    ideaList.value = mergeIdeaMessages([created], ideaList.value)
    ideaPagination.total += 1
    resetIdeaForm()
    appStore.showSuccess('灵感纸条已经贴上去了')
  } catch (error) {
    console.error('Failed to create idea message:', error)
    ideaError.value = '发布灵感失败，请稍后再试'
    appStore.showError('发布灵感失败')
  } finally {
    ideaSubmitting.value = false
  }
}

async function deleteIdeaMessage(message: IdeaMessage) {
  if (isIdeaActionLoading(message.id)) {
    return
  }

  setIdeaActionLoading(message.id, true)

  try {
    await ideaMessagesAPI.delete(message.id)
    ideaList.value = ideaList.value.filter((item) => item.id !== message.id)
    ideaPagination.total = Math.max(0, ideaPagination.total - 1)

    if (replyEditingId.value === message.id) {
      replyEditingId.value = null
    }

    if (ideaList.value.length === 0 && ideaPagination.total > 0) {
      await loadIdeaMessages({ reset: true })
    }

    appStore.showSuccess('这张灵感纸条已经收起来了')
  } catch (error) {
    console.error('Failed to delete idea message:', error)
    appStore.showError('删除灵感失败')
  } finally {
    setIdeaActionLoading(message.id, false)
  }
}

function startReplyEdit(message: IdeaMessage) {
  replyEditingId.value = message.id
  replyDrafts[message.id] = message.admin_reply || ''
}

function cancelReplyEdit() {
  replyEditingId.value = null
}

function getReplyLength(id: number) {
  return (replyDrafts[id] || '').length
}

function canSaveReply(id: number) {
  const length = (replyDrafts[id] || '').trim().length
  return length > 0 && length <= 1000
}

async function saveAdminReply(message: IdeaMessage) {
  if (!canSaveReply(message.id) || isIdeaActionLoading(message.id)) {
    return
  }

  setIdeaActionLoading(message.id, true)

  try {
    const updated = await adminAPI.ideaMessages.reply(message.id, {
      admin_reply: (replyDrafts[message.id] || '').trim()
    })

    ideaList.value = ideaList.value.map((item) => item.id === message.id ? updated : item)
    replyDrafts[message.id] = updated.admin_reply || ''
    replyEditingId.value = null
    appStore.showSuccess('官方便签已经贴好了')
  } catch (error) {
    console.error('Failed to save admin reply:', error)
    appStore.showError('保存回复失败')
  } finally {
    setIdeaActionLoading(message.id, false)
  }
}

function showAllIdeas() {
  if (!ideaMineOnly.value) {
    return
  }
  ideaMineOnly.value = false
  void loadIdeaMessages({ reset: true })
}

function showMineIdeas() {
  if (ideaMineOnly.value) {
    return
  }
  ideaMineOnly.value = true
  void loadIdeaMessages({ reset: true })
}

function loadMoreIdeaMessages() {
  if (!ideaHasMore.value) {
    return
  }
  void loadIdeaMessages()
}

function formatIdeaTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return ideaTimeFormatter.format(date)
}

function isIdeaActionLoading(id: number) {
  return ideaActionLoadingIds.value.includes(id)
}

function setIdeaActionLoading(id: number, loadingState: boolean) {
  if (loadingState) {
    if (!ideaActionLoadingIds.value.includes(id)) {
      ideaActionLoadingIds.value = [...ideaActionLoadingIds.value, id]
    }
    return
  }
  ideaActionLoadingIds.value = ideaActionLoadingIds.value.filter((item) => item !== id)
}

function mergeIdeaMessages(current: IdeaMessage[], incoming: IdeaMessage[]) {
  const merged = [...current]
  for (const item of incoming) {
    const index = merged.findIndex((existing) => existing.id === item.id)
    if (index === -1) {
      merged.push(item)
    } else {
      merged[index] = item
    }
  }
  return merged
}

onMounted(() => {
  void Promise.all([
    loadContactPage(),
    loadIdeaMessages({ reset: true })
  ])
})
</script>
