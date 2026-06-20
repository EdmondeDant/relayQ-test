# 联系页 AI 创业想法留言板计划

## Summary
- 目标：在当前联系页 `article`、`aside` 下方新增一个“AI 创业想法留言板”，风格与现有联系页的手账贴纸风保持一致。
- 权限规则：
  - 所有已登录客户都能看到全部已发布留言。
  - 客户只能发布留言、删除自己的留言。
  - 管理员可以删除任意留言，并对每条留言写一条官方回复。
- 设计方向：做成“公开留言墙 + 单条官方回复”的轻量内容模块，不做楼中楼，不做附件，不做实时推送，优先保证低资源占用、低复杂度、易维护。

## Current State Analysis

### 1. 当前页面结构
- 目标页面文件：`frontend/src/views/user/ContactSupportView.vue`
- 当前页面已有三块：
  - 顶部欢迎 `section`
  - 群二维码展示 `article`
  - 联系方式与管理员发布区 `aside`
- 当前联系页已经具备：
  - `isAdmin` 权限判断
  - `copyContact()` 成功/失败提示
  - `loadContactPage()` 页面初始化与公共设置拉取
  - 管理员二维码上传/发布/清空链路
- 结论：
  - 这个页面非常适合继续扩展一个新的完整 `section` 作为留言板主容器。
  - 不需要单独新建一个页面，直接在联系页下方追加即可满足“统一风格”和“功能聚合”。

### 2. 当前仓库可复用的后端模式
- 认证路由分层已存在：
  - 用户认证路由：`backend/internal/server/routes/user.go`
  - 管理员认证路由：`backend/internal/server/routes/admin.go`
- 分页返回标准已存在：
  - `backend/internal/pkg/response/response.go`
  - `frontend/src/types/index.ts` 中的 `BasePaginationResponse<T>`
- 管理端 CRUD 模式已存在，公告模块可直接作为参考：
  - 用户侧 handler：`backend/internal/handler/announcement_handler.go`
  - 管理侧 handler：`backend/internal/handler/admin/announcement_handler.go`
  - 前端管理 API：`frontend/src/api/admin/announcements.ts`
  - 前端管理视图：`frontend/src/views/admin/AnnouncementsView.vue`
- Redis 限流中间件已存在，可直接套用：
  - `backend/internal/middleware/rate_limiter.go`
  - `backend/internal/server/routes/auth.go` 展示了 `LimitWithOptions()` 的典型用法

### 3. 外部调研结论（用于指导低资源实现）
- 扁平评论/留言结构比楼中楼更容易分页和维护，尤其适合“顶级留言 + 单条官方回复”这种轻量场景。
- 深层回复线程虽然功能更全，但在前端树构建、分页、权限控制和数据更新上都会明显加重复杂度。
- 限流、分页和轻量返回是保护服务器资源的关键做法。
- 因此本方案采用：
  - 扁平留言表
  - 单条管理员官方回复
  - 分页列表
  - Redis 限流
  - 无附件 / 无实时推送 / 无富文本 Markdown 渲染

## Proposed Changes

### 1. 新增轻量留言数据模型
- 新增迁移文件：
  - `backend/migrations/145_add_ai_idea_messages.sql`
- 新增 Ent schema：
  - `backend/ent/schema/ai_idea_message.go`
- 生成并更新 Ent 相关文件：
  - `backend/ent/**`
  - `backend/cmd/server/wire_gen.go`（如代码生成链路需要）

#### 数据结构决策
- 表名建议：`ai_idea_messages`
- 字段设计：
  - `id`
  - `author_id`
  - `author_name`
  - `title`
  - `content`
  - `admin_reply`
  - `admin_reply_by`
  - `admin_reply_at`
  - `status`
  - `created_at`
  - `updated_at`
  - `deleted_at`
- `status` 枚举：
  - `active`
  - `user_deleted`
  - `admin_deleted`

#### 为什么这么设计
- 使用单表即可承载需求，不需要父子评论树。
- `author_name` 做快照，减少列表查询时的用户关联开销。
- 单条 `admin_reply` 字段直接解决“管理员回复”需求，不引入第二张回复表。
- 使用软删除状态而不是硬删除，便于保留审计与误删恢复空间，同时对前台直接隐藏删除内容。

#### 索引建议
- `(status, created_at desc)`
- `(author_id, created_at desc)`
- 可选：`(admin_reply_at desc)`，仅在后续需要按回复时间筛选时启用

