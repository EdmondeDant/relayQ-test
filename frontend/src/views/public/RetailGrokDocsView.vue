<template>
  <div class="min-h-screen bg-gray-50 px-6 py-10 dark:bg-dark-950">
    <div class="mx-auto max-w-5xl space-y-6">
      <div class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
        <div class="flex items-center justify-between gap-4">
          <div>
            <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">Grok 零售接口说明</h1>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
              零售 Key 独立走 <code>/retail/v1</code>，当前开放 Grok 4.3 推理、多模态识图、图片生成、图片编辑和视频生成。
            </p>
          </div>
          <RouterLink to="/retail/grok/key-usage" class="btn btn-secondary">返回查询页</RouterLink>
        </div>
      </div>

      <section class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">公共约定</h2>
        <ul class="mt-3 space-y-2 text-sm text-gray-600 dark:text-dark-300">
          <li>接口协议：兼容 OpenAI API 调用格式，请把 OpenAI SDK 或客户端的 <code>base_url</code> 设置为 <code>https://www.reayq.top/retail/v1</code>。</li>
          <li>鉴权方式：<code>Authorization: Bearer YOUR_RETAIL_GROK_KEY</code>。零售 Key 只能调用 <code>/retail/v1/*</code>，不能调用普通 <code>/v1/*</code>。</li>
          <li>文本/多模态接口：<code>POST https://www.reayq.top/retail/v1/chat/completions</code></li>
          <li>文生图接口：<code>POST https://www.reayq.top/retail/v1/images/generations</code></li>
          <li>图生图/图片编辑接口：<code>POST https://www.reayq.top/retail/v1/images/edits</code></li>
          <li>文生视频/图生视频接口：<code>POST https://www.reayq.top/retail/v1/videos/generations</code></li>
          <li>视频结果查询接口：<code>GET https://www.reayq.top/retail/v1/videos/REQUEST_ID</code></li>
          <li>支持的推理模型：<code>grok-4.3</code>，支持文本和图片理解。</li>
          <li>支持的图片模型：<code>grok-imagine-image</code>、<code>grok-imagine-image-quality</code>。</li>
          <li>支持的视频模型：<code>grok-imagine-video</code>。</li>
          <li>图片输入支持公网 URL 或 <code>data:image/jpeg;base64,...</code> / <code>data:image/png;base64,...</code>。</li>
        </ul>
      </section>

      <section class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Grok 4.3 多模态识图</h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          用 <code>messages[].content</code> 数组传入文本和图片。<code>detail</code> 可传 <code>low</code>、<code>high</code> 或 <code>auto</code>。
        </p>
        <pre class="mt-4 overflow-x-auto rounded-xl bg-gray-900 p-4 text-xs text-gray-100"><code>curl https://www.reayq.top/retail/v1/chat/completions \
  -H "Authorization: Bearer YOUR_RETAIL_GROK_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-4.3",
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "input_image",
            "image_url": "https://example.com/product.png",
            "detail": "high"
          },
          {
            "type": "input_text",
            "text": "识别图片里的商品，并给出三条卖点。"
          }
        ]
      }
    ],
    "stream": false
  }'</code></pre>
      </section>

      <section class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">文生图</h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          官方字段包含 <code>prompt</code>、<code>n</code>、<code>aspect_ratio</code>、<code>resolution</code>、<code>response_format</code>。清晰度通过模型和 <code>resolution</code> 控制。
        </p>
        <ul class="mt-3 space-y-2 text-sm text-gray-600 dark:text-dark-300">
          <li><code>aspect_ratio</code>：<code>1:1</code>、<code>16:9</code>、<code>9:16</code>、<code>4:3</code>、<code>3:4</code>、<code>3:2</code>、<code>2:3</code>、<code>2:1</code>、<code>1:2</code>、<code>auto</code> 等。</li>
          <li><code>resolution</code>：<code>1k</code> 或 <code>2k</code>。</li>
          <li><code>response_format</code>：<code>url</code> 或 <code>b64_json</code>。</li>
          <li><code>n</code>：一次生成图片数量，官方上限为 10。</li>
        </ul>
        <pre class="mt-4 overflow-x-auto rounded-xl bg-gray-900 p-4 text-xs text-gray-100"><code>curl https://www.reayq.top/retail/v1/images/generations \
  -H "Authorization: Bearer YOUR_RETAIL_GROK_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-image-quality",
    "prompt": "一张高端电商主图，白色背景，金属质感无线耳机，真实摄影光线",
    "n": 2,
    "aspect_ratio": "1:1",
    "resolution": "2k",
    "response_format": "url"
  }'</code></pre>
      </section>

      <section class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">图生图 / 图片编辑</h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          单图编辑使用 <code>image</code>，多参考图使用 <code>images</code>。多图编辑最多 3 张参考图，可在提示词里引用 <code>&lt;IMAGE_0&gt;</code>、<code>&lt;IMAGE_1&gt;</code>。
        </p>
        <pre class="mt-4 overflow-x-auto rounded-xl bg-gray-900 p-4 text-xs text-gray-100"><code>curl https://www.reayq.top/retail/v1/images/edits \
  -H "Authorization: Bearer YOUR_RETAIL_GROK_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-image-quality",
    "prompt": "把这张商品图改成黑金高端风格，保持主体结构不变。",
    "image": {
      "url": "https://example.com/source-product.png"
    },
    "aspect_ratio": "1:1",
    "resolution": "2k",
    "response_format": "url"
  }'</code></pre>

        <pre class="mt-4 overflow-x-auto rounded-xl bg-gray-900 p-4 text-xs text-gray-100"><code>curl https://www.reayq.top/retail/v1/images/edits \
  -H "Authorization: Bearer YOUR_RETAIL_GROK_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-image-quality",
    "prompt": "使用 &lt;IMAGE_0&gt; 的商品主体和 &lt;IMAGE_1&gt; 的背景风格，生成一张电商海报。",
    "images": [
      { "url": "https://example.com/product.png" },
      { "url": "https://example.com/background-style.png" }
    ],
    "aspect_ratio": "4:3",
    "resolution": "2k",
    "response_format": "url"
  }'</code></pre>
      </section>

      <section class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">文生视频</h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          视频生成是异步接口，提交后返回 <code>request_id</code>，再用轮询接口查询结果。
        </p>
        <ul class="mt-3 space-y-2 text-sm text-gray-600 dark:text-dark-300">
          <li><code>duration</code>：视频秒数，官方范围 <code>1-15</code>，默认 <code>8</code>。</li>
          <li><code>aspect_ratio</code>：<code>1:1</code>、<code>16:9</code>、<code>9:16</code>、<code>4:3</code>、<code>3:4</code>、<code>3:2</code>、<code>2:3</code>。</li>
          <li><code>resolution</code>：<code>480p</code>、<code>720p</code>、<code>1080p</code>。</li>
        </ul>
        <pre class="mt-4 overflow-x-auto rounded-xl bg-gray-900 p-4 text-xs text-gray-100"><code>curl https://www.reayq.top/retail/v1/videos/generations \
  -H "Authorization: Bearer YOUR_RETAIL_GROK_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-video",
    "prompt": "一辆银色跑车在雨夜城市街道缓慢驶过，电影感灯光，真实摄影风格",
    "duration": 10,
    "aspect_ratio": "16:9",
    "resolution": "720p"
  }'</code></pre>

        <pre class="mt-4 overflow-x-auto rounded-xl bg-gray-900 p-4 text-xs text-gray-100"><code>curl https://www.reayq.top/retail/v1/videos/REQUEST_ID \
  -H "Authorization: Bearer YOUR_RETAIL_GROK_KEY"</code></pre>
      </section>

      <section class="rounded-2xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">图生视频 / 参考图生成视频</h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          在视频生成接口中传 <code>image</code> 可做图生视频，传 <code>reference_images</code> 可用参考图控制风格或内容。
        </p>
        <pre class="mt-4 overflow-x-auto rounded-xl bg-gray-900 p-4 text-xs text-gray-100"><code>curl https://www.reayq.top/retail/v1/videos/generations \
  -H "Authorization: Bearer YOUR_RETAIL_GROK_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-video",
    "prompt": "让商品缓慢旋转，背景保持干净，适合电商展示。",
    "image": {
      "url": "https://example.com/product.png"
    },
    "duration": 6,
    "aspect_ratio": "1:1",
    "resolution": "720p"
  }'</code></pre>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>
