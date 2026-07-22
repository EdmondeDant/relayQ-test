<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-[1500px] flex-col gap-5 pb-8">
      <div class="grid gap-5 lg:grid-cols-[220px_minmax(0,1fr)]">
        <aside class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-900">
          <nav class="grid grid-cols-2 gap-2 sm:grid-cols-5 lg:grid-cols-1">
            <button v-for="tool in tools" :key="tool.id" type="button" :class="toolButtonClass(tool.id)" @click="selectTool(tool.id)">
              <Icon :name="tool.icon" size="sm" />
              <span>{{ tool.label }}</span>
            </button>
          </nav>
          <div class="mt-4 hidden border-t border-gray-200 pt-4 text-xs leading-5 text-gray-500 dark:border-dark-700 dark:text-dark-400 lg:block">
            图片和视频的具体计费方式请去看模型价格。
          </div>
        </aside>

        <main class="min-w-0">
          <section v-if="activeTool === 'home'" class="space-y-5">
            <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
              <button v-for="tool in creationTools" :key="tool.id" type="button" class="rounded-lg border border-gray-200 bg-white p-5 text-left transition hover:border-primary-300 hover:shadow-sm dark:border-dark-700 dark:bg-dark-900 dark:hover:border-primary-700" @click="selectTool(tool.id)">
                <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-950/40 dark:text-primary-300"><Icon :name="tool.icon" /></div>
                <h2 class="mt-4 font-semibold text-gray-950 dark:text-dark-50">{{ tool.label }}</h2>
                <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-dark-300">{{ tool.description }}</p>
                <div class="mt-4 text-sm font-medium text-primary-600 dark:text-primary-300">开始创作</div>
              </button>
            </div>
            <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
              <div class="flex items-center justify-between">
                <div>
                  <h2 class="font-semibold text-gray-950 dark:text-dark-50">最近创作</h2>
                  <p class="mt-1 text-sm text-gray-500">服务端默认保存最近 10 条；超过 10 条时会自动删除最早 1 条；所有记录会在 2 天后自动清理。</p>
                </div>
                <button class="text-sm text-primary-600" type="button" @click="selectTool('history')">查看全部</button>
              </div>
              <RecordList class="mt-4" :items="cloudRecords.slice(0, 6)" :loading="cloudLoading" @restore="restoreRecord" @remove="removeRecord" @download="downloadRecord" />
            </section>
          </section>

          <section v-else-if="activeTool === 'image' || activeTool === 'edit'" class="grid gap-5 xl:grid-cols-[minmax(320px,0.8fr)_minmax(0,1.2fr)]">
            <div class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
              <h2 class="text-lg font-semibold text-gray-950 dark:text-dark-50">{{ activeTool === 'image' ? 'AI 图片生成' : 'AI 图片编辑' }}</h2>
              <p class="mt-1 text-sm text-gray-500">{{ activeTool === 'image' ? '从场景模板开始，或直接描述需要的画面。' : '上传原图并说明需要修改的内容。' }}</p>

              <div v-if="activeTool === 'edit'" class="mt-5">
                <label class="input-label">原图</label>
                <label class="flex min-h-44 cursor-pointer items-center justify-center overflow-hidden rounded-lg border border-dashed border-gray-300 bg-gray-50 p-4 text-center dark:border-dark-600 dark:bg-dark-800">
                  <img v-if="editImage" :src="editImage" alt="待编辑图片" class="max-h-64 w-full object-contain" />
                  <span v-else class="text-sm text-gray-500">点击选择 JPG、PNG 或 WEBP，最大 8MB</span>
                  <input class="hidden" type="file" accept="image/jpeg,image/png,image/webp" @change="handleImageFile" />
                </label>
                <button v-if="editImage" type="button" class="mt-2 text-xs text-red-500" @click="editImage = ''">移除图片</button>
              </div>

              <div class="mt-5">
                <label class="input-label">API Key</label>
                <select v-model.number="selectedKeyId" class="input">
                  <option :value="null" disabled>请选择 API Key</option>
                  <option v-for="option in keyOptions" :key="String(option.value)" :value="option.value">{{ option.label }}</option>
                </select>
                <p v-if="keyModeError" class="mt-2 text-xs text-red-500">{{ keyModeError }}</p>
                <p v-else-if="resolvedGroupName" class="mt-2 text-xs text-gray-500">当前分组：{{ resolvedGroupName }} · 平台：{{ resolvedPlatformLabel }}</p>
              </div>
              <div class="mt-5">
                <label class="input-label">模型</label>
                <select v-model="selectedImageModel" class="input">
                  <option value="" disabled>请选择模型</option>
                  <option v-for="option in imageModelOptions" :key="String(option.value)" :value="String(option.value)">{{ option.label }}</option>
                </select>
              </div>
              <div class="mt-5">
                <label class="input-label">场景模板</label>
                <div class="flex flex-wrap gap-2">
                  <button v-for="template in imageTemplates" :key="template.label" type="button" class="rounded-md border border-gray-200 px-3 py-1.5 text-xs text-gray-600 hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:text-dark-300" @click="imagePrompt = template.prompt">{{ template.label }}</button>
                </div>
              </div>
              <div class="mt-5">
                <label class="input-label">{{ activeTool === 'image' ? '画面描述' : '编辑要求' }}</label>
                <textarea v-model="imagePrompt" class="input min-h-40 resize-y leading-6" :placeholder="activeTool === 'image' ? '描述主体、环境、光线、构图和用途…' : '例如：保留商品主体，将背景替换为明亮的现代厨房…'" />
              </div>
              <div class="mt-5">
                <label class="input-label">{{ isGrokImagineSelected ? '宽高比' : '图片尺寸' }}</label>
                <Select v-model="imageSize" :options="imageSizeOptions" />
              </div>
              <div v-if="isGrokImagineSelected" class="mt-5">
                <label class="input-label">分辨率</label>
                <Select v-model="imageQuality" :options="grokResolutionOptions" />
                <p class="mt-2 text-xs text-gray-500">Grok Imagine 使用 aspect_ratio + resolution（1k/2k）。quality 模型负责更高画质，分辨率仍按 1k/2k 提交；不支持 style/background。</p>
              </div>
              <div v-else class="mt-5 grid grid-cols-3 gap-3">
                <div>
                  <label class="input-label">画质</label>
                  <Select v-model="imageQuality" :options="imageQualityOptions" />
                </div>
                <div>
                  <label class="input-label">风格</label>
                  <Select v-model="imageStyle" :options="imageStyleOptions" />
                </div>
                <div>
                  <label class="input-label">背景</label>
                  <Select v-model="imageBackground" :options="imageBackgroundOptions" />
                </div>
              </div>
              <div class="mt-5 flex gap-3">
                <button class="btn btn-primary flex-1" :disabled="!canSubmitImage || submitting" @click="submitImage">{{ submitting ? '处理中…' : activeTool === 'image' ? '生成图片' : '开始编辑' }}</button>
                <button v-if="submitting" class="btn btn-secondary" type="button" @click="stopRequest">停止等待</button>
              </div>
            </div>
            <ResultPanel :loading="submitting" :error="error" :billing="lastBilling">
              <template #result>
                <div v-if="resultImage" class="flex h-full flex-col gap-4">
                  <div class="flex min-h-[420px] flex-1 items-center justify-center overflow-hidden rounded-lg bg-gray-100 dark:bg-dark-800"><img :src="resultImage" alt="生成结果" class="max-h-[620px] w-full object-contain" /></div>
                  <div class="flex flex-wrap gap-2"><button class="btn btn-secondary btn-sm" type="button" @click="downloadResultImage">下载图片</button><button class="btn btn-secondary btn-sm" type="button" @click="submitImage">再次生成</button></div>
                </div>
              </template>
            </ResultPanel>
          </section>

          <section v-else-if="activeTool === 'chat'" class="grid gap-5 xl:grid-cols-[280px_minmax(0,1fr)]">
            <div class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
              <h2 class="text-lg font-semibold">对话助手</h2>
              <div class="mt-5"><label class="input-label">API Key</label><select v-model.number="chatSelectedKeyId" class="input"><option :value="null" disabled>请选择 API Key</option><option v-for="option in keyOptions" :key="String(option.value)" :value="option.value">{{ option.label }}</option></select><p v-if="chatKeyModeError" class="mt-2 text-xs text-red-500">{{ chatKeyModeError }}</p><p v-else-if="chatResolvedGroupName" class="mt-2 text-xs text-gray-500">当前分组：{{ chatResolvedGroupName }} · 平台：{{ chatResolvedPlatformLabel }}</p></div>
              <div class="mt-5"><label class="input-label">模型</label><select v-model="chatSelectedModel" class="input"><option value="" disabled>请选择模型</option><option v-for="option in chatModelOptions" :key="String(option.value)" :value="String(option.value)">{{ option.label }}</option></select></div>
              <div class="mt-5"><label class="input-label">快捷场景</label><div class="grid gap-2"><button v-for="template in chatTemplates" :key="template.label" type="button" class="rounded-md border border-gray-200 p-3 text-left text-sm hover:border-primary-300 dark:border-dark-700" @click="chatInput = template.prompt"><strong>{{ template.label }}</strong><span class="mt-1 block text-xs text-gray-500">{{ template.description }}</span></button></div></div>
              <button class="btn btn-secondary mt-5 w-full" type="button" @click="clearChat">清空会话</button>
            </div>
            <div class="flex min-h-[680px] flex-col rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
              <div class="flex-1 space-y-3 overflow-y-auto rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
                <div v-if="!chatMessages.length" class="flex h-full items-center justify-center text-sm text-gray-500">选择快捷场景或输入问题开始对话。</div>
                <div v-for="(message, index) in chatMessages" :key="index" :class="message.role === 'user' ? 'flex justify-end' : 'flex justify-start'"><div :class="['max-w-[85%] whitespace-pre-wrap rounded-lg px-4 py-3 text-sm leading-6', message.role === 'user' ? 'bg-primary-600 text-white' : 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-dark-50']">{{ message.content || '…' }}</div></div>
              </div>
              <div v-if="error" class="mt-3 rounded-md bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-300">{{ error }}</div>
              <div class="mt-4 flex gap-3"><textarea v-model="chatInput" class="input min-h-20 flex-1 resize-none" placeholder="输入问题，Enter 发送，Shift + Enter 换行" @keydown.enter.exact.prevent="sendChat" /><button v-if="submitting" class="btn btn-secondary self-end" @click="stopRequest">停止</button><button v-else class="btn btn-primary self-end" :disabled="!chatInput.trim()" @click="sendChat">发送</button></div>
              <div v-if="requestId || lastBilling" class="mt-3 flex flex-wrap gap-4 text-xs text-gray-500"><span v-if="requestId">request_id：{{ requestId }}</span><span v-if="lastBilling?.amount">实扣：{{ formatMoney(lastBilling.amount) }}</span></div>
            </div>
          </section>

          <section v-else-if="activeTool === 'copywriting'" class="grid gap-5 xl:grid-cols-[minmax(320px,0.8fr)_minmax(0,1.2fr)]">
            <div class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
              <h2 class="text-lg font-semibold">商品文案生成</h2>
              <p class="mt-1 text-sm text-gray-500">根据商品信息生成标题、卖点、详情描述和社媒文案。</p>
              <div class="mt-5"><label class="input-label">API Key</label><select v-model.number="selectedKeyId" class="input"><option :value="null" disabled>请选择 API Key</option><option v-for="option in keyOptions" :key="String(option.value)" :value="option.value">{{ option.label }}</option></select><p v-if="keyModeError" class="mt-2 text-xs text-red-500">{{ keyModeError }}</p></div>
              <div class="mt-5"><label class="input-label">模型</label><select v-model="selectedChatModel" class="input"><option value="" disabled>请选择模型</option><option v-for="option in chatModelOptions" :key="String(option.value)" :value="String(option.value)">{{ option.label }}</option></select></div>
              <div class="mt-5"><label class="input-label">商品名称</label><input v-model="copywritingName" class="input" placeholder="例如：便携式榨汁杯" /></div>
              <div class="mt-5"><label class="input-label">商品信息</label><textarea v-model="copywritingBrief" class="input min-h-40 resize-y" placeholder="填写材质、功能、适用人群、核心优势和需要规避的表达…" /></div>
              <div class="mt-5 grid grid-cols-2 gap-3"><div><label class="input-label">目标平台</label><Select v-model="copywritingPlatform" :options="copywritingPlatformOptions" /></div><div><label class="input-label">语言</label><Select v-model="copywritingLanguage" :options="languageOptions" /></div></div>
              <button class="btn btn-primary mt-5 w-full" :disabled="!copywritingName.trim() || !copywritingBrief.trim() || submitting" @click="generateCopywriting">{{ submitting ? '生成中…' : '生成并保存到作品库' }}</button>
            </div>
            <ResultPanel :loading="submitting" :error="error" :billing="lastBilling"><template #result><div v-if="textResult" class="flex h-full flex-col"><pre class="flex-1 whitespace-pre-wrap rounded-lg bg-gray-50 p-5 text-sm leading-7 dark:bg-dark-800">{{ textResult }}</pre><button class="btn btn-secondary mt-4 self-start" type="button" @click="copyTextResult">复制文案</button></div></template></ResultPanel>
          </section>

          <section v-else-if="activeTool === 'translate'" class="grid gap-5 xl:grid-cols-[minmax(320px,0.8fr)_minmax(0,1.2fr)]">
            <div class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
              <h2 class="text-lg font-semibold">图片翻译</h2>
              <p class="mt-1 text-sm text-gray-500">保留原图构图、版式与视觉风格，仅将图上文字替换为目标语言，输出译后图片。</p>
              <div class="mt-5"><label class="input-label">API Key</label><select v-model.number="selectedKeyId" class="input"><option :value="null" disabled>请选择 API Key</option><option v-for="option in keyOptions" :key="String(option.value)" :value="option.value">{{ option.label }}</option></select><p v-if="keyModeError" class="mt-2 text-xs text-red-500">{{ keyModeError }}</p></div>
              <div class="mt-5"><label class="input-label">模型</label><select v-model="selectedImageModel" class="input"><option value="" disabled>请选择图片模型</option><option v-for="option in imageModelOptions" :key="String(option.value)" :value="String(option.value)">{{ option.label }}</option></select><p class="mt-2 text-xs text-gray-500">使用图片编辑/生图模型完成“图上文字本地化”。</p></div>
              <div class="mt-5"><label class="input-label">原图</label><label class="flex min-h-48 cursor-pointer items-center justify-center overflow-hidden rounded-lg border border-dashed border-gray-300 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-800"><img v-if="translateImage" :src="translateImage" alt="待翻译图片" class="max-h-72 object-contain" /><span v-else class="text-sm text-gray-500">选择包含文字的 JPG、PNG 或 WEBP</span><input class="hidden" type="file" accept="image/jpeg,image/png,image/webp" @change="handleTranslateFile" /></label></div>
              <div class="mt-5 grid grid-cols-2 gap-3"><div><label class="input-label">源语言</label><Select v-model="translateSource" :options="sourceLanguageOptions" /></div><div><label class="input-label">目标语言</label><Select v-model="translateTarget" :options="languageOptions" /></div></div>
              <div class="mt-5">
                <label class="input-label">{{ isGrokImagineSelected ? '宽高比' : '图片尺寸' }}</label>
                <Select v-model="imageSize" :options="imageSizeOptions" />
              </div>
              <div v-if="isGrokImagineSelected" class="mt-5">
                <label class="input-label">分辨率</label>
                <Select v-model="imageQuality" :options="grokResolutionOptions" />
                <p class="mt-2 text-xs text-gray-500">Grok Imagine 使用 aspect_ratio + resolution（1k/2k）。quality 模型负责更高画质，分辨率仍按 1k/2k 提交。</p>
              </div>
              <div v-else class="mt-5 grid grid-cols-3 gap-3">
                <div><label class="input-label">画质</label><Select v-model="imageQuality" :options="imageQualityOptions" /></div>
                <div><label class="input-label">风格</label><Select v-model="imageStyle" :options="imageStyleOptions" /></div>
                <div><label class="input-label">背景</label><Select v-model="imageBackground" :options="imageBackgroundOptions" /></div>
              </div>
              <div class="mt-5 rounded-lg bg-gray-50 p-4 text-xs leading-5 text-gray-500 dark:bg-dark-800">会尽量保持人物、商品、背景、排版位置不变，只替换图中可见文字为目标语言。</div>
              <button class="btn btn-primary mt-5 w-full" :disabled="!translateImage || !selectedImageModel || submitting" @click="translateImageText">{{ submitting ? '图片翻译中…' : '开始图片翻译' }}</button>
            </div>
            <ResultPanel :loading="submitting" :error="error" :billing="lastBilling"><template #result><div v-if="resultImage" class="flex h-full flex-col"><div class="flex min-h-[420px] flex-1 items-center justify-center overflow-hidden rounded-lg bg-gray-50 dark:bg-dark-800"><img :src="resultImage" alt="译后图片" class="max-h-[560px] max-w-full object-contain" /></div><div class="mt-4 flex flex-wrap gap-3"><button class="btn btn-secondary" type="button" @click="downloadResultImage">下载译后图片</button></div></div></template></ResultPanel>
          </section>

          <section v-else-if="activeTool === 'batch-main' || activeTool === 'batch-clone'" class="grid gap-5 xl:grid-cols-[minmax(340px,0.8fr)_minmax(0,1.2fr)]">
            <div class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
              <h2 class="text-lg font-semibold">{{ activeTool === 'batch-main' ? '批量商品主图' : '参考图批量克隆' }}</h2>
              <p class="mt-1 text-sm text-gray-500">{{ activeTool === 'batch-main' ? '一次处理最多 6 张商品图，逐张生成统一风格主图。' : '使用一张参考图指导最多 6 张商品图的构图和风格。' }}</p>
              <div v-if="activeTool === 'batch-clone'" class="mt-5"><label class="input-label">参考图</label><label class="flex min-h-32 cursor-pointer items-center justify-center overflow-hidden rounded-lg border border-dashed border-gray-300 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-800"><img v-if="referenceImage" :src="referenceImage" alt="参考图" class="max-h-44 object-contain" /><span v-else class="text-sm text-gray-500">选择参考构图或风格图</span><input class="hidden" type="file" accept="image/jpeg,image/png,image/webp" @change="handleReferenceFile" /></label></div>
              <div class="mt-5"><label class="input-label">商品图片（{{ batchInputs.length }}/6）</label><label class="flex min-h-32 cursor-pointer items-center justify-center rounded-lg border border-dashed border-gray-300 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-800"><span class="text-sm text-gray-500">选择多张 JPG、PNG 或 WEBP</span><input class="hidden" type="file" multiple accept="image/jpeg,image/png,image/webp" @change="handleBatchFiles" /></label><div v-if="batchInputs.length" class="mt-3 grid grid-cols-3 gap-2"><div v-for="item in batchInputs" :key="item.id" class="relative"><img :src="item.input" alt="商品图" class="aspect-square w-full rounded object-cover" /><button class="absolute right-1 top-1 rounded bg-black/70 px-1.5 text-xs text-white" @click="removeBatchItem(item.id)">×</button></div></div></div>
              <div class="mt-5"><label class="input-label">处理要求</label><textarea v-model="batchPrompt" class="input min-h-32 resize-y" :placeholder="activeTool === 'batch-main' ? '例如：生成纯白背景电商主图，商品居中，保留真实材质和比例…' : '例如：参考示例图的构图、光线与背景，但必须保留每个商品本身特征…'" /></div>
              <div class="mt-5"><label class="input-label">API Key</label><select v-model.number="selectedKeyId" class="input"><option :value="null" disabled>请选择 API Key</option><option v-for="option in keyOptions" :key="String(option.value)" :value="option.value">{{ option.label }}</option></select><p v-if="keyModeError" class="mt-2 text-xs text-red-500">{{ keyModeError }}</p></div>
              <div class="mt-5"><label class="input-label">模型</label><select v-model="selectedImageModel" class="input"><option value="" disabled>请选择模型</option><option v-for="option in imageModelOptions" :key="String(option.value)" :value="String(option.value)">{{ option.label }}</option></select></div>
              <div class="mt-5 rounded-lg bg-gray-50 p-4 text-sm dark:bg-dark-800"><div class="flex justify-between"><span>预计最多</span><strong>{{ batchImagePriceLabel }}</strong></div><p class="mt-2 text-xs text-gray-500">并发上限 2，按每张实际成功请求结算。</p></div>
              <div class="mt-5 flex gap-3"><button class="btn btn-primary flex-1" :disabled="!canRunBatch || batchRunning" @click="runBatchImages">{{ batchRunning ? `处理中 ${batchCompleted}/${batchInputs.length}` : '开始批量处理' }}</button><button v-if="batchRunning" class="btn btn-secondary" @click="stopBatch">停止</button></div>
            </div>
            <div class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900"><div class="flex items-center justify-between"><h2 class="text-lg font-semibold">批量结果</h2><span class="text-sm text-gray-500">成功 {{ batchSuccessCount }} / {{ batchInputs.length }}</span></div><div v-if="!batchInputs.length" class="mt-5 flex min-h-[520px] items-center justify-center rounded-lg border border-dashed border-gray-300 text-sm text-gray-500 dark:border-dark-600">上传商品图后开始处理。</div><div v-else class="mt-5 grid gap-4 sm:grid-cols-2"><article v-for="item in batchInputs" :key="item.id" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700"><div class="relative aspect-square overflow-hidden rounded bg-gray-100 dark:bg-dark-800"><img :src="item.output || item.input" alt="批量结果" class="h-full w-full object-contain" /><span class="absolute left-2 top-2 rounded bg-black/70 px-2 py-1 text-xs text-white">{{ batchStatusLabel(item.status) }}</span></div><p v-if="item.error" class="mt-2 text-xs text-red-500">{{ item.error }}</p><div class="mt-3 flex gap-3 text-xs"><button v-if="item.output" class="text-primary-600" @click="downloadImage(item.output, `relayq-${item.id}.png`)">下载</button><button v-if="item.status === 'failed'" class="text-primary-600" @click="retryBatchItem(item)">重试</button></div></article></div></div>
          </section>

          <section v-else-if="activeTool === 'watermark'" class="grid gap-5 xl:grid-cols-[minmax(340px,0.8fr)_minmax(0,1.2fr)]">
            <div class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900"><h2 class="text-lg font-semibold">水印处理</h2><p class="mt-1 text-sm text-gray-500">支持去除水印、添加文字水印或上传 logo 作为水印。仅处理你拥有合法使用权的图片。</p><div class="mt-5"><label class="input-label">处理模式</label><Select v-model="watermarkMode" :options="watermarkModeOptions" /></div><div class="mt-5"><label class="input-label">API Key</label><select v-model.number="selectedKeyId" class="input"><option :value="null" disabled>请选择 API Key</option><option v-for="option in keyOptions" :key="String(option.value)" :value="option.value">{{ option.label }}</option></select><p v-if="keyModeError" class="mt-2 text-xs text-red-500">{{ keyModeError }}</p></div><div class="mt-5"><label class="input-label">模型</label><select v-model="selectedImageModel" class="input"><option value="" disabled>请选择模型</option><option v-for="option in imageModelOptions" :key="String(option.value)" :value="String(option.value)">{{ option.label }}</option></select></div>
              <div class="mt-5">
                <label class="input-label">{{ isGrokImagineSelected ? '宽高比' : '图片尺寸' }}</label>
                <Select v-model="imageSize" :options="imageSizeOptions" />
              </div>
              <div v-if="isGrokImagineSelected" class="mt-5">
                <label class="input-label">分辨率</label>
                <Select v-model="imageQuality" :options="grokResolutionOptions" />
                <p class="mt-2 text-xs text-gray-500">Grok Imagine 使用 aspect_ratio + resolution（1k/2k）。quality 模型负责更高画质，分辨率仍按 1k/2k 提交。</p>
              </div>
              <div v-else class="mt-5 grid grid-cols-3 gap-3">
                <div><label class="input-label">画质</label><Select v-model="imageQuality" :options="imageQualityOptions" /></div>
                <div><label class="input-label">风格</label><Select v-model="imageStyle" :options="imageStyleOptions" /></div>
                <div><label class="input-label">背景</label><Select v-model="imageBackground" :options="imageBackgroundOptions" /></div>
              </div>
              <div class="mt-5 grid grid-cols-2 gap-3"><div><label class="input-label">原图</label><label class="flex aspect-square cursor-pointer items-center justify-center overflow-hidden rounded-lg border border-dashed border-gray-300 bg-gray-50 p-2 dark:border-dark-600 dark:bg-dark-800"><img v-if="watermarkImage" :src="watermarkImage" alt="原图" class="max-h-full object-contain" /><span v-else class="text-sm text-gray-500">选择原图</span><input class="hidden" type="file" accept="image/jpeg,image/png,image/webp" @change="handleWatermarkFile" /></label></div><div v-if="watermarkMode === 'remove'"><label class="input-label">蒙版（可选）</label><label class="flex aspect-square cursor-pointer items-center justify-center overflow-hidden rounded-lg border border-dashed border-gray-300 bg-gray-50 p-2 dark:border-dark-600 dark:bg-dark-800"><img v-if="watermarkMask" :src="watermarkMask" alt="蒙版" class="max-h-full object-contain" /><span v-else class="text-center text-xs text-gray-500">白色区域表示需要修复的位置</span><input class="hidden" type="file" accept="image/png,image/webp" @change="handleWatermarkMask" /></label></div><div v-else class="space-y-3"><div><label class="input-label">水印类型</label><Select v-model="watermarkAssetType" :options="watermarkAssetTypeOptions" /></div><div v-if="watermarkAssetType === 'text'"><label class="input-label">水印文字</label><input v-model="watermarkText" class="input" placeholder="例如：RelayQ / 品牌名 / 仅供演示" /></div><div v-else><label class="input-label">Logo 图片</label><label class="flex min-h-32 cursor-pointer items-center justify-center overflow-hidden rounded-lg border border-dashed border-gray-300 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-800"><img v-if="watermarkLogo" :src="watermarkLogo" alt="Logo 水印" class="max-h-28 object-contain" /><span v-else class="text-sm text-gray-500">上传 PNG、WEBP 或透明背景 Logo</span><input class="hidden" type="file" accept="image/png,image/webp,image/jpeg" @change="handleWatermarkLogo" /></label></div><div><label class="input-label">水印位置</label><Select v-model="watermarkPosition" :options="watermarkPositionOptions" /></div><div><label class="input-label">水印样式</label><Select v-model="watermarkStyle" :options="watermarkStyleOptions" /></div></div></div><div class="mt-5"><label class="input-label">{{ watermarkMode === 'remove' ? '修复说明' : '处理说明' }}</label><textarea v-model="watermarkPrompt" class="input min-h-32 resize-y" /></div><button class="btn btn-primary mt-5 w-full" :disabled="!watermarkImage || submitting || (watermarkMode === 'add' && ((watermarkAssetType === 'text' && !watermarkText.trim()) || (watermarkAssetType === 'logo' && !watermarkLogo)))" @click="processWatermark">{{ submitting ? '处理中…' : watermarkMode === 'remove' ? '去除并保存结果' : '添加并保存结果' }}</button></div>
            <ResultPanel :loading="submitting" :error="error" :billing="lastBilling"><template #result><div v-if="resultImage" class="flex h-full flex-col gap-4"><div class="flex min-h-[420px] flex-1 items-center justify-center rounded-lg bg-gray-100 dark:bg-dark-800"><img :src="resultImage" alt="水印处理结果" class="max-h-[620px] w-full object-contain" /></div><button class="btn btn-secondary self-start" @click="downloadImage(resultImage, watermarkMode === 'remove' ? 'relayq-watermark-removed.png' : 'relayq-watermark-added.png')">下载结果</button></div></template></ResultPanel>
          </section>

          <section v-else-if="activeTool === 'video'" class="grid gap-5 xl:grid-cols-[minmax(320px,0.8fr)_minmax(0,1.2fr)]">
            <div class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
              <h2 class="text-lg font-semibold">AI 视频</h2><p class="mt-1 text-sm text-gray-500">支持文生视频和首帧图生视频，提交后自动查询任务状态。</p>
              <div class="mt-5"><label class="input-label">API Key</label><select v-model.number="selectedKeyId" class="input"><option :value="null" disabled>请选择 API Key</option><option v-for="option in keyOptions" :key="String(option.value)" :value="option.value">{{ option.label }}</option></select><p v-if="keyModeError" class="mt-2 text-xs text-red-500">{{ keyModeError }}</p></div>
              <div class="mt-5"><label class="input-label">模型</label><select v-model="selectedVideoModel" class="input"><option value="" disabled>请选择模型</option><option v-for="option in videoModelOptions" :key="String(option.value)" :value="String(option.value)">{{ option.label }}</option></select></div>
              <div class="mt-5"><label class="input-label">首帧图片（可选）</label><label class="flex min-h-36 cursor-pointer items-center justify-center overflow-hidden rounded-lg border border-dashed border-gray-300 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-800"><img v-if="videoImage" :src="videoImage" alt="视频首帧" class="max-h-52 object-contain" /><span v-else class="text-sm text-gray-500">不上传则为文生视频</span><input class="hidden" type="file" accept="image/jpeg,image/png,image/webp" @change="handleVideoFile" /></label></div>
              <div class="mt-5"><label class="input-label">视频描述</label><textarea v-model="videoPrompt" class="input min-h-40 resize-y" placeholder="描述主体动作、镜头运动、场景与光线…" /></div>
              <div class="mt-5 grid grid-cols-3 gap-3"><div><label class="input-label">比例</label><Select v-model="videoAspectRatio" :options="videoRatioOptions" /></div><div><label class="input-label">时长</label><Select v-model="videoDuration" :options="videoDurationOptions" /></div><div><label class="input-label">分辨率</label><Select v-model="videoResolution" :options="videoResolutionSelectOptions" /></div></div>
              <div class="mt-5 flex gap-3"><button class="btn btn-primary flex-1" :disabled="!videoPrompt.trim() || submitting" @click="submitVideo">{{ submitting ? '提交中…' : '生成视频' }}</button><button v-if="submitting" class="btn btn-secondary" @click="stopRequest">停止等待</button></div>
            </div>
            <ResultPanel :loading="submitting || videoPolling" :error="error" :billing="lastBilling">
              <template #result><div v-if="videoUrl" class="space-y-4"><video class="w-full rounded-lg bg-black" :src="videoUrl" controls /><div class="flex flex-wrap gap-2"><a class="btn btn-secondary inline-flex" :href="videoUrl" target="_blank" rel="noreferrer">打开视频</a><button class="btn btn-secondary btn-sm" type="button" @click="downloadImage(videoUrl, 'relayq-video-result.mp4')">下载视频</button></div></div><div v-else-if="requestId" class="rounded-lg border border-dashed border-gray-300 p-8 text-center dark:border-dark-600"><div class="text-lg font-semibold">任务处理中</div><p class="mt-2 text-sm text-gray-500">状态：{{ videoStatus || 'queued' }}<span v-if="videoProgress !== undefined"> · {{ videoProgress }}%</span></p><button class="btn btn-secondary mt-4" type="button" @click="pollVideoOnce">立即查询</button></div></template>
            </ResultPanel>
          </section>

          <section v-else-if="activeTool === 'audio-transcribe'" class="grid gap-5 xl:grid-cols-[minmax(340px,0.8fr)_minmax(0,1.2fr)]">
            <div class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
              <h2 class="text-lg font-semibold">语音转写</h2><p class="mt-1 text-sm text-gray-500">上传音频，输出可复制的纯文本转写结果。</p>
              <div class="mt-5"><label class="input-label">API Key</label><select v-model.number="selectedKeyId" class="input"><option :value="null" disabled>请选择 API Key</option><option v-for="option in keyOptions" :key="String(option.value)" :value="option.value">{{ option.label }}</option></select><p v-if="keyModeError" class="mt-2 text-xs text-red-500">{{ keyModeError }}</p><p v-else-if="resolvedGroupName" class="mt-2 text-xs text-gray-500">当前分组：{{ resolvedGroupName }} · 平台：{{ resolvedPlatformLabel }}</p></div>
              <div class="mt-5"><label class="input-label">模型</label><select v-model="selectedAudioModel" class="input"><option value="" disabled>请选择语音转写模型</option><option v-for="option in audioModelOptions" :key="String(option.value)" :value="String(option.value)">{{ option.label }}</option></select><p class="mt-2 text-xs text-gray-500">默认优先使用当前分组可用的 mimo ASR 模型。</p></div>
              <div class="mt-5"><label class="input-label">音频文件</label><label class="flex min-h-32 cursor-pointer items-center justify-center rounded-lg border border-dashed border-gray-300 bg-gray-50 p-4 text-center dark:border-dark-600 dark:bg-dark-800"><span class="text-sm text-gray-500">{{ audioInputName || '上传 WAV 或 MP3 音频' }}</span><input class="hidden" type="file" accept="audio/mpeg,audio/wav,audio/mp3,.mp3,.wav" @change="handleAudioFile" /></label></div>
              <div class="mt-5"><label class="input-label">语言</label><Select v-model="audioLanguage" :options="audioLanguageOptions" placeholder="请选择音频语言" /></div>
              <div class="mt-5 rounded-lg bg-gray-50 p-4 text-xs leading-5 text-gray-500 dark:bg-dark-800">当前仅支持纯文本转写。默认自动识别语言，也可手动指定中文/英文；上传 wav/mp3 后即可开始。</div>
              <button class="btn btn-primary mt-5 w-full" :disabled="!audioInput || !selectedAudioModel || submitting" @click="submitAudioTranscription">{{ submitting ? '转写中…' : '开始转写' }}</button>
            </div>
            <ResultPanel :loading="submitting" :error="error" :billing="lastBilling"><template #result><div v-if="audioTranscriptText" class="flex h-full flex-col"><pre class="flex-1 whitespace-pre-wrap rounded-lg bg-gray-50 p-5 text-sm leading-7 dark:bg-dark-800">{{ audioTranscriptText }}</pre><div class="mt-4 flex flex-wrap gap-3"><button class="btn btn-secondary" type="button" @click="copyTextResult">复制转写结果</button><button class="btn btn-secondary" type="button" @click="downloadTranscriptTxt">下载 TXT</button></div></div></template></ResultPanel>
          </section>

          <section v-else-if="activeTool === 'audio-generate'" class="grid gap-5 xl:grid-cols-[minmax(340px,0.8fr)_minmax(0,1.2fr)]">
            <div class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
              <h2 class="text-lg font-semibold">AI 配音</h2><p class="mt-1 text-sm text-gray-500">支持标准配音、音色设计和声音克隆。</p>
              <div class="mt-5"><label class="input-label">模式</label><Select v-model="audioGenerateMode" :options="audioGenerateModeOptions" placeholder="请选择配音模式" /><p class="mt-2 text-xs text-gray-500">默认从标准配音开始，可随时切换到音色设计或声音克隆。</p></div>
              <div class="mt-5"><label class="input-label">API Key</label><select v-model.number="selectedKeyId" class="input"><option :value="null" disabled>请选择 API Key</option><option v-for="option in keyOptions" :key="String(option.value)" :value="option.value">{{ option.label }}</option></select><p v-if="keyModeError" class="mt-2 text-xs text-red-500">{{ keyModeError }}</p><p v-else-if="resolvedGroupName" class="mt-2 text-xs text-gray-500">当前分组：{{ resolvedGroupName }} · 平台：{{ resolvedPlatformLabel }}</p></div>
              <div class="mt-5"><label class="input-label">模型</label><select v-model="selectedTtsModel" class="input"><option value="" disabled>请选择配音模型</option><option v-for="option in ttsModelOptions" :key="String(option.value)" :value="String(option.value)">{{ option.label }}</option></select><p class="mt-2 text-xs text-gray-500">默认优先使用当前分组可用的 mimo TTS 模型。</p></div>
              <div class="mt-5"><label class="input-label">配音文本</label><textarea v-model="ttsText" class="input min-h-40 resize-y" placeholder="输入需要转换成语音的文案…" /></div>
              <div class="mt-5 grid grid-cols-2 gap-3"><div><label class="input-label">语言</label><Select v-model="ttsLanguage" :options="languageOptions" placeholder="请选择输出语言" /></div><div><label class="input-label">风格</label><Select v-model="ttsStyle" :options="ttsStyleOptions" placeholder="请选择配音风格" /></div></div>
              <div v-if="audioGenerateMode === 'standard'" class="mt-5"><label class="input-label">预设音色</label><Select v-model="ttsVoicePreset" :options="ttsVoicePresetOptions" placeholder="请选择预设音色" /></div>
              <div v-else-if="audioGenerateMode === 'voicedesign'" class="mt-5 space-y-3"><div><label class="input-label">音色描述</label><textarea v-model="ttsVoiceDescription" class="input min-h-24 resize-y" placeholder="例如：温柔、稳定、适合品牌讲解的女声" /></div><div><label class="input-label">人设类型</label><input v-model="ttsPersona" class="input" placeholder="例如：品牌讲解员 / 专业旁白" /></div></div>
              <div v-else class="mt-5 space-y-3"><div><label class="input-label">参考音频</label><label class="flex min-h-28 cursor-pointer items-center justify-center rounded-lg border border-dashed border-gray-300 bg-gray-50 p-4 text-center dark:border-dark-600 dark:bg-dark-800"><span class="text-sm text-gray-500">{{ ttsReferenceAudioName || '上传参考音频用于声音克隆' }}</span><input class="hidden" type="file" accept="audio/mpeg,audio/wav,audio/mp4,audio/webm,.mp3,.wav,.m4a,.webm" @change="handleTtsReferenceFile" /></label></div><label class="flex items-center gap-2 text-sm text-gray-600 dark:text-dark-300"><input v-model="ttsAuthorizationConfirmed" type="checkbox" />我确认已获得该声音样本的合法授权</label></div>
              <div class="mt-5 rounded-lg bg-gray-50 p-4 text-xs leading-5 text-gray-500 dark:bg-dark-800">首版默认值已预设为中文、自然讲述和通用女声。填写文案后可直接生成；声音克隆模式下还需要上传参考音频并确认授权。</div>
              <button class="btn btn-primary mt-5 w-full" :disabled="!canSubmitAudioGeneration" @click="submitAudioGeneration">{{ submitting ? '生成中…' : '生成配音' }}</button>
            </div>
            <ResultPanel :loading="submitting || audioPreviewLoading" :error="error" :billing="lastBilling"><template #result><div v-if="ttsResultUrl || ttsResultText" class="space-y-4"><div v-if="audioPreviewLoading" class="rounded-lg bg-gray-50 p-4 text-sm text-gray-500 dark:bg-dark-800">音频已生成，正在下载并解析媒体元数据…</div><audio v-if="ttsResultUrl && !audioPreviewLoading" :key="ttsResultUrl" class="w-full" :src="ttsResultUrl" controls preload="auto" @loadedmetadata="onPreviewAudioLoadedMetadata" @canplaythrough="onPreviewAudioCanPlayThrough" @error="onPreviewAudioError" /><pre v-if="ttsResultText" class="whitespace-pre-wrap rounded-lg bg-gray-50 p-5 text-sm leading-7 dark:bg-dark-800">{{ ttsResultText }}</pre><button v-if="ttsResultUrl && !audioPreviewLoading" class="btn btn-secondary" type="button" @click="downloadImage(ttsResultUrl, 'relayq-audio-result.mp3')">下载音频</button><div v-if="ttsDebug" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs leading-5 text-amber-800 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200">{{ ttsDebug }}</div></div></template></ResultPanel>
          </section>

          <section v-else class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
            <div class="flex items-center justify-between">
              <div>
                <h2 class="text-lg font-semibold">创作记录</h2>
                <p class="mt-1 text-sm text-gray-500">图片、视频、音频、文案统一保存在服务端：最多保留最近 10 条，超过 10 条会自动删除最早 1 条，且所有记录会在 2 天后自动清理。可直接恢复参数或下载结果。</p>
              </div>
              <button class="btn btn-secondary btn-sm" :disabled="cloudLoading" @click="loadCloudRecords">刷新</button>
            </div>
            <RecordList class="mt-5" :items="cloudRecords" :loading="cloudLoading" @restore="restoreRecord" @remove="removeRecord" @download="downloadRecord" />
          </section>
        </main>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { modelTestAPI, type ChatMessage, type PlaygroundBilling } from '@/api/modelTest'
