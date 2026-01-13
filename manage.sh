#!/bin/bash

# 项目根目录
PROJECT_ROOT=$(pwd)
APP_NAME="server"
PID_FILE="$PROJECT_ROOT/server.pid"
ENV_FILE="$PROJECT_ROOT/.env"

# 加载环境变量
if [ -f "$ENV_FILE" ]; then
    export $(grep -v '^#' "$ENV_FILE" | xargs)
    echo "Loaded environment variables from .env"
else
    echo "Warning: .env file not found. Make sure ARK_API_KEY and ARK_MODEL_ID are set in your shell."
fi

stop() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        echo "Stopping $APP_NAME (PID: $PID)..."
        kill $PID
        rm "$PID_FILE"
        sleep 1
    else
        # 兜底：通过端口查找并杀掉进程
        TPID=$(lsof -t -i:8080)
        if [ ! -z "$TPID" ]; then
            echo "Killing process on port 8080 (PID: $TPID)..."
            kill -9 $TPID
        else
            echo "$APP_NAME is not running."
        fi
    fi
}

start() {
    echo "Building $APP_NAME..."
    go build -o bin/$APP_NAME cmd/server/main.go
    
    echo "Starting $APP_NAME..."
    # 在后台运行，并将日志输出到 server.log
    nohup ./bin/$APP_NAME > server.log 2>&1 &
    echo $! > "$PID_FILE"
    echo "$APP_NAME started. Logs are in server.log"
    echo "Access at: http://localhost:8080"
}

status() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p $PID > /dev/null; then
            echo "$APP_NAME is running (PID: $PID)."
            return
        fi
    fi
    echo "$APP_NAME is stopped."
}

case "$1" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        stop
        start
        ;;
    status)
        status
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        exit 1
esac

