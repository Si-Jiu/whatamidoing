mod config;
mod foreground;
mod reporter;

use std::sync::{Arc, Mutex};

use tauri::menu::{Menu, MenuItem};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::{AppHandle, Manager};

pub struct AppState {
    cfg: Arc<Mutex<config::ClientConfig>>,
}

/// Rebuild the tray menu so the toggle item reflects the current sharing state.
fn rebuild_tray_menu(app: &AppHandle, enabled: bool) {
    let Some(tray) = app.tray_by_id("main") else { return };
    let toggle_text = if enabled { "停止共享" } else { "开始共享" };
    let Ok(toggle) = MenuItem::with_id(app, "toggle", toggle_text, true, None::<&str>) else { return };
    let Ok(settings) = MenuItem::with_id(app, "settings", "打开设置", true, None::<&str>) else { return };
    let Ok(quit) = MenuItem::with_id(app, "quit", "退出", true, None::<&str>) else { return };
    if let Ok(menu) = Menu::with_items(app, &[&toggle, &settings, &quit]) {
        let _ = tray.set_menu(Some(menu));
    }
}

#[tauri::command]
fn get_config(state: tauri::State<'_, AppState>) -> config::ClientConfig {
    state.cfg.lock().map(|g| g.clone()).unwrap_or_default()
}

#[tauri::command]
fn save_config(app: AppHandle, state: tauri::State<'_, AppState>, cfg: config::ClientConfig) -> Result<(), String> {
    *state.cfg.lock().map_err(|_| "配置锁被污染")? = cfg.clone();
    config::save(&cfg).map_err(|e| e.to_string())?;
    rebuild_tray_menu(&app, cfg.enabled);
    Ok(())
}

#[tauri::command]
fn set_sharing(app: AppHandle, state: tauri::State<'_, AppState>, enabled: bool) -> Result<(), String> {
    let mut c = state.cfg.lock().map_err(|_| "配置锁被污染")?.clone();
    c.enabled = enabled;
    *state.cfg.lock().map_err(|_| "配置锁被污染")? = c.clone();
    config::save(&c).map_err(|e| e.to_string())?;
    rebuild_tray_menu(&app, enabled);
    Ok(())
}

pub fn run() {
    env_logger::init();

    let cfg = Arc::new(Mutex::new(config::load()));

    // Background reporter thread — started once, toggled via cfg.enabled.
    let worker_cfg = Arc::clone(&cfg);
    std::thread::spawn(move || reporter::run(worker_cfg));

    tauri::Builder::default()
        .manage(AppState { cfg: cfg.clone() })
        .invoke_handler(tauri::generate_handler![get_config, save_config, set_sharing])
        .setup(move |app| {
            // Settings window, hidden until requested.
            let settings = tauri::WebviewWindowBuilder::new(
                app,
                "settings",
                tauri::WebviewUrl::App("index.html".into()),
            )
            .title("whatamidoing 设置")
            .inner_size(460.0, 720.0)
            .resizable(false)
            .build()?;
            settings.hide()?;

            // First run (no server configured): show settings.
            let first_run = {
                let c = cfg.lock().unwrap();
                c.server_url.is_empty() || c.token.is_empty()
            };
            if first_run {
                let _ = settings.show();
                let _ = settings.set_focus();
            }

            // Tray.
            let toggle = MenuItem::with_id(app, "toggle", "开始共享", true, None::<&str>)?;
            let settings_item = MenuItem::with_id(app, "settings", "打开设置", true, None::<&str>)?;
            let quit = MenuItem::with_id(app, "quit", "退出", true, None::<&str>)?;
            let menu = Menu::with_items(app, &[&toggle, &settings_item, &quit])?;

            let _tray = TrayIconBuilder::with_id("main")
                .icon(app.default_window_icon().unwrap().clone())
                .menu(&menu)
                .show_menu_on_left_click(false)
                .on_tray_icon_event(|tray, event| {
                    if let TrayIconEvent::Click {
                        button: MouseButton::Left,
                        button_state: MouseButtonState::Up,
                        ..
                    } = event
                    {
                        show_settings(tray.app_handle());
                    }
                })
                .on_menu_event(|app, event| match event.id.as_ref() {
                    "toggle" => {
                        let enabled = {
                            let state = app.state::<AppState>();
                            let mut g = state.cfg.lock().unwrap();
                            g.enabled = !g.enabled;
                            let enabled = g.enabled;
                            let _ = config::save(&g);
                            enabled
                        };
                        rebuild_tray_menu(app, enabled);
                    }
                    "settings" => show_settings(app),
                    "quit" => app.exit(0),
                    _ => {}
                })
                .build(app)?;

            rebuild_tray_menu(app.handle(), cfg.lock().unwrap().enabled);
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("无法启动 whatamidoing 客户端");
}

fn show_settings(app: &AppHandle) {
    if let Some(w) = app.get_webview_window("settings") {
        let _ = w.show();
        let _ = w.set_focus();
    }
}
