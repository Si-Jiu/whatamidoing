# whatamidoing 共享协议

本文件是客户端 ⇄ 服务端之间唯一的协议事实来源。三个客户端（Rust 桌面、Kotlin 安卓）
与服务端（Go）各自镜像本文定义的 JSON 结构。

## 枚举

- `platform`：`windows` | `macos` | `linux` | `android`
- 时间戳一律使用 **RFC 3339**（如 `2026-08-08T09:30:00Z`）。

## 鉴权模型

- **设备**：每台设备一个 token，由**管理面板**添加设备时自动生成。上报时用
  `Authorization: Bearer <设备 token>`。设备身份以注册的 token 为准，上报体里的
  `device_id` / `device_name` 仅作信息参考、被服务端忽略。
- **管理员**：首次使用需在网页设置管理员密码（持久化）。管理员登录后可管理设备、
  设置网页查看密码。
- **查看者**：可设网页查看密码（管理面板中设置）；不设则页面免密。

## 设备 → 服务端：上报前台状态

```
POST /api/v1/report
Authorization: Bearer <设备 token>
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
| `device_id` / `device_name` | 否 | 仅作参考，被服务端忽略（身份由 token 决定）。 |
| `platform` | 是 | 见枚举。 |
| `app` | 是 | 前台应用显示名。 |
| `window_title` | 否 | 窗口标题；**Android 恒为空**（无权限读取，见 README 限制）。 |
| `app_started_at` | 否 | 当前前台应用开始时刻；缺省视为上报时刻。 |

语义：

- 客户端**每 5 秒**兜底上报一次（幂等，重复内容无副作用）；**前台应用切换时立即上报**。
- 响应：`204 No Content`；`401`（token 无效）、`400`（字段缺失/非法）。

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
      "device_id":   "dev_xxx",
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

- 返回**管理面板注册的全部设备**；未上报过的设备显示 `online: false`、`app` 为空。
- `online`：`last_seen` 距今超过 `IDLE_TIMEOUT`（默认 30s）即为 `false`。
- 未设置网页密码时无需认证；设置后需要 `viewer_session` cookie（见登录）。

### 登录（仅当设置了网页密码）

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

需 `viewer_session` cookie（设置了密码时）。

- 连接建立后，服务端先推一条**全量快照**：

```json
{"type": "state", "devices": [ ... ]}
```

- 此后每次设备状态变更，推送**增量**：

```json
{"type": "update", "device": { ...同 state 中单设备结构... }}
```

- 查看端在 WS 断开时**退化为 5 秒轮询** `/api/v1/state`。

## 管理员 API（网页管理面板）

管理员会话 cookie：`admin_session`。除 `status` / `setup` 外均需管理员登录。

| 方法 & 路径 | 说明 |
|---|---|
| `GET /api/admin/status` | 返回 `{"initialized": bool}`（是否已设管理员）。 |
| `POST /api/admin/setup` | 首次初始化 `{"setup_token":"...","password":"..."}`（令牌见服务端启动日志，或 `SETUP_TOKEN` 环境变量）；成功后写入 `admin_session`。已初始化返回 `409`。 |
| `POST /api/admin/login` | `{"password":"..."}` → `admin_session`。 |
| `GET /api/admin/devices` | 设备列表 `{"devices":[{id,name,token}]}`。 |
| `POST /api/admin/devices` | 添加设备 `{"name":"..."}` → 返回新设备（含自动生成的 `token`）。 |
| `DELETE /api/admin/devices/{id}` | 删除设备（其 token 即失效）。 |
| `POST /api/admin/viewer-password` | 设置网页查看密码 `{"password":"..."}`；空字符串清除（免密）。 |

## 错误

所有 API 错误统一返回：

```json
{"error": "描述"}
```

HTTP 状态码：`400`（请求非法）、`401`（未认证/token 错）、`404`（不存在）、`405`（方法不允许）、`409`（冲突）。
