# 邀请码注册改为选填并兼容返利绑定计划

## Summary

当前后台“邀请码注册”开关开启后，普通邮箱注册会强制要求用户填写有效邀请码；用户期望改为：邀请码字段选填，不填写也能正常注册；填写有效邀请码时按现有邀请码逻辑处理，并且该输入也要尝试作为邀请返利码绑定邀请关系。

本计划只覆盖用户确认的范围：**仅邮箱注册**。OAuth/第三方首次注册流程不在本次修改范围内。

## Current State Analysis

### 1. 前端普通注册页强制邀请码必填

文件：[RegisterView.vue](file:///c:/work/RelayQ-test/frontend/src/views/auth/RegisterView.vue)

关键逻辑：

- 注册页通过 public settings 读取 `invitation_code_enabled`，开关开启时显示邀请码输入框。
- `validateForm()` 中当前逻辑在 `invitationCodeEnabled.value === true` 且 `formData.invitation_code` 为空时直接设置错误 `auth.invitationCodeRequired`，导致前端不允许提交。
- `handleRegister()` 中后续逻辑已经比较接近选填语义：只有 `formData.invitation_code.trim()` 有值且未验证时才触发邀请码验证；无效则阻止注册。
- 注册提交时目前会传：
  - `invitation_code: formData.invitation_code || undefined`
  - `aff_code` 只来自 URL/localStorage 的推广码。

### 2. 前端 API/store 不强制邀请码

文件：[auth.ts](file:///c:/work/RelayQ-test/frontend/src/api/auth.ts)

当前 `register(userData)` 只透传到 `/auth/register`，没有做邀请码必填校验。

### 3. 后端 Handler 不强制邀请码

文件：[auth_handler.go](file:///c:/work/RelayQ-test/backend/internal/handler/auth_handler.go)

`RegisterRequest` 中：

- `InvitationCode string json:"invitation_code"` 没有 `binding:"required"`。
- `AffCode string json:"aff_code"` 是独立返利码字段。

`Register()` 只是把 `req.InvitationCode` 和 `req.AffCode` 传给 `AuthService.RegisterWithVerification()`，不做必填判断。

### 4. 后端普通邮箱注册服务强制邀请码必填

文件：[auth_service.go](file:///c:/work/RelayQ-test/backend/internal/service/auth_service.go)

`RegisterWithVerification()` 当前在 `IsInvitationCodeEnabled(ctx)` 为 true 时：

- 如果 `invitationCode == ""`，直接返回 `ErrInvitationCodeRequired`。
- 如果有值，则通过 `redeemRepo.GetByCode()` 校验一次性邀请码。
- 创建用户后，如果 `invitationRedeemCode != nil`，调用 `redeemRepo.Use()` 标记邀请码已使用。
- 邀请返利绑定当前只使用 `affiliateCode`，即 `aff_code`，调用 `AffiliateService.BindInviterByCode()`。

### 5. 邀请返利绑定服务支持“无码静默跳过”

文件：[affiliate_service.go](file:///c:/work/RelayQ-test/backend/internal/service/affiliate_service.go)

`BindInviterByCode(ctx, userID, rawCode)` 当前行为：

- 空码直接返回 nil。
- 返利总开关关闭时返回 nil，不阻断注册。
- 无效返利码返回错误。

但在 `AuthService.RegisterWithVerification()` 中，绑定失败只记录日志，不阻断注册。因此将邀请码输入也作为返利码尝试绑定，不会破坏注册主流程。

## Proposed Changes

### 1. 修改前端普通注册页：邀请码字段从必填变为选填

文件：[RegisterView.vue](file:///c:/work/RelayQ-test/frontend/src/views/auth/RegisterView.vue)

修改内容：

1. 删除或改写 `validateForm()` 中的邀请码空值必填校验：
   - 当前：开关开启且邀请码为空，阻止注册。
   - 修改后：开关开启但邀请码为空，不报错，允许继续注册。

2. 保留 `handleRegister()` 中“填写了邀请码则必须有效”的逻辑：
   - 如果用户填写邀请码且验证中，提示等待。
   - 如果用户填写邀请码且验证无效，阻止注册。
   - 如果用户没有填写邀请码，跳过邀请码验证，继续注册。

3. 提交注册数据时，让 `invitation_code` 同时作为返利绑定兜底：
   - 计算 `const invitationCode = formData.invitation_code.trim()`。
   - 计算 `const affCode = formData.aff_code.trim() || loadAffiliateReferralCode() || invitationCode`。
   - 这样：
     - URL/localStorage 有专门推广码时，优先使用原来的 `aff_code`。
     - 没有推广码但用户在邀请码框填写了码时，也会把它作为 `aff_code` 传给后端尝试绑定返利。

4. 邮箱验证流程也要同步：
   - 写入 `sessionStorage.register_data` 时同样使用上述 `affCode` 逻辑。
   - 直接注册流程同样使用上述 `affCode` 逻辑。

5. 文案层面：
   - 把注释或可见提示从“邀请码必填”调整为“邀请码选填”。
   - 如现有 i18n 已有 `common.optional` 或类似字段，则复用；如果没有，不新增大范围文案，只避免展示“必填”语义。

### 2. 修改后端普通邮箱注册服务：邀请码有值才验证

文件：[auth_service.go](file:///c:/work/RelayQ-test/backend/internal/service/auth_service.go)

修改内容：

1. 在 `RegisterWithVerification()` 开头对输入做 trim：
   - `invitationCode = strings.TrimSpace(invitationCode)`
   - `affiliateCode = strings.TrimSpace(affiliateCode)`

2. 改造邀请码校验逻辑：
   - 当前：`IsInvitationCodeEnabled(ctx)` 为 true 且 `invitationCode == ""` 时返回 `ErrInvitationCodeRequired`。
   - 修改后：仅当 `IsInvitationCodeEnabled(ctx)` 为 true 且 `invitationCode != ""` 时，才校验邀请码。
   - 如果填写的邀请码无效，仍返回 `ErrInvitationCodeInvalid`，阻止注册。
   - 如果不填写邀请码，`invitationRedeemCode` 保持 nil，正常注册。

3. 保留 `redeemRepo.Use()`：
   - 只有 `invitationRedeemCode != nil` 时才消费邀请码。

4. 返利绑定逻辑：
   - 继续使用 `affiliateCode` 调用 `BindInviterByCode()`。
   - 前端已把邀请码输入作为 `aff_code` 兜底传入，因此后端无需新增复杂映射。
   - 绑定失败仍只记录日志，不影响注册。

### 3. 不修改 OAuth/第三方注册流程

用户已选择“仅邮箱注册”。因此以下文件暂不修改：

- [auth_oauth_email_flow.go](file:///c:/work/RelayQ-test/backend/internal/service/auth_oauth_email_flow.go)
- [auth_email_oauth.go](file:///c:/work/RelayQ-test/backend/internal/handler/auth_email_oauth.go)
- [auth_oauth_pending_flow.go](file:///c:/work/RelayQ-test/backend/internal/handler/auth_oauth_pending_flow.go)

OAuth 中 `invitation_required` 相关状态保持现状。

## Assumptions & Decisions

1. “邀请码注册”开关的新邮箱注册语义调整为：显示并启用邀请码输入能力，但邀请码不是必填。
2. 用户不填邀请码时，邮箱注册正常完成。
3. 用户填写邀请码时：
   - 如果该码是有效一次性邀请码，则照常消费邀请码。
   - 同时前端把该码作为 `aff_code` 传给后端尝试绑定邀请返利。
   - 如果该码不是有效返利码，返利绑定失败只记录日志，不影响注册。
4. 如果填写了无效一次性邀请码，仍应阻止注册；这是为了避免用户误以为已使用邀请码但实际没有生效。
5. URL/localStorage 中已有 `aff_code` 时，优先使用原有 `aff_code`，不被手动邀请码覆盖。
6. 本次不新增数据库字段、不新增系统设置、不改 OAuth 注册流程。

## Verification Steps

1. 前端静态检查：

```powershell
cd c:\work\RelayQ-test\frontend
pnpm run typecheck
pnpm run lint:check
```

2. 后端测试：

```powershell
cd c:\work\RelayQ-test\backend
go test ./internal/service ./internal/handler ./internal/server/routes
```

3. 手动验证普通邮箱注册：

- 后台打开“邀请码注册”开关。
- 打开注册页。
- 不填写邀请码，填写邮箱/密码，应该可以正常注册。
- 填写无效邀请码，应该提示邀请码无效并阻止注册。
- 填写有效邀请码，应该可以注册，并消费该邀请码。
- 填写一个有效返利码到邀请码输入框，在没有 URL 推广码时，注册后应尝试绑定邀请关系。
- URL/localStorage 已有推广码时，应优先使用原推广码进行返利绑定。

4. 回归确认：

- 关闭“邀请码注册”开关时，注册流程保持原有正常注册行为。
- 优惠码、Turnstile、邮箱验证码流程不受影响。
