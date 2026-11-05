export type AuthState =
  | "authenticated"
  | "not_logged_in"
  | "missing_credentials"
  | "offline"
  | "error";

export type MeetingPhase = "past" | "current" | "upcoming";

export type TabId = "overview" | "schedule" | "console" | "settings";

export interface AppSettings {
  onlyAccepted: boolean;
  notifyEnabled: boolean;
  notifyThreshold: string;
  pollIntervalSeconds: number;
  minimizeToTray: boolean;
  compactMode: boolean;
  alwaysOnTop: boolean;
  theme: "system" | "dark" | "light";
  customCliPath: string | null;
}

export interface EnrichedMeeting {
  id: string;
  summary: string;
  start: string;
  end: string;
  startEpochMs: number;
  endEpochMs: number;
  durationMinutes: number;
  isAllDay: boolean;
  location: string;
  hangoutLink: string;
  joinUrl: string | null;
  meetingProvider: string | null;
  attendees: number;
  selfResponseStatus: string;
  phase: MeetingPhase;
  notified: boolean;
  isOverlapping: boolean;
}

export interface DailyStats {
  totalMeetings: number;
  acceptedCount: number;
  remainingCount: number;
  completedCount: number;
  totalScheduledMinutes: number;
  remainingScheduledMinutes: number;
  freeUntilNextMinutes: number | null;
}

export interface CacheInfo {
  path: string;
  exists: boolean;
  timestamp: string | null;
  ageSeconds: number | null;
  ttlRemainingSeconds: number | null;
  isExpired: boolean;
  notifiedCount: number;
}

export interface CredentialsInfo {
  path: string;
  exists: boolean;
  valid: boolean;
  clientIdPreview: string | null;
  error: string | null;
}

export interface EngineInfo {
  binaryPath: string;
  source: string;
  available: boolean;
  binarySizeBytes: number | null;
}

export interface CliExecutionRecord {
  id: number;
  timestamp: string;
  command: string;
  args: string[];
  stdout: string;
  stderr: string;
  exitCode: number;
  durationMs: number;
  trigger: string;
  success: boolean;
}

export interface DashboardSnapshot {
  rawCliOutput: string;
  authState: AuthState;
  errorMessage: string | null;
  currentMeeting: EnrichedMeeting | null;
  nextMeeting: EnrichedMeeting | null;
  events: EnrichedMeeting[];
  stats: DailyStats;
  cacheInfo: CacheInfo;
  credentialsInfo: CredentialsInfo;
  engineInfo: EngineInfo;
  lastExecution: CliExecutionRecord | null;
  syncedAt: string;
}

export interface LoginProgressPayload {
  stage: "starting" | "waiting_browser" | "log" | "stderr" | "finished" | "cancelled";
  url: string | null;
  message: string;
  success: boolean | null;
}
