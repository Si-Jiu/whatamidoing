# whatamidoing

让朋友通过一个网站实时看到你**电脑/手机当前前台在干什么**。

设备上的客户端持续上报前台应用名 + 窗口标题到自托管服务端；朋友打开网页即可实时看到你在干什么。

```
┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│ Windows  │  │  macOS   │  │  Linux   │  │ Android  │
│ Rust/Tau │  │ Rust/Tau │  │ Rust/Tau │  │  Kotlin  │
└────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘
     └─────────────┴── HTTP ─────┴─────────────┘
                        │
                  ┌─────▼─────┐      ┌────────────┐
                  │ Go 服务端 │─────▶│ 查看者网页 │
                  │ (Docker)  │  WS  │ (实时推送) │
                  └───────────┘      └────────────┘
```

## 快速开始

### 服务端

#### Docker

复制 `server/docker-compose.yml` 以及 `server/.env.example` 到工作目录：

```bash
docker compose up -d
```

**首次打开网页会引导设置管理员密码**，需要**初始化令牌**——部署时通过 `SETUP_TOKEN`
环境变量设置（`docker-compose.yml` 已要求必填，`.env.example` 有示例）。之后在管理面板
（页面右上角「管理员」）里：添加设备（自动生成每设备 token，填到客户端配置）、设置
网页查看密码。数据（管理员、设备、token）持久化在 `./data/data.json`（compose 已挂载卷）。

> 若部署在反向代理（Cloudflare / Nginx / Caddy）之后，建议设置 `TRUSTED_PROXIES`
> 为代理网段（逗号分隔），登录限流才能正确识别访问者 IP。

打开 `http://<host>:8080` 即可看到查看页。

> 不想用 Docker？可从 [Release](https://github.com/Si-Jiu/whatamidoing/releases) 直接下载
> `whatamidoing-server-linux-amd64` / `whatamidoing-server-linux-arm64` 二进制，或见下方「自行编译」。

#### 配置

| 环境变量 | 必填 | 说明 |
| --- | --- | --- |
| `SETUP_TOKEN` | 首次初始化时 | 初始化管理员的令牌（首次打开网页输入用） |
| `DATA_FILE` | 否 | 持久化数据文件路径（管理员/设备/token），默认 `data.json`；Docker 内为 `/data/data.json` |
| `TRUSTED_PROXIES` | 否 | 可信反向代理 IP/CIDR（逗号分隔），设置后登录限流才信任 `X-Forwarded-For` |
| `PORT` | 否 | 监听端口，默认 `8080` |
| `IDLE_TIMEOUT` | 否 | 超过该时长无上报判为离线，默认 `30s` |

> 设备上报 token、网页查看密码不再用环境变量，统一在管理面板配置。

### 客户端

#### 安装

前往 [Releases](https://github.com/Si-Jiu/whatamidoing/releases) 页面下载对应平台的安装包

| 平台 | 安装包 |
| --- | --- |
| **Windows** | `whatamidoing_<ver>_x64-setup.exe`（安装向导） |
| **macOS** | `whatamidoing_<ver>_aarch64.dmg`（Intel 需自行编译） |
| **Linux** | `whatamidoing_<ver>_amd64.deb`或 `whatamidoing_<ver>_amd64.AppImage` |
| **Android** | `app-release.apk`（直接安装即可） |

#### 首次使用

- **桌面端**（Windows / macOS / Linux）：托盘图标控制共享开关。
  首次运行会自动弹出设置窗口：填服务端地址、设备名、设备 ID、上报 token，
  打开「共享前台状态」开关即可。

- **安卓端**（Android）：安装后打开 App：允许「使用情况访问」
  （打开共享开关时会引导到系统设置页），填写服务端地址与 token，打开共享。
  前台服务常驻通知栏持续上报。

#### 进程名映射表

检测到的前台应用名会按规则映射成友好显示名（如 `kitty` → 终端、
`chrome` → 浏览器）。内置一套默认映射表；要自定义，把
`dist/linux/rules.example.json` 复制为 `~/.config/whatamidoing/rules.json` 并按需修改
（存在该文件时完全以其为准）。支持四种匹配类型，按顺序先命中先用：

| `match_type` | 含义 | 示例 |
| --- | --- | --- |
| `exact` | 完全相等（不区分大小写） | `"kitty"` |
| `prefix` | 前缀匹配 | `"com.tencent"` |
| `contains` | 包含子串 | `"chrome"` → 匹配 `google-chrome`/`Google Chrome` |
| `regex` | 正则表达式 | `"^chrom.*"` |

## 自行编译

### 服务端（Go）

```bash
cd server
go build ./cmd/server           # 产出二进制
REPORT_TOKEN="YOUR_TOKEN" go run ./cmd/server   # 或直接开发运行
# 可选：VIEWER_PASSWORD="YOUR_PASSWORD" PORT=8080 IDLE_TIMEOUT=30s
```

需要 Go 1.26+。交叉编译（在任意平台出 Linux 产物）：

```bash
cd server
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath ./cmd/server
```

### 桌面端（Rust + Tauri）

```bash
cd clients/desktop/src-tauri
cargo run          # 开发运行
cargo build        # 构建调试二进制
npx tauri build    # 产出当前平台安装包
```

打包分发产物：

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
需在对应平台构建。跨平台一键出所有安装包见 `.github/workflows/release.yml`（CI 发布说明）。

### 安卓端（Android）

Kotlin + Jetpack Compose。用 Android Studio 打开
`clients/android` 直接运行，或命令行构建：

```bash
cd clients/android
./gradlew assembleDebug
```

**主题自适应**：自动检测 HyperOS / MIUI（小米/红米/POCO），是则用
[Miuix](https://github.com/compose-miuix-ui/miuix) 呈现 MIUI 原生视觉，否则 Material
Design 3。Miuix 要求 AGP 9 / compileSdk 37，构建需较新工具链（见
`clients/android/build.gradle.kts`）。

## 已知平台限制

| 平台 | 说明 |
| --- | --- |
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