### 2. 后端新增留言板领域、仓储、服务与接口
- 新增领域/服务/仓储文件：
  - `backend/internal/service/ai_idea_message.go`
  - `backend/internal/service/ai_idea_message_service.go`
  - `backend/internal/repository/ai_idea_message_repo.go`
  - `backend/internal/handler/dto/ai_idea_message.go`
  - `backend/internal/handler/ai_idea_message_handler.go`
  - `backend/internal/handler/admin/ai_idea_message_handler.go`
- 注入与装配更新：
  - `backend/internal/service/wire.go`
  - `backend/internal/handler/wire.go`
  - `backend/cmd/server/wire_gen.go`

#### 用户侧接口
- 在 `backend/internal/server/routes/user.go` 中新增一组认证路由，例如：
  - `GET /api/v1/idea-messages`
  - `POST /api/v1/idea-messages`
  - `DELETE /api/v1/idea-messages/:id`

#### 管理员侧接口
- 在 `backend/internal/server/routes/admin.go` 中新增管理员动作路由，例如：
  - `PUT /api/v1/admin/idea-messages/:id/reply`

#### 接口行为定义
- `GET /idea-messages`
  - 返回公开可见的 `active` 留言
  - 支持 `page`、`page_size`
  - 支持 `mine_only=1` 过滤自己的留言
  - 按 `created_at desc` 排序
- `POST /idea-messages`
  - 登录客户可发
  - 参数：`title`、`content`
  - 返回新建留言
- `DELETE /idea-messages/:id`
  - 客户只能删自己的留言
  - 管理员也可删任意留言
  - 实际执行软删除
- `PUT /admin/idea-messages/:id/reply`
  - 仅管理员
  - 对指定留言写入或覆盖单条官方回复

#### 返回 DTO 决策
- 列表项统一返回：
  - `id`
  - `author_id`
  - `author_name`
  - `title`
  - `content`
  - `admin_reply`
  - `admin_reply_at`
  - `created_at`
  - `updated_at`
  - `is_mine`
  - `can_delete`
  - `can_reply`
- 这样前端不需要重复推导复杂权限，直接按返回能力展示按钮。

#### 校验与限制
- 字段长度限制：
  - `title`：1-120
  - `content`：1-2000
  - `admin_reply`：1-1000
- 删除后的留言不再出现在普通列表中。
- 管理员回复仅保留一条，后续回复视为覆盖更新。

### 3. 接入限流与低资源保护
- 复用现有 Redis 限流中间件：
  - `backend/internal/middleware/rate_limiter.go`
- 在用户/管理员路由注册层直接加限流：
  - 发留言：每用户/IP 每分钟 5 次
  - 删除留言：每分钟 20 次
  - 管理员回复：每分钟 20 次

#### 低资源策略明确化
- 不做 WebSocket / SSE / 长轮询
- 不做图片上传、附件、富文本、Markdown 解析
- 不做多级回复树
- 列表分页默认 10 条，最大 30 条
- 前端仅按需加载下一页，不预取全部留言
- 前端操作后优先局部更新当前列表，避免每次全量刷新

### 4. 前端新增留言板 API 与类型
- 新增前端 API 文件：
  - `frontend/src/api/ideaMessages.ts`
  - `frontend/src/api/admin/ideaMessages.ts`
- 更新导出：
  - `frontend/src/api/index.ts`
  - `frontend/src/api/admin/index.ts`
- 更新类型定义：
  - `frontend/src/types/index.ts`

#### 类型设计
- 新增类型：
  - `IdeaMessage`
  - `CreateIdeaMessageRequest`
  - `AdminReplyIdeaMessageRequest`
- 继续复用：
  - `BasePaginationResponse<T>`
  - `FetchOptions`

### 5. 在联系页底部新增统一风格的留言板 section
- 修改文件：
  - `frontend/src/views/user/ContactSupportView.vue`

#### 页面布局决策
- 在当前二维码区和联系区下方新增一个独立 `section`
- 区块建议结构：
  - 顶部：标题 + 说明 + 小标签
  - 左侧/上方：发布留言表单
  - 右侧/下方：留言列表

#### 表单功能
- 客户与管理员都可看到发布表单
- 字段：
  - `title`
  - `content`
- 交互：
  - 提交按钮
  - 提交中状态
  - 长度校验提示
  - 失败 toast / 错误文案

