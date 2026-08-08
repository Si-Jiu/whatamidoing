#!/usr/bin/env bash
# whatamidoing 本地调试服务端（开发用，勿用于生产）
# 用法:
#   ./debug-server.sh start   启动（源码有改动自动重新编译）
#   ./debug-server.sh stop    停止
#   ./debug-server.sh restart 重启
#   ./debug-server.sh reset   清空数据重新初始化（首次初始化令牌: debug-setup-token）
#   ./debug-server.sh log     跟踪日志
# 页面: http://localhost:9090
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# 数据/日志/PID 放系统指定缓存目录，可显式覆盖
case "$(uname -s)" in
  Darwin)
    sys_cache="${XDG_CACHE_HOME:-$HOME/Library/Caches}"
    ;;
  Linux)
    sys_cache="${XDG_CACHE_HOME:-$HOME/.cache}"
    ;;
  *)
    sys_cache="${TMPDIR:-/tmp}"
    ;;
esac
BUILD_DIR="${WAID_DEBUG_DIR:-$sys_cache/whatamidoing-debug}"
mkdir -p "$BUILD_DIR"

PORT=9090
SETUP_TOKEN="debug-setup-token"
PIDFILE="$BUILD_DIR/server.pid"
LOGFILE="$BUILD_DIR/server.log"
BIN="$BUILD_DIR/server"
DATA_FILE="$BUILD_DIR/data.json"
# 开发模式：前端从项目目录实时读取，改 server/web 下文件刷新即生效
WAID_DEV_WEB_DIR="$ROOT/server/web"

ensure_built() {
  if [[ ! -x "$BIN" ]] || find "$ROOT/server" -name '*.go' -newer "$BIN" | grep -q .; then
    echo "编译服务端..."
    (cd "$ROOT/server" && go build -o "$BIN" ./cmd/server)
  fi
}

case "${1:-start}" in
  start)
    if [[ -f "$PIDFILE" ]] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
      echo "已在运行 (pid $(cat "$PIDFILE")): http://localhost:$PORT"
      exit 0
    fi
    ensure_built
    rm -f "$PIDFILE"
    PORT=$PORT SETUP_TOKEN=$SETUP_TOKEN DATA_FILE=$DATA_FILE WAID_DEV_WEB_DIR=$WAID_DEV_WEB_DIR nohup "$BIN" >>"$LOGFILE" 2>&1 &
    echo $! >"$PIDFILE"
    sleep 1
    echo "已启动 (pid $(cat "$PIDFILE")): http://localhost:$PORT"
    echo "数据文件: $DATA_FILE"
    echo "首次初始化令牌: $SETUP_TOKEN"
    echo "前端实时模式: $WAID_DEV_WEB_DIR"
    ;;
  stop)
    [[ -f "$PIDFILE" ]] || { echo "未在运行"; exit 0; }
    kill "$(cat "$PIDFILE")" 2>/dev/null || true
    rm -f "$PIDFILE"
    echo "已停止"
    ;;
  restart)
    "$0" stop; "$0" start
    ;;
  reset)
    "$0" stop
    rm -f "$DATA_FILE"
    echo "数据已清空"
    ;;
  log)
    tail -f "$LOGFILE"
    ;;
  *)
    echo "用法: $0 {start|stop|restart|reset|log}"
    exit 1
    ;;
esac
