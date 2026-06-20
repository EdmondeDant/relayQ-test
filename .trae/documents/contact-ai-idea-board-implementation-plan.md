# 联系页 AI 创业想法留言板实施计划

## Summary
- 目标：在现有联系页 `frontend/src/views/user/ContactSupportView.vue` 的二维码区 `article` 和联系方式区 `aside` 下方新增一个“AI 创业想法留言板” section，保持当前手账贴纸风格统一。
- 权限：
  - 所有已登录用户都能查看公开留言墙。
  - 普通用户可以发布留言、删除自己的留言。
  - 管理员可以删除任意留言，并为每条留言写入或覆盖一条官方回复。
- 技术方向：采用“单表扁平留言 + 单条管理员回复 + 分页 + Redis 限流”的轻量方案，不做楼中楼、不做附件、不做实时推送、不新增独立后台页面。
- 本文件基于当前仓库真实结构重新落地，补全了上一版计划里还未锁定的接线细节，尤其是路由限流接入链路与迁移编号。

## Current State Analysis

### 页面现状
- 目标页面文件是 `frontend/src/views/user/ContactSupportView.vue`。
- 当前页面只有三块主结构：
  - 顶部欢迎 `section`
  - 二维码展示 `article`
  - 联系方式与管理员二维码操作 `aside`
- 页面已具备可复用能力：
  - `isAdmin` 权限判断
  - `useAppStore()` 成功/失败 toast
  - `onMounted()` 初始化加载
  - 管理员二维码上传、发布、清空逻辑
- 当前文件中没有任何留言板状态、API 调用或交互逻辑，因此新增 section 不会与旧功能重叠。

### 后端结构现状
- 用户认证路由入口：`backend/internal/server/routes/user.go`
- 管理员路由入口：`backend/internal/server/routes/admin.go`
- 路由统一注册点：`backend/internal/server/router.go`
- 当前 `RegisterUserRoutes()` 和 `RegisterAdminRoutes()` 都没有接收 `redisClient`，因此如果要在留言板接口上复用 `backend/internal/middleware/rate_limiter.go`，必须连带修改这三个文件的函数签名和调用链。
- 现有内容模块可复用公告模式：
  - 用户 handler：`backend/internal/handler/announcement_handler.go`
  - 管理 handler：`backend/internal/handler/admin/announcement_handler.go`
  - service 抽象：`backend/internal/service/announcement.go`
  - service 实现：`backend/internal/service/announcement_service.go`
  - repository：`backend/internal/repository/announcement_repo.go`
  - DTO：`backend/internal/handler/dto/announcement.go`
- 依赖注入固定走 Wire：
  - handler provider：`backend/internal/handler/wire.go`
  - handler struct：`backend/internal/handler/handler.go`
  - service provider：`backend/internal/service/wire.go`

### Ent 与迁移现状
- 公告 schema 参考文件：`backend/ent/schema/announcement.go`
- 软删除 mixin 已存在：`backend/ent/schema/mixins/soft_delete.go`
- 迁移目录实际最新编号是 `144_add_opus48_to_model_mapping.sql`，因此新留言板迁移文件应使用 `145_add_ai_idea_messages.sql`，上一版计划里写的 `145` 在当前仓库中是成立的。
- 当前迁移命名规则由 `backend/migrations/README.md` 约束，要求 forward-only、顺序编号、不可修改已应用迁移。

### 前端 API 与类型现状
- 前端统一类型文件：`frontend/src/types/index.ts`
- 用户 API 汇总出口：`frontend/src/api/index.ts`
- 管理端 API 汇总出口：`frontend/src/api/admin/index.ts`
- 公告 API 文件可以直接作为风格参考：
  - `frontend/src/api/announcements.ts`
  - `frontend/src/api/admin/announcements.ts`
- 当前联系页没有单独 store；本功能更适合继续使用页面内本地状态，避免引入额外全局状态和常驻内存。

