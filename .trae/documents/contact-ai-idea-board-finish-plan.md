# 联系页 AI 创业想法留言板收尾实施计划

## Summary
- 目标：在 `frontend/src/views/user/ContactSupportView.vue` 当前 `article` 和 `aside` 下方，交付一个风格统一、低资源占用的“AI 创业想法留言板”，满足“管理员可删除/回复，客户可发布/删除自己的留言”。
- 当前仓库状态不是“从零开始”，而是“主要代码骨架已经接入，但尚未完成完整验证闭环”。执行阶段应优先做编译、测试、诊断、联调修正，而不是重复重建整套模块。
- 最终交付标准：
  - 联系页可稳定展示留言板 section，视觉延续当前手账贴纸感。
  - 用户可分页查看、发布、删除自己的留言。
  - 管理员可删除任意留言，并内联回复或覆盖回复。
  - 后端接口有分页与限流保护，不引入实时推送、富文本、附件、楼中楼等高资源功能。
  - 不影响原有群二维码上传、发布、清空和联系方式复制能力。

## Current State Analysis

### 页面与前端现状
- `frontend/src/views/user/ContactSupportView.vue` 已经不是空白页面扩展点，而是已经包含完整的留言板 UI 骨架：
  - 新 section 标题与说明文案已存在。
  - 发布表单已存在：`ideaForm.title`、`ideaForm.content`、`submitIdeaMessage()`、`resetIdeaForm()`。
  - 列表与分页已存在：`ideaList`、`ideaPagination`、`loadIdeaMessages()`、`loadMoreIdeaMessages()`。
  - 筛选已存在：`ideaMineOnly`、`showAllIdeas()`、`showMineIdeas()`。
  - 管理员回复交互已存在：`replyDrafts`、`replyEditingId`、`saveAdminReply()`。
  - 页面已在 `onMounted()` 里并行加载联系页数据和留言列表。
- `frontend/src/api/ideaMessages.ts` 已存在用户侧 API 封装：
  - `list(page, pageSize, params)`
  - `create(request)`
  - `deleteIdeaMessage(id)`，并以 `delete` 字段导出。
- `frontend/src/api/admin/ideaMessages.ts` 已存在管理员回复 API：
  - `reply(id, request)`
- `frontend/src/types/index.ts` 已加入留言板相关前端类型：
  - `IdeaMessage`
  - `IdeaMessageListParams`
  - `CreateIdeaMessageRequest`
  - `AdminReplyIdeaMessageRequest`

### 后端现状
- 数据底座已经创建：
  - `backend/migrations/145_add_ai_idea_messages.sql`
  - `backend/ent/schema/ai_idea_message.go`
- 领域模型、服务、仓储、DTO、handler 已落位：
  - `backend/internal/service/ai_idea_message.go`
  - `backend/internal/service/ai_idea_message_service.go`
  - `backend/internal/repository/ai_idea_message_repo.go`
  - `backend/internal/handler/dto/ai_idea_message.go`
  - `backend/internal/handler/ai_idea_message_handler.go`
  - `backend/internal/handler/admin/ai_idea_message_handler.go`
- 路由已经接入真实路径：
  - 用户侧：`backend/internal/server/routes/user.go`
  - 管理侧：`backend/internal/server/routes/admin.go`
  - 路由透传 `redisClient`：`backend/internal/server/router.go`
- 当前设计已经符合低资源目标：
  - 单表扁平结构
  - 单条管理员回复
  - 分页加载
  - Redis 限流
  - 软删除
  - 无实时推送、无附件、无嵌套楼层

### 测试与生成现状
- `backend/internal/service/ai_idea_message_service_test.go` 已存在聚焦型单测，覆盖：
  - 删除权限
  - 管理员回复权限
  - `mine_only` + 分页
  - 作者名快照
- 上轮上下文已经确认：
  - Ent 生成已执行
  - Wire 生成已执行
- 当前最大的未闭环项不是设计不明确，而是：
  - 后端编译与测试尚未完整跑完
  - 前端模板/类型尚未做最终诊断
  - 尚未完成真实功能回归验证

## Assumptions & Decisions
- 决策：留言板只对已登录用户开放，不做匿名浏览与匿名发布。
- 决策：留言结构继续保持扁平模型，每条留言只允许 1 条管理员官方回复；不新增楼中楼。
- 决策：继续使用现有分页模式，默认每页 10 条，前端以“加载更多”方式追加，减少首屏负载。
- 决策：继续使用 Redis 限流保护写接口，不引入轮询加速、推送订阅、消息队列等额外运行成本。
- 决策：不新增独立 store、不新增独立管理页，继续使用联系页本地状态，降低前端常驻复杂度。
- 决策：当前执行阶段以“验证并修正现有实现”为主；只有在验证暴露缺口时，才对现有代码做最小必要改动。
- 决策：管理员删除能力沿用现有后端服务权限设计；如果前端显示策略与用户要求不一致，优先修正 UI 入口，使管理员在卡片上明确可删除。

## Proposed Changes

### 1. 后端完整性核验与最小修正

#### 核验文件
- `backend/internal/service/ai_idea_message.go`
- `backend/internal/service/ai_idea_message_service.go`
- `backend/internal/repository/ai_idea_message_repo.go`
- `backend/internal/handler/ai_idea_message_handler.go`
- `backend/internal/handler/admin/ai_idea_message_handler.go`
- `backend/internal/handler/dto/ai_idea_message.go`
- `backend/internal/handler/handler.go`
- `backend/internal/handler/wire.go`
- `backend/internal/service/wire.go`
- `backend/internal/repository/wire.go`
- `backend/internal/server/routes/user.go`
- `backend/internal/server/routes/admin.go`
- `backend/internal/server/router.go`

