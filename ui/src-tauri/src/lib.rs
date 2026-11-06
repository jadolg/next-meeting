pub mod cli_engine;
pub mod models;
pub mod paths;
pub mod tray;

use std::fs;
use tauri::{AppHandle, Manager, State, WindowEvent};

use crate::cli_engine::{
    cancel_oauth_login, execute_cli, resolve_engine_info, start_oauth_login, sync_dashboard,
    sync_dashboard_locked, validate_notify_duration, EngineState,
};
use crate::models::{
    AppSettings, CliExecutionRecord, CredentialsInfo, DashboardSnapshot, EngineInfo,
};
use crate::paths::{
    app_config_dir, cache_path, delete_credentials_file, inspect_credentials, save_settings,
    write_credentials_content,
};
use crate::tray::{setup_tray, update_tray_from_snapshot};

#[tauri::command]
async fn get_dashboard_snapshot(
    app: AppHandle,
    state: State<'_, EngineState>,
    force_refresh: bool,
    trigger: Option<String>,
) -> Result<DashboardSnapshot, String> {
    let trig = trigger.unwrap_or_else(|| "manual".to_string());
    let snapshot = sync_dashboard(&state, force_refresh, &trig).await?;
    update_tray_from_snapshot(&app, &snapshot);
    Ok(snapshot)
}

#[tauri::command]
async fn get_settings(state: State<'_, EngineState>) -> Result<AppSettings, String> {
    Ok(state.settings.lock().await.clone())
}

#[tauri::command]
async fn update_settings(
    app: AppHandle,
    state: State<'_, EngineState>,
    mut settings: AppSettings,
) -> Result<AppSettings, String> {
    if settings.notify_enabled {
        settings.notify_threshold = validate_notify_duration(&settings.notify_threshold)?;
    }
    if let Some(custom) = &settings.custom_cli_path {
        if custom.trim().is_empty() {
            settings.custom_cli_path = None;
        }
    }

    save_settings(&settings)?;
    {
        let mut guard = state.settings.lock().await;
        *guard = settings.clone();
    }

    if let Some(window) = app.get_webview_window("main") {
        let _ = window.set_always_on_top(settings.always_on_top);
    }

    Ok(settings)
}

#[tauri::command]
async fn get_credentials_status() -> Result<CredentialsInfo, String> {
    Ok(inspect_credentials())
}

#[tauri::command]
async fn save_credentials_json(content: String) -> Result<CredentialsInfo, String> {
    write_credentials_content(&content)
}

#[tauri::command]
async fn remove_credentials_json() -> Result<CredentialsInfo, String> {
    delete_credentials_file()
}

/// Starts a sign-in attempt and returns its session id (echoed on every progress event).
#[tauri::command]
async fn start_login(app: AppHandle, state: State<'_, EngineState>) -> Result<u64, String> {
    start_oauth_login(app, &state).await
}

#[tauri::command]
async fn cancel_login(state: State<'_, EngineState>) -> Result<(), String> {
    cancel_oauth_login(&state).await;
    Ok(())
}

#[tauri::command]
async fn logout_account(
    app: AppHandle,
    state: State<'_, EngineState>,
) -> Result<DashboardSnapshot, String> {
    cancel_oauth_login(&state).await;

    // Hold the engine lock across the whole multi-command action so an automatic sync
    // cannot interleave between `--clear`, `--clear-cache`, and the final refresh.
    let _engine_guard = state.sync_lock.lock().await;
    let settings = state.settings.lock().await.clone();
    let engine = resolve_engine_info(settings.custom_cli_path.as_deref());
    if !engine.available {
        return Err("CLI engine binary is not available.".to_string());
    }

    let _ = execute_cli(
        &state,
        &engine.binary_path,
        vec!["--clear".to_string()],
        "auth",
    )
    .await?;

    let _ = execute_cli(
        &state,
        &engine.binary_path,
        vec!["--clear-cache".to_string()],
        "auth",
    )
    .await?;

    let snapshot = sync_dashboard_locked(&state, false, "auth").await?;
    update_tray_from_snapshot(&app, &snapshot);
    Ok(snapshot)
}

#[tauri::command]
async fn clear_cli_cache(
    app: AppHandle,
    state: State<'_, EngineState>,
) -> Result<CliExecutionRecord, String> {
    // Serialize with syncs so `--clear-cache` cannot race a cache write in progress.
    let _engine_guard = state.sync_lock.lock().await;
    let settings = state.settings.lock().await.clone();
    let engine = resolve_engine_info(settings.custom_cli_path.as_deref());
    if !engine.available {
        return Err("CLI engine binary is not available.".to_string());
    }

    let rec = execute_cli(
        &state,
        &engine.binary_path,
        vec!["--clear-cache".to_string()],
        "action",
    )
    .await?;

    if let Ok(snapshot) = sync_dashboard_locked(&state, false, "action").await {
        update_tray_from_snapshot(&app, &snapshot);
    }

    Ok(rec)
}

