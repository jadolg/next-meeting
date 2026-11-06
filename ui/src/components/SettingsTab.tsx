import {
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  FileJson,
  FolderOpen,
  HelpCircle,
  KeyRound,
  LogIn,
  LogOut,
  RotateCcw,
  Save,
  ShieldCheck,
  Sliders,
  Trash2,
  Upload,
} from "lucide-react";
import { type ChangeEvent, type DragEvent, useRef, useState } from "react";
import type { AppSettings, DashboardSnapshot } from "../types";

interface SettingsTabProps {
  snapshot: DashboardSnapshot;
  settings: AppSettings;
  onUpdateSettings: (next: AppSettings, resync?: boolean) => void;
  onSaveCredentials: (jsonText: string) => Promise<void>;
  onRemoveCredentials: () => Promise<void>;
  onStartLogin: () => void;
  onLogout: () => Promise<void>;
  onOpenUrl: (url: string) => void;
  onRevealDir: (target: "config" | "cache") => void;
}

export function SettingsTab({
  snapshot,
  settings,
  onUpdateSettings,
  onSaveCredentials,
  onRemoveCredentials,
  onStartLogin,
  onLogout,
  onOpenUrl,
  onRevealDir,
}: SettingsTabProps) {
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [showPasteBox, setShowPasteBox] = useState(false);
  const [showGuide, setShowGuide] = useState(
    !snapshot.credentialsInfo.valid
  );
  const [jsonDraft, setJsonDraft] = useState("");
  const [dragging, setDragging] = useState(false);
  const [credFeedback, setCredFeedback] = useState<{
    type: "ok" | "err";
    text: string;
  } | null>(null);
  const [customBinDraft, setCustomBinDraft] = useState(
    settings.customCliPath ?? ""
  );
  const [busyAuth, setBusyAuth] = useState(false);

  const { credentialsInfo, authState } = snapshot;

  const processCredentialsText = async (content: string) => {
    setCredFeedback(null);
    try {
      await onSaveCredentials(content);
      setJsonDraft("");
      setShowPasteBox(false);
      setCredFeedback({
        type: "ok",
        text: "credentials.json validated and saved securely.",
      });
    } catch (err) {
      setCredFeedback({
        type: "err",
        text: String(err),
      });
    }
  };

  const handleFilePicked = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const text = await file.text();
    e.target.value = "";
    await processCredentialsText(text);
  };

  const handleDrop = async (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setDragging(false);
    const file = e.dataTransfer.files?.[0];
    if (!file) return;
    const text = await file.text();
    await processCredentialsText(text);
  };

  const handleLogoutClick = async () => {
    setBusyAuth(true);
    try {
      await onLogout();
    } finally {
      setBusyAuth(false);
    }
  };

  return (
    <div className="tab-stack">
      {/* Authentication & Google Credentials Section */}
      <div className="panel-card">
        <div className="section-title-row">
          <div className="row-center">
            <ShieldCheck size={18} className="accent-icon" />
            <div>
              <h3>Google Calendar Authentication</h3>
              <p className="muted-text">
                Read-only calendar scope. OAuth tokens are stored exclusively in
                your OS keyring.
              </p>
            </div>
          </div>

          <span
            className={`pill ${
              authState === "authenticated"
                ? "pill-success"
                : authState === "offline"
                  ? "pill-warning"
                  : "pill-subtle"
            }`}
          >
            {authState === "authenticated"
              ? "Signed In"
              : authState === "not_logged_in"
                ? "Signed Out"
                : authState === "missing_credentials"
                  ? "Needs Credentials"
                  : authState === "offline"
                    ? "Offline"
                    : "Attention Needed"}
          </span>
        </div>

        {/* Credentials Status Box */}
        <div className="creds-status-box">
          <div className="creds-meta">
            <div className="row-center">
              <KeyRound size={15} />
              <strong>OAuth Desktop Client (credentials.json)</strong>
              {credentialsInfo.valid ? (
                <span className="pill pill-success">
                  <CheckCircle2 size={11} /> Valid
                </span>
              ) : (
                <span className="pill pill-warning">
                  {credentialsInfo.exists ? "Invalid" : "Missing"}
                </span>
              )}
            </div>
            <code className="code-path">{credentialsInfo.path}</code>
            {credentialsInfo.clientIdPreview && (
              <span className="muted-text small-text">
                Client ID: <code>{credentialsInfo.clientIdPreview}</code>
              </span>
            )}
            {credentialsInfo.error && (
              <span className="err-inline">{credentialsInfo.error}</span>
            )}
          </div>

          <div className="creds-buttons">
            <button
              type="button"
              className="btn btn-sm btn-secondary"
              onClick={() => onRevealDir("config")}
            >
              <FolderOpen size={14} />
              Open Folder
            </button>
            {credentialsInfo.exists && (
              <button
                type="button"
                className="btn btn-sm btn-ghost danger-hover"
                onClick={async () => {
                  setCredFeedback(null);
                  await onRemoveCredentials();
                }}
              >
                <Trash2 size={14} />
                Remove
              </button>
            )}
          </div>
        </div>

        {/* Dropzone / Import / Paste */}
        <div
          className={`dropzone ${dragging ? "dropzone-active" : ""}`}
          onDragOver={(e) => {
            e.preventDefault();
            setDragging(true);
          }}
          onDragLeave={() => setDragging(false)}
          onDrop={handleDrop}
        >
          <input
            ref={fileInputRef}
            type="file"
            accept=".json,application/json"
            style={{ display: "none" }}
            onChange={handleFilePicked}
          />
          <FileJson size={20} className="muted-icon" />
          <div className="dropzone-text">
            <strong>
              Drop <code>credentials.json</code> here
            </strong>
            <span>or import from disk / paste JSON content</span>
          </div>
          <div className="row-center">
            <button
              type="button"
              className="btn btn-sm btn-secondary"
              onClick={() => fileInputRef.current?.click()}
            >
              <Upload size={14} />
              Choose File
            </button>
            <button
              type="button"
              className="btn btn-sm btn-ghost"
              onClick={() => setShowPasteBox((v) => !v)}
            >
              {showPasteBox ? "Hide Editor" : "Paste JSON"}
            </button>
          </div>
        </div>

        {showPasteBox && (
          <div className="paste-editor">
            <textarea
              rows={5}
              placeholder='{"installed":{"client_id":"...","client_secret":"..."}}'
              value={jsonDraft}
              onChange={(e) => setJsonDraft(e.target.value)}
            />
            <div className="paste-actions">
              <button
                type="button"
                className="btn btn-sm btn-primary"
                disabled={!jsonDraft.trim()}
                onClick={() => processCredentialsText(jsonDraft)}
              >
                <Save size={14} />
                Validate & Save credentials.json
              </button>
            </div>
          </div>
        )}

        {credFeedback && (
          <div
            className={`inline-banner ${
              credFeedback.type === "ok" ? "banner-success" : "banner-danger"
            }`}
          >
            <span>{credFeedback.text}</span>
          </div>
        )}

        {/* Collapsible Setup Guide */}
        <button
          type="button"
          className="guide-toggle"
          onClick={() => setShowGuide((v) => !v)}
        >
          {showGuide ? <ChevronDown size={15} /> : <ChevronRight size={15} />}
          <HelpCircle size={14} />
          <span>How to generate Google Cloud Desktop OAuth credentials</span>
        </button>

        {showGuide && (
          <div className="guide-steps">
            <ol>
              <li>
                Open Google Cloud Console and enable the{" "}
                <strong>Google Calendar API</strong> on your project.
              </li>
              <li>
                Navigate to <strong>APIs &amp; Services → Credentials</strong>{" "}
                and click <strong>Create Credentials → OAuth client ID</strong>.
              </li>
              <li>
                Select <strong>Desktop app</strong> as the Application type (do
                not choose Web application).
              </li>
              <li>
                Download the JSON file and drop or import it using the box
                above.
              </li>
            </ol>
            <button
              type="button"
              className="link-btn"
              onClick={() =>
                onOpenUrl("https://console.cloud.google.com/apis/credentials")
              }
            >
              Open Google Cloud Credentials Console
              <ExternalLink size={13} />
            </button>
          </div>
        )}

        {/* Sign In / Sign Out Action Row */}
        <div className="auth-action-bar">
          <button
            type="button"
            className="btn btn-primary"
            disabled={!credentialsInfo.valid || busyAuth}
            onClick={onStartLogin}
          >
            <LogIn size={15} />
            {authState === "authenticated"
              ? "Re-authenticate with Google"
              : "Sign in with Google (--login)"}
          </button>

          {authState === "authenticated" && (
            <button
              type="button"
              className="btn btn-secondary danger-hover"
              disabled={busyAuth}
              onClick={handleLogoutClick}
            >
              <LogOut size={15} />
              Clear Keyring Token (--clear)
            </button>
          )}
        </div>
      </div>

      {/* Application & Engine Preferences */}
      <div className="panel-card">
        <div className="row-center section-head-gap">
          <Sliders size={18} className="accent-icon" />
          <h3>Preferences &amp; Runtime</h3>
        </div>

        <div className="prefs-list">
          <div className="pref-row">
            <div>
              <strong id="pref-poll-interval-title">Auto-Sync Interval</strong>
              <p className="muted-text small-text">
                How frequently the UI polls the <code>next-meeting</code> CLI
                engine (countdowns always tick locally every second).
              </p>
            </div>
            <select
              className="select-input"
              aria-labelledby="pref-poll-interval-title"
              value={settings.pollIntervalSeconds}
              onChange={(e) =>
                onUpdateSettings(
                  {
                    ...settings,
                    pollIntervalSeconds: Number(e.target.value),
                  },
                  false
                )
              }
            >
              <option value={30}>Every 30 seconds</option>
              <option value={60}>Every 1 minute</option>
              <option value={120}>Every 2 minutes</option>
              <option value={300}>Every 5 minutes</option>
              <option value={0}>Manual sync only</option>
            </select>
          </div>

          <div className="pref-row">
            <div>
              <strong id="pref-theme-title">Color Theme</strong>
              <p className="muted-text small-text">
                Match your operating system theme or lock dark/light appearance.
              </p>
            </div>
            <select
              className="select-input"
              aria-labelledby="pref-theme-title"
              value={settings.theme}
              onChange={(e) =>
                onUpdateSettings(
                  {
                    ...settings,
                    theme: e.target.value as AppSettings["theme"],
                  },
                  false
                )
              }
            >
              <option value="system">System Default</option>
              <option value="dark">Dark</option>
              <option value="light">Light</option>
            </select>
          </div>

          <div className="pref-row">
            <div>
              <strong id="pref-minimize-to-tray-title">
                Minimize to System Tray on Close
              </strong>
              <p className="muted-text small-text">
                Keep meeting reminders and tray status active in the background
                when closing the main window.
              </p>
            </div>
            <label className="switch">
              <input
                type="checkbox"
                aria-labelledby="pref-minimize-to-tray-title"
                checked={settings.minimizeToTray}
                onChange={(e) =>
                  onUpdateSettings(
                    { ...settings, minimizeToTray: e.target.checked },
                    false
                  )
                }
              />
              <span className="switch-slider" />
            </label>
          </div>

          <div className="pref-row">
            <div>
              <strong id="pref-always-on-top-title">
                Pin Window Always on Top
              </strong>
              <p className="muted-text small-text">
                Float Next Meeting above other desktop windows (great with
                Compact Widget mode).
              </p>
            </div>
            <label className="switch">
              <input
                type="checkbox"
                aria-labelledby="pref-always-on-top-title"
                checked={settings.alwaysOnTop}
                onChange={(e) =>
                  onUpdateSettings(
                    { ...settings, alwaysOnTop: e.target.checked },
                    false
                  )
                }
              />
              <span className="switch-slider" />
            </label>
          </div>

          <div className="pref-row pref-row-stack">
            <div>
              <strong id="pref-cli-path-title">
                CLI Binary Override (Optional)
              </strong>
              <p className="muted-text small-text">
                Leave blank to use the bundled optimized{" "}
                <code>next-meeting</code> sidecar binary.
              </p>
            </div>
            <div className="custom-path-row">
              <input
                type="text"
                className="text-input"
                aria-labelledby="pref-cli-path-title"
                placeholder="Default: Bundled next-meeting sidecar"
                value={customBinDraft}
                onChange={(e) => setCustomBinDraft(e.target.value)}
              />
              <button
                type="button"
                className="btn btn-sm btn-secondary"
                onClick={() =>
                  onUpdateSettings(
                    {
                      ...settings,
                      customCliPath: customBinDraft.trim() || null,
                    },
                    true
                  )
                }
              >
                Apply
              </button>
              {settings.customCliPath && (
                <button
                  type="button"
                  className="btn btn-sm btn-ghost"
                  onClick={() => {
                    setCustomBinDraft("");
                    onUpdateSettings({ ...settings, customCliPath: null }, true);
                  }}
                  title="Reset to bundled sidecar"
                >
                  <RotateCcw size={14} />
                  Reset
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
