import { invoke } from "@tauri-apps/api/core";
import { listen, type UnlistenFn } from "@tauri-apps/api/event";
import { getCurrentWindow, LogicalSize } from "@tauri-apps/api/window";
import type {
  AppSettings,
  CliExecutionRecord,
  CredentialsInfo,
  DashboardSnapshot,
  EngineInfo,
  LoginProgressPayload,
} from "./types";

export const api = {
  getDashboardSnapshot(
    forceRefresh = false,
    trigger = "manual"
  ): Promise<DashboardSnapshot> {
    return invoke<DashboardSnapshot>("get_dashboard_snapshot", {
      forceRefresh,
      trigger,
    });
  },

  getSettings(): Promise<AppSettings> {
    return invoke<AppSettings>("get_settings");
  },

  updateSettings(settings: AppSettings): Promise<AppSettings> {
    return invoke<AppSettings>("update_settings", { settings });
  },

  getCredentialsStatus(): Promise<CredentialsInfo> {
    return invoke<CredentialsInfo>("get_credentials_status");
  },

  saveCredentialsJson(content: string): Promise<CredentialsInfo> {
    return invoke<CredentialsInfo>("save_credentials_json", { content });
  },

  removeCredentialsJson(): Promise<CredentialsInfo> {
    return invoke<CredentialsInfo>("remove_credentials_json");
  },

  /** Starts `next-meeting --login` and resolves with the id of the new login session. */
  startLogin(): Promise<number> {
    return invoke<number>("start_login");
  },

  cancelLogin(): Promise<void> {
    return invoke<void>("cancel_login");
  },

  logoutAccount(): Promise<DashboardSnapshot> {
    return invoke<DashboardSnapshot>("logout_account");
  },

  clearCliCache(): Promise<CliExecutionRecord> {
    return invoke<CliExecutionRecord>("clear_cli_cache");
  },

  runCliPreset(
    preset: "status" | "only_accepted" | "clear_cache" | "notify_now" | "help"
  ): Promise<CliExecutionRecord> {
    return invoke<CliExecutionRecord>("run_cli_preset", { preset });
  },

  getExecutionHistory(): Promise<CliExecutionRecord[]> {
    return invoke<CliExecutionRecord[]>("get_execution_history");
  },

  clearExecutionHistory(): Promise<void> {
    return invoke<void>("clear_execution_history");
  },

  getEngineDiagnostics(): Promise<EngineInfo> {
    return invoke<EngineInfo>("get_engine_diagnostics");
  },

  openExternalUrl(url: string): Promise<void> {
    return invoke<void>("open_external_url", { url });
  },

  revealSystemDirectory(target: "config" | "cache"): Promise<void> {
    return invoke<void>("reveal_system_directory", { target });
  },

  onLoginProgress(
    handler: (payload: LoginProgressPayload) => void
  ): Promise<UnlistenFn> {
    return listen<LoginProgressPayload>("cli://login-progress", (event) => {
      handler(event.payload);
    });
  },

  onSnapshotUpdated(
    handler: (snapshot: DashboardSnapshot) => void
  ): Promise<UnlistenFn> {
    return listen<DashboardSnapshot>("cli://snapshot-updated", (event) => {
      handler(event.payload);
    });
  },

  async setWindowCompact(compact: boolean): Promise<void> {
    try {
      const win = getCurrentWindow();
      if (compact) {
        await win.setSize(new LogicalSize(410, 340));
      } else {
        await win.setSize(new LogicalSize(980, 680));
      }
    } catch {
      // Non-fatal if window sizing is constrained by a tiling WM on Linux
    }
  },
};