import { playgroundCloudAPI, type PlaygroundRecord, type PersistedMediaRef, type PlaygroundTask } from '@/api/playgroundCloud'
import { keysAPI } from '@/api/keys'
import { userChannelsAPI, type UserAvailableChannel, type UserAvailableGroup, type UserSupportedModel } from '@/api/channels'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import type { ApiKey, SelectOption } from '@/types'

type ToolId = 'home' | 'image' | 'edit' | 'chat' | 'copywriting' | 'translate' | 'batch-main' | 'batch-clone' | 'watermark' | 'video' | 'audio-transcribe' | 'audio-generate' | 'history'
type IconName = InstanceType<typeof Icon>['$props']['name']
interface BatchImageItem { id: string; input: string; output?: string; status: 'pending' | 'processing' | 'completed' | 'failed' | 'canceled'; error?: string; requestId?: string }

const tools: Array<{ id: ToolId; label: string; description: string; icon: IconName }> = [
  { id: 'home', label: '创作首页', description: '工具入口与最近任务', icon: 'home' },
  { id: 'image', label: 'AI 生图', description: '海报、商品图和视觉素材', icon: 'lightbulb' },
  { id: 'edit', label: '图片编辑', description: '换背景、改风格和局部调整', icon: 'edit' },
  { id: 'chat', label: '对话助手', description: '文案、翻译、摘要与提示词优化', icon: 'chat' },
  { id: 'copywriting', label: '商品文案', description: '标题、卖点和详情描述', icon: 'document' },
  { id: 'translate', label: '图片翻译', description: '图上文字本地化，输出译后图片', icon: 'globe' },
  { id: 'batch-main', label: '批量主图', description: '批量生成统一商品主图', icon: 'document' },
  { id: 'batch-clone', label: '批量克隆', description: '参考图指导批量商品图', icon: 'copy' },
  { id: 'watermark', label: '水印处理', description: '去除或添加图片水印', icon: 'edit' },
  { id: 'video', label: 'AI 视频', description: '文生视频和首帧图生视频', icon: 'play' },
  { id: 'audio-transcribe', label: '语音转写', description: '录音转稿，输出纯文本', icon: 'document' },
  { id: 'audio-generate', label: 'AI 配音', description: '标准配音、音色设计与声音克隆', icon: 'play' },
  { id: 'history', label: '创作记录', description: '统一查看、恢复和下载结果', icon: 'clock' },
]
const creationTools = tools.filter((tool) => !['home', 'history'].includes(tool.id))
const imageTemplates = [{ label: '商品白底图', prompt: '电商商品白底主图，主体居中，真实摄影质感，边缘清晰，光线均匀，无多余装饰。' }, { label: '商品场景图', prompt: '高端电商商品场景图，自然生活环境，柔和商业摄影光线，突出商品材质和功能。' }, { label: '社媒海报', prompt: '现代社交媒体宣传海报，视觉焦点明确，留出标题区域，商业设计感，高对比配色。' }, { label: '人物摄影', prompt: '专业人物摄影，自然肤色，柔和轮廓光，真实镜头质感，背景干净。' }]
const chatTemplates = [{ label: '商品文案', description: '生成标题、卖点和详情描述', prompt: '请根据我接下来提供的商品信息，生成商品标题、5条核心卖点和一段详情描述。' }, { label: '内容润色', description: '改善语气、结构和可读性', prompt: '请润色我接下来提供的内容，保留原意，使表达更清晰专业。' }, { label: '中英翻译', description: '自然准确地翻译内容', prompt: '请把我接下来提供的中文翻译成自然、专业的英文。' }, { label: '提示词优化', description: '补充画面细节和构图要求', prompt: '请把我接下来提供的图片提示词优化成适合高质量图像生成的版本。' }]

