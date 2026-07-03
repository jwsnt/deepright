# Remote 迭代 20260521-3 使用手册

## 变更说明

本次迭代为 `remote create` 增加 `--certificate` 参数，支持使用 PEM 证书创建 SSH 连接。

证书模式下，行为等价于系统命令：

```bash
ssh -i /a.pem ubuntu@1.2.3.4 -p 10086
```

## 命令用法

```bash
./remote create --agentId xxx --chatId yyy --remote ubuntu@1.2.3.4 --port 10086 --certificate /a.pem
```

## 认证模式

`remote create` 现在支持两种认证方式：

- 密码模式：`--password`
- 证书模式：`--certificate /path/to/id.pem`

说明：

- 两者至少提供一个
- 证书路径需要是存在的文件
- 证书模式会使用 `ssh -i <pem>` 建立 SSH 主连接
- 证书模式不会再走 `SSH_ASKPASS` 密码输入链路

## 与主手册关系

- 主手册：[../../USER_GUIDE.md](../../USER_GUIDE.md)
- 主需求：[../../REQUIREMENT.md](../../REQUIREMENT.md)
- 当前迭代需求：[REQUIREMENT.md](REQUIREMENT.md)

