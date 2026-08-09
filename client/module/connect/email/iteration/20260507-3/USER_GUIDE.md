# 20260507-3 User Guide

## 目标

本次迭代为 `email` 插件补充 `command` 命令，用于按插件规范返回当前支持的能力列表。

## 命令格式

```bash
../plugins/email command
```

返回值固定为：

```json
["command","help","name","param","init","send","start","stop"]
```

## 说明

- `command` 返回的列表需要覆盖当前插件实际支持的 CLI 能力
- `help` 也属于插件能力的一部分，因此需要包含在返回值里
- `name`、`param`、`command` 三个命令都应保持稳定输出，便于 `connect` / `integration` 做插件发现和能力展示

## 配套固定输出

```bash
../plugins/email name
../plugins/email param
```

对应固定输出：

```json
{"key":"email","name":"邮件"}
```

```json
["email","email_pop3","email_smtp","email_password","email_whitelist"]
```

## 验证方式

```bash
../plugins/email command
../plugins/email help
```

行为预期：

- `command` 返回合法 JSON string array
- 返回值中包含 `command`、`help`、`name`、`param`、`init`、`send`、`start`、`stop`
- `help` 能输出完整插件使用手册