#### 列表功能
- 默认显示全部留言
- 增加轻量筛选：
  - `全部灵感`
  - `我的留言`
- 每条留言卡片展示：
  - 标题
  - 正文
  - 作者名
  - 发布时间
  - 官方回复（如有）
- 客户可见动作：
  - 自己的留言显示“删除”
- 管理员可见动作：
  - 所有留言显示“删除”
  - 所有留言显示“回复/修改回复”
  - 回复以单条内联编辑区展开，不弹复杂管理面板

#### 视觉风格
- 延续当前联系页的手账贴纸感：
  - 圆角卡片
  - 轻渐变背景
  - 彩色贴纸标签
  - 回复区用“官方便签”视觉区分
- 与现有 `section/article/aside` 统一色彩语言，不引入另一套后台风格

### 6. 前端状态流与请求策略
- 在 `ContactSupportView.vue` 中新增留言板本地状态：
  - 列表数据
  - 当前页/总页数
  - `mineOnly`
  - 发布表单状态
  - 管理员回复编辑态
- 页面初始化时：
  - 继续执行现有 `loadContactPage()`
  - 并行拉取第一页留言列表
- 新增操作函数：
  - `loadIdeaMessages()`
  - `submitIdeaMessage()`
  - `deleteIdeaMessage()`
  - `saveAdminReply()`
  - `loadMoreIdeaMessages()`

#### 资源控制细节
- 用户切换 `全部/我的` 时重置页码并重新请求第一页
- “加载更多”采用追加模式，不反复请求前页
- 回复与删除成功后，优先直接更新当前本地项；必要时再拉取当前页

### 7. 测试与回归
- 后端至少补以下测试：
  - 作者删除自己留言成功
  - 普通客户删除他人留言失败
  - 管理员删除任意留言成功
  - 管理员回复写入成功
  - 普通客户调用回复接口失败
  - 列表分页与 `mine_only` 生效
- 前端至少补以下验证：
  - 联系页能正常加载留言板
  - 普通客户发布成功后列表出现新留言
  - 普通客户只能看到自己留言的删除按钮
  - 管理员能看到回复和删除入口
  - 删除/回复不影响现有二维码发布功能

## Assumptions & Decisions
- 决策：留言板只对已登录客户开放，不做匿名发布。
- 决策：留言是公开墙，所有已登录客户可见全部 `active` 留言。
- 决策：管理员回复采用“单条官方回复位”，不做多层回复线程。
- 决策：客户删除仅限自己的留言；管理员可删任意留言。
- 决策：删除采用软删除，普通列表中隐藏已删内容。
- 决策：不做附件、图片、Markdown、实时推送，以降低服务器压力和滥用风险。
- 决策：第一页默认 10 条，按时间倒序排列。
- 决策：不新增独立后台页面，管理员直接在联系页留言板内完成回复和删除。

## Verification Steps
- 数据结构验证：
  - 迁移成功创建 `ai_idea_messages` 表
  - Ent schema 与生成代码可正常编译
- 接口验证：
  - 用户可分页获取留言列表
  - 用户可发布留言
  - 用户仅能删除自己的留言
  - 管理员可回复任意留言
  - 管理员可删除任意留言
  - 超出限流时返回 `429`
- 页面验证：
  - 联系页下方新增留言板 section
  - 风格与现有页面统一
  - 普通用户只看到“发布 / 删除自己的留言”
  - 管理员看到“回复 / 删除”能力
  - 回复展示为单条官方便签，不出现多层评论树
- 回归验证：
  - 当前二维码展示、上传、发布、清空逻辑不受影响
  - 联系方式区与复制按钮不受影响
- 性能验证：
  - 首屏仅加载第一页留言
  - 无实时订阅、无附件上传、无大 payload 返回
  - 多页加载时保持分页请求，不发生整表拉取

## 参考实现依据
- Flat comments / on-demand loading 思路参考：
  - [Design a Comments Thread](https://www.frontenddigest.com/system-design/design-comments-thread)
- API 分页与低资源优化参考：
  - [Optimizing API Performance with Rate Limiting, Pagination, and Compression](https://blog.easecloud.io/cloud-infrastructure/master-api-performance-optimization/)
- 限流策略与按端点分级限制参考：
  - [Rate Limiting Best Practices: API Protection and Abuse Prevention](https://checkyourvibe.dev/blog/best-practices/rate-limiting)
