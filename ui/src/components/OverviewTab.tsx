import {
  AlertTriangle,
  Bell,
  BellOff,
  CalendarCheck2,
  Check,
  Clock,
  Coffee,
  Copy,
  ExternalLink,
  Filter,
  KeyRound,
  LogIn,
  MapPin,
  RefreshCw,
  Sparkles,
  Users,
  Video,
  WifiOff,
} from "lucide-react";
import { useState } from "react";
import type {
  AppSettings,
  DashboardSnapshot,
  EnrichedMeeting,
  TabId,
} from "../types";
import {
  formatClockRange,
  formatCountdown,
  formatMinutesHuman,
  providerDisplayName,
  responseStatusLabel,
} from "../utils/time";

interface OverviewTabProps {
  snapshot: DashboardSnapshot;
  settings: AppSettings;
  nowMs: number;
  syncing: boolean;
  onSync: (force: boolean) => void;
  onUpdateSettings: (next: AppSettings, resync?: boolean) => void;
  onOpenUrl: (url: string) => void;
  onStartLogin: () => void;
  onSwitchTab: (tab: TabId) => void;
}

const NOTIFY_PRESETS = ["1m", "5m", "10m", "15m", "30m", "1h"];

function MeetingHeroCard({
  meeting,
  variant,
  nowMs,
  notifyEnabled,
  notifyThreshold,
  onOpenUrl,
}: {
  meeting: EnrichedMeeting;
  variant: "current" | "next";
  nowMs: number;
  notifyEnabled: boolean;
  notifyThreshold: string;
  onOpenUrl: (url: string) => void;
}) {
  const [copied, setCopied] = useState(false);

  const isCurrent = variant === "current";
  const totalMs = Math.max(1, meeting.endEpochMs - meeting.startEpochMs);
  const elapsedMs = Math.max(0, Math.min(totalMs, nowMs - meeting.startEpochMs));
  const progressPct = isCurrent ? Math.round((elapsedMs / totalMs) * 100) : 0;

  const deltaMs = isCurrent
    ? Math.max(0, meeting.endEpochMs - nowMs)
    : Math.max(0, meeting.startEpochMs - nowMs);

  const rsvp = responseStatusLabel(meeting.selfResponseStatus);

  const handleCopyLink = async () => {
    if (!meeting.joinUrl) return;
    await navigator.clipboard.writeText(meeting.joinUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  };

  return (
    <div className={`hero-card ${isCurrent ? "hero-current" : "hero-next"}`}>
      <div className="hero-header">
        <div className="hero-badges">
          {isCurrent ? (
            <span className="pill pill-live">
              <span className="pulse-dot" />
              In Progress
            </span>
          ) : (
            <span className="pill pill-upcoming">
              <Clock size={13} />
              Up Next
            </span>
          )}
          <span className={`pill pill-${rsvp.tone}`}>{rsvp.label}</span>
          {meeting.isOverlapping && (
            <span className="pill pill-warning">
              <AlertTriangle size={12} />
              Overlap
            </span>
          )}
          {!isCurrent && notifyEnabled && (
            <span className="pill pill-subtle">
              <Bell size={12} />
              {meeting.notified ? "Alert sent" : `Alert at ${notifyThreshold}`}
            </span>
          )}
        </div>

        <div className="countdown-chip">
          <span className="countdown-label">
            {isCurrent ? "Ends in" : "Starts in"}
          </span>
          <span className="countdown-value">{formatCountdown(deltaMs)}</span>
        </div>
      </div>

      <h2 className="hero-title">{meeting.summary}</h2>

      <div className="hero-meta">
        <span className="meta-item">
          <Clock size={14} />
          {formatClockRange(meeting.start, meeting.end)} (
          {formatMinutesHuman(meeting.durationMinutes)})
        </span>
        {meeting.attendees > 0 && (
          <span className="meta-item">
            <Users size={14} />
            {meeting.attendees} {meeting.attendees === 1 ? "attendee" : "attendees"}
          </span>
        )}
        {meeting.location && (
          <span className="meta-item meta-truncate" title={meeting.location}>
            <MapPin size={14} />
            {meeting.location}
          </span>
        )}
      </div>

      {isCurrent && (
        <div className="progress-track" aria-label="Meeting elapsed progress">
          <div
            className="progress-fill"
            style={{ width: `${progressPct}%` }}
          />
        </div>
      )}

      {meeting.joinUrl && (
        <div className="hero-actions">
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => onOpenUrl(meeting.joinUrl!)}
          >
            <Video size={16} />
            Join {providerDisplayName(meeting.meetingProvider)}
            <ExternalLink size={13} />
          </button>
          <button
            type="button"
            className="btn btn-ghost"
            onClick={handleCopyLink}
            title="Copy meeting link"
          >
            {copied ? <Check size={15} /> : <Copy size={15} />}
            {copied ? "Copied" : "Copy Link"}
          </button>
        </div>
      )}
    </div>
  );
}

