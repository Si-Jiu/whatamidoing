/* whatamidoing viewer SPA — 原生 JS，无框架 */
/* 含：首次初始化管理员、管理面板（设备/网页密码）、查看端实时状态 */

const $ = (id) => document.getElementById(id);

const viewSetup = $("view-setup");
const viewAdminLogin = $("view-admin-login");
const viewAdmin = $("view-admin");
const viewLogin = $("view-login");
const viewDevices = $("view-devices");
const adminBtn = $("admin-btn");
const empty = $("empty");
const devicesEl = $("devices");

/** device_id → rendered card element */
const cards = new Map();
/** last snapshot for duration ticking */
let lastDevices = [];

/* ---------- 视图切换 ---------- */

function hideAll() {
  [viewSetup, viewAdminLogin, viewAdmin, viewLogin, viewDevices].forEach((v) => {
    v.hidden = true;
  });
}
const showSetup = () => { hideAll(); viewSetup.hidden = false; };
const showAdminLogin = () => { hideAll(); viewAdminLogin.hidden = false; };
const showAdminPanel = () => { hideAll(); viewAdmin.hidden = false; };
const showViewerLogin = () => { hideAll(); viewLogin.hidden = false; };
function showDevices() {
  hideAll();
  viewDevices.hidden = false;
  adminBtn.hidden = false;
  refreshPresence();
}

/* ---------- 平台/发行版图标（SVG 文件在 assets/ 下） ---------- */

const PLATFORM_META = {
  windows: { label: "Windows", icon: "platform/windows" },
  macos: { label: "macOS", icon: "platform/macos" },
  linux: { label: "Linux", icon: "platform/linux" },
  android: { label: "Android", icon: "platform/android" },
};

const DISTRO_META = {
  alpinelinux: { label: "Alpine Linux", icon: "distro/alpinelinux" },
  archlinux: { label: "Arch Linux", icon: "distro/archlinux" },
  cachyos: { label: "CachyOS", icon: "distro/cachyos" },
  centos: { label: "CentOS", icon: "distro/centos" },
  debian: { label: "Debian", icon: "distro/debian" },
  deepin: { label: "deepin", icon: "distro/deepin" },
  fedora: { label: "Fedora", icon: "distro/fedora" },
  linuxmint: { label: "Linux Mint", icon: "distro/linuxmint" },
  manjaro: { label: "Manjaro", icon: "distro/manjaro" },
  nixos: { label: "NixOS", icon: "distro/nixos" },
  opensuse: { label: "openSUSE", icon: "distro/opensuse" },
  popos: { label: "Pop!_OS", icon: "distro/popos" },
  redhat: { label: "Red Hat", icon: "distro/redhat" },
  rockylinux: { label: "Rocky Linux", icon: "distro/rockylinux" },
  steamos: { label: "SteamOS", icon: "distro/steamos" },
  ubuntu: { label: "Ubuntu", icon: "distro/ubuntu" },
};

function platformMeta(p) {
  return PLATFORM_META[p] || { label: p || "其他", icon: "platform/custom" };
}

function distroMeta(d) {
  return DISTRO_META[d] || (d ? { label: d, icon: "platform/linux" } : null);
}

/** 平台显示名；Linux 且登记了发行版时显示"Linux · 发行版" */
function platformLabel(d) {
  const base = platformMeta(d.platform).label;
  if (d.platform === "linux" && d.distro) {
    const dm = distroMeta(d.distro);
    return dm ? `${base} · ${dm.label}` : `${base} · ${d.distro}`;
  }
  return base;
}

/** 平台 logo（mask 着色：颜色跟随 currentColor，图标文件在 assets/）。
 *  Linux 且登记了发行版时用发行版图标。 */
function platformLogo(d) {
  const distro = d.platform === "linux" && d.distro ? distroMeta(d.distro) : null;
  const icon = distro ? distro.icon : platformMeta(d.platform).icon;
  return `<span class="device__logo" style="--platform-logo:url('assets/${icon}.svg')" aria-hidden="true"></span>`;
}

/* ---------- 时长格式化 ---------- */

function formatDuration(startedAt, now) {
  if (!startedAt) return "";
  const secs = Math.max(0, Math.floor((now - Date.parse(startedAt)) / 1000));
  if (secs < 60) return `刚刚开始 · ${secs} 秒`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `已持续 ${mins} 分钟`;
  const h = Math.floor(mins / 60);
  const m = mins % 60;
  return m ? `已持续 ${h} 小时 ${m} 分钟` : `已持续 ${h} 小时`;
}

/* ---------- 设备渲染 ---------- */

