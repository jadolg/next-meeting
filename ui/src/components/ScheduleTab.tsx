import {
  AlertTriangle,
  Bell,
  Calendar,
  Check,
  Coffee,
  Copy,
  ExternalLink,
  MapPin,
  Search,
  Users,
  Video,
} from "lucide-react";
import { useMemo, useState } from "react";
import type { EnrichedMeeting } from "../types";
import {
  formatClockRange,
  formatCountdown,
  formatMinutesHuman,
  providerDisplayName,
  responseStatusLabel,
} from "../utils/time";

interface ScheduleTabProps {
  events: EnrichedMeeting[];
  nowMs: number;
  onOpenUrl: (url: string) => void;
}

type ScheduleFilter = "all" | "upcoming" | "accepted" | "unconfirmed";

export function ScheduleTab({ events, nowMs, onOpenUrl }: ScheduleTabProps) {
  const [filter, setFilter] = useState<ScheduleFilter>("all");
  const [query, setQuery] = useState("");
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const filteredEvents = useMemo(() => {
    const q = query.trim().toLowerCase();
    return events.filter((ev) => {
      if (filter === "upcoming" && ev.phase === "past") return false;
      if (
        filter === "accepted" &&
        ev.selfResponseStatus !== "accepted" &&
        ev.selfResponseStatus !== "tentative"
      ) {
        return false;
      }
      if (
        filter === "unconfirmed" &&
        (ev.selfResponseStatus === "accepted" ||
          ev.selfResponseStatus === "tentative")
      ) {
        return false;
      }
      if (q) {
        const haystack =
          `${ev.summary} ${ev.location} ${ev.meetingProvider ?? ""}`.toLowerCase();
        if (!haystack.includes(q)) return false;
      }
      return true;
    });
  }, [events, filter, query]);

  const handleCopy = async (id: string, url: string) => {
    await navigator.clipboard.writeText(url);
    setCopiedId(id);
    setTimeout(() => setCopiedId((prev) => (prev === id ? null : prev)), 1600);
  };

  return (
    <div className="tab-stack">
      <div className="schedule-toolbar">
        <div className="search-box">
          <Search size={15} className="search-icon" />
          <input
            type="text"
            placeholder="Filter meetings by title, room, or provider..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          {query && (
            <button
              type="button"
              className="clear-input-btn"
              onClick={() => setQuery("")}
            >
              ×
            </button>
          )}
        </div>

        <div className="preset-pills">
          {(
            [
              ["all", `All (${events.length})`],
              [
                "upcoming",
                `Active & Upcoming (${events.filter((e) => e.phase !== "past").length})`,
              ],
              ["accepted", "Accepted"],
              ["unconfirmed", "Pending / Declined"],
            ] as const
          ).map(([id, label]) => (
            <button
              key={id}
              type="button"
              className={`preset-btn ${filter === id ? "active" : ""}`}
              onClick={() => setFilter(id)}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {filteredEvents.length === 0 ? (
        <div className="empty-list-card">
          <Calendar size={28} className="muted-icon" />
          <h3>No matching events</h3>
          <p className="muted-text">
            {events.length === 0
              ? "Your calendar has no timed events scheduled for today."
              : "Try clearing your search or switching the status filter above."}
          </p>
        </div>
      ) : (
        <div className="timeline-list">
          {filteredEvents.map((meeting, idx) => {
            const prev = idx > 0 ? filteredEvents[idx - 1] : null;
            const gapMinutes =
              prev && meeting.startEpochMs > prev.endEpochMs
                ? Math.floor((meeting.startEpochMs - prev.endEpochMs) / 60_000)
                : 0;

            const rsvp = responseStatusLabel(meeting.selfResponseStatus);
            const isLive =
              nowMs >= meeting.startEpochMs && nowMs < meeting.endEpochMs;
            const isPast = nowMs >= meeting.endEpochMs;

            return (
              <div key={`${meeting.id}-${meeting.start}`} className="timeline-block">
                {gapMinutes >= 15 && (
                  <div className="gap-indicator">
                    <Coffee size={12} />
                    <span>{formatMinutesHuman(gapMinutes)} open break</span>
                  </div>
                )}

                <div
                  className={`timeline-card ${
                    isLive
                      ? "timeline-live"
                      : isPast
                        ? "timeline-past"
                        : "timeline-upcoming"
                  }`}
                >
                  <div className="timeline-time-col">
                    <span className="timeline-range">
                      {formatClockRange(meeting.start, meeting.end)}
                    </span>
                    <span className="timeline-dur">
                      {formatMinutesHuman(meeting.durationMinutes)}
                    </span>
                  </div>

                  <div className="timeline-main-col">
                    <div className="timeline-title-row">
                      <h3 className="timeline-title">{meeting.summary}</h3>
                      <div className="timeline-pills">
                        {isLive && (
                          <span className="pill pill-live">
                            <span className="pulse-dot" />
                            {formatCountdown(meeting.endEpochMs - nowMs)} left
                          </span>
                        )}
                        {!isLive && !isPast && (
                          <span className="pill pill-upcoming">
                            in {formatCountdown(meeting.startEpochMs - nowMs)}
                          </span>
                        )}
                        {isPast && <span className="pill pill-subtle">Ended</span>}
                        <span className={`pill pill-${rsvp.tone}`}>
                          {rsvp.label}
                        </span>
                        {meeting.isOverlapping && (
                          <span className="pill pill-warning">
                            <AlertTriangle size={11} />
                            Conflict
                          </span>
                        )}
                        {meeting.notified && (
                          <span
                            className="pill pill-subtle"
                            title="Desktop notification already sent by CLI"
                          >
                            <Bell size={11} />
                            Notified
                          </span>
                        )}
                      </div>
                    </div>

                    <div className="timeline-submeta">
                      {meeting.attendees > 0 && (
                        <span>
                          <Users size={13} /> {meeting.attendees} attendees
                        </span>
                      )}
                      {meeting.location && (
                        <span className="meta-truncate" title={meeting.location}>
                          <MapPin size={13} /> {meeting.location}
                        </span>
                      )}
                    </div>
                  </div>

                  {meeting.joinUrl && (
                    <div className="timeline-actions-col">
                      <button
                        type="button"
                        className={`btn btn-sm ${
                          isLive ? "btn-primary" : "btn-secondary"
                        }`}
                        onClick={() => onOpenUrl(meeting.joinUrl!)}
                      >
                        <Video size={14} />
                        {providerDisplayName(meeting.meetingProvider)}
                        <ExternalLink size={12} />
                      </button>
                      <button
                        type="button"
                        className="icon-btn"
                        onClick={() => handleCopy(meeting.id, meeting.joinUrl!)}
                        title="Copy meeting URL"
                      >
                        {copiedId === meeting.id ? (
                          <Check size={14} />
                        ) : (
                          <Copy size={14} />
                        )}
                      </button>
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
