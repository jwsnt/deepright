# 迭代 20260530-4：设置页复制 `deviceId` 与远程连接

本次迭代为 Site 设置页补充了两组能力：

- 在 `设置` 文案右侧新增慢速旋转的复制小图标
- 点击后会复制当前共享 Agent 元数据中的 `deviceId`
- 复制成功后，会在左侧虚拟文件系统 Tips 位置提示 `注意唯一ID的安全性`

- 在 Agent 蜂群图标右侧新增 `R` 小图标，旋转节奏与蜂群图标一致
- 点击后会在蜂群配置位置展开外部设备连接面板
- 支持多行输入，每行一个连接，单项最长 `50` 个字符，并带小眼睛切换显示
- 再次点击 `R` 会收起面板，点击蜂群图标会切回蜂群配置
- 点击 `确认` 后先收起面板，最终会随设置保存一起写回当前 Agent 的 `config.json.router_remote`

转发行为：

- 居中会话区发送 `/v1/chat/completions` 时，如果当前 Agent 已配置 `router_remote`，会把它写入 `metadata.router_remote`
- 右侧备忘录生成的任务明细在执行时也会附带同样的 `metadata.router_remote`
- 通过插件桥接生成的备忘录任务明细同样会继承当前 Agent 的 `router_remote`