const authStore = useAuthStore()
const appStore = useAppStore()
const activeTool = ref<ToolId>('home')
const availableKeys = ref<ApiKey[]>([])
const availableChannels = ref<UserAvailableChannel[]>([])
const selectedKeyId = ref<number | null>(null)
const selectedImageModel = ref('')
const selectedChatModel = ref('')
const chatSelectedKeyId = ref<number | null>(null)
const chatSelectedModel = ref('')
const selectedVideoModel = ref('')
const selectedAudioModel = ref('')
const selectedTtsModel = ref('')
const imagePrompt = ref(imageTemplates[0].prompt)
const imageSize = ref('1:1')
const imageQuality = ref('high')
const imageStyle = ref('natural')
const imageBackground = ref('opaque')
const editImage = ref('')
const resultImage = ref('')
const chatInput = ref('')
const chatMessages = ref<ChatMessage[]>([])
const copywritingName = ref('')
const copywritingBrief = ref('')
const copywritingPlatform = ref('电商详情页')
const copywritingLanguage = ref('中文')
const translateImage = ref('')
const translateSource = ref('自动识别')
const translateTarget = ref('英文')
const textResult = ref('')
const cloudRecords = ref<PlaygroundRecord[]>([])
const cloudLoading = ref(false)
const batchInputs = ref<BatchImageItem[]>([])
const batchPrompt = ref('生成专业电商商品主图，商品主体居中，保留真实材质、颜色和比例，背景干净，光线均匀，无文字和水印。')
const referenceImage = ref('')
const batchRunning = ref(false)
const watermarkImage = ref('')
const watermarkMask = ref('')
const watermarkMode = ref<'remove' | 'add'>('remove')
const watermarkAssetType = ref<'text' | 'logo'>('text')
const watermarkText = ref('')
const watermarkLogo = ref('')
const watermarkPosition = ref('右下角')
const watermarkStyle = ref('半透明白字')
const watermarkPrompt = ref('移除图片中的水印或覆盖文字，使用周围纹理自然修复，保持主体、构图、颜色和清晰度不变。')
const videoPrompt = ref('')
const videoImage = ref('')
const videoAspectRatio = ref('16:9')
const videoDuration = ref('10')
const videoResolution = ref('720p')
const videoStatus = ref('')
const videoProgress = ref<number>()
const videoUrl = ref('')
const videoPolling = ref(false)
const audioInput = ref('')
const audioInputName = ref('')
const audioLanguage = ref('自动识别')
const audioTranscriptText = ref('')
const audioGenerateMode = ref<'standard' | 'voicedesign' | 'voiceclone'>('standard')
const ttsText = ref('')
const ttsLanguage = ref('中文')
const ttsStyle = ref('自然讲述')
const ttsVoicePreset = ref('冰糖')
const ttsVoiceDescription = ref('温和、清晰、适合电商讲解')
const ttsPersona = ref('品牌讲解员')
const ttsReferenceAudio = ref('')
const ttsReferenceAudioName = ref('')
const ttsAuthorizationConfirmed = ref(false)
const ttsResultUrl = ref('')
const ttsResultText = ref('')
const ttsDebug = ref('')
const audioPreviewLoading = ref(false)
const submitting = ref(false)
const error = ref('')
const requestId = ref('')
const currentTaskId = ref<number | null>(null)
const failedAssetUrls = new Set<string>()
const DEBUG_EVENT_URL = String(import.meta.env.VITE_DEBUG_EVENT_URL || '').trim()
const ENABLE_DEBUG_EVENT_REPORT = import.meta.env.DEV && DEBUG_EVENT_URL.length > 0
const lastBilling = ref<PlaygroundBilling>()
let abortController: AbortController | null = null
let pollTimer: number | null = null