#### 执行内容
- 先用编译和单测确认以下点是否全部成立：
  - Ent 生成后的实体命名与 repository 使用一致。
  - Wire provider 签名与 `Handlers` 结构体字段一致。
  - 路由新增的 `redisClient` 透传没有破坏既有注册链路。
  - handler 中的分页、DTO 映射、权限上下文读取都能成功编译。
- 只在验证失败时做最小修正：
  - 修正包名/字段名/方法签名不匹配。
  - 修正分页结果映射或错误返回格式。
  - 修正删除/回复权限判定和状态写回细节。

#### 为什么这样做
- 当前模块大部分代码已在仓库中，重复重写风险高，且容易覆盖已经正确的实现。
- 最省资源、最稳妥的推进方式是先编译暴露真实问题，再按报错最小修复。

### 2. 前端类型、API 与页面交互核验

#### 核验文件
- `frontend/src/types/index.ts`
- `frontend/src/api/index.ts`
- `frontend/src/api/ideaMessages.ts`
- `frontend/src/api/admin/index.ts`
- `frontend/src/api/admin/ideaMessages.ts`
- `frontend/src/views/user/ContactSupportView.vue`

#### 执行内容
- 检查以下约定是否保持一致：
  - `IdeaMessage` 字段与后端 DTO 一致。
  - 用户 API 与管理 API 的返回结构能被当前页面直接消费。
  - 页面内调用 `ideaMessagesAPI.delete(message.id)` 与 API barrel export 没有命名冲突。
  - `adminAPI.ideaMessages.reply(...)` 在 `frontend/src/api/admin/index.ts` 已正确注册。
  - `loadIdeaMessages()`、`submitIdeaMessage()`、`deleteIdeaMessage()`、`saveAdminReply()` 在空态、翻页、切换“我的留言”时不会产生本地状态错乱。
- 只在核验发现问题时调整：
  - 类型字段名
  - API barrel export
  - 模板条件判断
  - 分页 total/page/pages 的本地更新逻辑

#### 重点关注
- 用户要求“管理员有权删除或者回复，客户只能选择发布和删除”。当前模板里删除按钮取决于 `message.can_delete`，需要在验证时特别确认管理员视角下每条卡片都能看到删除入口。
- 当前发布成功后直接把新留言插入 `ideaList` 并递增 `total`，需要验证在“我的留言”与“全部灵感”两个视角下都不产生重复项。

### 3. 联系页视觉与交互统一性收尾

#### 核验文件
- `frontend/src/views/user/ContactSupportView.vue`

#### 执行内容
- 逐项确认留言板 section 与当前联系页三块主内容的视觉语言一致：
  - 大圆角卡片
  - 贴纸标签
  - 渐变背景
  - 深浅色模式类名
- 检查留言板没有压缩、干扰原有二维码区和联系区布局。
- 如果需要微调，仅做统一性收尾：
  - 文案顺滑化
  - 间距/边框/按钮态统一
  - 空态和加载态与既有页面语气一致

#### 为什么这样做
- 用户明确要求“风格统一”，但当前仓库已经完成一轮联系页俏皮化改造，因此新增内容应融入既有风格，而不是另起一套后台式评论区 UI。

### 4. 验证与回归

#### 后端验证
- 运行留言板 service 单测，确认：
  - 普通用户不能删他人
  - 普通用户能删自己
  - 只有管理员能回复
  - `mine_only` 与分页可用
  - 作者名快照正确
- 运行后端编译，确认新增模块与 Wire/Ent 生成代码兼容。

#### 前端验证
- 对 `frontend/src/views/user/ContactSupportView.vue` 做诊断，确认：
  - 模板无类型错误
  - API 调用与响应类型匹配
  - 新增 section 没有引入明显语法或响应式错误

#### 联调回归
- 登录普通用户后验证：
  - 能加载留言列表
  - 能发布留言
  - 只能删除自己的留言
- 登录管理员后验证：
  - 能看到删除入口
  - 能回复或覆盖回复
  - 删除后列表即时更新
- 最后回归原联系页功能：
  - 群二维码预览与发布逻辑仍正常
  - 联系方式复制按钮仍正常

## Verification Steps
- 数据层：
  - `backend/migrations/145_add_ai_idea_messages.sql` 可正常应用，表与索引存在。
  - `backend/ent/schema/ai_idea_message.go` 生成代码后可被 repository 正常引用。
- 后端：
  - `GET /api/v1/idea-messages` 返回分页列表，`mine_only` 生效。
  - `POST /api/v1/idea-messages` 正常创建并返回 DTO。
  - `DELETE /api/v1/idea-messages/:id` 正确执行作者/管理员权限控制。
  - `PUT /api/v1/admin/idea-messages/:id/reply` 仅管理员可调用。
  - 写接口超限时返回限流结果。
- 前端：
  - 联系页展示新的留言板 section，样式与当前联系页统一。
  - 发布、删除、管理员回复都能直接更新当前列表，不需要整页刷新。
  - “全部灵感 / 我的留言 / 加载更多” 三种状态切换正常。
- 回归：
  - 原 `loadContactPage()`、`publishQrCode()`、`clearQrCode()`、`copyContact()` 行为保持可用。

## 执行顺序
1. 先跑后端单测、后端编译、前端诊断，收集真实报错。
2. 按报错最小修正后端接线与前端类型/模板问题。
3. 再做页面功能回归与管理员/普通用户权限验证。
4. 最后检查留言板风格统一性与原联系页功能无回归。
