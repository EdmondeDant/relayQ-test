# RelayQ 一键安装模型命名调整计划

## Summary

本计划只覆盖首页一键安装链路，不扩散到全站其他工具导出或历史配置模板。

一句最终标准：

> 客户在首页一键安装 `OpenClaw` 时，系统必须把与网站协议一致的正确 `baseURL`、客户当时明确选择的模型、以及对应的 `apikey`，完整且一致地写入安装后的 `OpenClaw`，使其安装完成后无需二次配置即可立即正常运行。

最终目标：

* 客户安装 `OpenClaw` 完成后，立刻就能使用

本轮只保证以下 3 件事全部正确：

1. `baseURL` 要对，并且和网站协议地址规则一致
2. 模型要和用户安装时选择的模型一致
3. `apikey` 要正确写入安装好的 `OpenClaw` 中，并被实际运行的 provider 正确读取

为达成上述目标，本计划的实现目标是：

* 把一键安装链路里的模型命名语义，从当前的 `openai/<model>` 调整为 `relayq/<model>`

* 同时保持 `baseURL` 规则单独正确：
  - OpenAI 兼容链路使用 `https://www.relayq.top/v1`
  - Anthropic 兼容链路使用 `https://www.relayq.top`

* 保持“模型必须来自客户安装时选择”的要求，禁止预设或偷偷回落到 `gpt-5.5` 等默认模型

用户已明确的决策：

* 模型命名采用 `relayq/*`

* 模型必须是客户在一键安装时选择的值，不能预先固定

* 改动范围仅限一键安装，不改全站其他导出

* 当前阶段先讨论和规划，不执行代码修改

## Current State Analysis

基于仓库现状，当前实现并不是 `relayq/*`，而是 OpenAI 兼容语义：

### 1. OpenClaw 安装链路当前固定为 `openai/*`

文件：`backend/internal/handler/install_script_handler.go`

* `OpenClaw` Windows onboarding 当前显式带有：

  * `--custom-provider-id openai`

  * `--custom-compatibility openai`

* `OpenClaw` Linux onboarding 同样如此

* 这意味着即使 `base_url` 指向 `https://www.relayq.top/v1`，当前写入到 OpenClaw 的 provider namespace 仍是 `openai/*`

对应位置：

* `backend/internal/handler/install_script_handler.go`

### 2. Hermes 当前不是 `relayq/*`，而是 `custom`

文件：`backend/internal/handler/install_script_handler.go`

* `Hermes` Linux 脚本写入：

  * `config set model.provider custom`

  * `config set model.base_url "$BASE_URL"`

  * `config set model.default "$MODEL_NAME"`

* 也就是说，Hermes 当前没有 `relayq/*` 这层 provider namespace

### 3. 前端首页一键安装当前已做到“模型必选”

文件：`frontend/src/views/HomeView.vue`

* 安装命令生成逻辑要求同时具备：

  * `apikey`

  * `installModel`

* 否则返回提示文案，不生成命令

* `/v1/models` 拉取成功后，也不会再自动选择第一个模型

这部分已经和“模型必须由客户选择”对齐，不需要重新做产品决策，只需要在后续实现中保持不回退。

### 4. 仓库其他导出逻辑仍普遍使用 `openai/*`

文件：`frontend/src/utils/toolConfigExport.ts`

* 当前 OpenAI 兼容导出统一生成：

  * `providerId: 'openai'`

  * `modelRef: openai/${modelName}`

但用户已经明确，本轮范围 **不改这些全站导出逻辑**。

### 5. 当前项目对不同协议根路径的处理规则已存在

文件：`frontend/src/utils/toolConfigExport.ts`

当前仓库的真实规则是：

* `OpenAI-compatible` 根路径使用 `${root}/v1`

* `Anthropic-compatible` 根路径使用 `${root}`

* `Gemini-compatible` 根路径使用 `${root}/v1beta`

也就是说，以你的站点为例：

* 站点根地址：`https://www.relayq.top`

* OpenAI 兼容地址：`https://www.relayq.top/v1`

* Anthropic 兼容地址：`https://www.relayq.top`

这与用户刚补充的产品判断一致，应在后续实现与文案中继续保持一致。

## Proposed Changes

本计划只处理首页一键安装链路相关文件。

### A. 调整 OpenClaw 一键安装的 provider namespace

文件：

* `backend/internal/handler/install_script_handler.go`

* `backend/internal/handler/install_script_handler_test.go`

改动方向：

* 把 `OpenClaw` Windows / Linux onboarding 中的：

  * `--custom-provider-id openai`

* 调整为：

  * `--custom-provider-id relayq`

保持不变的部分：

* `--custom-base-url` 仍使用当前 RelayQ 对外 `base_url`

* `--custom-model-id` 仍使用客户安装时选择的模型

* `--custom-compatibility openai` 仍保留

原因：

* `provider-id` 决定最终模型引用前缀

* `compatibility openai` 决定底层仍以 OpenAI 兼容协议对接 RelayQ 网关

* 这样可以实现“外层模型标识是 `relayq/<model>`，底层协议还是 OpenAI-compatible”

风险点：

* 之前修复过一次 `openai` provider 鉴权错配问题；改为 `relayq` 后，需要重新验证 OpenClaw 在 `provider-id=relayq` 时，认证是否正确落到该 provider 下