const balance = computed(() => authStore.user?.balance ?? 0)
const normalizedSelectedKeyId = computed(() => Number(selectedKeyId.value || 0) || null)
const selectedKey = computed(() => availableKeys.value.find((item) => item.id === normalizedSelectedKeyId.value) || null)
const resolvedGroup = computed<UserAvailableGroup | null>(() => {
  if (!selectedKey.value?.group_id) return null
  for (const channel of availableChannels.value) {
    for (const section of channel.platforms) {
      const match = section.groups.find((group) => group.id === selectedKey.value?.group_id)
      if (match) return match
    }
  }
  return null
})
const resolvedPlatformLabel = computed(() => resolvedGroup.value?.platform || '未匹配')
const resolvedGroupName = computed(() => resolvedGroup.value?.name || '')
const normalizedChatSelectedKeyId = computed(() => Number(chatSelectedKeyId.value || 0) || null)
const chatSelectedKey = computed(() => availableKeys.value.find((item) => item.id === normalizedChatSelectedKeyId.value) || null)
const chatResolvedGroup = computed<UserAvailableGroup | null>(() => {
  if (!chatSelectedKey.value?.group_id) return null
  for (const channel of availableChannels.value) {
    for (const section of channel.platforms) {
      const match = section.groups.find((group) => group.id === chatSelectedKey.value?.group_id)
      if (match) return match
    }
  }
  return null
})
const chatResolvedPlatformLabel = computed(() => chatResolvedGroup.value?.platform || '未匹配')
const chatResolvedGroupName = computed(() => chatResolvedGroup.value?.name || '')
const groupModels = computed<UserSupportedModel[]>(() => {
  if (!resolvedGroup.value) return []
  for (const channel of availableChannels.value) {
    const section = channel.platforms.find((item) => item.platform === resolvedGroup.value?.platform && item.groups.some((group) => group.id === resolvedGroup.value?.id))
    if (section) return section.supported_models
  }
  return []
})
const imageModels = computed(() => groupModels.value.filter((model) => isImageModel(model)))
const chatGroupModels = computed<UserSupportedModel[]>(() => {
  if (!chatResolvedGroup.value) return []
  for (const channel of availableChannels.value) {
    const section = channel.platforms.find((item) => item.platform === chatResolvedGroup.value?.platform && item.groups.some((group) => group.id === chatResolvedGroup.value?.id))
    if (section) return section.supported_models
  }
  return []
})
const chatModels = computed(() => groupModels.value.filter((model) => {
  if (isImageModel(model) || isVideoModel(model)) return false
  const billingMode = String(model.pricing?.billing_mode || '').toLowerCase().trim()
  if (billingMode === 'image') return false
  if (/mimo-v2\.5-asr|mimo-v2\.5-tts|mimo-v2-tts/i.test(model.name)) return false
  return true
}))
const chatToolModels = computed(() => chatGroupModels.value.filter((model) => {
  if (isImageModel(model) || isVideoModel(model)) return false
  const billingMode = String(model.pricing?.billing_mode || '').toLowerCase().trim()
  if (billingMode === 'image') return false
  if (/mimo-v2\.5-asr|mimo-v2\.5-tts|mimo-v2-tts/i.test(model.name)) return false
  return true
}))
const videoModels = computed(() => groupModels.value.filter((model) => isVideoModel(model)))
const audioModels = computed(() => groupModels.value.filter((model) => /mimo-v2\.5-asr/i.test(model.name)))
const ttsModels = computed(() => groupModels.value.filter((model) => /mimo-v2\.5-tts|mimo-v2-tts/i.test(model.name)))
function modelValue(model: UserSupportedModel) {
  return String(model.id || model.name || '').trim()
}
const currentImageModel = computed(() => imageModels.value.find((model) => modelValue(model) === selectedImageModel.value) || imageModels.value[0] || null)
const currentChatModel = computed(() => chatToolModels.value.find((model) => modelValue(model) === chatSelectedModel.value) || chatToolModels.value[0] || null)
const imageModelOptions = computed<SelectOption[]>(() => imageModels.value.map((model) => ({ value: modelValue(model), label: model.name })))
const chatModelOptions = computed<SelectOption[]>(() => chatToolModels.value.map((model) => ({ value: modelValue(model), label: model.name })))
const videoModelOptions = computed<SelectOption[]>(() => videoModels.value.map((model) => ({ value: modelValue(model), label: model.name })))
const audioModelOptions = computed<SelectOption[]>(() => audioModels.value.map((model) => ({ value: modelValue(model), label: model.name })))
const ttsModelOptions = computed<SelectOption[]>(() => ttsModels.value.map((model) => ({ value: modelValue(model), label: model.name })))
const keyOptions = computed<SelectOption[]>(() => availableKeys.value.map((key) => ({ value: key.id, label: `${key.name} · ${key.status === 'active' ? '可用' : '不可用'}` })))
const keyModeError = computed(() => {
  if (!availableKeys.value.length) return '请先创建并绑定 API Key。'
  if (!selectedKey.value) return '请选择一个 API Key。'
  if (selectedKey.value.status !== 'active') return '当前 API Key 已停用，请更换。'
  if (!selectedKey.value.group_id) return '当前 API Key 没有绑定分组，请先到 API Key 页面绑定分组。'
  if (!resolvedGroup.value) return '当前 API Key 所属分组没有匹配到可用渠道。'
  return ''
})
const chatKeyModeError = computed(() => {
  if (!availableKeys.value.length) return '请先创建并绑定 API Key。'
  if (!chatSelectedKey.value) return '请选择一个 API Key。'
  if (chatSelectedKey.value.status !== 'active') return '当前 API Key 已停用，请更换。'
  if (!chatSelectedKey.value.group_id) return '当前 API Key 没有绑定分组，请先到 API Key 页面绑定分组。'
  if (!chatResolvedGroup.value) return '当前 API Key 所属分组没有匹配到可用渠道。'
  return ''
})
const batchImagePriceLabel = computed(() => estimateBatchPrice(currentImageModel.value, imageSize.value, batchInputs.value.length))
// 外联 gpt-image-2 文档使用比例 size（如 16:9），不是 1024x1024 像素串
const imageSizeOptions = [
  { value: '1:1', label: '方图 1:1' },
  { value: '16:9', label: '横图 16:9' },
  { value: '9:16', label: '竖图 9:16' },
  { value: '3:2', label: '横图 3:2' },
  { value: '2:3', label: '竖图 2:3' },
]
const imageQualityOptions = [
  { value: 'low', label: '低' },
  { value: 'medium', label: '中' },
  { value: 'high', label: '高' },
]
// Grok 分辨率映射：UI 仍复用 imageQuality，medium→1k，high→2k（xAI 当前仅支持 1k/2k）
const grokResolutionOptions = [
  { value: 'medium', label: '1k 标准' },
  { value: 'high', label: '2k 高清' },
]
const imageStyleOptions = [
  { value: 'natural', label: '自然 natural' },
  { value: 'vivid', label: '鲜艳 vivid' },
]
const imageBackgroundOptions = [
  { value: 'opaque', label: '不透明' },
  { value: 'transparent', label: '透明' },
]
const isGrokImagineSelected = computed(() => /^grok-imagine-image/i.test(selectedImageModel.value || ''))
const watermarkModeOptions = [{ value: 'remove', label: '去除水印' }, { value: 'add', label: '添加水印' }]
const watermarkAssetTypeOptions = [{ value: 'text', label: '文字水印' }, { value: 'logo', label: 'Logo 水印' }]
const watermarkPositionOptions = [{ value: '右下角', label: '右下角' }, { value: '右上角', label: '右上角' }, { value: '左下角', label: '左下角' }, { value: '居中', label: '居中' }]
const watermarkStyleOptions = [{ value: '半透明白字', label: '半透明白字' }, { value: '半透明黑字', label: '半透明黑字' }, { value: '浅色描边', label: '浅色描边' }, { value: '品牌签名', label: '品牌签名' }]
const videoRatioOptions = [{ value: '16:9', label: '横屏 16:9' }, { value: '9:16', label: '竖屏 9:16' }, { value: '1:1', label: '方形 1:1' }]
const videoDurationOptions = [{ value: '5', label: '5 秒' }, { value: '10', label: '10 秒' }, { value: '15', label: '15 秒' }, { value: '20', label: '20 秒' }]
const videoResolutionOptions = [{ value: '480p', label: '480p 标清' }, { value: '720p', label: '720p 高清' }, { value: '1080p', label: '1080p 全高清' }]
const videoResolutionSelectOptions = computed(() => {
  const model = String(selectedVideoModel.value || '').trim().toLowerCase()
  if (model.startsWith('grok-imagine-video')) {
    return videoResolutionOptions.filter((option) => option.value !== '1080p')
  }
  return videoResolutionOptions
})
const copywritingPlatformOptions = [{ value: '电商详情页', label: '电商详情页' }, { value: '小红书', label: '小红书' }, { value: '抖音', label: '抖音' }, { value: '亚马逊', label: '亚马逊' }]
const languageOptions = [
  { value: '中文', label: '中文（简体）' },
  { value: '中文繁体', label: '中文（繁体）' },
  { value: '英文', label: '英文' },
  { value: '日文', label: '日文' },
  { value: '韩文', label: '韩文' },
  { value: '法文', label: '法文' },
  { value: '德文', label: '德文' },
  { value: '西班牙文', label: '西班牙文' },
  { value: '葡萄牙文', label: '葡萄牙文' },
  { value: '意大利文', label: '意大利文' },
  { value: '俄文', label: '俄文' },
  { value: '阿拉伯文', label: '阿拉伯文' },
  { value: '泰文', label: '泰文' },
  { value: '越南文', label: '越南文' },
  { value: '印尼文', label: '印尼文' },
  { value: '马来文', label: '马来文' },
  { value: '印地文', label: '印地文' },
  { value: '土耳其文', label: '土耳其文' },
  { value: '荷兰文', label: '荷兰文' },
  { value: '波兰文', label: '波兰文' },
  { value: '瑞典文', label: '瑞典文' },
  { value: '希腊文', label: '希腊文' },
  { value: '希伯来文', label: '希伯来文' },
  { value: '乌克兰文', label: '乌克兰文' },
  { value: '捷克文', label: '捷克文' },
]
const audioLanguageOptions = [
  { value: '自动识别', label: '自动识别' },
  { value: '中文', label: '中文' },
  { value: '英文', label: '英文' },
]
const audioGenerateModeOptions = [{ value: 'standard', label: '标准配音' }, { value: 'voicedesign', label: '音色设计' }, { value: 'voiceclone', label: '声音克隆' }]
const ttsStyleOptions = [{ value: '自然讲述', label: '自然讲述' }, { value: '商品讲解', label: '商品讲解' }, { value: '温柔陪伴', label: '温柔陪伴' }, { value: '专业旁白', label: '专业旁白' }]
const ttsVoicePresetOptions = [{ value: 'mimo_default', label: 'MiMo 默认' }, { value: '冰糖', label: '冰糖' }, { value: '茉莉', label: '茉莉' }, { value: '苏打', label: '苏打' }, { value: '白桦', label: '白桦' }, { value: 'Mia', label: 'Mia' }, { value: 'Chloe', label: 'Chloe' }, { value: 'Milo', label: 'Milo' }, { value: 'Dean', label: 'Dean' }]
const sourceLanguageOptions = [{ value: '自动识别', label: '自动识别' }, ...languageOptions]
const canSubmitImage = computed(() => !keyModeError.value && selectedImageModel.value && imagePrompt.value.trim() && (activeTool.value !== 'edit' || editImage.value))
const canRunBatch = computed(() => !keyModeError.value && selectedImageModel.value && batchInputs.value.length > 0 && batchPrompt.value.trim() && (activeTool.value !== 'batch-clone' || referenceImage.value))
const batchCompleted = computed(() => batchInputs.value.filter((item) => ['completed', 'failed', 'canceled'].includes(item.status)).length)
const batchSuccessCount = computed(() => batchInputs.value.filter((item) => item.status === 'completed').length)
const canSubmitAudioGeneration = computed(() => !keyModeError.value && selectedTtsModel.value && ttsText.value.trim() && (audioGenerateMode.value !== 'voiceclone' || (ttsReferenceAudio.value && ttsAuthorizationConfirmed.value)))

