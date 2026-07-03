# Site 迭代 20260422_2 使用手册

## 变更说明

非本地访问时（host 不是 localhost 或 127.0.0.1）自动隐藏"打开目录"相关按钮。

## 隐藏的按钮

- 虚拟文件系统的"打开目录"
- 对话框输入栏右侧"打开 Agent 目录"
- 设置中选择 Agent 的"打开 Agent 目录"

## 实现方式

- 通过 `location.hostname` 检测，非本地时给 `<html>` 加 `.remote` class
- `.remote .local-only` 隐藏按钮，flex 布局自动调整无错位
