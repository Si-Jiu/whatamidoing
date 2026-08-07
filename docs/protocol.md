# whatamidoing 共享协议

本文件是客户端 ⇄ 服务端之间唯一的协议事实来源。三个客户端（Rust 桌面、Kotlin 安卓）
与服务端（Go）各自镜像本文定义的 JSON 结构。

## 枚举

- `platform`：`windows` | `macos` | `linux` | `android`
- 时间戳一律使用 **RFC 3339**（如 `2026-08-08T09:30:00Z`）。

## 设备 → 服务端：上报前台状态

```
POST /api/v1/report
Authorization: Bearer <REPORT_TOKEN>
Content-Type: application/json
```

请求体：

```json
{
  "device_id":     "desktop-main",
  "device_name":   "我的电脑",
  "platform":      "windows",
  "app":           "Google Chrome",
  "window_title":  "GitHub · whatamidoing",
  "app_started_at": "2026-08-08T09:30:00Z"
}
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `device_id` | 是 | 设备唯一 ID（客户端配置，如 `desktop-main` / `phone`）。同 ID 上报视为同一设备。 |
| `device_name` | 是 | 展示用名称。 |
| `platform` | 是 | 见枚举。 |
| `app` | 是 | 前台应用显示名。 |
| `window_title` | 否 | 窗口标题；**Android 恒为空**（无权限读取，见 README 限制）。 |
| `app_started_at` | 否 | 当前前台应用开始时刻；缺省视为上报时刻。 |

语义：

- 客户端**每 5 秒**兜底上报一次（幂等，重复内容无副作用）；**前台应用切换时立即上报**。
- 响应：`204 No Content`；`401`（token 错）、`400`（字段缺失/非法）。

## 查看者 → 服务端

### 获取全量状态

```
GET /api/v1/state
```

响应 `200`：

```json
{
  "devices": [
    {
      "device_id":   "desktop-main",
      "device_name": "我的电脑",
      "platform":    "windows",
      "app":         "Google Chrome",
      "window_title":"GitHub · whatamidoing",
      "app_started_at": "2026-08-08T09:30:00Z",
      "last_seen":   "2026-08-08T09:35:00Z",
      "online":      true
    }
  ]
}
```

- `online`：`last_seen` 距今超过 `IDLE_TIMEOUT`（默认 30s）即为 `false`。
- 未配置 `VIEWER_PASSWORD` 时无需认证；配置后需要 `viewer_session` cookie（见登录）。

### 登录（仅当配置了 `VIEWER_PASSWORD`）

```
POST /login
Content-Type: application/json
{"password": "..."}
```

- `200`：设置 `viewer_session` cookie（有效期 24h），返回 `{"ok":true}`。
- `401`：密码错误。

### 实时推送（WebSocket）

```
WS /ws
```

需 `viewer_session` cookie（配置了密码时）。

- 连接建立后，服务端先推一条**全量快照**：

```json
{"type": "state", "devices": [ ... ]}
```

- 此后每次设备状态变更，推送**增量**：

```json
{"type": "update", "device": { ...同 state 中单设备结构... }}
```

- 查看端在 WS 断开时**退化为 5 秒轮询** `/api/v1/state`。

## 错误

所有 API 错误统一返回：

```json
{"error": "描述"}
```

HTTP 状态码：`400`（请求非法）、`401`（未认证/token 错）、`404`（不存在）、`405`（方法不允许）。