watch(selectedVideoModel, (model) => {
  const normalized = String(model || '').trim().toLowerCase()
  if (normalized.startsWith('grok-imagine-video') && videoResolution.value === '1080p') {
    videoResolution.value = '720p'
  }
})

function selectTool(tool: ToolId) { stopRequest(); activeTool.value = tool; error.value = ''; requestId.value = ''; lastBilling.value = undefined; if (tool === 'history' || tool === 'home') void loadCloudRecords() }
function toolButtonClass(tool: ToolId) { return ['flex items-center gap-2 rounded-md px-3 py-2.5 text-left text-sm transition', activeTool.value === tool ? 'bg-primary-50 font-semibold text-primary-700 dark:bg-primary-950/40 dark:text-primary-300' : 'text-gray-600 hover:bg-gray-50 dark:text-dark-300 dark:hover:bg-dark-800'] }

function getAuthContext() {
  if (!selectedKey.value?.key) throw new Error('请选择一个可用的 API Key。')
  if (selectedKey.value.status !== 'active') throw new Error('当前 API Key 不可用。')
  if (!selectedKey.value.group_id) throw new Error('当前 API Key 未绑定分组。')
  return { apiKey: selectedKey.value.key }
}

function getExecutionMetadata() {
  return {
    api_key_id: selectedKey.value?.id,
    api_key_name: selectedKey.value?.name,
    group_id: selectedKey.value?.group_id,
    group_name: resolvedGroupName.value || undefined,
    platform: resolvedPlatformLabel.value !== '未匹配' ? resolvedPlatformLabel.value : undefined,
  }
}

async function reportAsyncImageEditDebugEvent(hypothesisId: string, location: string, msg: string, data: Record<string, unknown>) {
  if (!ENABLE_DEBUG_EVENT_REPORT) return
  try {
    await fetch(DEBUG_EVENT_URL, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        sessionId: 'image-edit-failure',
        runId: 'pre-fix',
        hypothesisId,
        location,
        msg,
        data,
        ts: Date.now(),
      }),
    })
  } catch {
    // Ignore debug reporting failures.
  }
}

async function blobObjectUrlToDataUrl(url: string): Promise<string> {
  if (url.startsWith('data:')) return url
  const response = await fetch(url)
  if (!response.ok) throw new Error('读取本地文件失败。')
  const blob = await response.blob()
  return await new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result || ''))
    reader.onerror = () => reject(new Error('转换本地文件失败。'))
    reader.readAsDataURL(blob)
  })
}

async function normalizeLocalMediaUrl(url?: string): Promise<string> {
  const value = String(url || '').trim()
  if (!value) return ''
  if (value.startsWith('blob:')) return blobObjectUrlToDataUrl(value)
  return value
}

async function buildPlaygroundImageMedia(images: string[], mask?: string) {
  const resolvedImages = await Promise.all(images.map((item) => normalizeLocalMediaUrl(item)))
  const resolvedMask = mask ? await normalizeLocalMediaUrl(mask) : ''
  return {
    images: resolvedImages.filter(Boolean).map((url) => ({ url })),
    ...(resolvedMask ? { mask: { url: resolvedMask } } : {}),
  }
}

async function buildPlaygroundAudioMedia(url?: string, filename?: string, field: 'audio' | 'reference_audio' = 'audio') {
  const resolved = await normalizeLocalMediaUrl(url)
  if (!resolved) return undefined
  return {
    [field]: {
      url: resolved,
      ...(filename ? { filename } : {}),
    },
  }
}

async function buildPlaygroundVideoMedia(url?: string) {
  const resolved = await normalizeLocalMediaUrl(url)
  if (!resolved) return undefined
  return {
    input_reference: { url: resolved },
  }
}

function delayWithSignal(ms: number, signal?: AbortSignal) {
  return new Promise<void>((resolve, reject) => {
    const timer = window.setTimeout(() => {
      cleanup()
      resolve()
    }, ms)
    const onAbort = () => {
      cleanup()
      reject(new DOMException('Aborted', 'AbortError'))
    }
    const cleanup = () => {
      window.clearTimeout(timer)
      signal?.removeEventListener('abort', onAbort)
    }
    signal?.addEventListener('abort', onAbort, { once: true })
  })
}

async function submitAsyncPlaygroundJob(kind: string, model: string, requestPayload: Record<string, unknown>) {
  const auth = getAuthContext()
  // #region debug-point G:frontend-async-job-submit
  if (kind === 'edit') {
    await reportAsyncImageEditDebugEvent('G', 'PlaygroundView.vue:submitAsyncPlaygroundJob', '[DEBUG] frontend async edit job submit', {
      kind,
      model,
      api_key_id: selectedKey.value?.id,
      image_count: Array.isArray((requestPayload.media as any)?.images) ? (requestPayload.media as any).images.length : 0,
      prompt_length: String(requestPayload.prompt || '').length,
    })
  }
  // #endregion
  const task = await playgroundCloudAPI.submitJob({
    kind,
    model,
    api_key: auth.apiKey,
    request_payload: requestPayload,
  })
  // #region debug-point H:frontend-async-job-created
  if (kind === 'edit') {
    await reportAsyncImageEditDebugEvent('H', 'PlaygroundView.vue:submitAsyncPlaygroundJob', '[DEBUG] frontend async edit job created', {
      task_id: task.id,
      status: task.status,
      request_id: task.request_id || '',
    })
  }
  // #endregion
  currentTaskId.value = task.id
  void loadCloudRecords()
  return task
}

async function waitForPlaygroundTask(taskId: number, signal?: AbortSignal): Promise<PlaygroundTask> {
  while (true) {
    if (signal?.aborted) throw new DOMException('Aborted', 'AbortError')
    const task = await playgroundCloudAPI.getTask(taskId)
    currentTaskId.value = task.id
    requestId.value = task.request_id || requestId.value
    // #region debug-point I:frontend-async-job-poll
    if (task.kind === 'edit') {
      await reportAsyncImageEditDebugEvent('I', 'PlaygroundView.vue:waitForPlaygroundTask', '[DEBUG] frontend async edit job polled', {
        task_id: task.id,
        status: task.status,
        request_id: task.request_id || '',
        error_message: task.error_message || '',
      })
    }
    // #endregion
    if (task.status === 'succeeded') return task
    if (task.status === 'failed') throw new Error(task.error_message || '任务执行失败。')
    if (task.status === 'canceled') throw new Error('任务已取消。')
    await delayWithSignal(document.hidden ? 5000 : 3000, signal)
  }
}

async function refreshAndRestoreTask(task: PlaygroundTask) {
  await loadCloudRecords()
  const saved = cloudRecords.value.find((item) => item.id === task.id || (task.request_id && item.request_id === task.request_id))
  if (saved) await restoreRecord(saved)
}

function isImageModel(model: UserSupportedModel) {
  // image_pricing 优先；名称兜底覆盖外联 gpt-image / gemini-banana / adobe 等
  return Boolean(model.image_pricing)
    || /image|flux|sdxl|recraft|midjourney|banana|adobe|dall-?e|imagen/i.test(model.name)
}

function isVideoModel(model: UserSupportedModel) {
  return /video|veo|grok-imagine-video|kling|runway/i.test(model.name)
}

function ensureModelSelections() {
  selectedImageModel.value = imageModels.value.some((model) => modelValue(model) === selectedImageModel.value)
    ? selectedImageModel.value
    : (modelValue(imageModels.value[0] as UserSupportedModel) || '')
  selectedChatModel.value = chatModels.value.some((model) => modelValue(model) === selectedChatModel.value)
    ? selectedChatModel.value
    : (modelValue(chatModels.value[0] as UserSupportedModel) || '')
  selectedVideoModel.value = videoModels.value.some((model) => modelValue(model) === selectedVideoModel.value)
    ? selectedVideoModel.value
    : (modelValue(videoModels.value[0] as UserSupportedModel) || '')
  selectedAudioModel.value = audioModels.value.some((model) => modelValue(model) === selectedAudioModel.value)
    ? selectedAudioModel.value
    : (modelValue(audioModels.value[0] as UserSupportedModel) || '')
  selectedTtsModel.value = ttsModels.value.some((model) => modelValue(model) === selectedTtsModel.value)
    ? selectedTtsModel.value
    : (modelValue(ttsModels.value[0] as UserSupportedModel) || '')
}

function ensureChatToolSelections() {
  chatSelectedModel.value = chatToolModels.value.some((model) => modelValue(model) === chatSelectedModel.value)
    ? chatSelectedModel.value
    : (modelValue(chatToolModels.value[0] as UserSupportedModel) || '')
}

watch(normalizedSelectedKeyId, (keyId, previousKeyId) => {
  if (keyId === previousKeyId) return
  selectedImageModel.value = ''
  selectedChatModel.value = ''
  selectedVideoModel.value = ''
  selectedAudioModel.value = ''
  selectedTtsModel.value = ''
  ensureModelSelections()
})

watch(normalizedChatSelectedKeyId, (keyId, previousKeyId) => {
  if (keyId === previousKeyId) return
  chatSelectedModel.value = ''
  ensureChatToolSelections()
})

const syncPlaygroundDebugState = () => {
  if (!import.meta.env.DEV || typeof window === 'undefined') return
  ;(window as typeof window & { __playgroundDebug?: Record<string, unknown> }).__playgroundDebug = {
    selectedKeyId: selectedKeyId.value,
    normalizedSelectedKeyId: normalizedSelectedKeyId.value,
    selectedKey: selectedKey.value ? { id: selectedKey.value.id, name: selectedKey.value.name, group_id: selectedKey.value.group_id } : null,
    resolvedGroup: resolvedGroup.value ? { id: resolvedGroup.value.id, name: resolvedGroup.value.name, platform: resolvedGroup.value.platform } : null,
    chatSelectedKeyId: chatSelectedKeyId.value,
    chatSelectedKey: chatSelectedKey.value ? { id: chatSelectedKey.value.id, name: chatSelectedKey.value.name, group_id: chatSelectedKey.value.group_id } : null,
    chatResolvedGroup: chatResolvedGroup.value ? { id: chatResolvedGroup.value.id, name: chatResolvedGroup.value.name, platform: chatResolvedGroup.value.platform } : null,
    chatModels: chatToolModels.value.map((model) => ({ id: model.id, name: model.name, platform: model.platform })),
    imageModels: imageModels.value.map((model) => ({ id: model.id, name: model.name, platform: model.platform })),
  }
}

watch([selectedKeyId, normalizedSelectedKeyId, selectedKey, resolvedGroup, chatSelectedKeyId, normalizedChatSelectedKeyId, chatSelectedKey, chatResolvedGroup, chatToolModels, imageModels], syncPlaygroundDebugState, { immediate: true })

async function loadKeyModeData() {
  const [keysResult, channels] = await Promise.all([
    keysAPI.list(1, 100, { status: 'active' }),
    userChannelsAPI.getAvailable(),
  ])
  availableKeys.value = keysResult.items || []
  availableChannels.value = channels
  if (!selectedKeyId.value || !availableKeys.value.some((item) => item.id === selectedKeyId.value)) {
    selectedKeyId.value = availableKeys.value[0]?.id ?? null
  }
  if (!chatSelectedKeyId.value || !availableKeys.value.some((item) => item.id === chatSelectedKeyId.value)) {
    chatSelectedKeyId.value = availableKeys.value[0]?.id ?? null
  }
  ensureModelSelections()
  ensureChatToolSelections()
}

function estimateBatchPrice(model: UserSupportedModel | null, size: string, count: number) {
  if (!count) return formatMoney(0)
  if (!model) return '以渠道结算为准'
  if (model.image_pricing) {
    const unit = size.includes('1536') ? model.image_pricing.price_2k : model.image_pricing.price_1k
    return unit ? formatMoney(unit * count) : '以渠道结算为准'
  }
  const unit = model.pricing?.per_request_price ?? model.pricing?.image_output_price
  return unit ? formatMoney(unit * count) : '以渠道结算为准'
}

async function submitImage() {
  if (!canSubmitImage.value || submitting.value) return
  const balanceBefore = balance.value
  startRequest()
  try {
    const kind = activeTool.value === 'edit' ? 'edit' : 'image'
    const title = kind === 'edit' ? '图片编辑' : 'AI 生图'
    const requestPayload: Record<string, unknown> = {
      title,
      asset_kind: 'image',
      prompt: imagePrompt.value.trim(),
      size: imageSize.value,
      quality: imageQuality.value,
      style: imageStyle.value,
      background: imageBackground.value,
      metadata: getExecutionMetadata(),
    }
    if (kind === 'edit') requestPayload.media = await buildPlaygroundImageMedia([editImage.value].filter(Boolean))
    const task = await submitAsyncPlaygroundJob(kind, selectedImageModel.value, requestPayload)
    await waitForPlaygroundTask(task.id, abortController?.signal)
    lastBilling.value = await resolveBilling(undefined, balanceBefore)
    await refreshAndRestoreTask(task)
  } catch (cause) { handleError(cause, '图片处理失败，本次不应扣费。') } finally { endRequest() }
}

async function sendChat() {
  const text = chatInput.value.trim(); if (!text || submitting.value) return
  const balanceBefore = balance.value
  chatInput.value = ''; chatMessages.value.push({ role: 'user', content: text }); const assistantIndex = chatMessages.value.push({ role: 'assistant', content: '' }) - 1; startRequest()
  try {
    await modelTestAPI.streamPlaygroundChat({
      auth: getAuthContext(),
      model: currentChatModel.value?.name || chatSelectedModel.value,
      messages: chatMessages.value.slice(0, assistantIndex) as ChatMessage[],
      signal: abortController?.signal,
      onDelta(delta) {
        chatMessages.value[assistantIndex].content += delta
      },
      onBilling(billing, id) {
        requestId.value = id || requestId.value
        lastBilling.value = billing || lastBilling.value
      },
      async onDone() {
        lastBilling.value = await resolveBilling(lastBilling.value, balanceBefore)
      },
    })
  } catch (cause) { if (!isAbortError(cause) && !chatMessages.value[assistantIndex].content) chatMessages.value.splice(assistantIndex, 1); handleError(cause, '对话请求失败，本次不应扣费。') } finally { endRequest() }
}

