# 专属业务员提现申请与管理员打款记录功能计划

## Summary

根据最新讨论，第一版要做的不是简单展示联系方式，而是一个轻量提现申请闭环：

- 专属业务员在用户侧“邀请返利”页面输入提现金额，提交提现申请。
- 用户侧显示自己的提现记录，并能看到状态：待打款 / 已打款。
- 管理员侧在“邀请返利管理”下面新增“提现记录”页面。
- 管理员线下打款后，在后台点击“标记已打款”。
- 标记后，业务员端提现记录同步显示“已打款”。
- 现有“返佣转余额”功能保留，与提现申请并存。

第一版不做线上支付出款，不收集银行卡/支付宝等收款信息，不做驳回/撤销流程；管理员线下付款，系统只负责申请、记录和状态确认。

## Current State Analysis

### 1. 用户侧邀请返利页面

用户页面：

- [AffiliateView.vue](file:///c:/work/RelayQ-test/frontend/src/views/user/AffiliateView.vue)

当前已有：

- 邀请码/邀请链接
- 邀请人数
- 当前生效返利比例
- 可用返佣额度 `aff_quota`
- 冻结返佣额度 `aff_frozen_quota`
- 历史累计返佣 `aff_history_quota`
- 被邀请用户列表
- “转入余额”按钮

用户 API：

- [user.ts](file:///c:/work/RelayQ-test/frontend/src/api/user.ts)

当前已有：

- `getAffiliateDetail()` -> `GET /api/v1/user/aff`
- `transferAffiliateQuota()` -> `POST /api/v1/user/aff/transfer`

用户类型：

- [index.ts](file:///c:/work/RelayQ-test/frontend/src/types/index.ts)

当前 `UserAffiliateDetail` 已包含返佣额度字段，但没有提现申请记录类型。

### 2. 管理员侧邀请返利管理

管理端路由：

- [router/index.ts](file:///c:/work/RelayQ-test/frontend/src/router/index.ts)

当前已有 affiliate 子路由：

- `/admin/affiliates/invites` 邀请记录
- `/admin/affiliates/rebates` 返利记录
- `/admin/affiliates/transfers` 转余额记录

管理端导航：

- [AppSidebar.vue](file:///c:/work/RelayQ-test/frontend/src/components/layout/AppSidebar.vue)

当前“邀请返利管理”是可折叠菜单，children 已有邀请记录、返利记录、转余额记录。

管理端记录页模式：

- [AdminAffiliateRecordsTable.vue](file:///c:/work/RelayQ-test/frontend/src/views/admin/affiliates/AdminAffiliateRecordsTable.vue)
- [AdminAffiliateTransfersView.vue](file:///c:/work/RelayQ-test/frontend/src/views/admin/affiliates/AdminAffiliateTransfersView.vue)

现有记录页以通用表格展示记录，支持搜索、日期过滤、排序、分页。

### 3. 后端 affiliate 路由与服务

用户路由：

- [user.go](file:///c:/work/RelayQ-test/backend/internal/server/routes/user.go)

当前已有：

- `GET /api/v1/user/aff`
- `POST /api/v1/user/aff/transfer`

用户 handler：

- [user_handler.go](file:///c:/work/RelayQ-test/backend/internal/handler/user_handler.go)

当前已有：

- `GetAffiliate`
- `TransferAffiliateQuota`

管理端路由：

- [admin.go](file:///c:/work/RelayQ-test/backend/internal/server/routes/admin.go)

当前 `registerAffiliateRoutes` 已有：

- `GET /api/v1/admin/affiliates/invites`
- `GET /api/v1/admin/affiliates/rebates`
- `GET /api/v1/admin/affiliates/transfers`
- `/api/v1/admin/affiliates/users/*` 专属用户配置相关接口

管理端 affiliate handler：

- [affiliate_handler.go](file:///c:/work/RelayQ-test/backend/internal/handler/admin/affiliate_handler.go)

当前已有邀请记录、返佣记录、转余额记录列表方法。

Affiliate service：

- [affiliate_service.go](file:///c:/work/RelayQ-test/backend/internal/service/affiliate_service.go)

当前已有：

- 返佣计算
- 冻结返佣
- 可用额度
- 转入余额
- 邀请/返佣/转余额记录查询
- 专属用户设置

Affiliate repository：

- [affiliate_repo.go](file:///c:/work/RelayQ-test/backend/internal/repository/affiliate_repo.go)

当前 `TransferQuotaToBalance` 已有资金扣减/转余额事务模式：

- 锁定 `user_affiliates`
- 解冻已成熟冻结额度
- 消耗 `aff_quota`
- 更新用户余额
- 写 affiliate ledger

提现申请也应复用类似事务思想，但不增加用户余额。

### 4. 当前数据库结构

相关迁移：

- [130_add_user_affiliates.sql](file:///c:/work/RelayQ-test/backend/migrations/130_add_user_affiliates.sql)
- [131_affiliate_rebate_hardening.sql](file:///c:/work/RelayQ-test/backend/migrations/131_affiliate_rebate_hardening.sql)
- [133_affiliate_rebate_freeze.sql](file:///c:/work/RelayQ-test/backend/migrations/133_affiliate_rebate_freeze.sql)
- [134_affiliate_ledger_audit_snapshots.sql](file:///c:/work/RelayQ-test/backend/migrations/134_affiliate_ledger_audit_snapshots.sql)

现有核心表：

- `user_affiliates`
  - `aff_quota` 可用返佣
  - `aff_frozen_quota` 冻结返佣
  - `aff_history_quota` 历史累计返佣
- `user_affiliate_ledger`
  - 当前主要记录 `accrue` / `transfer`

现有 ledger 不适合直接作为提现申请表，因为提现申请需要状态：待打款、已打款。

## Proposed Changes

### 1. 新增提现申请表

新增迁移文件：

- `backend/migrations/135_affiliate_withdrawals.sql`

新增表：

```sql
CREATE TABLE IF NOT EXISTS user_affiliate_withdrawals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(20,8) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMPTZ NULL,
    paid_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    remark TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_user_affiliate_withdrawals_amount_positive CHECK (amount > 0),
    CONSTRAINT chk_user_affiliate_withdrawals_status CHECK (status IN ('pending', 'paid'))
);
```

新增索引：

```sql
CREATE INDEX IF NOT EXISTS idx_user_affiliate_withdrawals_user_created
  ON user_affiliate_withdrawals(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_affiliate_withdrawals_status_created
  ON user_affiliate_withdrawals(status, created_at DESC);
```

扩展 ledger action 注释：

```sql
COMMENT ON COLUMN user_affiliate_ledger.action IS 'accrue|transfer|withdraw_request|withdraw_paid';
```

决策：

- 用户提交提现申请时立即扣减 `aff_quota`。
- 管理员标记已打款时只更新提现申请状态，不再次扣减金额。

原因：

- 避免用户同时多次申请或同时使用“转入余额”导致双花。
- 管理员打款动作只做状态确认，账务风险更低。

### 2. 扩展后端 service 和 repository

文件：

- [affiliate_service.go](file:///c:/work/RelayQ-test/backend/internal/service/affiliate_service.go)
- [affiliate_repo.go](file:///c:/work/RelayQ-test/backend/internal/repository/affiliate_repo.go)

新增类型：

```go
type AffiliateWithdrawalStatus string

const (
    AffiliateWithdrawalPending AffiliateWithdrawalStatus = "pending"
    AffiliateWithdrawalPaid    AffiliateWithdrawalStatus = "paid"
)

type AffiliateWithdrawalRecord struct {
    ID          int64      `json:"id"`
    UserID      int64      `json:"user_id"`
    UserEmail   string     `json:"user_email,omitempty"`
    Username    string     `json:"username,omitempty"`
    Amount      float64    `json:"amount"`
    Status      string     `json:"status"`
    RequestedAt time.Time  `json:"requested_at"`
    PaidAt      *time.Time `json:"paid_at,omitempty"`
    PaidBy      *int64     `json:"paid_by,omitempty"`
    Remark      string     `json:"remark,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}
```

扩展 `AffiliateRepository`：

```go
CreateAffiliateWithdrawal(ctx context.Context, userID int64, amount float64) (*AffiliateWithdrawalRecord, error)
ListUserAffiliateWithdrawals(ctx context.Context, userID int64, params PaginationParams) ([]AffiliateWithdrawalRecord, int64, error)
ListAffiliateWithdrawalRecords(ctx context.Context, filter AffiliateRecordFilter, status string) ([]AffiliateWithdrawalRecord, int64, error)
MarkAffiliateWithdrawalPaid(ctx context.Context, id int64, operatorID int64, remark string) (*AffiliateWithdrawalRecord, error)
```

`CreateAffiliateWithdrawal` 事务逻辑：

1. 确保用户 affiliate profile 存在。
2. 解冻已到期冻结返佣。
3. `SELECT aff_quota FROM user_affiliates WHERE user_id = $1 FOR UPDATE`。
4. 校验金额：
   - `amount > 0`
   - `amount <= aff_quota`
5. `UPDATE user_affiliates SET aff_quota = aff_quota - amount`。
6. 插入 `user_affiliate_withdrawals(status='pending')`。
7. 插入 `user_affiliate_ledger(action='withdraw_request')`，记录本次提现申请扣减。
8. 返回申请记录。

`MarkAffiliateWithdrawalPaid` 事务逻辑：

1. `SELECT * FROM user_affiliate_withdrawals WHERE id=$1 FOR UPDATE`。
2. 仅允许 `status='pending'`。
3. 更新：
   - `status='paid'`
   - `paid_at=NOW()`
   - `paid_by=operatorID`
   - `remark`
4. 插入 `user_affiliate_ledger(action='withdraw_paid')` 作为状态审计记录。
5. 返回更新后的记录。

错误处理：

- 金额小于等于 0：返回参数错误。
- 金额大于可用返佣：返回余额不足。
- 已打款记录重复标记：返回“已打款”错误或冲突响应。

### 3. 新增用户侧提现接口

文件：

- [user.go](file:///c:/work/RelayQ-test/backend/internal/server/routes/user.go)
- [user_handler.go](file:///c:/work/RelayQ-test/backend/internal/handler/user_handler.go)

新增路由：

```go
user.GET("/aff/withdrawals", h.User.ListAffiliateWithdrawals)
user.POST("/aff/withdrawals", h.User.CreateAffiliateWithdrawal)
```

请求：

```json
{
  "amount": 100
}
```

响应：

```json
{
  "id": 1,
  "amount": 100,
  "status": "pending",
  "requested_at": "..."
}
```

说明：

- 使用当前登录用户身份，不允许传 user_id。
- 提交成功后，用户 `aff_quota` 已被扣减。

### 4. 新增管理端提现记录接口

文件：

- [admin.go](file:///c:/work/RelayQ-test/backend/internal/server/routes/admin.go)
- [affiliate_handler.go](file:///c:/work/RelayQ-test/backend/internal/handler/admin/affiliate_handler.go)

新增路由：

```go
affiliates.GET("/withdrawals", h.Admin.Affiliate.ListWithdrawalRecords)
affiliates.POST("/withdrawals/:id/mark-paid", h.Admin.Affiliate.MarkWithdrawalPaid)
```

列表支持参数：

- `page`
- `page_size`
- `search`
- `status`
- `start_at`
- `end_at`
- `sort_by`
- `sort_order`

`mark-paid` 请求：

```json
{
  "remark": "已通过微信转账"
}
```

说明：

- 仅管理员可访问。
- `paid_by` 尽量记录当前管理员 user id。
- 如果当前 admin middleware 不方便取 ID，则先保留字段，mark-paid 时传 0 或空值；但优先尝试从现有鉴权上下文获取。

### 5. 扩展前端用户 API 和类型

文件：

- [user.ts](file:///c:/work/RelayQ-test/frontend/src/api/user.ts)
- [index.ts](file:///c:/work/RelayQ-test/frontend/src/types/index.ts)

新增类型：

```ts
export type AffiliateWithdrawalStatus = 'pending' | 'paid'

export interface AffiliateWithdrawalRecord {
  id: number
  user_id: number
  amount: number
  status: AffiliateWithdrawalStatus
  requested_at: string
  paid_at?: string | null
  remark?: string
  created_at: string
  updated_at: string
}
```

新增 API：

```ts
createAffiliateWithdrawal(amount: number)
listAffiliateWithdrawals(params?: { page?: number; page_size?: number })
```

### 6. 扩展用户侧 AffiliateView

文件：

- [AffiliateView.vue](file:///c:/work/RelayQ-test/frontend/src/views/user/AffiliateView.vue)

新增提现申请卡片：

- 输入框：提现金额
- 显示最大可提现金额：`detail.aff_quota`
- 按钮：`提交提现申请`
- 校验：
  - 金额必须大于 0
  - 金额不能大于 `aff_quota`
- 成功后：
  - 清空输入框
  - 重新加载 affiliate detail
  - 重新加载提现记录

新增提现记录表：

字段：

- 申请时间
- 提现金额
- 状态：待打款 / 已打款
- 打款时间
- 备注

状态展示：

- `pending`：待打款
- `paid`：已打款

保留现有“转入余额”按钮。

### 7. 新增管理端提现记录页面

新增文件：

- `frontend/src/views/admin/affiliates/AdminAffiliateWithdrawalsView.vue`

实现：

- 独立页面，不强行塞进 `AdminAffiliateRecordsTable.vue`。
- 复用项目已有表格风格：
  - 搜索
  - 状态筛选
  - 日期筛选
  - 分页
  - 金额格式化
- 待打款记录显示按钮：`标记已打款`
- 点击按钮弹出确认，可输入备注。
- 成功后刷新列表。

原因：

- 提现记录有状态流转和操作按钮，比普通记录展示复杂。
- 独立页面更清晰，避免现有三类记录表组件继续膨胀。

### 8. 扩展管理端 affiliate API

文件：

- [affiliates.ts](file:///c:/work/RelayQ-test/frontend/src/api/admin/affiliates.ts)

新增：

```ts
listWithdrawalRecords(params)
markWithdrawalPaid(id: number, payload?: { remark?: string })
```

新增管理端提现记录类型，包含：

- 用户邮箱
- 用户名
- 金额
- 状态
- 申请时间
- 打款时间
- 备注

### 9. 新增路由和导航

文件：

- [router/index.ts](file:///c:/work/RelayQ-test/frontend/src/router/index.ts)
- [AppSidebar.vue](file:///c:/work/RelayQ-test/frontend/src/components/layout/AppSidebar.vue)

新增路由：

```ts
{
  path: '/admin/affiliates/withdrawals',
  name: 'AdminAffiliateWithdrawals',
  component: () => import('@/views/admin/affiliates/AdminAffiliateWithdrawalsView.vue'),
  meta: {
    requiresAuth: true,
    requiresAdmin: true,
    title: 'Affiliate Withdrawals',
    titleKey: 'nav.affiliateWithdrawalRecords',
    descriptionKey: 'admin.affiliates.withdrawalsDescription'
  }
}
```

导航位置：

- 放在“邀请返利管理”children 下。
- 文案：`提现记录`。
- 与“邀请记录 / 返利记录 / 转余额记录”并列。

### 10. i18n 文案

文件：

- [zh.ts](file:///c:/work/RelayQ-test/frontend/src/i18n/locales/zh.ts)
- [en.ts](file:///c:/work/RelayQ-test/frontend/src/i18n/locales/en.ts)

新增中文文案：

- `提现申请`
- `提现金额`
- `提交提现申请`
- `最大可提现金额`
- `提现记录`
- `待打款`
- `已打款`
- `标记已打款`
- `申请时间`
- `打款时间`
- `打款备注`
- `管理员线下打款后，请点击标记已打款`

英文文案按项目现有双语模式补齐。

## Assumptions & Decisions

1. 提现申请提交时立即扣减 `aff_quota`。
   - 防止重复申请和转余额双花。

2. 第一版只有两个状态：
   - `pending` 待打款
   - `paid` 已打款

3. 第一版不做驳回。
   - 因为驳回需要退回额度和额外流水，后续可扩展。

4. 管理员打款是线下完成。
   - 系统按钮只负责标记“已打款”。

5. 专属业务员仍保留“转入余额”。
   - 两个出口共享同一个 `aff_quota`，后端事务保证不会超额。

6. 该功能应只对有返佣额度的用户有意义，但接口不需要限制“必须是专属用户”。
   - 页面可以对所有能访问 affiliate 的用户显示提现申请。
   - 如果后续要只给专属业务员，可增加 `cashout_enabled` 字段做开关。
   - 本版按用户当前重点，优先实现业务闭环；不额外引入专属用户开关，避免复杂化。

7. 管理端“提现记录”受邀请返利功能开关控制。
   - 导航放在现有 affiliate 管理菜单下。

## Verification steps

### 后端验证

1. 运行迁移。

2. 执行后端测试/构建：

```powershell
cd C:\work\RelayQ-test\backend
go test ./...
```

3. 用户提交提现申请：

- 准备一个 `aff_quota > 0` 的用户。
- 调用 `POST /api/v1/user/aff/withdrawals`。
- 确认：
  - 返回 `pending` 记录。
  - `user_affiliates.aff_quota` 被扣减。
  - `user_affiliate_withdrawals` 有记录。
  - `user_affiliate_ledger` 有 `withdraw_request` 记录。

4. 用户查看提现记录：

- 调用 `GET /api/v1/user/aff/withdrawals`。
- 能看到刚提交的 pending 记录。

5. 管理员标记已打款：

- 调用 `POST /api/v1/admin/affiliates/withdrawals/:id/mark-paid`。
- 确认状态变为 `paid`，写入 `paid_at`。
- 用户侧再次查询，显示已打款。

6. 双花验证：

- 当 `aff_quota=100` 时，提交 `80` 提现成功。
- 再提交 `30` 应失败。
- 或提交提现后再点转余额，只能转剩余金额。

### 前端验证

1. 前端类型检查：

```powershell
cd C:\work\RelayQ-test\frontend
pnpm run typecheck
```

2. 前端 lint：

```powershell
cd C:\work\RelayQ-test\frontend
pnpm run lint:check
```

3. 用户侧页面：

- 打开 `/affiliate`。
- 输入提现金额。
- 提交申请。
- 看到提现记录状态为“待打款”。
- 可用返佣金额减少。

4. 管理端页面：

- 打开 `/admin/affiliates/withdrawals`。
- 能看到提现申请。
- 点击“标记已打款”。
- 状态更新为“已打款”。

5. 用户侧状态同步：

- 回到业务员账号 `/affiliate`。
- 提现记录显示“已打款”。

## Future Enhancements（不纳入第一版）

- 驳回提现并退回返佣额度。
- 用户填写收款方式。
- 管理员上传付款凭证。
- 提现最低金额。
- 提现手续费。
- 每个业务员单独提现规则。
- 只允许专属业务员提现。
- 完整财务对账报表。