import {
  Clock,
  ExternalLink,
  Maximize2,
  Pin,
  RefreshCw,
  Sparkles,
  Video,
} from "lucide-react";
import type { AppSettings, DashboardSnapshot } from "../types";
import {
  formatClockRange,
  formatCountdown,
  providerDisplayName,
} from "../utils/time";

interface CompactWidgetProps {
  snapshot: DashboardSnapshot;
  settings: AppSettings;
  nowMs: number;
  syncing: boolean;
  onSync: (force: boolean) => void;
  onTogglePin: () => void;
  onExitCompact: () => void;
  onOpenUrl: (url: string) => void;
}

export function CompactWidget({
  snapshot,
  settings,
  nowMs,
  syncing,
  onSync,
  onTogglePin,
  onExitCompact,
  onOpenUrl,
}: CompactWidgetProps) {
  const focusMeeting = snapshot.currentMeeting ?? snapshot.nextMeeting;
  const isLive = Boolean(snapshot.currentMeeting);

  const deltaMs = focusMeeting
    ? isLive
      ? Math.max(0, focusMeeting.endEpochMs - nowMs)
      : Math.max(0, focusMeeting.startEpochMs - nowMs)
    : 0;

  const progressPct =
    focusMeeting && isLive
      ? Math.round(
          (Math.max(0, nowMs - focusMeeting.startEpochMs) /
            Math.max(1, focusMeeting.endEpochMs - focusMeeting.startEpochMs)) *
            100
        )
      : 0;

  return (
    <div className="compact-shell">
      <header className="compact-topbar">
        <code className="compact-cli-pill" title={snapshot.rawCliOutput}>
          {snapshot.rawCliOutput}
        </code>
        <div className="compact-controls">
          <button
            type="button"
            className={`icon-btn ${settings.alwaysOnTop ? "icon-btn-active" : ""}`}
            onClick={onTogglePin}
            title={
              settings.alwaysOnTop
                ? "Unpin Always on Top"
                : "Pin Always on Top"
            }
          >
            <Pin size={14} />
          </button>
          <button
            type="button"
            className="icon-btn"
            onClick={() => onSync(false)}
            disabled={syncing}
            title="Sync now"
          >
            <RefreshCw size={14} className={syncing ? "spin" : ""} />
          </button>
          <button
            type="button"
            className="icon-btn"
            onClick={onExitCompact}
            title="Expand to full view"
          >
            <Maximize2 size={14} />
          </button>
        </div>
      </header>

      <main className="compact-body">
        {focusMeeting ? (
          <div className="compact-card">
            <div className="row-between">
              {isLive ? (
                <span className="pill pill-live">
                  <span className="pulse-dot" />
                  Live · {formatCountdown(deltaMs)} left
                </span>
              ) : (
                <span className="pill pill-upcoming">
                  <Clock size={12} />
                  In {formatCountdown(deltaMs)}
                </span>
              )}
              <span className="muted-text small-text">
                {formatClockRange(focusMeeting.start, focusMeeting.end)}
              </span>
            </div>

            <h2 className="compact-title">{focusMeeting.summary}</h2>

            {isLive && (
              <div className="progress-track">
                <div
                  className="progress-fill"
                  style={{ width: `${progressPct}%` }}
                />
              </div>
            )}

            <div className="compact-footer">
              {focusMeeting.joinUrl ? (
                <button
                  type="button"
                  className="btn btn-sm btn-primary compact-join-btn"
                  onClick={() => onOpenUrl(focusMeeting.joinUrl!)}
                >
                  <Video size={14} />
                  Join {providerDisplayName(focusMeeting.meetingProvider)}
                  <ExternalLink size={12} />
                </button>
              ) : (
                <span className="muted-text small-text">
                  {focusMeeting.location || "No video link attached"}
                </span>
              )}

              {snapshot.currentMeeting && snapshot.nextMeeting && (
                <span
                  className="muted-text small-text compact-next-hint"
                  title={snapshot.nextMeeting.summary}
                >
                  Next: {snapshot.nextMeeting.summary}
                </span>
              )}
            </div>
          </div>
        ) : (
          <div className="compact-empty">
            <Sparkles size={20} className="accent-icon" />
            <div>
              <strong>No upcoming meetings</strong>
              <p className="muted-text small-text">
                {snapshot.authState === "authenticated"
                  ? "Your schedule is clear for the rest of today."
                  : "Expand window to sign in or configure Google OAuth."}
              </p>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