async function generateCopywriting() {
  if (!copywritingName.value.trim() || !copywritingBrief.value.trim() || submitting.value) return
  const balanceBefore = balance.value
  startRequest(); textResult.value = ''
  try {
    const prompt = `商品名称：${copywritingName.value}\n商品信息：${copywritingBrief.value}\n目标平台：${copywritingPlatform.value}\n输出语言：${copywritingLanguage.value}\n请输出：3个标题、5条核心卖点、详情描述、1段社媒短文案。`
    const task = await submitAsyncPlaygroundJob('copywriting', selectedChatModel.value, {
      title: copywritingName.value,
      asset_kind: 'text',
      prompt,
      metadata: getExecutionMetadata(),
    })
    await waitForPlaygroundTask(task.id, abortController?.signal)
    lastBilling.value = await resolveBilling(undefined, balanceBefore)
    await refreshAndRestoreTask(task)
  } catch (cause) { handleError(cause, '商品文案生成失败。') } finally { endRequest() }
}

function buildImageTranslatePrompt() {
  return [
    '请对输入图片做“图上文字本地化”，并输出一张新的图片。',
    `源语言：${translateSource.value}。`,
    `目标语言：${translateTarget.value}。`,
    '要求：',
    '1. 完整保留原图的构图、主体、人物、商品、背景、光线、色彩、材质和版式位置。',
    '2. 仅将图中可见文字翻译并替换为目标语言，语义准确、自然。',
    '3. 译文尽量匹配原文字号、字重、颜色、对齐方式与所在区域，不要额外增加装饰文字。',
    '4. 不要改变 logo 图形；若 logo 含可读文字，按目标语言自然处理，无法确认时保持原样。',
    '5. 不要输出解释或文本，只输出译后图片。',
  ].join('\n')
}

async function translateImageText() {
  if (!translateImage.value || !selectedImageModel.value || submitting.value) return
  const balanceBefore = balance.value
  startRequest()
  resultImage.value = ''
  textResult.value = ''
  try {
    const prompt = buildImageTranslatePrompt()
    const task = await submitAsyncPlaygroundJob('image-translate', selectedImageModel.value, {
      title: `图片翻译 · ${translateTarget.value}`,
      asset_kind: 'image',
      prompt,
      source_language: translateSource.value,
      target_language: translateTarget.value,
      size: imageSize.value,
      quality: imageQuality.value,
      style: imageStyle.value,
      background: imageBackground.value,
      media: await buildPlaygroundImageMedia([translateImage.value]),
      metadata: { source_language: translateSource.value, target_language: translateTarget.value, ...getExecutionMetadata() },
    })
    await waitForPlaygroundTask(task.id, abortController?.signal)
    lastBilling.value = await resolveBilling(undefined, balanceBefore)
    await refreshAndRestoreTask(task)
  } catch (cause) {
    handleError(cause, '图片翻译失败。')
  } finally {
    endRequest()
  }
}

async function runBatchImages() {
  if (!canRunBatch.value || batchRunning.value) return
  batchRunning.value = true
  error.value = ''
  batchInputs.value.forEach((item) => { if (item.status !== 'completed') { item.status = 'processing'; item.error = undefined } })
  const taskKind = activeTool.value === 'batch-clone' ? 'batch-clone' : 'batch-main'
  startRequest()
  try {
    const cloning = activeTool.value === 'batch-clone'
    const items = await Promise.all(batchInputs.value.map(async (item) => {
      const prompt = cloning ? `${batchPrompt.value}\n第一张是待处理商品图，第二张是参考图。参考第二张的构图、光线和背景，但保留第一张商品的外观、颜色、文字和比例。` : batchPrompt.value
      return {
        prompt,
        media: await buildPlaygroundImageMedia(cloning ? [item.input, referenceImage.value] : [item.input]),
      }
    }))
    const task = await submitAsyncPlaygroundJob(taskKind, selectedImageModel.value, {
      title: taskKindLabel(taskKind),
      asset_kind: 'image',
      prompt: batchPrompt.value,
      count: batchInputs.value.length,
      size: imageSize.value,
      quality: imageQuality.value,
      style: imageStyle.value,
      background: imageBackground.value,
      batch: { items },
      metadata: getExecutionMetadata(),
    })
    await waitForPlaygroundTask(task.id, abortController?.signal)
    await loadCloudRecords()
    const saved = cloudRecords.value.find((item) => item.id === task.id)
    const assets = saved?.assets?.filter((item) => item.kind === 'image') || []
    await Promise.all(batchInputs.value.map(async (item, index) => {
      const asset = assets[index]
      item.status = asset ? 'completed' : 'failed'
      item.error = asset ? undefined : '未生成成功'
      item.output = asset ? await toDisplayImageUrl(stableAssetUrl(asset)) : item.output
    }))
  } catch (cause) {
    handleError(cause, '批量任务提交失败。')
    batchInputs.value.forEach((item) => { if (item.status === 'processing') item.status = 'failed' })
  } finally {
    batchRunning.value = false
    endRequest()
  }
}

async function processBatchItem(item: BatchImageItem) {
  item.status = 'processing'
  try {
    const cloning = activeTool.value === 'batch-clone'
    const prompt = cloning ? `${batchPrompt.value}\n第一张是待处理商品图，第二张是参考图。参考第二张的构图、光线和背景，但保留第一张商品的外观、颜色、文字和比例。` : batchPrompt.value
    const result = await modelTestAPI.editPlaygroundImage({
      auth: getAuthContext(),
      model: selectedImageModel.value,
      prompt,
      images: cloning ? [item.input, referenceImage.value] : [item.input],
      size: imageSize.value,
      quality: imageQuality.value,
      style: imageStyle.value,
      background: imageBackground.value,
    })
    const output = result.images[0]?.url
    if (!output) throw new Error('未返回图片')
    item.output = output
    item.requestId = result.requestId
    item.status = 'completed'
  } catch (cause) {
    item.status = 'failed'
    item.error = cause instanceof Error ? cause.message : '处理失败'
  }
}

function resolveAsrLanguage(label: string): 'auto' | 'zh' | 'en' {
  if (label === '中文') return 'zh'
  if (label === '英文' || label === '英语') return 'en'
  return 'auto'
}

async function submitAudioTranscription() {
  if (!audioInput.value || !selectedAudioModel.value || submitting.value) return
  if (!audioInput.value.startsWith('data:audio/')) {
    error.value = '请上传 wav 或 mp3 音频文件。'
    return
  }
  const balanceBefore = balance.value
  startRequest(); audioTranscriptText.value = ''
  try {
    const task = await submitAsyncPlaygroundJob('audio-transcribe', selectedAudioModel.value, {
      title: `语音转写 · ${audioInputName.value || '未命名音频'}`,
      asset_kind: 'text',
      filename: audioInputName.value,
      language: audioLanguage.value,
      asr_language: resolveAsrLanguage(audioLanguage.value),
      output_mode: '纯文本',
      media: await buildPlaygroundAudioMedia(audioInput.value, audioInputName.value),
      metadata: { source_kind: 'audio-transcribe', ...getExecutionMetadata() },
    })
    await waitForPlaygroundTask(task.id, abortController?.signal)
    lastBilling.value = await resolveBilling(undefined, balanceBefore)
    await refreshAndRestoreTask(task)
  } catch (cause) {
    handleError(cause, '语音转写失败。')
  } finally {
    endRequest()
  }
}

async function submitAudioGeneration() {
  if (!canSubmitAudioGeneration.value || submitting.value) return
  const balanceBefore = balance.value
  startRequest(); ttsResultUrl.value = ''; ttsResultText.value = ''; ttsDebug.value = ''; audioPreviewLoading.value = false
  try {
    const taskKind = audioGenerateMode.value === 'voiceclone' ? 'audio-voice-clone' : audioGenerateMode.value === 'voicedesign' ? 'audio-voice-design' : 'audio-generate'
    const title = audioGenerateMode.value === 'voiceclone' ? '声音克隆配音' : audioGenerateMode.value === 'voicedesign' ? '音色设计配音' : '标准配音'
    const requestPayload: Record<string, unknown> = {
      title,
      asset_kind: 'audio',
      mode: audioGenerateMode.value,
      text: ttsText.value.trim(),
      language: ttsLanguage.value,
      style: ttsStyle.value,
      voice_preset: audioGenerateMode.value === 'standard' ? ttsVoicePreset.value : undefined,
      voice_description: audioGenerateMode.value === 'voicedesign' ? ttsVoiceDescription.value : undefined,
      persona: audioGenerateMode.value === 'voicedesign' ? ttsPersona.value : undefined,
      has_reference_audio: audioGenerateMode.value === 'voiceclone' ? Boolean(ttsReferenceAudio.value) : undefined,
      authorization_confirmed: audioGenerateMode.value === 'voiceclone' ? ttsAuthorizationConfirmed.value : undefined,
      metadata: { source_kind: taskKind, mode: audioGenerateMode.value, ...getExecutionMetadata() },
    }
    if (audioGenerateMode.value === 'voiceclone') requestPayload.media = await buildPlaygroundAudioMedia(ttsReferenceAudio.value, ttsReferenceAudioName.value, 'reference_audio')
    const task = await submitAsyncPlaygroundJob(taskKind, selectedTtsModel.value, requestPayload)
    await waitForPlaygroundTask(task.id, abortController?.signal)
    lastBilling.value = await resolveBilling(undefined, balanceBefore)
    audioPreviewLoading.value = true
    await refreshAndRestoreTask(task)
    audioPreviewLoading.value = false
  } catch (cause) { handleError(cause, 'AI 配音失败。') } finally { endRequest() }
}

function stopBatch() { batchRunning.value = false }
async function retryBatchItem(item: BatchImageItem) { if (batchRunning.value) return; item.status = 'pending'; item.error = undefined; batchRunning.value = true; await processBatchItem(item); batchRunning.value = false }
function batchStatusLabel(status: BatchImageItem['status']) { return ({ pending: '等待中', processing: '处理中', completed: '已完成', failed: '失败', canceled: '已停止' } as const)[status] }

function buildWatermarkPrompt() {
  if (watermarkMode.value === 'add') {
    if (watermarkAssetType.value === 'logo') {
      return `${watermarkPrompt.value.trim() || '在原图上添加 logo 水印。'} 第二张图是要叠加的 logo 水印，请把它自然叠加到第一张原图上。位置：${watermarkPosition.value}。样式：${watermarkStyle.value}。保持原图主体、构图、颜色和清晰度不变，不要新增其他元素，不要改变 logo 内容。`
    }
    return `${watermarkPrompt.value.trim() || '在原图上添加水印。'} 水印内容：${watermarkText.value.trim()}。位置：${watermarkPosition.value}。样式：${watermarkStyle.value}。保持原图主体、构图、颜色和清晰度不变，不要新增其他元素。`
  }
  return watermarkPrompt.value.trim()
}

async function processWatermark() {
  if (!watermarkImage.value || submitting.value) return
  const balanceBefore = balance.value
  startRequest()
  try {
    const images = watermarkMode.value === 'add' && watermarkAssetType.value === 'logo' && watermarkLogo.value ? [watermarkImage.value, watermarkLogo.value] : [watermarkImage.value]
    const title = watermarkMode.value === 'remove' ? '水印去除' : '添加水印'
    const task = await submitAsyncPlaygroundJob('watermark', selectedImageModel.value, {
      title,
      asset_kind: 'image',
      prompt: buildWatermarkPrompt(),
      has_mask: watermarkMode.value === 'remove' && Boolean(watermarkMask.value),
      mode: watermarkMode.value,
      asset_type: watermarkMode.value === 'add' ? watermarkAssetType.value : undefined,
      watermark_text: watermarkMode.value === 'add' && watermarkAssetType.value === 'text' ? watermarkText.value.trim() : undefined,
      has_logo: watermarkMode.value === 'add' && watermarkAssetType.value === 'logo' ? Boolean(watermarkLogo.value) : undefined,
      watermark_position: watermarkMode.value === 'add' ? watermarkPosition.value : undefined,
      watermark_style: watermarkMode.value === 'add' ? watermarkStyle.value : undefined,
      size: imageSize.value,
      quality: imageQuality.value,
      style: imageStyle.value,
      background: imageBackground.value,
      media: await buildPlaygroundImageMedia(images, watermarkMode.value === 'remove' ? watermarkMask.value || undefined : undefined),
      metadata: { mode: watermarkMode.value, asset_type: watermarkMode.value === 'add' ? watermarkAssetType.value : undefined, ...getExecutionMetadata() },
    })
    await waitForPlaygroundTask(task.id, abortController?.signal)
    lastBilling.value = await resolveBilling(undefined, balanceBefore)
    await refreshAndRestoreTask(task)
  } catch (cause) { handleError(cause, watermarkMode.value === 'remove' ? '水印去除失败，本次不应扣费。' : '添加水印失败，本次不应扣费。') } finally { endRequest() }
}

async function fetchAuthedAsset(url: string): Promise<{ objectUrl: string, blob: Blob }> {
  const value = String(url || '').trim()
  if (!value) throw new Error('媒体地址为空')
  if (failedAssetUrls.has(value)) throw new Error('媒体地址已失效')
  if (value.startsWith('blob:')) return { objectUrl: value, blob: new Blob() }
  if (value.startsWith('data:')) {
    return { objectUrl: value, blob: new Blob() }
  }
  if (/^https?:\/\//i.test(value)) {
    return { objectUrl: value, blob: new Blob() }
  }
  const token = localStorage.getItem('auth_token')
  console.info('[playground-media] fetchAuthedAsset:start', {
    url: value,
    hasToken: Boolean(token),
  })
  const response = await fetch(value, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    credentials: 'include',
  })
  console.info('[playground-media] fetchAuthedAsset:result', {
    url: value,
    status: response.status,
    ok: response.ok,
  })
  if (!response.ok) {
    failedAssetUrls.add(value)
    throw new Error(`加载媒体失败：${response.status}`)
  }
  const blob = await response.blob()
  return { objectUrl: URL.createObjectURL(blob), blob }
}