## Assumptions & Decisions
- 决策：留言板仅面向已登录用户，不做匿名访问或匿名发布。
- 决策：留言墙对所有已登录用户公开可见，默认展示全部有效留言。
- 决策：留言模型为扁平结构，每条留言只允许 1 条管理员官方回复，不做多级评论树。
- 决策：删除采用软删除，普通列表中不返回已删记录。
- 决策：管理员回复支持覆盖更新；不单独设计“清空回复”按钮，管理员如需移除内容可直接覆盖或删除整条留言。
- 决策：普通用户删除权限仅限自己的留言；管理员删除权限覆盖全部留言。
- 决策：默认每页 10 条，最大 30 条，按 `created_at desc, id desc` 排序。
- 决策：前端不新增独立后台页面，管理员直接在联系页留言卡片内回复和删除。
- 决策：不做 WebSocket/SSE、附件上传、Markdown 富文本、点赞、收藏、审核工作流，以控制服务器和前端复杂度。

## Proposed Changes

### 1. 数据模型与数据库迁移

#### 新增文件
- `backend/migrations/145_add_ai_idea_messages.sql`
- `backend/ent/schema/ai_idea_message.go`

#### 迁移内容
- 创建表 `ai_idea_messages`
- 字段：
  - `id BIGSERIAL PRIMARY KEY`
  - `author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT`
  - `author_name VARCHAR(120) NOT NULL`
  - `title VARCHAR(120) NOT NULL`
  - `content TEXT NOT NULL`
  - `admin_reply TEXT DEFAULT NULL`
  - `admin_reply_by BIGINT DEFAULT NULL REFERENCES users(id) ON DELETE SET NULL`
  - `admin_reply_at TIMESTAMPTZ DEFAULT NULL`
  - `status VARCHAR(20) NOT NULL DEFAULT 'active'`
  - `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
  - `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
  - `deleted_at TIMESTAMPTZ DEFAULT NULL`
- 索引：
  - `idx_ai_idea_messages_status_created_at` on `(status, created_at DESC, id DESC)`
  - `idx_ai_idea_messages_author_id_created_at` on `(author_id, created_at DESC, id DESC)`
- 注释说明：
  - `status` 仅允许 `active`、`user_deleted`、`admin_deleted`
  - `admin_reply` 为单条官方回复位

#### Ent schema 设计
- 参考 `backend/ent/schema/announcement.go` 定义字段与索引。
- 复用 `backend/ent/schema/mixins/soft_delete.go`，让普通查询自动过滤软删除记录。
- schema 中额外保留 `status` 字段，目的是区分“被谁删除”，便于后续审计与行为分析；普通列表仍只返回 `status=active`。

### 2. 后端领域模型、仓储与服务

#### 新增文件
- `backend/internal/service/ai_idea_message.go`
- `backend/internal/service/ai_idea_message_service.go`
- `backend/internal/repository/ai_idea_message_repo.go`
- `backend/internal/handler/dto/ai_idea_message.go`
- `backend/internal/handler/ai_idea_message_handler.go`
- `backend/internal/handler/admin/ai_idea_message_handler.go`

#### 修改文件
- `backend/internal/service/wire.go`
- `backend/internal/handler/wire.go`
- `backend/internal/handler/handler.go`

#### service 层职责
- 在 `ai_idea_message.go` 中定义：
  - 状态常量：`active`、`user_deleted`、`admin_deleted`
  - 领域模型：`AIdeaMessage`
  - repository 接口
  - service 错误常量，例如：
    - 找不到留言
    - 标题/正文/回复长度非法
    - 非作者删除他人留言
    - 非管理员回复
- 在 `ai_idea_message_service.go` 中实现：
  - `List(ctx, actorID, actorIsAdmin, params, filters)`
  - `Create(ctx, input)`
  - `Delete(ctx, actorID, actorIsAdmin, id)`
  - `Reply(ctx, actorID, id, reply)`
- 校验规则：
  - `title`: trim 后 1-120
  - `content`: trim 后 1-2000
  - `admin_reply`: trim 后 1-1000
- 返回模型中直接计算好：
  - `is_mine`
  - `can_delete`
  - `can_reply`
  让前端不用重复推权限。

#### repository 层职责
- 参考 `backend/internal/repository/announcement_repo.go` 实现分页查询、创建、更新。
- 查询只返回 `status=active` 且未软删除记录。
- 列表支持两个过滤项：
  - `mine_only`
  - `page/page_size`
- 排序固定使用 `created_at desc, id desc`，不开放多字段排序，降低接口复杂度。

### 3. 路由与限流接线

#### 修改文件
- `backend/internal/server/routes/user.go`
- `backend/internal/server/routes/admin.go`
- `backend/internal/server/router.go`

