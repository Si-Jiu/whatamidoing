#!/usr/bin/env bash
# 构建并安装 Linux 桌面客户端到 ~/.local
# 用法: ./dist/linux/install.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BIN_DEST="${HOME}/.local/bin/whatamidoing-client"
APP_DEST="${HOME}/.local/share/applications/whatamidoing.desktop"
ICON_SRC="${REPO_ROOT}/clients/desktop/src-tauri/icons/128x128.png"
ICON_DEST="${HOME}/.local/share/icons/hicolor/128x128/apps/whatamidoing.png"

echo "==> 构建 release 版本（首次较慢）..."
( cd "${REPO_ROOT}/clients/desktop/src-tauri" && cargo build --release )

echo "==> 安装二进制到 ${BIN_DEST}"
mkdir -p "${HOME}/.local/bin"
cp "${REPO_ROOT}/clients/desktop/src-tauri/target/release/whatamidoing-client" "${BIN_DEST}"
chmod +x "${BIN_DEST}"

echo "==> 安装桌面启动器与图标"
mkdir -p "$(dirname "${APP_DEST}")" "$(dirname "${ICON_DEST}")"
sed "s|^Exec=.*|Exec=${BIN_DEST}|" "${REPO_ROOT}/dist/linux/whatamidoing.desktop" > "${APP_DEST}"
cp "${ICON_SRC}" "${ICON_DEST}"

echo "==> 完成。可在应用菜单找到 whatamidoing，或直接运行 ${BIN_DEST}"
echo "    首次运行会弹出设置窗口：填服务端地址与上报 token 即可。"
