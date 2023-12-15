#!/bin/bash

# 设置要查找的端口号
PORT=8002


function startapi(){
  go build -o cj-server main.go
  chmod +x cj-serer
  nohup ./cj-server &
}
# 检查端口是否被占用
if netstat -tuln | grep -q "$PORT"; then
    echo "端口 $PORT 已被占用"
    # 获取占用该端口的进程ID
    PID=$(lsof -t -i:$PORT)
    # 终止进程
    if [ -n "$PID" ]; then
        echo "正在终止进程 $PID"
        kill $PID
        echo "进程已终止"
    fi
fi

startapi