export function OverviewTab({
  snapshot,
  settings,
  nowMs,
  syncing,
  onSync,
  onUpdateSettings,
  onOpenUrl,
  onStartLogin,
  onSwitchTab,
}: OverviewTabProps) {
  const { authState, currentMeeting, nextMeeting, stats } = snapshot;

  if (authState === "missing_credentials") {
    return (
      <div className="callout-panel">
        <div className="callout-icon callout-icon-warn">
          <KeyRound size={26} />
        </div>
        <h2>Connect Google Calendar OAuth Credentials</h2>
        <p className="muted-text">
          To read your schedule privately using the <code>next-meeting</code>{" "}
          CLI engine, import or paste your Desktop OAuth{" "}
          <code>credentials.json</code> file. All tokens stay locally in your OS
          keyring.
        </p>
        <div className="callout-actions">
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => onSwitchTab("settings")}
          >
            <KeyRound size={16} />
            Configure credentials.json
          </button>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() =>
              onOpenUrl("https://console.cloud.google.com/apis/credentials")
            }
          >
            Open Google Cloud Console
            <ExternalLink size={14} />
          </button>
        </div>
      </div>
    );
  }

  if (authState === "not_logged_in") {
    return (
      <div className="callout-panel">
        <div className="callout-icon callout-icon-accent">
          <LogIn size={26} />
        </div>
        <h2>Sign in to Google Calendar</h2>
        <p className="muted-text">
          Your OAuth client configuration is ready. Authorize read-only calendar
          access via <code>next-meeting --login</code> to store your token
          securely in your system keyring.
        </p>
        <div className="callout-actions">
          <button
            type="button"
            className="btn btn-primary"
            onClick={onStartLogin}
          >
            <LogIn size={16} />
            Sign in with Google
          </button>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => onSwitchTab("settings")}
          >
            Review Setup
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="tab-stack">
      {authState === "offline" && (
        <div className="inline-banner banner-warning">
          <WifiOff size={16} />
          <span>
            Calendar API is unreachable. Showing your most recent cached
            schedule.
          </span>
          <button
            type="button"
            className="btn btn-xs btn-secondary"
            onClick={() => onSync(true)}
            disabled={syncing}
          >
            <RefreshCw size={13} className={syncing ? "spin" : ""} />
            Retry
          </button>
        </div>
      )}

      {authState === "error" && snapshot.errorMessage && (
        <div className="inline-banner banner-danger">
          <AlertTriangle size={16} />
          <span>{snapshot.errorMessage}</span>
          <button
            type="button"
            className="btn btn-xs btn-secondary"
            onClick={() => onSwitchTab("console")}
          >
            Inspect CLI
          </button>
        </div>
      )}

      {/* Active & Upcoming Focus Cards */}
      {currentMeeting || nextMeeting ? (
        <div
          className={`hero-grid ${
            currentMeeting && nextMeeting ? "hero-grid-dual" : ""
          }`}
        >
          {currentMeeting && (
            <MeetingHeroCard
              meeting={currentMeeting}
              variant="current"
              nowMs={nowMs}
              notifyEnabled={settings.notifyEnabled}
              notifyThreshold={settings.notifyThreshold}
              onOpenUrl={onOpenUrl}
            />
          )}
          {nextMeeting && (
            <MeetingHeroCard
              meeting={nextMeeting}
              variant="next"
              nowMs={nowMs}
              notifyEnabled={settings.notifyEnabled}
              notifyThreshold={settings.notifyThreshold}
              onOpenUrl={onOpenUrl}
            />
          )}
        </div>
      ) : (
        <div className="empty-focus-card">
          <div className="empty-focus-icon">
            <Sparkles size={24} />
          </div>
          <div className="empty-focus-body">
            <h2>Clear schedule ahead</h2>
            <p className="muted-text">
              {stats.completedCount > 0
                ? `You wrapped up ${stats.completedCount} ${
                    stats.completedCount === 1 ? "meeting" : "meetings"
                  } earlier today. No more meetings remaining.`
                : "No meetings on your calendar for the rest of today. Enjoy uninterrupted focus time."}
            </p>
          </div>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => onSync(true)}
            disabled={syncing}
          >
            <RefreshCw size={15} className={syncing ? "spin" : ""} />
            Force Refresh
          </button>
        </div>
      )}

      {/* Daily Metrics */}
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-top">
            <span className="stat-label">Today&apos;s Meetings</span>
            <CalendarCheck2 size={15} className="stat-icon" />
          </div>
          <div className="stat-value">{stats.totalMeetings}</div>
          <div className="stat-sub">
            {stats.acceptedCount} accepted · {stats.completedCount} done
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-top">
            <span className="stat-label">Remaining</span>
            <Clock size={15} className="stat-icon" />
          </div>
          <div className="stat-value">{stats.remainingCount}</div>
          <div className="stat-sub">
            {formatMinutesHuman(stats.remainingScheduledMinutes)} left today
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-top">
            <span className="stat-label">Scheduled Time</span>
            <Users size={15} className="stat-icon" />
          </div>
          <div className="stat-value">
            {formatMinutesHuman(stats.totalScheduledMinutes)}
          </div>
          <div className="stat-sub">Total meeting load today</div>
        </div>

        <div className="stat-card">
          <div className="stat-top">
            <span className="stat-label">Focus Window</span>
            <Coffee size={15} className="stat-icon" />
          </div>
          <div className="stat-value">
            {currentMeeting
              ? "In Call"
              : stats.freeUntilNextMinutes !== null
                ? formatMinutesHuman(stats.freeUntilNextMinutes)
                : "Open"}
          </div>
          <div className="stat-sub">
            {currentMeeting
              ? "Currently in an active meeting"
              : nextMeeting
                ? "Free before next meeting"
                : "Rest of day is open"}
          </div>
        </div>
      </div>

      {/* Interactive CLI Engine Flags Strip */}
      <div className="panel-card engine-flags-bar">
        <div className="flag-group">
          <div className="flag-title">
            <Filter size={15} />
            <div>
              <strong>Accepted Meetings Only</strong>
              <span className="flag-code">--only-accepted</span>
            </div>
          </div>
          <label className="switch">
            <input
              type="checkbox"
              aria-label="Accepted meetings only (--only-accepted)"
              checked={settings.onlyAccepted}
              onChange={(e) =>
                onUpdateSettings(
                  { ...settings, onlyAccepted: e.target.checked },
                  true
                )
              }
            />
            <span className="switch-slider" />
          </label>
        </div>

        <div className="flag-divider" />

        <div className="flag-group flag-group-wide">
          <div className="flag-title">
            {settings.notifyEnabled ? <Bell size={15} /> : <BellOff size={15} />}
            <div>
              <strong>Desktop Notification</strong>
              <span className="flag-code">
                {settings.notifyEnabled
                  ? `--notify ${settings.notifyThreshold}`
                  : "disabled"}
              </span>
            </div>
          </div>

          <div className="preset-pills">
            <button
              type="button"
              className={`preset-btn ${!settings.notifyEnabled ? "active" : ""}`}
              onClick={() =>
                onUpdateSettings({ ...settings, notifyEnabled: false }, true)
              }
            >
              Off
            </button>
            {NOTIFY_PRESETS.map((dur) => (
              <button
                key={dur}
                type="button"
                className={`preset-btn ${
                  settings.notifyEnabled && settings.notifyThreshold === dur
                    ? "active"
                    : ""
                }`}
                onClick={() =>
                  onUpdateSettings(
                    {
                      ...settings,
                      notifyEnabled: true,
                      notifyThreshold: dur,
                    },
                    true
                  )
                }
              >
                {dur}
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
