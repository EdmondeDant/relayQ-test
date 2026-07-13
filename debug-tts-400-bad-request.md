# [OPEN] tts-400-bad-request

## 症状

- AI 配音提交后返回 `请求失败：400 Bad Request`

## 当前假设

- 假设 1：不同配音模式的请求体字段不符合上游要求
- 假设 2：TTS 模型需要额外字段而不是当前 chat messages 结构
- 假设 3：data URL 音频输入不被上游接受
- 假设 4：后端 raw chat 转发时改坏了音频字段
- 假设 5：前端吞掉了响应体里的真实错误原因

## 当前阶段

- 已通过浏览器复现：MiMo TTS 返回 200，并成功返回 WAV Base64 音频
- 后续 `POST /api/v1/playground/tasks` 因携带约 51 KB 的内联音频返回 500
- 页面将记录保存失败误报为 AI 配音失败
- 已修复：内联音频不入库，记录保存失败不再覆盖已成功生成的音频，WAV MIME 正确标记