async function fetchAuthedAssetUrl(url: string): Promise<string> {
  const value = String(url || '').trim()
  if (!value || value.startsWith('data:') || value.startsWith('blob:') || /^https?:\/\//i.test(value)) return value
  const { objectUrl } = await fetchAuthedAsset(value)
  return objectUrl
}

async function toPlayableMediaUrl(url: string): Promise<string> {
  return fetchAuthedAssetUrl(url)
}

async function waitForAudioMetadata(url: string): Promise<void> {
  const value = String(url || '').trim()
  if (!value) return
  console.info('[playground-audio] waitForAudioMetadata:start', { url: value })
  await new Promise<void>((resolve) => {
    const audio = new Audio()
    const done = (event: string) => {
      console.info('[playground-audio] waitForAudioMetadata:done', {
        event,
        url: value,
        duration: Number.isFinite(audio.duration) ? audio.duration : null,
        readyState: audio.readyState,
        networkState: audio.networkState,
        error: audio.error ? { code: audio.error.code, message: audio.error.message } : null,
      })
      audio.onloadedmetadata = null
      audio.oncanplaythrough = null
      audio.onerror = null
      resolve()
    }
    audio.onloadedmetadata = () => done('loadedmetadata')
    audio.oncanplaythrough = () => done('canplaythrough')
    audio.onerror = () => done('error')
    audio.preload = 'auto'
    audio.src = value
    audio.load()
  })
}

async function resolvePlayableAudioUrl(url: string): Promise<string> {
  console.info('[playground-audio] resolvePlayableAudioUrl:start', { url })
  const playable = await toPlayableMediaUrl(url)
  console.info('[playground-audio] resolvePlayableAudioUrl:fetched', { url, playable })
  await waitForAudioMetadata(playable)
  console.info('[playground-audio] resolvePlayableAudioUrl:ready', { url, playable })
  return playable
}

async function toDisplayImageUrl(url: string): Promise<string> {
  return fetchAuthedAssetUrl(url)
}

function inferExtension(contentType: string, fallback = 'bin') {
  const value = String(contentType || '').toLowerCase()
  if (value.includes('image/png')) return 'png'
  if (value.includes('image/jpeg')) return 'jpg'
  if (value.includes('image/webp')) return 'webp'
  if (value.includes('image/gif')) return 'gif'
  if (value.includes('video/mp4')) return 'mp4'
  if (value.includes('video/webm')) return 'webm'
  if (value.includes('audio/mpeg')) return 'mp3'
  if (value.includes('audio/mp3')) return 'mp3'
  if (value.includes('audio/wav') || value.includes('audio/x-wav') || value.includes('audio/wave')) return 'wav'
  if (value.includes('audio/mp4') || value.includes('audio/m4a')) return 'm4a'
  return fallback
}

function recordMediaAsset(record: PlaygroundRecord, kinds?: string[]) {
  const allow = Array.isArray(kinds) && kinds.length ? new Set(kinds) : null
  const matches = (asset: any) => {
    if (!asset) return false
    if (allow && !allow.has(String(asset.kind || ''))) return false
    return (asset.content && String(asset.content).startsWith('data:')) || Boolean(stableAssetUrl(asset))
  }
  const primary = record.primary_asset
  if (matches(primary)) return primary
  return record.assets?.find((asset) => matches(asset))
}

async function persistMediaAsset(input: {
  taskId: number
  kind: 'image' | 'audio' | 'video'
  title: string
  sourceUrl?: string
  sourceContent?: string
  contentType?: string
  metadata?: Record<string, unknown>
}): Promise<PersistedMediaRef | null> {
  if (input.sourceContent) {
    const asset = await playgroundCloudAPI.createAsset({
      task_id: input.taskId,
      kind: input.kind,
      title: input.title,
      content: input.sourceContent,
      content_type: input.contentType,
      metadata: input.metadata,
    })
    return playgroundCloudAPI.toPersistedMediaRef(asset)
  }
  if (input.sourceUrl) {
    const asset = await playgroundCloudAPI.createAsset({
      task_id: input.taskId,
      kind: input.kind,
      title: input.title,
      url: input.sourceUrl,
      content_type: input.contentType,
      metadata: input.metadata,
    })
    return playgroundCloudAPI.toPersistedMediaRef(asset)
  }
  return null
}

async function loadCloudRecords() {
  cloudLoading.value = true
  try {
    const result = await playgroundCloudAPI.listRecords({ page_size: 10 })
    const items = Array.isArray(result?.items)
      ? result.items
      : (Array.isArray((result as any)?.data?.items) ? (result as any).data.items : [])

    cloudRecords.value = items.map((item: PlaygroundRecord) => ({
      ...item,
      assets: Array.isArray(item.assets)
        ? item.assets.map((asset) => ({
            ...asset,
            content: asset?.kind === 'text'
              ? asset.content
              : (String(asset?.content || '').startsWith('data:') ? undefined : asset.content),
          }))
        : [],
      primary_asset: item.primary_asset
        ? {
            ...item.primary_asset,
            content: item.primary_asset.kind === 'text'
              ? item.primary_asset.content
              : (String(item.primary_asset.content || '').startsWith('data:') ? undefined : item.primary_asset.content),
          }
        : undefined,
    }))
  } catch (cause) {
    cloudRecords.value = []
    console.error('加载创作记录失败', cause)
    const message = cause instanceof Error
      ? cause.message
      : (typeof cause === 'object' && cause && 'message' in cause ? String((cause as any).message || '') : '')
    if (!error.value) {
      error.value = message && message !== '加载创作记录失败。'
        ? `加载创作记录失败：${message}`
        : '加载创作记录失败。'
    }
  } finally {
    cloudLoading.value = false
  }
}

async function removeRecord(id: number) {
  try {
    await playgroundCloudAPI.deleteRecord(id)
    cloudRecords.value = cloudRecords.value.filter((item) => item.id !== id)
  } catch (cause) {
    handleError(cause, '删除创作记录失败。')
  }
}

function taskKindLabel(kind: string) {
  return ({
    copywriting: '商品文案',
    'image-translate': '图片翻译',
    translate: '图片翻译',
    chat: '对话助手',
    'batch-main': '批量商品主图',
    'batch-clone': '参考图批量克隆',
    watermark: '水印处理',
    image: 'AI 生图',
    edit: '图片编辑',
    video: 'AI 视频',
    'audio-transcribe': '语音转写',
    'audio-generate': 'AI 配音',
    'audio-voice-design': '音色设计',
    'audio-voice-clone': '声音克隆',
  } as Record<string, string>)[kind] || kind
}

function normalizeRecord(value: unknown) {
  if (!value) return {}
  if (typeof value === 'string') {
    try {
      const parsed = JSON.parse(value)
      return parsed && typeof parsed === 'object' ? parsed as Record<string, any> : {}
    } catch {
      return {}
    }
  }
  return typeof value === 'object' ? value as Record<string, any> : {}
}

function recordPrompt(record: PlaygroundRecord) {
  const payload = normalizeRecord(record.request_payload)
  const result = normalizeRecord(record.result_payload)
  return String(payload.prompt || payload.text || payload.filename || result.content || record.primary_asset?.title || taskKindLabel(record.kind) || '')
}

function isPlayableMediaUrl(url: string) {
  const value = String(url || '').trim()
  return Boolean(value)
}

function stableAssetUrl(asset: PlaygroundRecord['primary_asset'] | PlaygroundRecord['assets'][number] | undefined) {
  const storageKey = String((asset as any)?.storage_key || '').trim()
  if (storageKey) {
    return `/api/v1/playground/assets/content/${encodeURIComponent(storageKey)}`
  }
  const directUrl = String((asset as any)?.url || '').trim()
  if (directUrl) return directUrl
  const assetId = Number((asset as any)?.id || 0)
  if (assetId > 0) {
    return `/api/v1/playground/assets/${assetId}/content`
  }
  return ''
}

function recordResultUrl(record: PlaygroundRecord) {
  const payload = normalizeRecord(record.result_payload)
  const primary = record.primary_asset
  if (primary?.content && String(primary.content).startsWith('data:')) return primary.content
  const primaryUrl = stableAssetUrl(primary)
  if (primaryUrl && isPlayableMediaUrl(primaryUrl)) return primaryUrl
  const media = record.assets?.find((asset) => (asset.content && String(asset.content).startsWith('data:')) || Boolean(stableAssetUrl(asset)))
  if (media?.content && String(media.content).startsWith('data:')) return media.content
  const mediaUrl = stableAssetUrl(media)
  if (mediaUrl && isPlayableMediaUrl(mediaUrl)) return mediaUrl
  const fallback = String(payload.url || payload.audio_url || payload.video_url || '')
  return isPlayableMediaUrl(fallback) ? fallback : ''
}

function recordResultText(record: PlaygroundRecord) {
  const payload = normalizeRecord(record.result_payload)
  const textAsset = record.assets?.find((asset) => asset.kind === 'text' && asset.content)
  return String(textAsset?.content || payload.content || payload.transcript || payload.text || '')
}

function recordDownloadUrl(record: PlaygroundRecord) {
  return recordResultUrl(record)
}

async function downloadRecord(record: PlaygroundRecord) {
  const raw = recordDownloadUrl(record)
  if (raw) {
    if (raw.startsWith('blob:') || raw.startsWith('data:')) {
      const asset = recordMediaAsset(record)
      const ext = inferExtension(String((asset as any)?.content_type || ''), record.kind === 'video' ? 'mp4' : record.kind.includes('audio') ? 'mp3' : 'png')
      downloadImage(raw, `relayq-${record.kind}-${record.id}.${ext}`)
      return
    }
    try {
      const { objectUrl, blob } = await fetchAuthedAsset(raw)
      const asset = recordMediaAsset(record)
      const ext = inferExtension(blob.type || String((asset as any)?.content_type || ''), record.kind === 'video' ? 'mp4' : record.kind.includes('audio') ? 'mp3' : 'png')
      downloadImage(objectUrl, `relayq-${record.kind}-${record.id}.${ext}`)
      if (objectUrl.startsWith('blob:')) setTimeout(() => URL.revokeObjectURL(objectUrl), 60_000)
      return
    } catch (cause) {
      console.error('下载媒体失败', cause)
    }
  }
  const text = recordResultText(record)
  if (!text) return
  const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
  const objectUrl = URL.createObjectURL(blob)
  downloadImage(objectUrl, `relayq-${record.kind}-${record.id}.txt`)
  URL.revokeObjectURL(objectUrl)
}

async function restoreRecord(record: PlaygroundRecord) {
  const payload = normalizeRecord(record.request_payload)
  const prompt = recordPrompt(record)
  const resultUrl = recordResultUrl(record)
  const resultText = recordResultText(record)
  error.value = ''
  requestId.value = record.request_id || ''
  lastBilling.value = undefined
  resultImage.value = ''
  textResult.value = ''
  audioTranscriptText.value = ''
  ttsResultUrl.value = ''
  ttsResultText.value = ''
  videoUrl.value = ''
  videoStatus.value = ''
  videoProgress.value = undefined

  const kind = record.kind
  if (kind === 'chat') {
    activeTool.value = 'chat'
    chatInput.value = String(payload.prompt || prompt)
    if (resultText) chatMessages.value = [{ role: 'user', content: String(payload.prompt || prompt) }, { role: 'assistant', content: resultText }]
    return
  }
  if (kind === 'copywriting') {
    activeTool.value = 'copywriting'
    copywritingBrief.value = String(payload.prompt || prompt)
    textResult.value = resultText
    return
  }
  if (kind === 'image-translate' || kind === 'translate') {
    activeTool.value = 'translate'
    translateSource.value = String(payload.source_language || translateSource.value || '自动识别')
    translateTarget.value = String(payload.target_language || translateTarget.value || '英文')
    // 旧版文本翻译记录兼容展示；新版以图片结果为主
    if (resultUrl) {
      resultImage.value = await toDisplayImageUrl(resultUrl)
    } else if (resultText) {
      textResult.value = resultText
    }
    return
  }
  if (kind === 'video') {
    activeTool.value = 'video'
    videoPrompt.value = String(payload.prompt || prompt)
    if (resultUrl) {
      videoUrl.value = await toPlayableMediaUrl(resultUrl)
    } else if (record.request_id) {
      scheduleVideoPoll()
    }
    return
  }
  if (['audio-generate', 'audio-voice-design', 'audio-voice-clone'].includes(kind)) {
    activeTool.value = 'audio-generate'
    ttsText.value = String(payload.text || payload.prompt || prompt)
    if (resultUrl) {
      audioPreviewLoading.value = true
      ttsResultUrl.value = await resolvePlayableAudioUrl(resultUrl)
      audioPreviewLoading.value = false
    } else {
      ttsResultUrl.value = ''
    }
    ttsResultText.value = resultText
    if (payload.mode === 'voicedesign' || kind === 'audio-voice-design') audioGenerateMode.value = 'voicedesign'
    else if (payload.mode === 'voiceclone' || kind === 'audio-voice-clone') audioGenerateMode.value = 'voiceclone'
    else audioGenerateMode.value = 'standard'
    return
  }
  if (kind === 'audio-transcribe') {
    activeTool.value = 'audio-transcribe'
    audioTranscriptText.value = resultText
    if (!audioTranscriptText.value) error.value = '已恢复参数，未找到可回显的转写结果，请重新转写。'
    return
  }
  if (kind === 'watermark') {
    activeTool.value = 'watermark'
    watermarkPrompt.value = String(payload.prompt || prompt)
    if (resultUrl) {
      resultImage.value = await toDisplayImageUrl(resultUrl)
    } else {
      error.value = '已恢复参数，未找到可回显的图片结果，请重新生成。'
    }
    return
  }
  if (kind === 'edit') {
    activeTool.value = 'edit'
    imagePrompt.value = String(payload.prompt || prompt)
    if (resultUrl) {
      resultImage.value = await toDisplayImageUrl(resultUrl)
    } else {
      error.value = '已恢复参数，未找到可回显的图片结果，请重新生成。'
    }
    return
  }
  if (kind === 'image' || kind === 'batch-main' || kind === 'batch-clone') {
    activeTool.value = kind === 'batch-main' || kind === 'batch-clone' ? kind : 'image'
    imagePrompt.value = String(payload.prompt || prompt)
    batchPrompt.value = String(payload.prompt || prompt)
    if (resultUrl) {
      resultImage.value = await toDisplayImageUrl(resultUrl)
    } else if (kind === 'image') {
      error.value = '已恢复参数，未找到可回显的图片结果，请重新生成。'
    }
    return
  }
  activeTool.value = 'history'
}

async function copyTextResult() {
  const value = textResult.value || audioTranscriptText.value
  if (value) await navigator.clipboard.writeText(value)
}

function downloadTranscriptTxt() {
  const value = audioTranscriptText.value.trim()
  if (!value) return
  const baseName = (audioInputName.value || 'transcript')
    .replace(/\.[^.]+$/, '')
    .replace(/[\\/:*?"<>|]+/g, '_')
    .trim() || 'transcript'
  const stamp = new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')
  const blob = new Blob([value], { type: 'text/plain;charset=utf-8' })
  const objectUrl = URL.createObjectURL(blob)
  downloadImage(objectUrl, `relayq-transcript-${baseName}-${stamp}.txt`)
  URL.revokeObjectURL(objectUrl)
}

async function submitVideo() {
  if (!videoPrompt.value.trim() || submitting.value) return
  startRequest()
  try {
    const model = selectedVideoModel.value
    if (!model) throw new Error('当前 API Key 所属分组没有可用的视频模型。')
    const requestPayload: Record<string, unknown> = {
      title: 'AI 视频',
      asset_kind: 'video',
      prompt: videoPrompt.value.trim(),
      duration: Number(videoDuration.value),
      aspect_ratio: videoAspectRatio.value,
      resolution: videoResolution.value,
      has_image: Boolean(videoImage.value),
      metadata: getExecutionMetadata(),
    }
    if (videoImage.value) requestPayload.media = await buildPlaygroundVideoMedia(videoImage.value)
    const task = await submitAsyncPlaygroundJob('video', model, requestPayload)
    videoStatus.value = 'queued'
    videoUrl.value = ''
    const done = await waitForPlaygroundTask(task.id, abortController?.signal)
    requestId.value = done.request_id || requestId.value
    await refreshAndRestoreTask(done)
  } catch (cause) { handleError(cause, '视频体验组尚未配置或任务提交失败。') } finally { endRequest() }
}

async function pollVideoOnce() {
  if (!requestId.value || videoPolling.value) return
  videoPolling.value = true
  try {
    const result = await modelTestAPI.getPlaygroundVideo(getAuthContext(), requestId.value)
    videoStatus.value = result.status || videoStatus.value
    videoProgress.value = result.progress
    lastBilling.value = result.billing || lastBilling.value
    if (result.videoUrl) {
      let taskId = cloudRecords.value.find((item) => item.request_id === requestId.value)?.id
      if (!taskId) {
        const created = await playgroundCloudAPI.createTask({ kind: 'video', status: 'succeeded', model: selectedVideoModel.value, request_id: requestId.value || undefined, request_payload: { prompt: videoPrompt.value.trim(), ...getExecutionMetadata() }, result_payload: { status: videoStatus.value, progress: videoProgress.value, ...getExecutionMetadata() } }).catch(() => undefined)
        taskId = created?.id
      }
      const mediaRef = taskId
        ? await persistMediaAsset({
            taskId,
            kind: 'video',
            title: 'AI 视频',
            sourceUrl: result.videoUrl,
            contentType: 'video/mp4',
            metadata: { request_id: requestId.value, auth_token: getAuthContext().apiKey, ...getExecutionMetadata() },
          }).catch(() => null)
        : null
      videoUrl.value = result.videoUrl
      if (mediaRef?.url) {
        try {
          videoUrl.value = await toPlayableMediaUrl(mediaRef.url)
        } catch (cause) {
          console.warn('视频结果已生成，但持久化资源回读失败，保留当前结果预览', cause)
        }
      }
      void loadCloudRecords()
      setTimeout(() => {
        const saved = cloudRecords.value.find((item) => item.request_id === requestId.value)
        if (saved) void restoreRecord(saved)
      }, 1200)
    } else if (!['failed', 'error'].includes(videoStatus.value.toLowerCase())) {
      scheduleVideoPoll()
    }
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '查询视频任务失败，可稍后手动重试。'
  } finally {
    videoPolling.value = false
  }
}
function scheduleVideoPoll() { if (pollTimer !== null) window.clearTimeout(pollTimer); pollTimer = window.setTimeout(pollVideoOnce, document.hidden ? 15000 : 5000) }

function startRequest() { stopRequest(); abortController = new AbortController(); submitting.value = true; error.value = ''; requestId.value = ''; lastBilling.value = undefined }
function endRequest() { submitting.value = false; abortController = null }
function stopRequest() { abortController?.abort(); abortController = null; submitting.value = false; if (pollTimer !== null) window.clearTimeout(pollTimer); pollTimer = null; videoPolling.value = false }
function describeError(cause: unknown, fallback: string) {
  if (cause instanceof Error) return cause.message || fallback
  if (typeof cause === 'string' && cause.trim()) return cause.trim()
  if (cause && typeof cause === 'object') {
    const record = cause as Record<string, unknown>
    const response = (record.response && typeof record.response === 'object') ? record.response as Record<string, unknown> : null
    const responseData = (response?.data && typeof response.data === 'object') ? response.data as Record<string, unknown> : null
    const responseBody = typeof response?.data === 'string' && response.data.trim() ? response.data.trim() : ''
    const nestedError = (responseData?.error && typeof responseData.error === 'object') ? responseData.error as Record<string, unknown> : null
    const direct = typeof record.message === 'string' && record.message.trim() ? record.message.trim() : ''
    const reason = typeof record.reason === 'string' && record.reason.trim() ? record.reason.trim() : ''
    const responseMessage = typeof responseData?.message === 'string' && responseData.message.trim() ? responseData.message.trim() : ''
    const nestedMessage = typeof nestedError?.message === 'string' && nestedError.message.trim() ? nestedError.message.trim() : ''
    const details = responseData?.metadata ?? responseData?.details ?? nestedError?.details ?? record.metadata ?? record.details ?? record.detail
    const preferred = responseMessage || nestedMessage || direct
    if (preferred && details) {
      const extra = typeof details === 'string' ? details : JSON.stringify(details)
      return `${preferred}（${extra}）`
    }
    if (preferred) return preferred
    if (responseBody) return responseBody
    if (direct && details) {
      const extra = typeof details === 'string' ? details : JSON.stringify(details)
      return `${direct}（${extra}）`
    }
    if (direct) return direct
    if (reason) return reason
  }
  return fallback
}
function handleError(cause: unknown, fallback: string) {
  if (isAbortError(cause)) return
  const message = describeError(cause, fallback)
  if (/no available accounts/i.test(message)) {
    error.value = '当前 API Key 所属分组下没有可用账号支持所选模型，请切换模型或 API Key。'
    return
  }
  error.value = message
}
async function resolveBilling(billing: PlaygroundBilling | undefined, before: number) { try { const user = await authStore.refreshUser(); const after = user.balance ?? balance.value; const amount = billing?.amount ?? Math.max(0, Number((before - after).toFixed(6))); return { ...billing, amount: amount || billing?.amount, balance_after: billing?.balance_after ?? after } } catch { return billing } }

function onPreviewAudioLoadedMetadata(event: Event) {
  const audio = event.target as HTMLAudioElement | null
  console.info('[playground-audio] preview:loadedmetadata', {
    src: audio?.currentSrc || audio?.src || '',
    duration: audio && Number.isFinite(audio.duration) ? audio.duration : null,
    readyState: audio?.readyState,
    networkState: audio?.networkState,
  })
}

function onPreviewAudioCanPlayThrough(event: Event) {
  const audio = event.target as HTMLAudioElement | null
  console.info('[playground-audio] preview:canplaythrough', {
    src: audio?.currentSrc || audio?.src || '',
    duration: audio && Number.isFinite(audio.duration) ? audio.duration : null,
    readyState: audio?.readyState,
    networkState: audio?.networkState,
  })
}

function onPreviewAudioError(event: Event) {
  const audio = event.target as HTMLAudioElement | null
  console.info('[playground-audio] preview:error', {
    src: audio?.currentSrc || audio?.src || '',
    duration: audio && Number.isFinite(audio.duration) ? audio.duration : null,
    readyState: audio?.readyState,
    networkState: audio?.networkState,
    error: audio?.error ? { code: audio.error.code, message: audio.error.message } : null,
  })
}

async function readImageFile(event: Event) { const file = (event.target as HTMLInputElement).files?.[0]; if (!file) return ''; if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) throw new Error('仅支持 JPG、PNG 和 WEBP。'); if (file.size > 8 * 1024 * 1024) throw new Error('图片不能超过 8MB。'); return URL.createObjectURL(file) }
async function handleImageFile(event: Event) { try { editImage.value = await readImageFile(event) } catch (cause) { error.value = cause instanceof Error ? cause.message : '读取图片失败。' } }
async function handleVideoFile(event: Event) { try { videoImage.value = await readImageFile(event) } catch (cause) { error.value = cause instanceof Error ? cause.message : '读取图片失败。' } }
async function handleTranslateFile(event: Event) { try { translateImage.value = await readImageFile(event) } catch (cause) { error.value = cause instanceof Error ? cause.message : '读取图片失败。' } }
async function handleReferenceFile(event: Event) { try { referenceImage.value = await readImageFile(event) } catch (cause) { error.value = cause instanceof Error ? cause.message : '读取参考图失败。' } }
async function handleWatermarkFile(event: Event) { try { watermarkImage.value = await readImageFile(event) } catch (cause) { error.value = cause instanceof Error ? cause.message : '读取原图失败。' } }
async function handleWatermarkMask(event: Event) { try { watermarkMask.value = await readImageFile(event) } catch (cause) { error.value = cause instanceof Error ? cause.message : '读取蒙版失败。' } }
async function handleWatermarkLogo(event: Event) { try { watermarkLogo.value = await readImageFile(event) } catch (cause) { error.value = cause instanceof Error ? cause.message : '读取 Logo 失败。' } }
async function handleAudioFile(event: Event) { try { const file = (event.target as HTMLInputElement).files?.[0]; if (!file) return; audioInputName.value = file.name; audioInput.value = await new Promise<string>((resolve, reject) => { const reader = new FileReader(); reader.onload = () => resolve(String(reader.result || '')); reader.onerror = () => reject(new Error('读取音频失败。')); reader.readAsDataURL(file) }) } catch (cause) { error.value = cause instanceof Error ? cause.message : '读取音频失败。' } }
async function handleTtsReferenceFile(event: Event) { try { const file = (event.target as HTMLInputElement).files?.[0]; if (!file) return; ttsReferenceAudioName.value = file.name; ttsReferenceAudio.value = await new Promise<string>((resolve, reject) => { const reader = new FileReader(); reader.onload = () => resolve(String(reader.result || '')); reader.onerror = () => reject(new Error('读取参考音频失败。')); reader.readAsDataURL(file) }) } catch (cause) { error.value = cause instanceof Error ? cause.message : '读取参考音频失败。' } }
async function handleBatchFiles(event: Event) { const input = event.target as HTMLInputElement; const files = Array.from(input.files || []).slice(0, Math.max(0, 6 - batchInputs.value.length)); for (const file of files) { try { batchInputs.value.push({ id: createId(), input: await readImage(file), status: 'pending' }) } catch (cause) { error.value = cause instanceof Error ? cause.message : '读取商品图失败。' } } input.value = '' }
function removeBatchItem(id: string) { if (!batchRunning.value) batchInputs.value = batchInputs.value.filter((item) => item.id !== id) }
async function readImage(file: File) { if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) throw new Error(`${file.name} 格式不支持。`); if (file.size > 8 * 1024 * 1024) throw new Error(`${file.name} 超过 8MB。`); return URL.createObjectURL(file) }

function clearChat() { chatMessages.value = []; error.value = ''; requestId.value = ''; lastBilling.value = undefined }
function downloadResultImage() { if (!resultImage.value) return; const link = document.createElement('a'); link.href = resultImage.value; link.download = 'relayq-creation.png'; link.click() }
function downloadImage(url: string, filename: string) { const link = document.createElement('a'); link.href = url; link.download = filename; link.target = '_blank'; link.click() }
function formatMoney(value: number) { return `¥${Number(value || 0).toFixed(2)}` }
function createId() { return typeof crypto !== 'undefined' && 'randomUUID' in crypto ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}` }
function isAbortError(cause: unknown) { return cause instanceof DOMException && cause.name === 'AbortError' }

const ResultPanel = defineComponent({ props: { loading: Boolean, error: String, billing: Object as () => PlaygroundBilling | undefined }, setup(props, { slots }) { return () => h('section', { class: 'flex min-h-[620px] flex-col rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900' }, [h('div', { class: 'mb-4 flex items-center justify-between' }, [h('h2', { class: 'text-lg font-semibold' }, '结果预览'), props.billing?.amount ? h('span', { class: 'text-sm font-medium text-emerald-600' }, `实扣 ¥${Number(props.billing.amount).toFixed(2)}`) : null]), props.loading ? h('div', { class: 'flex flex-1 items-center justify-center text-gray-500' }, '正在处理真实任务…') : props.error ? h('div', { class: 'flex flex-1 items-center justify-center rounded-lg bg-red-50 p-8 text-center text-sm text-red-700 dark:bg-red-950/30 dark:text-red-300' }, props.error) : slots.result?.() || h('div', { class: 'flex flex-1 items-center justify-center rounded-lg border border-dashed border-gray-300 text-sm text-gray-500 dark:border-dark-600' }, '配置参数后开始创作')]) } })

const RecordList = defineComponent({
  props: {
    items: { type: Array as () => PlaygroundRecord[], required: true },
    loading: Boolean,
  },
  emits: ['restore', 'remove', 'download'],
  setup(props, { emit }) {
    return () => {
      if (props.loading) return h('div', { class: 'rounded-lg border border-dashed border-gray-300 p-8 text-center text-sm text-gray-500 dark:border-dark-600' }, '加载中…')
      if (!props.items.length) return h('div', { class: 'rounded-lg border border-dashed border-gray-300 p-8 text-center text-sm text-gray-500 dark:border-dark-600' }, '还没有创作记录。')
      return h('div', { class: 'grid gap-3 md:grid-cols-2 xl:grid-cols-3' }, props.items.map((item) => {
        const prompt = recordPrompt(item)
        const rawUrl = recordResultUrl(item)
        const resultText = recordResultText(item)
        const isAudio = item.kind.includes('audio') || item.primary_asset?.kind === 'audio'
        const isVideo = item.kind === 'video' || item.primary_asset?.kind === 'video'
        const mediaPreview = rawUrl && (isAudio || isVideo)
          ? (isAudio
            ? h('audio', {
                key: rawUrl || `audio-${item.id}`,
                class: 'mt-3 w-full',
                src: rawUrl,
                controls: true,
                preload: 'auto',
              })
            : h('video', {
                key: rawUrl || `video-${item.id}`,
                class: 'mt-3 max-h-40 w-full rounded object-cover bg-black',
                src: rawUrl,
                controls: true,
                preload: 'metadata',
              }))
          : rawUrl
            ? h('img', { key: rawUrl, class: 'mt-3 h-40 w-full rounded object-cover bg-gray-50 dark:bg-dark-800', src: rawUrl, alt: taskKindLabel(item.kind) })
            : resultText
              ? h('pre', { class: 'mt-3 line-clamp-5 whitespace-pre-wrap text-sm leading-6 text-gray-600 dark:text-dark-300' }, resultText)
              : h('p', { class: 'mt-3 line-clamp-2 text-sm text-gray-600 dark:text-dark-300' }, prompt || '暂无可预览内容')
        return h('div', { class: 'rounded-lg border border-gray-200 p-4 dark:border-dark-700' }, [
          h('div', { class: 'flex items-start justify-between gap-3' }, [
            h('div', [
              h('div', { class: 'font-medium text-gray-950 dark:text-dark-50' }, taskKindLabel(item.kind)),
              h('div', { class: 'mt-1 text-xs text-gray-500' }, new Date(item.created_at).toLocaleString()),
            ]),
            h('span', { class: 'text-xs text-gray-500' }, item.status),
          ]),
          mediaPreview,
          h('div', { class: 'mt-4 flex flex-wrap gap-3 text-xs' }, [
            h('button', { class: 'text-primary-600', onClick: () => emit('restore', item) }, '恢复参数'),
            (rawUrl || resultText) ? h('button', { class: 'text-primary-600', onClick: () => emit('download', item) }, '下载') : null,
            h('button', { class: 'text-red-500', onClick: () => emit('remove', item.id) }, '删除'),
          ].filter(Boolean)),
        ])
      }))
    }
  },
})

onBeforeUnmount(stopRequest)
onMounted(async () => {
  appStore.setSidebarCollapsed(true)
  try {
    await loadKeyModeData()
    await loadCloudRecords()
    syncPlaygroundDebugState()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '加载 API Key 与可用模型失败。'
  }
})
</script>
