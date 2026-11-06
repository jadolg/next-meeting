import {
  CheckCircle2,
  Cpu,
  Database,
  FolderOpen,
  Play,
  Terminal,
  Trash2,
  XCircle,
} from "lucide-react";
import { useState } from "react";
import type { CliExecutionRecord, DashboardSnapshot } from "../types";
import {
  formatBytes,
  formatClockTime,
  formatRelativeSeconds,
} from "../utils/time";

interface ConsoleTabProps {
  snapshot: DashboardSnapshot;
  history: CliExecutionRecord[];
  runningPreset: string | null;
  onRunPreset: (
    preset: "status" | "only_accepted" | "clear_cache" | "notify_now" | "help"
  ) => void;
  onClearHistory: () => void;
  onRevealDir: (target: "config" | "cache") => void;
}

export function ConsoleTab({
  snapshot,
  history,
  runningPreset,
  onRunPreset,
  onClearHistory,
  onRevealDir,
}: ConsoleTabProps) {
  const [expandedId, setExpandedId] = useState<number | null>(
    history[0]?.id ?? null
  );

  const { engineInfo, cacheInfo } = snapshot;

  return (
    <div className="tab-stack">
      {/* Engine Diagnostics Grid */}
      <div className="diagnostics-grid">
        <div className="panel-card diag-card">
          <div className="diag-header">
            <Cpu size={16} className="accent-icon" />
            <h3>CLI Execution Engine</h3>
            <span
              className={`pill ${
                engineInfo.available ? "pill-success" : "pill-danger"
              }`}
            >
              {engineInfo.available ? engineInfo.source : "Unavailable"}
            </span>
          </div>
          <code className="code-path" title={engineInfo.binaryPath}>
            {engineInfo.binaryPath}
          </code>
          <div className="diag-footer">
            <span>Binary size: {formatBytes(engineInfo.binarySizeBytes)}</span>
            <span>Mode: Subprocess IPC (UTF-8)</span>
          </div>
        </div>

        <div className="panel-card diag-card">
          <div className="diag-header">
            <Database size={16} className="accent-icon" />
            <h3>Calendar Event Cache (30m TTL)</h3>
            <span
              className={`pill ${
                cacheInfo.exists && !cacheInfo.isExpired
                  ? "pill-success"
                  : "pill-subtle"
              }`}
            >
              {cacheInfo.exists
                ? cacheInfo.isExpired
                  ? "Expired"
                  : `Valid · ${formatRelativeSeconds(cacheInfo.ttlRemainingSeconds)} left`
                : "Empty"}
            </span>
          </div>
          <code className="code-path" title={cacheInfo.path}>
            {cacheInfo.path}
          </code>
          <div className="diag-footer">
            <span>
              Age: {formatRelativeSeconds(cacheInfo.ageSeconds)} · Tracked alerts:{" "}
              {cacheInfo.notifiedCount}
            </span>
            <button
              type="button"
              className="link-btn"
              onClick={() => onRevealDir("cache")}
            >
              <FolderOpen size={13} />
              Open Temp Dir
            </button>
          </div>
        </div>
      </div>

      {/* Interactive CLI Preset Runner */}
      <div className="panel-card">
        <div className="section-title-row">
          <div>
            <h3>Interactive CLI Runner</h3>
            <p className="muted-text">
              Invoke <code>next-meeting</code> flags directly and inspect raw
              standard output, error streams, and execution latency.
            </p>
          </div>
        </div>

        <div className="runner-buttons">
          {(
            [
              ["status", "next-meeting", "Check Status"],
              [
                "only_accepted",
                "next-meeting --only-accepted",
                "Accepted Only",
              ],
              ["notify_now", "next-meeting --notify", "Evaluate Alert"],
              ["clear_cache", "next-meeting --clear-cache", "Clear Cache"],
              ["help", "next-meeting -h", "Print Flags (-h)"],
            ] as const
          ).map(([preset, cmdHint, label]) => (
            <button
              key={preset}
              type="button"
              className="btn btn-secondary runner-btn"
              disabled={runningPreset !== null}
              onClick={() => onRunPreset(preset)}
              title={cmdHint}
            >
              <Play size={13} />
              <span>{label}</span>
              <code>{cmdHint.replace("next-meeting", "").trim() || "default"}</code>
            </button>
          ))}
        </div>
      </div>

      {/* Execution History Log */}
      <div className="panel-card">
        <div className="section-title-row">
          <div className="row-center">
            <Terminal size={16} className="accent-icon" />
            <h3>CLI Execution Log ({history.length})</h3>
          </div>
          {history.length > 0 && (
            <button
              type="button"
              className="btn btn-xs btn-ghost"
              onClick={onClearHistory}
            >
              <Trash2 size={13} />
              Clear Log
            </button>
          )}
        </div>

        {history.length === 0 ? (
          <p className="muted-text empty-inline">
            No CLI executions recorded in this session yet.
          </p>
        ) : (
          <div className="exec-log-list">
            {history.map((item) => {
              const isOpen = expandedId === item.id;
              return (
                <div
                  key={item.id}
                  className={`exec-item ${!item.success ? "exec-failed" : ""}`}
                >
                  <button
                    type="button"
                    className="exec-summary"
                    onClick={() => setExpandedId(isOpen ? null : item.id)}
                  >
                    <div className="exec-left">
                      {item.success ? (
                        <CheckCircle2 size={15} className="ok-icon" />
                      ) : (
                        <XCircle size={15} className="err-icon" />
                      )}
                      <code className="exec-cmd">{item.command}</code>
                    </div>
                    <div className="exec-right">
                      <span className="pill pill-subtle">{item.trigger}</span>
                      <span className="mono-meta">{item.durationMs}ms</span>
                      <span className="mono-meta">exit {item.exitCode}</span>
                      <span className="mono-meta">
                        {formatClockTime(item.timestamp)}
                      </span>
                    </div>
                  </button>

                  {isOpen && (
                    <div className="exec-details">
                      {item.stdout && (
                        <div className="stream-block">
                          <span className="stream-label">stdout</span>
                          <pre>{item.stdout}</pre>
                        </div>
                      )}
                      {item.stderr && (
                        <div className="stream-block stream-stderr">
                          <span className="stream-label">stderr</span>
                          <pre>{item.stderr}</pre>
                        </div>
                      )}
                      {!item.stdout && !item.stderr && (
                        <span className="muted-text">
                          (Process exited cleanly with empty output)
                        </span>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
