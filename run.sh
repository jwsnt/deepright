#!/bin/bash
# 定义 JAR 包名称
JAR_NAME="deepright-1.0.jar"
# JVM 内存及优化参数配置（优先使用环境变量 JVM_OPTS）
# -Xms3g: 初始堆内存 3GB
# -Xmx3g: 最大堆内存 3GB
# -XX:+UseG1GC: 针对 3GB 及以上内存，推荐使用更现代高效的 G1 垃圾回收器，替代您之前默认的 SerialGC
JVM_OPTS="${JVM_OPTS:--Xms3g -Xmx3g -XX:+UseG1GC}"
# 检查程序是否已经在运行
PID=$(ps -ef | grep "$JAR_NAME" | grep -v grep | awk '{print $2}')
if [ -n "$PID" ]; then
    echo "警告: $JAR_NAME 已经在运行中 (PID: $PID)，请先停止它！"
    exit 1
fi
echo "正在启动 $JAR_NAME，JVM 参数: $JVM_OPTS"
# 执行后台启动命令
nohup java $JVM_OPTS -jar "$JAR_NAME" &> /dev/null &
# 等待 2 秒检查是否成功启动
sleep 2
NEW_PID=$(ps -ef | grep "$JAR_NAME" | grep -v grep | awk '{print $2}')
if [ -n "$NEW_PID" ]; then
    echo "启动成功！PID 进程号为: $NEW_PID"
else
    echo "启动失败，请检查程序本身是否有其他配置错误。"
fi