#[tauri::command]
async fn run_cli_preset(
    app: AppHandle,
    state: State<'_, EngineState>,
    preset: String,
) -> Result<CliExecutionRecord, String> {
    // Serialize with syncs: an automatic sync and "Evaluate Alert" (`--notify`) running
    // concurrently can both pass the CLI's notified check and send duplicate alerts.
    let _engine_guard = state.sync_lock.lock().await;
    let settings = state.settings.lock().await.clone();
    let engine = resolve_engine_info(settings.custom_cli_path.as_deref());
    if !engine.available {
        return Err("CLI engine binary is not available.".to_string());
    }

    let args = match preset.as_str() {
        "status" => Vec::new(),
        "only_accepted" => vec!["--only-accepted".to_string()],
        "clear_cache" => vec!["--clear-cache".to_string()],
        "notify_now" => {
            let dur = validate_notify_duration(&settings.notify_threshold)
                .unwrap_or_else(|_| "15m".to_string());
            vec!["--notify".to_string(), dur]
        }
        "help" => vec!["-h".to_string()],
        _ => return Err(format!("Unsupported CLI preset: {preset}")),
    };

    let record = execute_cli(&state, &engine.binary_path, args, "console").await?;

    if preset != "help" {
        if let Ok(snapshot) = sync_dashboard_locked(&state, false, "console").await {
            update_tray_from_snapshot(&app, &snapshot);
        }
    }

    Ok(record)
}

#[tauri::command]
async fn get_execution_history(
    state: State<'_, EngineState>,
) -> Result<Vec<CliExecutionRecord>, String> {
    let history = state.history.lock().await;
    Ok(history.iter().cloned().collect())
}

#[tauri::command]
async fn clear_execution_history(state: State<'_, EngineState>) -> Result<(), String> {
    state.history.lock().await.clear();
    Ok(())
}

#[tauri::command]
async fn get_engine_diagnostics(state: State<'_, EngineState>) -> Result<EngineInfo, String> {
    let settings = state.settings.lock().await.clone();
    Ok(resolve_engine_info(settings.custom_cli_path.as_deref()))
}

#[tauri::command]
async fn open_external_url(url: String) -> Result<(), String> {
    let trimmed = url.trim();
    if !(trimmed.starts_with("https://") || trimmed.starts_with("http://")) {
        return Err("Only http:// and https:// URLs can be opened externally.".to_string());
    }
    open::that_detached(trimmed).map_err(|e| format!("Failed to open URL in browser: {e}"))
}

#[tauri::command]
async fn reveal_system_directory(target: String) -> Result<(), String> {
    let dir = match target.as_str() {
        "config" => {
            let d = app_config_dir()?;
            fs::create_dir_all(&d)
                .map_err(|e| format!("Failed to create config directory: {e}"))?;
            d
        }
        "cache" => cache_path()
            .parent()
            .map(|p| p.to_path_buf())
            .ok_or_else(|| "Unable to determine cache directory".to_string())?,
        _ => return Err(format!("Unknown directory target: {target}")),
    };

    open::that_detached(&dir).map_err(|e| format!("Failed to open folder {}: {e}", dir.display()))
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let engine_state = EngineState::new();

    tauri::Builder::default()
        .manage(engine_state)
        .setup(|app| {
            setup_tray(app.handle())?;

            if let Some(win) = app.get_webview_window("main") {
                let _ = win.show();
                let _ = win.unminimize();
                let _ = win.set_focus();
            }

            // Apply initial window pin setting if enabled
            let app_handle = app.handle().clone();
            tauri::async_runtime::spawn(async move {
                let st = app_handle.state::<EngineState>();
                let always_on_top = st.settings.lock().await.always_on_top;
                if always_on_top {
                    if let Some(win) = app_handle.get_webview_window("main") {
                        let _ = win.set_always_on_top(true);
                    }
                }
            });

            Ok(())
        })
        .on_window_event(|window, event| {
            if window.label() == "main" {
                if let WindowEvent::CloseRequested { api, .. } = event {
                    let app = window.app_handle();
                    let state = app.state::<EngineState>();
                    let minimize = tauri::async_runtime::block_on(async {
                        state.settings.lock().await.minimize_to_tray
                    });
                    if minimize {
                        api.prevent_close();
                        let _ = window.hide();
                    }
                }
            }
        })
        .invoke_handler(tauri::generate_handler![
            get_dashboard_snapshot,
            get_settings,
            update_settings,
            get_credentials_status,
            save_credentials_json,
            remove_credentials_json,
            start_login,
            cancel_login,
            logout_account,
            clear_cli_cache,
            run_cli_preset,
            get_execution_history,
            clear_execution_history,
            get_engine_diagnostics,
            open_external_url,
            reveal_system_directory,
        ])
        .run(tauri::generate_context!())
        .expect("error while running Next Meeting desktop application");
}
