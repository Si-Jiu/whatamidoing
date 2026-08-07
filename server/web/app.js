
const $ = (id) => document.getElementById(id);

const viewLogin = $("view-login");
const viewDevices = $("view-devices");
const empty = $("empty");
const devicesEl = $("devices");

/** device_id → rendered card element */
const cards = new Map();
/** last snapshot for duration ticking */
let lastDevices = [];

/* ---------- 平台图标 ---------- */

const PLATFORM_META = {
  windows: { label: "Windows", icon: "🪟" },
  macos: { label: "macOS", icon: "" },
  linux: { label: "Linux", icon: "🐧" },
  android: { label: "Android", icon: "🤖" },
};

function platformMeta(p) {
  return PLATFORM_META[p] || { label: p, icon: "📟" };
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

/* ---------- 渲染 ---------- */

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
  const meta = platformMeta(d.platform);
  card.className = `md-card device${d.online ? "" : " device--offline"}`;
  card.innerHTML = `
    <div class="device__header">
      <span class="device__name"></span>
      <span class="device__platform"></span>
      <span class="device__status"><span class="status-dot"></span><span class="status-label"></span></span>
    </div>
    <div class="device__app"></div>
    <div class="device__window"></div>
    <div class="device__meta"></div>
  `;
  card.querySelector(".device__name").textContent = d.device_name || "未命名设备";
  const plat = card.querySelector(".device__platform");
  plat.textContent = `${meta.icon} ${meta.label}`;
  const dot = card.querySelector(".status-dot");
  dot.classList.toggle("status-dot--online", d.online);
  card.querySelector(".status-label").textContent = d.online ? "在线" : "离线";
  card.querySelector(".device__app").textContent = d.app || "未知";
  card.querySelector(".device__window").textContent = d.window_title || "";
  const metaEl = card.querySelector(".device__meta");
  metaEl.innerHTML = "";
  if (d.online) {
    const since = document.createElement("span");
    since.className = "duration";
    since.textContent = formatDuration(d.app_started_at, Date.now());
    metaEl.appendChild(since);
  } else {
    const last = document.createElement("span");
    last.textContent = lastSeenText(d.last_seen);
    metaEl.appendChild(last);
  }
}

function lastSeenText(lastSeen) {
  const mins = Math.max(1, Math.round((Date.now() - Date.parse(lastSeen)) / 60000));
  return mins < 60 ? `${mins} 分钟前离线` : `${Math.floor(mins / 60)} 小时前离线`;
}

/* 每 30 秒刷新一次"已持续"计数 */
setInterval(() => {
  for (const d of lastDevices) {
    if (d.online && cards.has(d.device_id)) {
      const meta = cards.get(d.device_id).querySelector(".device__meta .duration");
      if (meta) meta.textContent = formatDuration(d.app_started_at, Date.now());
    }
  }
}, 30_000);

/* ---------- 登录 ---------- */

function showLogin() {
  viewDevices.hidden = true;
  viewLogin.hidden = false;
  $("password").focus();
}

function showDevices() {
  viewLogin.hidden = true;
  viewDevices.hidden = false;
}

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
  } catch (e2) {
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

  // 登录成功：立即隐藏密码框并清空密码，再加载状态。
  // 状态加载失败也靠轮询重试，不再退回登录框。
  $("password").value = "";
  showDevices();
  connectWS();
  try {
    await loadState();
  } catch {
    startPolling();
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
  try {
    await loadState();
    showDevices();
    connectWS();
  } catch (e) {
    if (e.message === "unauthorized") showLogin();
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
    // 指数退避重连
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

init();
