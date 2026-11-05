use std::sync::Mutex;

use tauri::menu::{MenuBuilder, MenuItem, MenuItemBuilder};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::{AppHandle, Emitter, Manager, Wry};

use crate::cli_engine::{cancel_oauth_login, sync_dashboard, EngineState};
use crate::models::DashboardSnapshot;

pub struct TrayHandles {
    status_item: MenuItem<Wry>,
    join_item: MenuItem<Wry>,
    join_url: Mutex<Option<String>>,
}

pub fn show_main_window(app: &AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.show();
        let _ = window.unminimize();
        let _ = window.set_focus();
    }
}

pub fn setup_tray(app: &AppHandle) -> Result<(), Box<dyn std::error::Error>> {
    let status_item = MenuItemBuilder::with_id("status", "Syncing schedule...")
        .enabled(false)
        .build(app)?;
    let join_item = MenuItemBuilder::with_id("join", "No active meeting link")
        .enabled(false)
        .build(app)?;
    let open_item = MenuItemBuilder::with_id("open", "Open Next Meeting").build(app)?;
    let sync_item = MenuItemBuilder::with_id("sync", "Sync Schedule").build(app)?;
    let force_sync_item =
        MenuItemBuilder::with_id("force_sync", "Force Refresh (Clear Cache)").build(app)?;
    let quit_item = MenuItemBuilder::with_id("quit", "Quit Next Meeting").build(app)?;

    let menu = MenuBuilder::new(app)
        .item(&status_item)
        .item(&join_item)
        .separator()
        .item(&open_item)
        .item(&sync_item)
        .item(&force_sync_item)
        .separator()
        .item(&quit_item)
        .build()?;

    let mut builder = TrayIconBuilder::with_id("main-tray")
        .menu(&menu)
        .tooltip("Next Meeting")
        .show_menu_on_left_click(cfg!(target_os = "macos"))
        .on_menu_event(|app, event| match event.id().as_ref() {
            "open" => {
                show_main_window(app);
            }
            "join" => {
                if let Some(handles) = app.try_state::<TrayHandles>() {
                    if let Ok(guard) = handles.join_url.lock() {
                        if let Some(url) = guard.as_ref() {
                            let _ = open::that_detached(url);
                        }
                    }
                }
            }
            "sync" | "force_sync" => {
                let force = event.id().as_ref() == "force_sync";
                let app_clone = app.clone();
                tauri::async_runtime::spawn(async move {
                    let state = app_clone.state::<EngineState>();
                    if let Ok(snapshot) = sync_dashboard(&state, force, "tray").await {
                        update_tray_from_snapshot(&app_clone, &snapshot);
                        let _ = app_clone.emit("cli://snapshot-updated", &snapshot);
                    }
                });
            }
            "quit" => {
                let app_clone = app.clone();
                tauri::async_runtime::spawn(async move {
                    let state = app_clone.state::<EngineState>();
                    cancel_oauth_login(&state).await;
                    app_clone.exit(0);
                });
            }
            _ => {}
        })
        .on_tray_icon_event(|tray, event| {
            if !cfg!(target_os = "macos") {
                if let TrayIconEvent::Click {
                    button: MouseButton::Left,
                    button_state: MouseButtonState::Up,
                    ..
                } = event
                {
                    show_main_window(tray.app_handle());
                }
            }
        });

    if let Some(icon) = app.default_window_icon() {
        builder = builder.icon(icon.clone());
    }

    builder.build(app)?;

    app.manage(TrayHandles {
        status_item,
        join_item,
        join_url: Mutex::new(None),
    });

    Ok(())
}

pub fn update_tray_from_snapshot(app: &AppHandle, snapshot: &DashboardSnapshot) {
    let summary_line = snapshot.raw_cli_output.lines().next().unwrap_or("Next Meeting");
    if let Some(tray) = app.tray_by_id("main-tray") {
        let _ = tray.set_tooltip(Some(format!("Next Meeting — {summary_line}")));
    }

    let Some(handles) = app.try_state::<TrayHandles>() else {
        return;
    };

    let _ = handles.status_item.set_text(summary_line);

    let actionable_meeting = snapshot
        .current_meeting
        .as_ref()
        .filter(|m| m.join_url.is_some())
        .or_else(|| {
            snapshot
                .next_meeting
                .as_ref()
                .filter(|m| m.join_url.is_some())
        });

    if let Some(meeting) = actionable_meeting {
        let short_title: String = meeting.summary.chars().take(36).collect();
        let _ = handles.join_item.set_text(format!("Join: {short_title}"));
        let _ = handles.join_item.set_enabled(true);
        if let Ok(mut guard) = handles.join_url.lock() {
            *guard = meeting.join_url.clone();
        }
    } else {
        let _ = handles.join_item.set_text("No active meeting link");
        let _ = handles.join_item.set_enabled(false);
        if let Ok(mut guard) = handles.join_url.lock() {
            *guard = None;
        }
    }
}