* 不能只改字符串，必须重新做一次隔离状态复现实验和真实安装脚本验证

### B. 保持并强化“模型必选”

文件：

* `frontend/src/views/HomeView.vue`

* `backend/internal/handler/install_script_handler.go`

* `backend/internal/handler/install_script_handler_test.go`

当前状态：

* 前端已做到不自动选默认模型

* OpenClaw / Hermes 脚本已要求显式传模型

计划要求：

* 实现 `relayq/*` 时不能回退这些保护

* 所有测试断言继续保证：

  * 不存在 `gpt-5.5` 或其他默认模型兜底

  * 无模型时不生成安装命令

  * 无模型时脚本直接报错

### C. 只调整首页一键安装相关展示语义

文件：

* `frontend/src/views/HomeView.vue`

改动方向：

* 如果首页安装区有任何直接展示模型前缀或 provider 语义的地方，需要改成 `relayq/*`

* 如果当前首页只展示纯模型名而不展示前缀，则不额外制造新显示项

原因：

* 用户要求只改一键安装范围

* 因此只处理首页安装区可见语义，不碰其他页面、导出面板或历史工具模板

### D. 暂不改动全站导出与其他客户端模板

明确不在本轮范围内的文件：

* `frontend/src/utils/toolConfigExport.ts`

* `frontend/src/views/user/StarterInstallView.vue`

* 其他 `UseKeyModal`、工具导入页、下载页等非首页一键安装场景

原因：

* 用户已明确“只改一键安装”

* 如果这些区域仍然保留 `openai/*`，属于后续潜在统一任务，不在本轮执行

## Assumptions & Decisions

### 已确认决策

* `https://www.relayq.top/v1` 的首页一键安装模型语义，应改为 `relayq/<model>`

* 模型值必须由客户安装时选择，不能预先固定为 `gpt-5.5`

- 验收只看三件事：

  * `baseURL` 是否和网站协议规则一致

  * 模型是否和用户选择一致

  * `apikey` 是否真正写入并被安装后的 `OpenClaw` 正确读取

### 关键实现假设

* `OpenClaw` 的 `--custom-provider-id` 可以安全改为 `relayq`

* 同时保留 `--custom-compatibility openai`，仍能使用 RelayQ 的 OpenAI-compatible `/v1` 接口

- `relayq/*` 主要是 provider namespace / 模型引用语义调整，不应改变用户本地已选模型值本身；真正需要重点防的是 auth store 与 provider id 是否仍然对齐

### 需要重点验证的未知点

* OpenClaw 在 `provider-id=relayq` 时，是否会再次出现与之前相似的 provider auth store 错位问题

* OpenClaw 启动日志中 `agent model` 最终是否会显示为 `relayq/<selected-model>`

* Hermes 是否需要仅在展示层使用 `relayq/<model>`，还是需要有更底层的 provider id 映射方案

## Verification Steps

实现时需要按以下顺序验证：

### 1. 静态验证

* 检查 `OpenClaw` 安装脚本已从 `--custom-provider-id openai` 改为 `relayq`

* 检查没有重新引入默认模型兜底

* 检查首页生成命令仍要求显式模型

### 2. 定向测试

运行与安装脚本相关的定向测试：

```powershell
go test ./internal/handler -run "Test(BuildOpenClawWindowsIncludesPowerShellGuidance|BuildOpenClawLinuxUsesOfficialOnboarding|BuildHermesWindowsIncludesWSLGuidance|BuildHermesLinuxStartsHermesImmediately)$"
```

必要时补充新断言：

* `OpenClaw` 脚本包含 `--custom-provider-id relayq`

* 不再包含旧的 `--custom-provider-id openai`

* 仍要求模型必选

### 3. 隔离复现实验

使用隔离 `OPENCLAW_STATE_DIR` 执行非交互 onboarding：

* 验证 `openclaw.json` 中：

  * `agents.defaults.model.primary = relayq/<selected-model>`

  * `models.providers.relayq` 存在

* 验证没有生成 `custom-*` 或错误回退到 `openai/*`

### 4. 运行态验证

重建并发布后，直接检查：

* `/install/openclaw-windows`

* `/install/openclaw-linux`

* 首页一键安装命令预览

* `/install/hermes-windows`

* `/install/hermes-linux`

重点确认：

* 无模型时命令不生成

* 有模型时命令携带客户所选模型

* OpenClaw 启动后日志中的模型前缀符合 `relayq/*`

### 5. 最终验收标准

满足以下条件才算完成：

* `baseURL` 正确：

  * OpenAI 兼容链路使用 `https://www.relayq.top/v1`

  * Anthropic 兼容链路使用 `https://www.relayq.top`

* 首页一键安装链路中，客户选择什么模型，就安装什么模型

* 不再预设或回退到 `gpt-5.5` 或其他默认模型

* OpenClaw 安装完成后最终模型语义为 `relayq/<selected-model>`

* 安装后的 `OpenClaw` 能从正确的 provider/auth store 中读到用户安装时传入的 `apikey`

* 启动后不再出现 provider auth 缺失类报错，例如：

  * `No API key found for provider ...`

* 本轮没有改动全站其他导出逻辑
