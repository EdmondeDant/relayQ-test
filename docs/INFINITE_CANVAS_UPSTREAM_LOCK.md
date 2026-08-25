# Infinite Canvas 上游锁定

- 仓库：`https://github.com/basketikun/infinite-canvas`
- 版本：`v0.16.0`
- 提交：`9414048f9d0a099386aa15d81bedb5376b79ee61`
- 许可证：MIT
- 服务路径：`/canvas-app/`

当前阶段使用固定上游版本承载画布 UI；RelayQ 前端 `/canvas` 负责登录校验和 bootstrap，所有生成能力必须通过 RelayQ 网关接入。

v0.16.0 引入多图展开控制、默认移动工具、缩放稳定性与生成次数修复。RelayQ overlay 已按该版本源码重放，继续保留万能 Key、模型目录、余额、退出入口、请求来源头和异步视频任务路由。
