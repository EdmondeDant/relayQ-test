[OPEN] Playground API Key / 模型串台调试

- 现象：切换 API Key 后，当前分组文案和模型下拉仍显示其他分组的数据。
- 期望：模型下拉只显示当前 API Key 绑定分组允许的模型。

## 假设

1. selectedKeyId 已变化，但 selectedKey 没同步。
2. resolvedGroup/groupModels 读取了旧状态。
3. 页面运行的是旧 bundle。
4. watcher 在切换后把状态回写成旧值。
5. 共享状态被其它区块覆盖。

## 计划

1. 只加运行时埋点，不改业务逻辑。
2. 复现切 key，收集 selectedKeyId / selectedKey / resolvedGroup / chatModels 证据。
3. 依据证据做最小修复。
