# whatamidoing

让朋友通过一个网站实时看到你**电脑/手机当前前台在干什么**。

设备上的客户端持续上报前台应用名 + 窗口标题到自托管服务端；朋友打开网页即可实时看到
你的状态（文字，不含屏幕截图）。Material Design 3 界面，亮/暗双主题。

```
┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│ Windows  │  │  macOS   │  │  Linux   │  │ Android  │
│ Rust/Tau │  │ Rust/Tau │  │ Rust/Tau │  │  Kotlin  │
└────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘
     └─────────────┴── HTTP ─────┴─────────────┘
                          │
                    ┌─────▼─────┐     ┌────────────┐
                    │  Go 服务端 │────▶│ 查看者网页  │
                    │  (Docker) │  WS │  (实时推送) │
                    └───────────┘     └────────────┘
```

## 快速开始（服务端）

### 方式一：Docker（推荐）

```bash
cd server
cp .env.example .env        # 编辑 REPORT_TOKEN，可加 VIEWER_PASSWORD
docker compose up -d --build
```

### 方式二：直接运行

```bash
cd server
REPORT_TOKEN=dev go run ./cmd/server
# 可选：VIEWER_PASSWORD=secret PORT=8080 IDLE_TIMEOUT=30s
```

打开 `http://<host>:8080` 即可看到查看页。

### 配置

| 环境变量 | 必填 | 说明 |
|---|---|---|
| `REPORT_TOKEN` | 是 | 设备上报鉴权 token，与客户端配置一致 |
| `VIEWER_PASSWORD` | 否 | 设置后查看网页需先输入密码（会话 24h） |
| `PORT` | 否 | 监听端口，默认 `8080` |
| `IDLE_TIMEOUT` | 否 | 超过该时长无上报判为离线，默认 `30s` |

## 客户端

### 桌面端（Windows / macOS / Linux）

Rust + Tauri 2.x，一套代码库。托盘图标控制共享开关，设置窗口配置服务端。

```bash
cd clients/desktop/src-tauri
cargo run          # 开发运行
cargo build        # 构建二进制
```

首次运行会自动弹出设置窗口：填服务端地址、设备名、设备 ID、上报 token，打开「共享前台状态」开关即可。

**打包分发产物**：

```bash
cd clients/desktop/src-tauri
npx tauri build --bundles appimage     # 产出单文件 .AppImage（含 webkit 运行时）
```

无头环境（无 FUSE）打包 AppImage 时需设置环境变量，并确保安装了 `patchelf`（Arch 可从
`pacman` 安装，或解包官方包到 `~/.local/bin`）：

```bash
export APPIMAGE_EXTRACT_AND_RUN=1 NO_STRIP=1
npx tauri build --bundles appimage
```

`.deb` 打包需要 `dpkg-deb`（Debian/Ubuntu 系自带）；Windows 的 `.msi`、macOS 的 `.dmg`
需在对应平台构建。跨平台一键出所有安装包见下方「CI 发布」说明。

### 安卓端（Android）

Kotlin + Jetpack Compose（Material 3 + MIUI X 视觉）。用 Android Studio 打开
`clients/android` 直接运行，或命令行构建：

```bash
cd clients/android
gradle wrapper      # 首次需系统安装 Gradle
./gradlew assembleDebug
```

安装后打开 App：允许「使用情况访问」（打开共享开关时会引导到系统设置页），填写
服务端地址与 token，打开共享。前台服务常驻通知栏持续上报。

## 已知平台限制

| 平台 | 说明 |
|---|---|
| **macOS** | 当前仅上报前台应用名。窗口标题需要「屏幕录制」权限 + CGWindowList API，未实现（见 `clients/desktop/src-tauri/src/foreground/macos.rs`）。 |
| **Linux** | Hyprland（`hyprctl`）、X11 完全支持；wlroots 系 Wayland 视合成器而定；**GNOME/KDE 纯 Wayland 无全局前台 API**，无法读取（会停止上报，保持上次状态）。 |
| **Android** | 只能读取前台**应用名**（包名→应用标签），无法读取窗口标题（如"和小明聊天"）——那需要无障碍服务，超出当前范围。需要「使用情况访问」特殊权限。 |
| **全部** | 手机锁屏/睡眠期间服务暂停；恢复后自动续报。 |

## 协议

客户端 ⇄ 服务端 JSON 协议见 [`docs/protocol.md`](docs/protocol.md)（唯一事实来源）。

## 目录结构

```
server/        Go 服务端（含内嵌网页、Docker）
clients/desktop/   Rust/Tauri 桌面客户端（Win/macOS/Linux）
clients/android/   Kotlin/Compose 安卓客户端
docs/protocol.md   共享协议
```

## 隐私说明

- 只传输前台应用名与窗口标题的**文字**，无截图、无音频。
- 数据仅存于服务端内存，重启即清空；不落盘。
- 页面访问可设置密码（`VIEWER_PASSWORD`）；设备上报用 `REPORT_TOKEN` 鉴权。
- 自托管，数据不出你的服务器。
