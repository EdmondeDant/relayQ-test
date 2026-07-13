# [OPEN] task-center-loading

## 症状

- 任务中心一直显示加载中
- 浏览器曾出现 `Cannot read properties of undefined (reading 'length')`

## 假设

- 任务列表接口结构与前端预期不一致
- 音频任务元数据缺少摘要渲染所需字段
- 模板对可选值直接读取 `.length`
- 渲染异常导致加载状态无法正常展示

## 当前阶段

- 收集浏览器运行时证据