#### 接口定义
- 用户侧：
  - `GET /api/v1/idea-messages`
  - `POST /api/v1/idea-messages`
  - `DELETE /api/v1/idea-messages/:id`
- 管理侧：
  - `PUT /api/v1/admin/idea-messages/:id/reply`

#### 限流接入方式
- 复用 `backend/internal/middleware/rate_limiter.go`
- 因为当前 `RegisterUserRoutes()` / `RegisterAdminRoutes()` 没有 `redisClient` 入参，所以要：
  - 在 `backend/internal/server/router.go` 中把 `redisClient` 继续向下传
  - 更新 `RegisterUserRoutes()` 签名
  - 更新 `RegisterAdminRoutes()` 签名
- 在两个 routes 文件内部各自创建 `rateLimiter := middleware.NewRateLimiter(redisClient)`。
- 限流策略：
  - `POST /idea-messages`: 每分钟 5 次，`fail-close`
  - `DELETE /idea-messages/:id`: 每分钟 20 次，`fail-close`
  - `PUT /admin/idea-messages/:id/reply`: 每分钟 20 次，`fail-close`
- 说明：这里继续沿用 `backend/internal/server/routes/auth.go` 的保护策略，避免 Redis 故障时高风险写操作无上限放大。

### 4. DTO 与接口返回

#### 新增文件
- `backend/internal/handler/dto/ai_idea_message.go`

#### 返回结构
- 单条留言 DTO 统一包含：
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
- 列表接口返回继续复用现有分页响应格式，由 `response.Paginated(...)` 输出。

#### 请求结构
- 用户发布请求：
  - `title`
  - `content`
- 管理员回复请求：
  - `admin_reply`

### 5. 前端类型与 API 封装

#### 新增文件
- `frontend/src/api/ideaMessages.ts`
- `frontend/src/api/admin/ideaMessages.ts`

#### 修改文件
- `frontend/src/api/index.ts`
- `frontend/src/api/admin/index.ts`
- `frontend/src/types/index.ts`

#### 类型新增
- `IdeaMessage`
- `CreateIdeaMessageRequest`
- `AdminReplyIdeaMessageRequest`
- `IdeaMessageListParams`

#### API 设计
- `frontend/src/api/ideaMessages.ts`
  - `list(page?, pageSize?, params?)`
  - `create(request)`
  - `deleteIdeaMessage(id)`
- `frontend/src/api/admin/ideaMessages.ts`
  - `reply(id, request)`
- 用户列表方法返回 `BasePaginationResponse<IdeaMessage>`，与仓库现有分页约定一致。

### 6. 联系页 UI 与交互

#### 修改文件
- `frontend/src/views/user/ContactSupportView.vue`

#### 页面结构
- 在当前第二个 `section` 之后新增一个完整的留言板 `section`。
- 布局采用单 section、上下结构：
  - 顶部：标题、说明文案、彩色标签
  - 中部：左侧发布表单，右侧筛选和说明
  - 下部：留言卡片列表
- 这样在窄屏和桌面端都更稳定，且不会压缩现有 `article/aside` 区块。

#### 新增页面状态
- `ideaLoading`
- `ideaSubmitting`
- `ideaActionLoadingIds`
- `ideaError`
- `ideaList`
- `ideaPagination`
- `ideaHasMore`
- `ideaMineOnly`
- `ideaForm = { title, content }`
- `replyDrafts: Record<number, string>`
- `replyEditingId`

#### 页面初始化
- 保留当前 `loadContactPage()` 不动。
- 在 `onMounted()` 中并行执行：
  - `loadContactPage()`
  - `loadIdeaMessages({ reset: true })`

#### 交互函数
- `loadIdeaMessages({ reset?: boolean })`
- `submitIdeaMessage()`
- `deleteIdeaMessage(message: IdeaMessage)`
- `startReplyEdit(message: IdeaMessage)`
- `saveAdminReply(message: IdeaMessage)`
- `loadMoreIdeaMessages()`
- `resetIdeaForm()`

#### 表单行为
- 所有登录用户可见发布表单。
- 标题和正文输入使用页面内原生表单控件，不引入新组件库。
- 提交前前端做 trim 和长度校验，减少无效请求。
- 发布成功后：
  - 若当前筛选是“全部灵感”，将新留言直接插入列表头部，并同步 `total`
  - 若当前筛选是“我的留言”，同样插入头部
  - 表单清空
