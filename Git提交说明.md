# 2026-07-22 Git 提交说明

## 提交标题

`feat: 完善在线体验工作台并接入 MiMo 音频能力`

## 提交说明

- 重构在线体验工作台的 API Key + 动态模型选择链路，统一图片、视频、音频与文本工具的模型选择体验。
- 修复创作记录恢复后图片结果丢失的问题，支持按 `request_id` 回查云端作品补回结果图。
- 将水印工具升级为“水印处理”，补齐文字水印、Logo 水印与相关界面文案。
- 新增 MiMo 音频工作台能力，接入语音转写、标准配音、音色设计、声音克隆四类能力。
- 修正 MiMo 音频请求格式与任务保存链路，解决 AI 配音 `400 Bad Request` / 任务保存失败问题。
- 优化任务中心与云端作品的音频展示，支持播放器卡片、任务摘要和空态兜底。
- 修正 `/interface-docs` 音频章节，统一为当前可用的 `POST /v1/chat/completions` 接入格式。

## 可直接使用的提交正文

```text
feat: 完善在线体验工作台并接入 MiMo 音频能力

- 重构工作台 API Key 与动态模型选择链路
- 补齐多工具模型选择器并优化选择框中文占位
- 修复创作记录恢复后图片结果丢失问题
- 将水印工具升级为支持文字与 Logo 的水印处理
- 新增 MiMo 语音转写、标准配音、音色设计、声音克隆能力
- 修正 MiMo 音频请求格式、任务保存与播放器展示链路
- 修复任务中心持续加载问题并增强云端作品音频卡片展示
- 更新接口文档中的音频示例为 chat/completions 实际格式
```

## 拆分提交建议

### 提交一

`feat: 完善在线体验工作台与图片工具体验`

包含：

- API Key + 动态模型选择链路
- 多工具模型选择器补齐
- 创作记录恢复补图
- 水印处理能力升级
- Select 中文占位与交互优化

### 提交二

`feat: 接入 MiMo 音频工作台并修正文档`

包含：

- 语音转写与 AI 配音面板
- MiMo 请求格式修正
- 任务中心 / 云端作品音频展示
- AI 配音问题修复
- `/interface-docs` 音频章节更新

---

## 追加：7 月 14 日本轮变更建议

### 建议提交标题一

`fix: 修正工作台与后台控制台收尾问题`

包含：

- 修复 Ops Dashboard 中 `AbortController` 生命周期问题
- 清理后台控制台 3 条日志：`null.signal` 与连带 `ERR_ABORTED`
- 删除分组倍率展示徽标但保留执行逻辑
- 清理倍率 UI 删除后残留的未使用计算属性和变量
- 修正文案 `导入到 CCS` 为 `导入到 CC Switch`

### 建议提交标题二

`docs: 同步 Agent 文档与接口页面说明`

包含：

- 新增并补全 `relayq-agent-api-reference.md`
- 补齐 `gpt-image-2-adobe` JSON 请求格式
- 补齐 `grok-imagine-image*` JSON 请求格式
- 补齐 `grok-imagine-video` 异步提交 / 轮询 / 下载格式
- 将 `APIDocsView.vue` 页面正文与下载版文档完全对齐
- 删除 APIDocs 页头无意义的标签元素

### 可直接使用的提交正文一

```text
fix: 修正工作台与后台控制台收尾问题

- 修复 Ops Dashboard 请求中止时读取空 signal 的问题
- 清理后台控制台中的 snapshot-v2 / throughput-trend 连带中止日志
- 删除分组倍率的前端展示徽标但保留执行倍率逻辑
- 清理 GroupBadge 与 GroupOptionItem 中残留的未使用代码
- 将“导入到 CCS”文案修正为“导入到 CC Switch”
```

### 可直接使用的提交正文二

```text
docs: 同步 Agent 文档与接口页面说明

- 新增 relayq-agent-api-reference.md 详细文档
- 补齐 gpt-image-2-adobe 的文生图与图片编辑 JSON 示例
- 补齐 grok-imagine-image 的图片生成与编辑 JSON 示例
- 补齐 grok-imagine-video 的异步提交、轮询与 content 下载示例
- 将 APIDocs 页面正文与下载版 Agent 文档统一为同一口径
- 删除 APIDocs 页头无意义标签以简化展示
```

### 如果要合并成一个提交

建议标题：

