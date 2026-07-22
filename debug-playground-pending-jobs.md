# Debug Session: playground-pending-jobs [OPEN]

## Symptom
- Playground 创作记录里的图片任务长时间停留在 `pending`，用户怀疑后端没有正常推进到 `running` / `succeeded` / `failed`。

## Scope
- 仅做运行时证据收集与分析。
- 在拿到证据前不修改业务逻辑。

## Initial Hypotheses
1. Worker/调度器没有真正启动，任务创建后无人消费。
2. 任务被消费了，但状态更新事务没有落库，前端只能看到旧的 `pending`。
3. 任务选择条件过严，导致新建任务没有被 worker 扫描到。
4. 上游调用卡住或重试过长，但在进入处理前没有先把状态置为 `running`。
5. 前端轮询或记录查询命中了旧记录，而不是这次最新任务。

## Evidence Log
- 运行时接口证据：
- `GET /api/v1/playground/records` 返回任务 `6/7/8` 均为 `pending`，且 `updated_at == created_at`，说明后端没有推进状态，不是前端显示延迟。
- `GET /api/v1/playground/tasks/8` 在提交 2 秒后仍为 `pending`，且 `result_payload` 为空。
- 已启动本地 debug server 监听 `127.0.0.1:7777`，重新提交任务 `8` 后没有收到任何 `runJob` / `update running` / `execute result` 事件。
- 运行进程证据：
- 当前监听 `8080` 的进程为 `C:\Users\Administrator\.openclaw\workspace\realyq-test\.localdata_clean\server.exe`。
- 该二进制 `LastWriteTime = 2026-07-22 02:23:52 +08:00`。
- 当前源码文件 `backend/internal/service/playground_jobs.go` `LastWriteTime = 2026-07-22 04:11:16 +08:00`。
- 结论：当前实际运行的后端不是最新源码编译产物，源码里的异步任务逻辑/埋点未被加载。
- 修复验证证据：
- 已停止旧进程 `17028`，使用当前源码重新编译 `.localdata_clean/server.exe`，新二进制 `LastWriteTime = 2026-07-22 15:47:34 +08:00`。
- 重启后再次提交任务 `9`，debug server 收到如下事件链：
- `N playground runJob start`
- `O playground runJob update running result err=<nil>`
- `P playground runJob execute result err=Invalid API key`
- `GET /api/v1/playground/tasks/9` 返回：
- `status = failed`
- `started_at/completed_at` 已写入
- `error_message = Invalid API key`
- 结论：最新后端已能正确推进异步任务状态，旧的“永久 pending”问题已消失；当前剩余问题是提交时使用了无效 API Key，因此任务正确失败而非卡住。
- 第二阶段运行时证据：
- 用户复测后，新提交任务不再 `pending`，而是先进入 `running`。
- `task 11`（`edit`）事件链完整：
- `runJob start -> update running -> backend async edit request start -> upstream headers received -> backend image edit completed -> execute result err=nil`
- `task 11` 最终已写入图片资产并完成，`completed_at = 2026-07-22 15:57:50 +08:00`。
- `task 10`（`image`）事件链停在：
- `runJob start -> update running -> http_upstream.Do invoked (target=/v1/images/generations)`
- 之后没有 `execute result`，`GET /api/v1/playground/tasks/10` 仍为 `running`，且 `updated_at` 仅停留在开始执行时刻。
- 代码证据：
- `doPlaygroundJSONRequest()` 使用 `http.DefaultClient.Do(req)`，未设置专用超时，也未在外层为图片任务设置执行超时。
- 结论：`pending` 根因已修复；当前新的卡点是图片生成请求在执行阶段可能无限等待上游响应，导致任务长时间停留在 `running`。

## Next Step
- 重编译并重启后端到最新源码，再复现提交任务，验证状态是否从 `pending` 进入 `running` / `succeeded` / `failed`。
