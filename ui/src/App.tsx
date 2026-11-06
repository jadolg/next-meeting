import {
  AlertTriangle,
  CalendarClock,
  CalendarRange,
  Check,
  Copy,
  LayoutDashboard,
  Minimize2,
  Pin,
  RefreshCw,
  RotateCw,
  Settings,
  Terminal,
  X,
} from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "./api";
import { CompactWidget } from "./components/CompactWidget";
import { ConsoleTab } from "./components/ConsoleTab";
import { LoginModal } from "./components/LoginModal";
import { OverviewTab } from "./components/OverviewTab";
import { ScheduleTab } from "./components/ScheduleTab";
import { SettingsTab } from "./components/SettingsTab";
import type {
  AppSettings,
  CliExecutionRecord,
  DashboardSnapshot,
  TabId,
} from "./types";
import { formatRelativeSeconds } from "./utils/time";

export function App() {
  const [activeTab, setActiveTab] = useState<TabId>("overview");
  const [snapshot, setSnapshot] = useState<DashboardSnapshot | null>(null);
  const [settings, setSettings] = useState<AppSettings | null>(null);
  const [history, setHistory] = useState<CliExecutionRecord[]>([]);
  const [syncing, setSyncing] = useState(false);
  const [runningPreset, setRunningPreset] = useState<string | null>(null);
  const [copiedCli, setCopiedCli] = useState(false);
  const [nowMs, setNowMs] = useState(() => Date.now());
  // Startup failure shown on the boot screen (no snapshot yet).
  const [bootError, setBootError] = useState<string | null>(null);
  // Failure of a later sync/action/settings save, shown as a dismissible banner on top of
  // the (possibly stale) schedule and cleared on the next successful operation.
  const [runtimeError, setRuntimeError] = useState<string | null>(null);
  const snapshotRef = useRef<DashboardSnapshot | null>(null);
  useEffect(() => {
    snapshotRef.current = snapshot;
  }, [snapshot]);

  // Login modal state
  const [loginOpen, setLoginOpen] = useState(false);
  const [loginUrl, setLoginUrl] = useState<string | null>(null);
  const [loginStatus, setLoginStatus] = useState(
    "Preparing OAuth authorization..."
  );
  const [loginError, setLoginError] = useState<string | null>(null);

  const boundaryTriggeredRef = useRef<string>("");
  // Id of the newest `--login` attempt; progress events from older attempts are ignored.
  const loginSessionRef = useRef(0);
  const loginStartPendingRef = useRef(false);

  const refreshHistory = useCallback(async () => {
    try {
      const records = await api.getExecutionHistory();
      setHistory(records);
    } catch {
      // Ignore transient errors
    }
  }, []);

  const performSync = useCallback(
    async (force = false, trigger = "manual") => {
      setSyncing(true);
      try {
        const snap = await api.getDashboardSnapshot(force, trigger);
        setSnapshot(snap);
        setRuntimeError(null);
        await refreshHistory();
      } catch (err) {
        const message = String(err);
        if (snapshotRef.current) {
          setRuntimeError(message);
        } else {
          setBootError(message);
        }
      } finally {
        setSyncing(false);
      }
    },
    [refreshHistory]
  );

  // Full startup load: settings, snapshot and history together. "Retry Initialization"
  // reuses it so a failed startup can never leave `settings` null after recovery (which
  // would keep the app on the boot loader forever).
  const initialize = useCallback(
    async (trigger: string, isActive: () => boolean = () => true) => {
      const [loadedSettings, initialSnap, initialHistory] = await Promise.all([
        api.getSettings(),
        api.getDashboardSnapshot(false, trigger),
        api.getExecutionHistory(),
      ]);
      if (!isActive()) return;
      setSettings(loadedSettings);
      setSnapshot(initialSnap);
      setHistory(initialHistory);
      if (loadedSettings.compactMode) {
        await api.setWindowCompact(true);
      }
    },
    []
  );

  // Initial load
  useEffect(() => {
    let mounted = true;
    initialize("startup", () => mounted).catch((err) => {
      if (mounted) {
        setBootError(String(err));
      }
    });
    return () => {
      mounted = false;
    };
  }, [initialize]);

  // Apply theme attribute to document root
  useEffect(() => {
    if (!settings) return;
    const root = document.documentElement;
    if (settings.theme === "system") {
      root.removeAttribute("data-theme");
    } else {
      root.setAttribute("data-theme", settings.theme);
    }
  }, [settings?.theme, settings]);

  // Visibility-aware 1-second local clock ticker (zero idle CPU while hidden)
  useEffect(() => {
    let intervalId: number | null = null;

    const startTicker = () => {
      if (intervalId !== null) return;
      setNowMs(Date.now());
      intervalId = window.setInterval(() => {
        setNowMs(Date.now());
      }, 1000);
    };

    const stopTicker = () => {
      if (intervalId !== null) {
        clearInterval(intervalId);
        intervalId = null;
      }
    };

    const onVisibilityChange = () => {
      if (document.hidden) {
        stopTicker();
      } else {
        startTicker();
      }
    };

    if (!document.hidden) {
      startTicker();
    }
    document.addEventListener("visibilitychange", onVisibilityChange);
    return () => {
      stopTicker();
      document.removeEventListener("visibilitychange", onVisibilityChange);
    };
  }, []);

  // Auto-sync when current meeting ends or next meeting begins
  useEffect(() => {
    if (!snapshot) return;
    const currEnd = snapshot.currentMeeting?.endEpochMs ?? 0;
    const nextStart = snapshot.nextMeeting?.startEpochMs ?? 0;

    if (currEnd > 0 && nowMs >= currEnd) {
      const key = `end-${currEnd}`;
      if (boundaryTriggeredRef.current !== key) {
        boundaryTriggeredRef.current = key;
        void performSync(false, "boundary");
      }
    } else if (nextStart > 0 && nowMs >= nextStart) {
      const key = `start-${nextStart}`;
      if (boundaryTriggeredRef.current !== key) {
        boundaryTriggeredRef.current = key;
        void performSync(false, "boundary");
      }
    }
  }, [nowMs, snapshot, performSync]);

  // Background polling timer for CLI sync & notifications
  useEffect(() => {
    const secs = settings?.pollIntervalSeconds ?? 0;
    if (secs <= 0) return;
    const timer = window.setInterval(() => {
      void performSync(false, "auto");
    }, secs * 1000);
    return () => clearInterval(timer);
  }, [settings?.pollIntervalSeconds, performSync]);

  // Subscribe to backend events (login progress & system tray refreshes)
  useEffect(() => {
    // `listen()` registers asynchronously, so cleanup can run before an unlisten function
    // has been assigned (StrictMode's setup/cleanup replay does exactly this). Track
    // disposal and unsubscribe immediately when a registration resolves late, otherwise
    // the stale subscription survives and handlers run twice.
    let disposed = false;
    let unlistenLogin: (() => void) | null = null;
    let unlistenSnap: (() => void) | null = null;

    void api
      .onLoginProgress((payload) => {
        // Drop events from a sign-in attempt that was cancelled or replaced.
        if (payload.sessionId < loginSessionRef.current) return;
        loginSessionRef.current = payload.sessionId;
        // While a new attempt is starting, a late "cancelled" can only belong to the
        // previous one (the frontend closes the modal itself on user cancel).
        if (payload.stage === "cancelled" && loginStartPendingRef.current) return;

        if (payload.url) {
          setLoginUrl(payload.url);
        }
        if (payload.stage === "stderr") {
          setLoginError(payload.message);
        } else if (payload.stage === "finished") {
          if (payload.success) {
            setLoginOpen(false);
            setLoginError(null);
            setLoginUrl(null);
            void performSync(true, "auth");
          } else {
            setLoginError(payload.message);
          }
        } else if (payload.stage === "cancelled") {
          setLoginOpen(false);
          setLoginError(null);
        } else {
          setLoginStatus(payload.message);
        }
      })
      .then((fn) => {
        if (disposed) {
          fn();
        } else {
          unlistenLogin = fn;
        }
      })
      .catch(() => {});

    void api
      .onSnapshotUpdated((updated) => {
        setSnapshot(updated);
        void refreshHistory();
      })
      .then((fn) => {
        if (disposed) {
          fn();
        } else {
          unlistenSnap = fn;
        }
      })
      .catch(() => {});

    return () => {
      disposed = true;
      unlistenLogin?.();
      unlistenSnap?.();
    };
  }, [performSync, refreshHistory]);

  const handleUpdateSettings = useCallback(
    async (next: AppSettings, resync = false) => {
      const previous = settings;
      const prevCompact = previous?.compactMode ?? false;
      setSettings(next);
      try {
        const saved = await api.updateSettings(next);
        setSettings(saved);
        setRuntimeError(null);
        if (saved.compactMode !== prevCompact) {
          await api.setWindowCompact(saved.compactMode);
        }
      } catch (err) {
        // Roll back the optimistic update so the UI never shows settings that were not
        // actually saved, and surface the failure.
        setSettings(previous);
        setRuntimeError(String(err));
        return;
      }
      if (resync) {
        // performSync reports its own failures.
        await performSync(false, "settings");
      }
    },
    [settings, performSync]
  );

  const handleStartLogin = useCallback(async () => {
    setLoginUrl(null);
    setLoginError(null);
    setLoginStatus(
      "Launching `next-meeting --login` and opening Google authorization..."
    );
    setLoginOpen(true);
    loginStartPendingRef.current = true;
    try {
      const sessionId = await api.startLogin();
      loginSessionRef.current = Math.max(loginSessionRef.current, sessionId);
    } catch (err) {
      setLoginError(String(err));
    } finally {
      loginStartPendingRef.current = false;
    }
  }, []);

  const handleCancelLogin = useCallback(async () => {
    await api.cancelLogin().catch(() => {});
    setLoginOpen(false);
    setLoginError(null);
  }, []);

  const handleOpenUrl = useCallback(async (url: string) => {
    await api.openExternalUrl(url);
  }, []);

  // Power-user keyboard shortcuts
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null;
      const inEditable =
        target &&
        (target.tagName === "INPUT" ||
          target.tagName === "TEXTAREA" ||
          target.tagName === "SELECT" ||
          target.isContentEditable);

      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "r") {
        e.preventDefault();
        void performSync(e.shiftKey, "shortcut");
        return;
      }

      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "j") {
        const joinLink =
          snapshot?.currentMeeting?.joinUrl ?? snapshot?.nextMeeting?.joinUrl;
        if (joinLink) {
          e.preventDefault();
          void handleOpenUrl(joinLink);
        }
        return;
      }

      if (!inEditable && !e.ctrlKey && !e.metaKey && !e.altKey) {
        if (e.key === "1") setActiveTab("overview");
        if (e.key === "2") setActiveTab("schedule");
        if (e.key === "3") setActiveTab("console");
        if (e.key === "4") setActiveTab("settings");
      }
    };

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [performSync, snapshot, handleOpenUrl]);

  if (bootError && !snapshot) {
    return (
      <div className="boot-screen">
        <div className="callout-panel">
          <h2>Desktop Runtime Unavailable</h2>
          <p className="muted-text">{bootError}</p>
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => {
              setBootError(null);
              void initialize("retry").catch((err) => setBootError(String(err)));
            }}
          >
            Retry Initialization
          </button>
        </div>
      </div>
    );
  }

  if (!snapshot || !settings) {
    return (
      <div className="boot-screen">
        <div className="boot-loader">
          <CalendarClock size={30} className="accent-icon" />
          <span>Starting Next Meeting CLI Engine...</span>
        </div>
      </div>
    );
  }

  if (settings.compactMode) {
    return (
      <CompactWidget
        snapshot={snapshot}
        settings={settings}
        nowMs={nowMs}
        syncing={syncing}
        errorMessage={runtimeError}
        onDismissError={() => setRuntimeError(null)}
        onSync={(force) => void performSync(force, "compact")}
        onTogglePin={() =>
          void handleUpdateSettings(
            { ...settings, alwaysOnTop: !settings.alwaysOnTop },
            false
          )
        }
        onExitCompact={() =>
          void handleUpdateSettings({ ...settings, compactMode: false }, false)
        }
        onOpenUrl={(url) => void handleOpenUrl(url)}
      />
    );
  }

  const copyCliOutput = async () => {
    await navigator.clipboard.writeText(snapshot.rawCliOutput);
    setCopiedCli(true);
    setTimeout(() => setCopiedCli(false), 1600);
  };

  return (
    <div className="app-shell">
      {/* Top Header & Live CLI Status Bar */}
      <header className="topbar">
        <div className="brand-block">
          <div className="brand-logo">
            <CalendarClock size={19} />
          </div>
          <div>
            <h1 className="brand-title">Next Meeting</h1>
            <span className="brand-sub">CLI-Powered Schedule Monitor</span>
          </div>
        </div>

        <div className="cli-status-strip" title="Live output from next-meeting CLI">
          <span className="cli-prompt">$</span>
          <code className="cli-output-text">{snapshot.rawCliOutput}</code>
          <button
            type="button"
            className="icon-btn icon-btn-xs"
            onClick={copyCliOutput}
            title="Copy CLI output"
          >
            {copiedCli ? <Check size={13} /> : <Copy size={13} />}
          </button>
        </div>

        <div className="topbar-actions">
          {snapshot.cacheInfo.exists && !snapshot.cacheInfo.isExpired && (
            <span
              className="cache-badge"
              title="Events cached locally by next-meeting CLI (30m TTL)"
            >
              Cache {formatRelativeSeconds(snapshot.cacheInfo.ttlRemainingSeconds)}
            </span>
          )}

          <button
            type="button"
            className={`icon-btn ${settings.alwaysOnTop ? "icon-btn-active" : ""}`}
            onClick={() =>
              void handleUpdateSettings(
                { ...settings, alwaysOnTop: !settings.alwaysOnTop },
                false
              )
            }
            title={
              settings.alwaysOnTop
                ? "Unpin Always on Top"
                : "Pin Always on Top"
            }
          >
            <Pin size={15} />
          </button>

          <button
            type="button"
            className="icon-btn"
            onClick={() =>
              void handleUpdateSettings({ ...settings, compactMode: true }, false)
            }
            title="Switch to Compact Desk Widget"
          >
            <Minimize2 size={15} />
          </button>

          <button
            type="button"
            className="btn btn-sm btn-secondary"
            disabled={syncing}
            onClick={() => void performSync(false, "manual")}
            title="Sync via CLI (Ctrl+R)"
          >
            <RefreshCw size={14} className={syncing ? "spin" : ""} />
            Sync
          </button>

          <button
            type="button"
            className="icon-btn"
            disabled={syncing}
            onClick={() => void performSync(true, "force")}
            title="Clear cache & fetch fresh from Google Calendar (Ctrl+Shift+R)"
          >
            <RotateCw size={15} />
          </button>
        </div>
      </header>

      {runtimeError && (
        <div className="inline-banner banner-danger runtime-banner" role="alert">
          <AlertTriangle size={16} />
          <span>{runtimeError}</span>
          <button
            type="button"
            className="icon-btn icon-btn-xs"
            onClick={() => setRuntimeError(null)}
            aria-label="Dismiss error"
          >
            <X size={14} />
          </button>
        </div>
      )}

      {/* Navigation Tabs */}
      <nav className="nav-tabs" aria-label="Main navigation">
        {(
          [
            ["overview", "Overview", LayoutDashboard, null],
            [
              "schedule",
              "Today's Schedule",
              CalendarRange,
              snapshot.events.length > 0 ? String(snapshot.events.length) : null,
            ],
            [
              "console",
              "CLI Engine",
              Terminal,
              snapshot.lastExecution
                ? `${snapshot.lastExecution.durationMs}ms`
                : null,
            ],
            [
              "settings",
              "Auth & Settings",
              Settings,
              snapshot.authState !== "authenticated" ? "!" : null,
            ],
          ] as const
        ).map(([id, label, Icon, badge]) => (
          <button
            key={id}
            type="button"
            className={`nav-tab ${activeTab === id ? "nav-tab-active" : ""}`}
            onClick={() => setActiveTab(id)}
          >
            <Icon size={15} />
            <span>{label}</span>
            {badge && (
              <span
                className={`tab-badge ${
                  badge === "!" ? "tab-badge-warn" : ""
                }`}
              >
                {badge}
              </span>
            )}
          </button>
        ))}
      </nav>

      {/* Main Content Area */}
      <main className="content-area">
        {activeTab === "overview" && (
          <OverviewTab
            snapshot={snapshot}
            settings={settings}
            nowMs={nowMs}
            syncing={syncing}
            onSync={(force) => void performSync(force, "manual")}
            onUpdateSettings={(next, resync) =>
              void handleUpdateSettings(next, resync)
            }
            onOpenUrl={(url) => void handleOpenUrl(url)}
            onStartLogin={() => void handleStartLogin()}
            onSwitchTab={setActiveTab}
          />
        )}

        {activeTab === "schedule" && (
          <ScheduleTab
            events={snapshot.events}
            nowMs={nowMs}
            onOpenUrl={(url) => void handleOpenUrl(url)}
          />
        )}

        {activeTab === "console" && (
          <ConsoleTab
            snapshot={snapshot}
            history={history}
            runningPreset={runningPreset}
            onRunPreset={async (preset) => {
              setRunningPreset(preset);
              try {
                await api.runCliPreset(preset);
                const updated = await api.getDashboardSnapshot(false, "console");
                setSnapshot(updated);
                setRuntimeError(null);
                await refreshHistory();
              } catch (err) {
                setRuntimeError(String(err));
              } finally {
                setRunningPreset(null);
              }
            }}
            onClearHistory={async () => {
              await api.clearExecutionHistory();
              setHistory([]);
            }}
            onRevealDir={(target) => void api.revealSystemDirectory(target)}
          />
        )}

        {activeTab === "settings" && (
          <SettingsTab
            snapshot={snapshot}
            settings={settings}
            onUpdateSettings={(next, resync) =>
              void handleUpdateSettings(next, resync)
            }
            onSaveCredentials={async (content) => {
              await api.saveCredentialsJson(content);
              await performSync(false, "credentials");
            }}
            onRemoveCredentials={async () => {
              try {
                await api.removeCredentialsJson();
              } catch (err) {
                setRuntimeError(String(err));
                return;
              }
              await performSync(false, "credentials");
            }}
            onStartLogin={() => void handleStartLogin()}
            onLogout={async () => {
              try {
                const snap = await api.logoutAccount();
                setSnapshot(snap);
                setRuntimeError(null);
                await refreshHistory();
              } catch (err) {
                setRuntimeError(String(err));
              }
            }}
            onOpenUrl={(url) => void handleOpenUrl(url)}
            onRevealDir={(target) => void api.revealSystemDirectory(target)}
          />
        )}
      </main>

      <LoginModal
        open={loginOpen}
        authUrl={loginUrl}
        statusMessage={loginStatus}
        errorMessage={loginError}
        onOpenUrl={(url) => void handleOpenUrl(url)}
        onCancel={() => void handleCancelLogin()}
      />
    </div>
  );
}