`fix: 收尾工作台控制台问题并同步图片视频接口文档`

可直接使用的提交正文：

```text
fix: 收尾工作台控制台问题并同步图片视频接口文档

- 修复 Ops Dashboard 请求取消时读取空 signal 的问题
- 清理后台控制台中的 3 条相关错误日志
- 删除分组倍率展示徽标并清理残留未使用代码
- 将“导入到 CCS”文案改为“导入到 CC Switch”
- 新增 relayq-agent-api-reference.md 并补齐图片视频模型 JSON 接口格式
- 将 APIDocs 页面正文与下载版 Agent 文档统一口径
- 删除 APIDocs 页头多余标签元素
```

---

## 追加：7 月 14 日本轮变更建议（二）— 一键安装中转下载

### 建议提交标题三

`feat: 为一键安装接入站内中转下载与自动回退`

包含：

- 为 OpenClaw Windows 一键安装增加 Node.js / Git 自动检测与静默安装
- 为 Hermes Windows 一键安装增加 Node.js / Git 自动检测与静默安装
- 依赖下载优先走 RelayQ 站内 `/downloads/...` 中转地址
- 中转失败时自动回退官方下载源
- 为 Hermes 官方安装脚本和仓库快照增加站内中转入口
- 调整下载目录校验逻辑，允许 `SizeBytes = 0` 的资源进入缓存代理

### 可直接使用的提交正文三

```text
feat: 为一键安装接入站内中转下载与自动回退

- 为 OpenClaw Windows 安装脚本增加 Node.js 与 Git 自动安装
- 为 Hermes Windows 安装脚本增加 Node.js 与 Git 自动安装
- 让依赖下载优先走 RelayQ 站内 /downloads 中转地址
- 中转失败时自动回退到官方源，保持一键安装无需人工干预
- 为 Hermes install.ps1 与仓库 ZIP 快照增加站内下载入口
- 调整下载目录校验逻辑以支持未知体积资源的缓存代理
```

### 如果这一轮要与安装相关改动单独提交

建议标题：

`feat: 优化 OpenClaw 与 Hermes 一键安装的国内下载成功率`

可直接使用的提交正文：

```text
feat: 优化 OpenClaw 与 Hermes 一键安装的国内下载成功率

- 为 Windows 一键安装链路增加 Node.js 与 Git 的自动检查和静默安装
- 优先通过 RelayQ 自有 /downloads 中转 Node.js、Git 和 Hermes 依赖资源
- 当站内中转不可用时自动回退到官方下载源
- 为 Hermes 官方安装脚本和源码快照增加站内代理下载入口
- 继续保持用户侧一条命令自动安装，无需手工替换源或修改命令
```

---

## 追加：7 月 22 日本轮变更建议（三）— Playground 异步图片链路修复

### 建议提交标题四

`fix: 修复 Playground 异步图片任务的回环地址与状态回写`

包含：

- 将 Playground 图片能力统一收口到后端主导的异步方案 B
- 前端仅提交规范化业务参数，不再直接拼接上游 provider 请求体
- 修复异步 worker 内部仍回调 `127.0.0.1:8080` 的错误默认地址
- 修复图片资产相对 URL 解析仍写死 `8080` 的问题
- 修复 `playground_tasks` 状态更新 SQL 参数类型冲突导致任务长期停留 `pending`
- 补充 Playground 图片链路运行时调试埋点，便于继续排查真实错误

### 可直接使用的提交正文四

```text
fix: 修复 Playground 异步图片任务的回环地址与状态回写

- 将 Playground 图片任务切换为后端主导的异步方案 B
- 前端仅提交规范化业务参数，由后端按任务类型构造上游请求
- 修复异步 worker 内部错误回调到 127.0.0.1:8080 的问题
- 修复图片资产相对路径解析仍使用 8080 默认地址的问题
- 修复 playground_tasks 状态更新 SQL 参数冲突导致任务卡在 pending 的问题
- 补充图片链路运行时调试埋点并保留真实错误透传能力
```

### 如果本轮想和异步架构调整一起提交

建议标题：

`fix: 收敛 Playground 异步图片协议并修复任务状态卡 pending`

可直接使用的提交正文：