function renderSnapshot(devices) {
  lastDevices = devices;
  const seen = new Set();
  for (const d of devices) {
    seen.add(d.device_id);
    const card = cards.get(d.device_id);
    if (card) {
      updateCard(card, d);
    } else {
      cards.set(d.device_id, createCard(d));
    }
  }
  for (const id of [...cards.keys()]) {
    if (!seen.has(id)) {
      cards.get(id).remove();
      cards.delete(id);
    }
  }
  empty.hidden = cards.size > 0;
  devicesEl.hidden = cards.size === 0;
}

function createCard(d) {
  const card = document.createElement("article");
  card.className = "md-card device";
  devicesEl.appendChild(card);
  updateCard(card, d);
  return card;
}

function updateCard(card, d) {
  card.className = `md-card device${d.online ? "" : " device--offline"}`;
  card.innerHTML = `
    <div class="device__header">
      <span class="device__logo-wrap">${platformLogo(d)}</span>
      <span class="device__name"></span>
      <span class="device__platform"></span>
      <span class="device__status"><span class="status-dot"></span><span class="status-label"></span></span>
    </div>
    <div class="device__app"></div>
    <div class="device__appid"></div>
    <div class="device__window"></div>
    <div class="device__meta"></div>
  `;
  card.querySelector(".device__name").textContent = d.device_name || "未命名设备";
  const plat = card.querySelector(".device__platform");
  plat.textContent = platformLabel(d);
  const dot = card.querySelector(".status-dot");
  dot.classList.toggle("status-dot--online", d.online);
  card.querySelector(".status-label").textContent = d.online ? "在线" : "离线";
  card.querySelector(".device__app").textContent = d.online ? d.app || d.app_id : "";
  card.querySelector(".device__appid").textContent = d.online ? ((d.app === d.app_id) ? "" : d.app_id) : "";
  card.querySelector(".device__window").textContent = d.online ? d.window_title || "" : "";
  const metaEl = card.querySelector(".device__meta");
  metaEl.innerHTML = "";
  if (d.online) {
    const since = document.createElement("span");
    since.className = "duration";
    since.textContent = formatDuration(d.app_started_at, Date.now());
    metaEl.appendChild(since);
  } else {
    const last = document.createElement("span");
    last.textContent = d.last_seen ? lastSeenText(d.last_seen) : "尚未上报";
    metaEl.appendChild(last);
  }
}

function lastSeenText(lastSeen) {
  const t = Date.parse(lastSeen);
  if (!t || isNaN(t)) return "尚未上报";
  const mins = Math.max(1, Math.round((Date.now() - t) / 60000));
  return mins < 60 ? `${mins} 分钟前离线` : `${Math.floor(mins / 60)} 小时前离线`;
}

setInterval(() => {
  for (const d of lastDevices) {
    if (d.online && cards.has(d.device_id)) {
      const meta = cards.get(d.device_id).querySelector(".device__meta .duration");
      if (meta) meta.textContent = formatDuration(d.app_started_at, Date.now());
    }
  }
}, 30_000);

/* ---------- 查看者登录 ---------- */

$("login-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const err = $("login-error");
  err.hidden = true;

  let res;
  try {
    res = await fetch("/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ password: $("password").value }),
    });
  } catch {
    err.textContent = "无法连接服务器";
    err.hidden = false;
    return;
  }
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    err.textContent = data.error || "登录失败";
    err.hidden = false;
    return;
  }

  $("password").value = "";
  showDevices();
  connectWS();
  try {
    await loadState();
  } catch {
    startPolling();
  }
});

/* ---------- 管理员：初始化 / 登录 ---------- */

$("setup-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const err = $("setup-error");
  err.hidden = true;
  const res = await fetch("/api/admin/setup", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      setup_token: $("setup-token").value,
      password: $("setup-password").value,
    }),
  });
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    err.textContent = data.error || "初始化失败";
    err.hidden = false;
    return;
  }
  $("setup-password").value = "";
  await openAdminPanel();
});

$("admin-login-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const err = $("admin-login-error");
  err.hidden = true;
  const res = await fetch("/api/admin/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password: $("admin-password").value }),
  });
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    err.textContent = data.error || "登录失败";
    err.hidden = false;
    return;
  }
  $("admin-password").value = "";
  await openAdminPanel();
});

adminBtn.addEventListener("click", async () => {
  try {
    const status = await (await fetch("/api/admin/status")).json();
    if (!status.initialized) {
      showSetup();
      return;
    }
    await openAdminPanel();
  } catch {
    showAdminLogin();
  }
});

$("admin-back").addEventListener("click", async () => {
  try {
    await loadState();
    showDevices();
    connectWS();
  } catch {
    showViewerLogin();
  }
});

/* ---------- 管理面板 ---------- */

async function openAdminPanel() {
  const res = await fetch("/api/admin/devices");
  if (res.status === 401) {
    showAdminLogin();
    return;
  }
  renderAdminDevices((await res.json()).devices || []);
  // 加载离线阈值设置（持久化值或环境变量默认）
  try {
    const s = await (await fetch("/api/admin/settings")).json();
    $("idle-timeout").value = s.idle_timeout_secs ?? "";
  } catch { /* 保留空值 */ }
  loadMappings();
  showAdminPanel();
}