- 发布失败只展示轻量错误文案和 toast，不清空用户输入。

#### 列表行为
- 顶部提供两个筛选：
  - `全部灵感`
  - `我的留言`
- 切换筛选时：
  - 重置页码为 1
  - 清空当前列表
  - 重新请求第一页
- 每条卡片展示：
  - 标题
  - 正文
  - 作者名
  - 发布时间
  - 官方回复便签（如有）
- 动作按钮：
  - 普通用户：仅自己的留言显示 `删除`
  - 管理员：所有留言显示 `删除`
  - 管理员：所有留言显示 `回复` 或 `修改回复`

#### 管理员回复交互
- 回复编辑区内联展开在卡片底部，不开弹窗。
- 管理员点击 `回复/修改回复` 后才展开对应卡片编辑区，始终只允许一个 `replyEditingId` 处于编辑态，避免页面状态过重。
- 回复成功后直接更新当前列表项的 `admin_reply` 和 `admin_reply_at`，不额外全量刷新。

#### 删除交互
- 删除成功后直接从当前列表移除该项，并同步减少 `total`。
- 如果当前页删空且仍有下一页，可补拉一页填充；否则保持当前状态即可。

#### “加载更多”策略
- 首屏只拉第一页 10 条。
- 当 `page < pages` 时显示 `加载更多` 按钮。
- 点击后请求下一页并追加到列表尾部，不重刷已加载数据。

#### 视觉风格
- 复用当前页面已有语言：
  - 大圆角卡片
  - 轻渐变背景
  - 彩色贴纸标签
  - 深浅色模式对应类名
- 新留言板 section 的视觉关键词：
  - 标题像“灵感收纳板”
  - 回复区像“官方便签”
  - 留言卡片像“贴在墙上的小纸条”
- 不引入后台表格、抽屉或密集型管理 UI，保持与当前联系页一致的轻松感。

### 7. Wire 与生成代码

#### 修改文件
- `backend/internal/service/wire.go`
- `backend/internal/handler/wire.go`
- `backend/internal/handler/handler.go`
- 受生成链路影响的文件：
  - `backend/ent/**`
  - `backend/cmd/server/wire_gen.go`

#### 具体接线
- 在 service ProviderSet 中新增留言板 service 构造器与 repository provider。
- 在 handler ProviderSet 中新增用户侧和管理员侧留言板 handler。
- 在 `AdminHandlers` / `Handlers` 结构体中加入对应字段。
- 重新生成 Ent 与 Wire，使 `router.go` 可直接访问 `h.IdeaMessage`、`h.Admin.IdeaMessage` 一类的新句柄。

### 8. 测试与验证

#### 后端测试
- 新增或补充针对留言板 service/handler 的测试，至少覆盖：
  - 普通用户发布成功
  - 标题/正文超限失败
  - 普通用户删除自己的留言成功
  - 普通用户删除他人留言失败
  - 管理员删除任意留言成功
  - 管理员回复成功
  - 普通用户调用回复接口失败
  - `mine_only` 过滤与分页返回正确

#### 前端验证
- 联系页加载时留言板首屏正常出现
- 普通用户可以发布并看到新留言
- 普通用户仅自己的卡片出现删除按钮
- 管理员卡片出现删除和回复入口
- 回复成功后卡片内出现官方回复便签
- 删除/回复后二维码区和联系方式区行为不受影响

## Verification Steps
- 数据层：
  - 应用 `145_add_ai_idea_messages.sql` 后成功生成 `ai_idea_messages` 表和索引
  - Ent 代码生成成功，项目可编译
- 后端接口：
  - `GET /api/v1/idea-messages` 支持分页和 `mine_only`
  - `POST /api/v1/idea-messages` 正常创建留言
  - `DELETE /api/v1/idea-messages/:id` 正确执行作者权限校验
  - `PUT /api/v1/admin/idea-messages/:id/reply` 仅管理员可调用
  - 高频调用写接口时返回 `429`
- 前端页面：
  - 联系页出现新的留言板 section
  - 样式与现有 `section/article/aside` 一致
  - 首屏只加载第一页
  - “加载更多”以追加方式工作
  - 本地更新策略生效，不因单次回复/删除触发全页重载
- 回归：
  - 原二维码上传、发布、清空功能保持可用
  - 原联系方式展示与复制按钮保持可用