```text
fix: 收敛 Playground 异步图片协议并修复任务状态卡 pending

- 将 Playground 图片生成与图片编辑统一改为后端主导的异步任务方案
- 前端仅传 kind、model、prompt、media 等规范化业务参数
- 修复 worker 内部回环地址错误导致图片任务无法正常执行的问题
- 修复资产下载链路中相对 URL 仍写死 8080 的问题
- 修复 playground_tasks 更新 SQL 的参数类型冲突，恢复 running/succeeded 状态推进
- 保留后端真实 error_message 透传，避免前端继续显示无意义通用报错
```

### 如果本轮只想提交最小修复

建议标题：

`fix: 修复 Playground 图片任务内部回环地址错误`

可直接使用的提交正文：

```text
fix: 修复 Playground 图片任务内部回环地址错误

- 从当前请求推导 Playground 内部 base URL 并透传给异步 worker
- 修复 worker 内部请求仍默认访问 127.0.0.1:8080 的问题
- 修复图片资产相对路径解析时仍写死 8080 的问题
- 为图片任务补充定向测试，验证内部 base URL 归一化逻辑
```

---

## 追加：7 月 22 日本轮变更建议（四）— 修复 priority 模式下 cache write 计费偏低

### 建议提交标题五

`fix: 对齐 priority 模式下的 cache write 计费逻辑`

包含：

- 为计费模型补齐 `CacheCreationPricePerTokenPriority`
- 为动态价格解析补齐 `cache_creation_input_token_cost_priority`
- 修复 `priority` service tier 下缓存写入仍按普通单价计费的问题
- 让渠道 `CacheWritePrice` 覆盖同时作用于普通价和 `priority` 价
- 补充定向测试，验证动态价格、渠道覆写和 `priority cache write` 计费

### 可直接使用的提交正文五

```text
fix: 对齐 priority 模式下的 cache write 计费逻辑

- 为计费模型补齐 CacheCreationPricePerTokenPriority 字段
- 为动态价格解析补齐 cache_creation_input_token_cost_priority 映射
- 修复 priority service tier 下 cache write 仍按普通单价计费的问题
- 让渠道 CacheWritePrice 同时覆盖普通价和 priority 价
- 补充定向测试验证 priority cache write 计费结果
```

### 如果本轮只想提交最小修复

建议标题：

`fix: 修复 priority 模式下 cache write 未按高优单价计费`

可直接使用的提交正文：

```text
fix: 修复 priority 模式下 cache write 未按高优单价计费

- 补齐 CacheCreationPricePerTokenPriority 字段与动态价格映射
- 修复 priority service tier 下 cache write 仍沿用普通单价的问题
- 让渠道 CacheWritePrice 覆盖同时作用于 priority 价
- 补充定向测试覆盖 priority cache write 计费
```

---

## 追加：7 月 22 日本轮变更建议（六）— 统一 clean 库运行环境并收口管理员账号

### 建议提交标题六

`fix: 统一 clean 库运行环境并收口非 clean 管理员`

包含：

- 停掉旧的 `3000` 实例并改为显式使用 `.localdata_clean` 配置启动后端
- 统一本地默认验证环境到 clean 库 `sub2api_relayq_clean_20260721`
- 将非 clean 库 `sub2api` 中的 `363164954@qq.com` 管理员退役
- 避免后续在 `admin@sub2api.local` 与 `363164954@qq.com` 两套管理员之间来回串库
- 实测验证 `admin@sub2api.local / admin123456` 在 clean 实例上登录成功

### 可直接使用的提交正文六

```text
fix: 统一 clean 库运行环境并收口非 clean 管理员

- 显式使用 .localdata_clean 配置启动后端并统一到 clean 库环境
- 停掉旧的 3000 实例，避免同端口下继续混用不同数据库配置
- 将 sub2api 中的 363164954@qq.com 管理员退役为 inactive 普通用户
- 保留 clean 库中的 admin@sub2api.local 作为唯一默认管理员
- 实测验证 admin@sub2api.local / admin123456 可在 clean 实例上登录成功
```

### 如果本轮只想提交最小收口

建议标题：

`fix: 锁定 clean 库管理员并停用非 clean 管理员`

可直接使用的提交正文：

```text
fix: 锁定 clean 库管理员并停用非 clean 管理员

- 锁定本地后端运行配置到 .localdata_clean
- 停用 sub2api 中的 363164954@qq.com 管理员，避免继续串库
- 保留 admin@sub2api.local 作为 clean 库默认管理员
- 实测验证 clean 实例上的管理员登录链路恢复正常
```
