export function formatCountdown(deltaMs: number): string {
  if (deltaMs <= 0) {
    return "now";
  }
  const totalSeconds = Math.floor(deltaMs / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (hours > 0) {
    return `${hours}h ${String(minutes).padStart(2, "0")}m ${String(seconds).padStart(2, "0")}s`;
  }
  if (minutes > 0) {
    return `${minutes}m ${String(seconds).padStart(2, "0")}s`;
  }
  return `${seconds}s`;
}

export function formatMinutesHuman(totalMinutes: number): string {
  if (totalMinutes <= 0) {
    return "0m";
  }
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  if (hours > 0 && minutes > 0) {
    return `${hours}h ${minutes}m`;
  }
  if (hours > 0) {
    return `${hours}h`;
  }
  return `${minutes}m`;
}

export function formatClockTime(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return iso;
  }
  return date.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function formatClockRange(startIso: string, endIso: string): string {
  return `${formatClockTime(startIso)} – ${formatClockTime(endIso)}`;
}

export function formatRelativeSeconds(seconds: number | null): string {
  if (seconds === null || seconds < 0) {
    return "—";
  }
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const mins = Math.floor(seconds / 60);
  const remSecs = seconds % 60;
  if (mins < 60) {
    return remSecs > 0 ? `${mins}m ${remSecs}s` : `${mins}m`;
  }
  const hrs = Math.floor(mins / 60);
  return `${hrs}h ${mins % 60}m`;
}

export function formatBytes(bytes: number | null): string {
  if (bytes === null || bytes <= 0) {
    return "—";
  }
  const units = ["B", "KB", "MB", "GB"];
  let value = bytes;
  let idx = 0;
  while (value >= 1024 && idx < units.length - 1) {
    value /= 1024;
    idx++;
  }
  return `${value.toFixed(idx === 0 ? 0 : 1)} ${units[idx]}`;
}

export function providerDisplayName(provider: string | null): string {
  switch (provider) {
    case "google_meet":
      return "Google Meet";
    case "zoom":
      return "Zoom";
    case "teams":
      return "Microsoft Teams";
    case "webex":
      return "Webex";
    case "link":
      return "Meeting Link";
    default:
      return "Join Call";
  }
}

export function responseStatusLabel(status: string): {
  label: string;
  tone: "success" | "warning" | "danger" | "neutral";
} {
  switch (status) {
    case "accepted":
      return { label: "Accepted", tone: "success" };
    case "tentative":
      return { label: "Tentative", tone: "warning" };
    case "declined":
      return { label: "Declined", tone: "danger" };
    case "needsAction":
      return { label: "Needs RSVP", tone: "neutral" };
    default:
      return { label: "Scheduled", tone: "neutral" };
  }
}
