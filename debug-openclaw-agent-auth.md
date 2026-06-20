# Debug Session: openclaw-agent-auth

- Status: OPEN
- Started: 2026-06-17
- Goal: 确认 `OpenClaw` 一键安装后，为什么内嵌 agent 仍提示 `No API key found for provider "openai"`，并将链路修到可确定复现、可确定验证。

## Symptoms

- 安装后可以启动 `gateway` / `dashboard`
- 但执行真实任务时失败
- 关键报错：
  - `ProviderAuthError: No API key found for provider "openai"`
  - auth store 指向 `C:\Users\Administritor\.openclaw\agents\main\agent\openclaw-agent.sqlite`

## Hypotheses

1. `openclaw onboard --non-interactive ... --custom-api-key ...` 只写入了主配置，没有把 provider auth 写入实际工作 agent 的 auth store。
2. onboarding 已写入 auth，但后续 `install-daemon` / `dashboard` 启动了另一个 agent 上下文，读取的是不同路径下的 sqlite。
3. 当前 `OpenClaw` 版本对 `custom-compatibility openai` 的非交互 onboarding 行为有变化，命令执行成功但没有持久化 agent 认证。
4. Windows 一键安装脚本传入的 `BaseUrl` / `ModelName` / `ApiKey` 虽然通过了参数校验，但某个字段被清洗后为空或不符合 onboarding 的持久化条件。
5. 需要额外执行官方的 agent/auth 初始化命令，单纯 `onboard --install-daemon` 不能保证 embedded agent 可立即工作。

## Evidence Plan

- 先查官方 CLI 是否存在独立的 agent/auth 写入命令或 profile 复制命令。
- 再用最小插桩记录安装脚本实际执行的命令、返回码和关键配置路径。
- 复现一次失败链路，对比 `pre-fix` 与 `post-fix` 日志。

## Progress Log

- 已创建调试会话文件，等待补充官方命令证据与最小插桩方案。
- 已启动本地 Debug Server：`http://127.0.0.1:7777`
- 已对 `backend/internal/handler/install_script_handler.go` 增加最小采样点，仅记录安装输入、onboarding 输出和 gateway 状态，不改变业务逻辑。

## Evidence

### 官方文档证据

- `openclaw onboard` 的自定义 provider 官方自动化示例包含 `--custom-provider-id "my-custom"`，说明 provider id 是 onboarding 的显式输入之一。
- 官方文档说明 agent / auth profile 是按 agent 维度存储，不是单纯全局配置。

### Pre-fix 复现实验

在隔离状态目录中，执行与当前项目脚本等价的 `OpenClaw 2026.6.8` onboarding 命令，但 **不带** `--custom-provider-id`：

```powershell
npx --yes openclaw@2026.6.8 onboard --non-interactive --mode local --auth-choice custom-api-key --custom-base-url 'https://example.com/v1' --custom-model-id 'gpt-5.5' --custom-api-key 'sk-test' --secret-input-mode plaintext --custom-compatibility openai ...
```

结果写出的 `openclaw.json` 关键字段：

- `agents.defaults.model.primary = custom-example-com/gpt-5.5`
- `models.providers.custom-example-com.apiKey = sk-test`

这说明当前脚本并没有把 provider 固定到 `openai`，而是让 OpenClaw 根据域名自动生成了 `custom-example-com`。

### Post-fix 复现实验

在同样的隔离状态目录实验中，仅额外加入：

```powershell
--custom-provider-id openai
```

结果写出的 `openclaw.json` 关键字段变为：

- `agents.defaults.model.primary = openai/gpt-5.5`
- `models.providers.openai.apiKey = sk-test`

这与用户故障日志中的请求模型标识完全对齐：

- `requested=openai/gpt-5.5`
- `No API key found for provider "openai"`

## Hypothesis Status

1. `onboard` 没把 auth 写到实际工作 agent store：证据不足，暂不作为主根因。
2. daemon / dashboard 切换了别的 agent 上下文：未发现支持证据。
3. 非交互 onboarding 在新版中不再正确持久化 auth：主因不成立，更像是 provider id 对不上。
4. 输入字段被清洗为空导致 auth 丢失：不成立，隔离复现中 `apiKey` 已正常写入配置。
5. 需要额外 agent/auth 初始化命令：目前不是主因，主因已收敛为 provider id 错配。

## Root Cause

当前项目生成的 `OpenClaw` 安装脚本在自定义 OpenAI 兼容网关 onboarding 时，没有显式传入 `--custom-provider-id openai`。

因此新版 `OpenClaw` 会按域名自动生成 provider id，例如：

- `custom-example-com`

而运行时真实请求的模型 provider 却是：

- `openai`

最终导致：

- 配置里有 API key
- 但 `openai` provider 下没有对应 auth
- embedded agent 报 `No API key found for provider "openai"`

## Fix

- Windows `OpenClaw` 安装脚本 onboarding 命令增加：`--custom-provider-id openai`
- Linux `OpenClaw` 安装脚本 onboarding 命令增加：`--custom-provider-id openai`
- 对应测试已新增断言，防止回归

## Verification

- 定向测试已通过：

```text
go test ./internal/handler -run "Test(BuildOpenClawWindowsIncludesPowerShellGuidance|BuildOpenClawLinuxUsesOfficialOnboarding|NormalizeOpenAIBaseURL)$"
ok github.com/Wei-Shaw/sub2api/internal/handler
```