/* ---------- 进程映射管理（app_id → 显示名） ---------- */

async function loadMappings() {
  const res = await fetch("/api/admin/mappings");
  if (!res.ok) return;
  const { mappings } = await res.json();
  const list = $("map-list");
  list.innerHTML = "";
  const entries = Object.entries(mappings || {}).sort((a, b) => a[0].localeCompare(b[0]));
  $("map-empty").hidden = entries.length > 0;
  for (const [id, name] of entries) {
    const row = document.createElement("div");
    row.className = "map-row";
    row.innerHTML = `
      <span class="map-row__id"></span>
      <span class="map-row__arrow">→</span>
      <span class="map-row__name"></span>
      <button class="md-button md-button--text map-del">删除</button>`;
    row.querySelector(".map-row__id").textContent = id;
    row.querySelector(".map-row__name").textContent = name;
    row.querySelector(".map-del").addEventListener("click", async () => {
      await fetch(`/api/admin/mappings/${encodeURIComponent(id)}`, { method: "DELETE" });
      loadMappings();
    });
    list.appendChild(row);
  }
}

$("map-add").addEventListener("click", async () => {
  const appId = $("map-appid").value.trim();
  const name = $("map-name").value.trim();
  if (!appId || !name) return;
  const res = await fetch("/api/admin/mappings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ app_id: appId, name }),
  });
  if (res.ok) {
    $("map-appid").value = "";
    $("map-name").value = "";
    loadMappings();
  }
});

$("map-export").addEventListener("click", async () => {
  const res = await fetch("/api/admin/mappings/export");
  if (!res.ok) return;
  const blob = await res.blob();
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = "waid-mappings.json";
  a.click();
  URL.revokeObjectURL(a.href);
});

$("map-import").addEventListener("click", () => $("map-import-file").click());
$("map-import-file").addEventListener("change", async (e) => {
  const file = e.target.files[0];
  e.target.value = "";
  if (!file) return;
  const text = await file.text();
  let data;
  try { data = JSON.parse(text); } catch { return; }
  const res = await fetch("/api/admin/mappings/import", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (res.ok) loadMappings();
});

function renderAdminDevices(devices) {
  const list = $("admin-devices");
  list.innerHTML = "";
  $("admin-empty").hidden = devices.length > 0;
  for (const d of devices) {
    const row = document.createElement("div");
    row.className = "admin-device";
    row.innerHTML = `
      <div class="admin-device__info">
        <div class="admin-device__name"></div>
        <div class="admin-device__token"></div>
      </div>
      <div class="admin-device__actions">
        <button class="md-button md-button--text copy-token">复制 token</button>
        <button class="md-button md-button--text delete-device">删除</button>
      </div>`;
    const nameEl = row.querySelector(".admin-device__name");
    nameEl.textContent = d.name;
    if (d.platform) {
      const badge = document.createElement("span");
      badge.className = "admin-device__platform";
      badge.textContent = platformLabel(d);
      nameEl.appendChild(badge);
    }
    row.querySelector(".admin-device__token").textContent = d.token;
    row.querySelector(".copy-token").addEventListener("click", () => {
      navigator.clipboard.writeText(d.token).then(() => {
        row.querySelector(".copy-token").textContent = "已复制 ✓";
        setTimeout(() => (row.querySelector(".copy-token").textContent = "复制 token"), 1200);
      });
    });
    row.querySelector(".delete-device").addEventListener("click", async () => {
      if (!confirm(`删除设备「${d.name}」？其 token 将失效。`)) return;
      await fetch(`/api/admin/devices/${encodeURIComponent(d.id)}`, { method: "DELETE" });
      openAdminPanel();
    });
    list.appendChild(row);
  }
}

/** 用 meta 表填充 select 的 option（数据驱动，不在 HTML 写死）；extra 追加末尾选项 */
function populateSelect(id, meta, extra) {
  const sel = $(id);
  sel.innerHTML = "";
  for (const [value, m] of Object.entries(meta)) {
    const opt = document.createElement("option");
    opt.value = value;
    opt.textContent = m.label;
    sel.appendChild(opt);
  }
  if (extra) {
    const opt = document.createElement("option");
    opt.value = "";
    opt.textContent = extra;
    sel.appendChild(opt);
  }
}

populateSelect("new-device-platform", PLATFORM_META);
populateSelect("new-device-distro", DISTRO_META, "自定义");
$("new-device-platform").value = "linux"; // 默认选中 Linux

function syncPlatformFields() {
  const isLinux = $("new-device-platform").value === "linux";
  $("distro-wrap").hidden = !isLinux;
  // 仅 Linux 且发行版选了"自定义"时显示自定义输入框
  $("distro-custom-wrap").hidden = !(isLinux && $("new-device-distro").value === "");
}
$("new-device-platform").addEventListener("change", syncPlatformFields);
$("new-device-distro").addEventListener("change", syncPlatformFields);
syncPlatformFields();

$("add-device").addEventListener("click", async () => {
  const name = $("new-device-name").value.trim();
  if (!name) return;
  const platform = $("new-device-platform").value;
  let distro = "";
  if (platform === "linux") {
    const v = $("new-device-distro").value;
    distro = v === "" ? $("new-device-distro-custom").value.trim() : v;
    if (!distro) return;
  }
  const res = await fetch("/api/admin/devices", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, platform, distro }),
  });
  if (res.ok) {
    $("new-device-name").value = "";
    $("new-device-distro-custom").value = "";
    openAdminPanel();
  }
});
$("new-device-name").addEventListener("keydown", (e) => {
  if (e.key === "Enter") $("add-device").click();
});

