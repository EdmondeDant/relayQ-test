# 首页命令展示断联计划

## Summary

- 目标：将 `frontend/src/views/HomeView.vue` 顶部英雄区中选中的两块命令展示改为固定占位命令，不再跟随下面安装实验室里的 OpenClaw/Hermes 一键安装命令联动。
- 范围：仅修改上方两个展示块的显示数据来源；保留下面安装区的真实命令生成、预览展示、复制命令、模型联动和脚本链接行为不变。

## Current State Analysis

- 文件：`frontend/src/views/HomeView.vue`
- 当前模板绑定：
  - 顶部终端行使用 `heroCommandPreview`，位置在英雄区命令行展示处。
  - 顶部右侧命令卡片 `code` 使用 `generatedInstallCommand`。
  - 下面安装实验室命令区也使用 `generatedInstallCommand`，复制按钮 `copyInstallCommand()` 同样复制该值。
- 当前数据关系：
  - `heroCommandPreview` 由 `generatedInstallCommand` 截断得到。
  - `generatedInstallCommand` 会根据 `installToken`、`installModel`、`selectedInstaller`、`selectedInstallerOs` 动态生成 OpenClaw/Hermes 的真实安装命令。
  - 因为顶部和底部共享同一个计算属性，所以顶部两个展示块会和下面一键安装命令完全联动。

## Proposed Changes

### 1. `frontend/src/views/HomeView.vue`

- 新增一个专用于顶部英雄区的固定命令常量或独立计算属性，例如：
  - 一个用于 `span.hero-command-inline` 的短命令预览文本。
  - 一个用于顶部 `hero-command-card-pre > code` 的完整占位命令文本。
- 将顶部模板绑定改为使用新的固定展示值，而不是继续引用：
  - `heroCommandPreview`
  - `generatedInstallCommand`
- 保持以下区域继续使用现有真实联动逻辑，不做行为变更：
  - `install-command-pre > code`
  - `copyInstallCommand()`
  - `generatedInstallCommand`
  - `fetchAvailableModels()`
  - `currentInstallScriptUrl` 及相关脚本入口展示

## What / Why / How

- What：
  - 把顶部两块“视觉展示命令”从真实安装命令链路中拆出来。
- Why：
  - 满足“随便写点什么命令行，但不要跟下面一键安装 OpenClaw 的命令联动”的需求。
  - 避免顶部内容因 token、model、installer、os 切换而变化。
- How：
  - 维持底部安装实验室现有真实命令逻辑不动。
  - 顶部单独引入静态命令字符串，作为纯展示文案使用。
  - 如顶部短行需要省略效果，可继续保留现有样式，直接提供较短文本，避免再依赖截断逻辑。

## Assumptions & Decisions

- 已确认用户选择“只改上面两块”，因此不改下面安装区的命令预览和复制逻辑。
- 顶部两块的命令内容无需真实可执行，只需看起来像命令行即可。
- 顶部展示块和底部安装区继续共存在同一组件内，本次不做组件拆分。
- 若 `heroCommandPreview` 仅服务于顶部展示，实施时可以：
  - 直接重定义为固定展示值；或
  - 新增更明确的顶部专用字段并替换模板引用。
  - 优先选择影响面更小、可读性更高的方案。

## Verification Steps

1. 打开首页，确认顶部终端行中的 `span.hero-command-inline` 显示固定命令文本。
2. 确认顶部右侧命令卡片中的 `code` 显示固定命令文本。
3. 在安装实验室中切换：
   - OpenClaw / Hermes
   - Windows / Linux / macOS
   - token / model 输入值
4. 验证顶部两块内容不发生变化。
5. 验证底部 `install-command-pre > code` 仍随 token / model / installer / os 正常变化。
6. 点击“复制命令”，确认复制的仍是底部真实安装命令，而不是顶部占位命令。
7. 运行前端类型检查或现有开发校验，确认没有引入模板或脚本错误。
