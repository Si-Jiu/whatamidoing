// whatamidoing 设置窗口 — Tauri 2 后端命令调用
const invoke = window.__TAURI_INTERNALS__.invoke;
const $ = (id) => document.getElementById(id);

async function load() {
  const cfg = await invoke("get_config");
  $("enabled").checked = cfg.enabled;
  $("server_url").value = cfg.server_url || "";
  $("device_name").value = cfg.device_name || "";
  $("device_id").value = cfg.device_id || "";
  $("token").value = cfg.token || "";
  $("interval_secs").value = cfg.interval_secs || 5;
}

$("settings-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const err = $("error");
  err.hidden = true;
  $("saved-hint").hidden = true;

  const cfg = {
    enabled: $("enabled").checked,
    server_url: $("server_url").value.trim(),
    device_name: $("device_name").value.trim(),
    device_id: $("device_id").value.trim(),
    token: $("token").value.trim(),
    interval_secs: parseInt($("interval_secs").value, 10) || 5,
  };
  try {
    await invoke("save_config", { cfg });
    $("saved-hint").hidden = false;
  } catch (ex) {
    err.textContent = String(ex);
    err.hidden = false;
  }
});

// 开关即时生效
$("enabled").addEventListener("change", async () => {
  try {
    await invoke("set_sharing", { enabled: $("enabled").checked });
  } catch (ex) {
    console.error("set_sharing failed:", ex);
  }
});

load();
