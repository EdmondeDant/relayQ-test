# 支持模型外层框删除计划

## Summary
可以删除“支持模型”区域当前最外层的那一层框，而且最小改动只需要动一个文件：[AvailableChannelsTable.vue](file:///c:/work/RelayQ-test/frontend/src/components/channels/AvailableChannelsTable.vue)。当前页面父级布局 [TablePageLayout.vue](file:///c:/work/RelayQ-test/frontend/src/components/layout/TablePageLayout.vue) 已经为 `#table` 插槽提供了一层统一的 `card table-scroll-container` 容器，因此子组件内再次包一层 `div.card overflow-hidden` 是重复的视觉框。计划是在不改数据流、不改分页、不改空态、不改父组件调用的前提下，移除这一层重复容器。

## Current State Analysis

### 1. 当前子组件结构
文件：[AvailableChannelsTable.vue](file:///c:/work/RelayQ-test/frontend/src/components/channels/AvailableChannelsTable.vue#L1-L228)

当前模板最外层是：

```vue
<div class="card overflow-hidden">
  <table class="w-full table-fixed border-collapse text-sm">
    ...
  </table>
</div>
```

这层 `card` 会额外产生：
- 圆角
- 背景
- 边框
- 阴影
- 一层独立的“框感”

而组件内部本身还包含：
- 每个平台 section 容器：`rounded-lg border ...`
- 每个模型卡片容器：`rounded-lg border ... shadow-sm ...`
- 分页栏
- 加载态 / 空态

所以“支持模型”区域当前实际是多层框叠加。

### 2. 父组件已经有统一容器
文件：[TablePageLayout.vue](file:///c:/work/RelayQ-test/frontend/src/components/layout/TablePageLayout.vue#L13-L18)

父布局已经对 `#table` slot 做了统一包裹：

```vue
<div class="layout-section-scrollable">
  <div class="card table-scroll-container">
    <slot name="table" />
  </div>
</div>
```

也就是说实际结构是：

```text
TablePageLayout
└─ div.card.table-scroll-container
   └─ AvailableChannelsTable
      └─ div.card.overflow-hidden
         └─ table
```

因此删除子组件最外层框，不会让页面失去整体容器；只是去掉重复的内层框。

### 3. 父页面的数据流不会受影响
文件：[AvailableChannelsView.vue](file:///c:/work/RelayQ-test/frontend/src/views/user/AvailableChannelsView.vue#L35-L45)

父页面只是把数据传给表格组件：
- `rows`
- `loading`
- `user-group-rates`
- 文案 props

数据来源为：
- [channels.ts](file:///c:/work/RelayQ-test/frontend/src/api/channels.ts#L73-L79) 的 `getAvailable()`
- `groups.ts` 的 `getUserGroupRates()`（父组件里调用）

删除子组件最外层框不会影响：
- API 请求
- 搜索过滤
- 平台分组
- 模型分页状态
- 空态 / 加载态

### 4. 全局 card 样式不应修改
文件：[style.css](file:///c:/work/RelayQ-test/frontend/src/style.css#L214-L220)

全局 `.card` 是通用样式，很多页面复用。当前需求只是去掉“支持模型”这个局部区域的内层框，不应通过修改全局 `.card` 来实现，否则会波及全站。

## Proposed Changes

### 变更文件 1
文件：[AvailableChannelsTable.vue](file:///c:/work/RelayQ-test/frontend/src/components/channels/AvailableChannelsTable.vue)

#### 变更内容
把模板最外层从：

```vue
<div class="card overflow-hidden">
  <table class="w-full table-fixed border-collapse text-sm">
    ...
  </table>
</div>
```

改成以下两种之一，优先第一种：

**方案 A（推荐）**

```vue
<table class="w-full table-fixed border-collapse text-sm">
  ...
</table>
```

**方案 B（仅在实际测试发现需要裁剪时才用）**

```vue
<div class="overflow-hidden">
  <table class="w-full table-fixed border-collapse text-sm">
    ...
  </table>
</div>
```

#### 为什么这么改
- 删除当前用户口中的“支持模型整个这个框”。
- 保留父层 `TablePageLayout` 提供的统一框体。
- 只去掉重复的视觉容器，不动业务结构。

#### 如何改
- 仅调整模板最外层包裹结构。
- 不改内部 `tbody`、`section`、`model` 卡片、分页按钮。
- 不改 script 内任何分页、价格、计费逻辑。
- 不改 `style scoped`，除非移除容器后出现非常轻微的边界样式问题。

### 不变更文件
以下文件保持不动：
- [AvailableChannelsView.vue](file:///c:/work/RelayQ-test/frontend/src/views/user/AvailableChannelsView.vue)
- [TablePageLayout.vue](file:///c:/work/RelayQ-test/frontend/src/components/layout/TablePageLayout.vue)
- [channels.ts](file:///c:/work/RelayQ-test/frontend/src/api/channels.ts)
- [style.css](file:///c:/work/RelayQ-test/frontend/src/style.css)

原因：这些文件与“删除内层框”无直接必要关系，改动只会扩大影响面。

## Assumptions & Decisions
- 这里的“把 `div` 支持模型整个这个框都删掉”解释为：删除 [AvailableChannelsTable.vue](file:///c:/work/RelayQ-test/frontend/src/components/channels/AvailableChannelsTable.vue) 最外层 `div.card overflow-hidden`。
- 不删除平台 section 容器。
- 不删除单模型卡片容器。
- 不重构当前每页 5 个模型的分页逻辑。
- 不调整父级通用布局容器。
- 若删除后视觉上仍觉得“框太多”，下一轮再单独计划处理 section 容器或 model 卡片容器，但不和本次最小改动混在一起。

## Verification
执行时按下面顺序验证：

1. 运行组件诊断，确认 [AvailableChannelsTable.vue](file:///c:/work/RelayQ-test/frontend/src/components/channels/AvailableChannelsTable.vue) 无 Vue / TypeScript 报错。
2. 运行前端构建：

```bash
pnpm run build
```

3. 用浏览器页面确认：
- “支持模型”区域最外层重复框消失；
- 页面仍保留父层整体卡片容器；
- 加载态正常；
- 空态正常；
- 每个平台的模型卡片仍正常显示；
- 每页 5 个分页仍正常；
- 上一页 / 下一页仍可用。

4. 如有条件，优先用 Chrome DevTools MCP 实际查看 DOM，确认移除的是子组件顶层框，而不是父层公共容器。