$("viewer-pw-save").addEventListener("click", async () => {
  const res = await fetch("/api/admin/viewer-password", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password: $("viewer-password").value }),
  });
  if (res.ok) {
    const hint = $("viewer-pw-hint");
    hint.hidden = false;
    setTimeout(() => (hint.hidden = true), 2000);
  }
});

$("idle-save").addEventListener("click", async () => {
  const secs = parseInt($("idle-timeout").value, 10);
  if (!secs || secs < 5 || secs > 3600) return;
  const res = await fetch("/api/admin/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ idle_timeout_secs: secs }),
  });
  if (res.ok) {
    const hint = $("idle-hint");
    hint.hidden = false;
    setTimeout(() => (hint.hidden = true), 2000);
  }
});

/* ---------- 数据加载 ---------- */

async function loadState() {
  const res = await fetch("/api/v1/state");
  if (res.status === 401) throw new Error("unauthorized");
  const data = await res.json();
  renderSnapshot(data.devices || []);
}

async function init() {
  let status;
  try {
    status = await (await fetch("/api/admin/status")).json();
  } catch {
    showSetup();
    return;
  }
  if (!status.initialized) {
    showSetup();
    return;
  }
  try {
    await loadState();
    showDevices();
    connectWS();
  } catch {
    showViewerLogin();
  }
}

/* ---------- WebSocket（失败时退化为轮询） ---------- */

let ws = null;
let wsTries = 0;
let pollTimer = null;

function connectWS() {
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return;
  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  ws = new WebSocket(`${proto}//${location.host}/ws`);

  ws.onopen = () => {
    wsTries = 0;
    stopPolling();
  };

  ws.onmessage = (ev) => {
    let msg;
    try {
      msg = JSON.parse(ev.data);
    } catch {
      return;
    }
    if (msg.type === "state" && Array.isArray(msg.devices)) {
      renderSnapshot(msg.devices);
    } else if (msg.type === "update" && msg.device) {
      mergeUpdate(msg.device);
    }
  };

  ws.onclose = () => {
    ws = null;
    startPolling();
    const delay = Math.min(30, 1.5 ** wsTries) * 1000;
    wsTries++;
    setTimeout(connectWS, delay);
  };

  ws.onerror = () => ws.close();
}

function mergeUpdate(device) {
  const idx = lastDevices.findIndex((d) => d.device_id === device.device_id);
  if (idx >= 0) {
    lastDevices[idx] = device;
  } else {
    lastDevices.push(device);
  }
  renderSnapshot(lastDevices);
}

function startPolling() {
  if (pollTimer) return;
  pollTimer = setInterval(async () => {
    try {
      await loadState();
    } catch {
      /* keep polling */
    }
  }, 5_000);
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = null;
  }
}

/* ---------- 在线查看者计数 ---------- */

// 每个查看标签页一个独立 id（sessionStorage，刷新不变、关闭重开新 id），
// 服务端据此统计"有多少人在查看"（同 IP 多开也算多个查看者）。
function viewerId() {
  let id = sessionStorage.getItem("waid_viewer_id");
  if (!id) {
    id = Math.random().toString(36).slice(2) + Math.random().toString(36).slice(2);
    sessionStorage.setItem("waid_viewer_id", id);
  }
  return id;
}

async function refreshPresence() {
  try {
    const res = await fetch(`/api/v1/presence?id=${viewerId()}`, { cache: "no-store" });
    if (!res.ok) return;
    const n = (await res.json()).viewers || 0;
    const text = `${n} 人正在查看`;
    $("viewer-count").textContent = text;
    $("viewer-count").hidden = false;
  } catch { /* 未登录或网络错误，保持隐藏 */ }
}
setInterval(refreshPresence, 15_000);
refreshPresence();

init();
