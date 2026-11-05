use chrono::{DateTime, FixedOffset};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AppSettings {
    pub only_accepted: bool,
    pub notify_enabled: bool,
    pub notify_threshold: String,
    pub poll_interval_seconds: u64,
    pub minimize_to_tray: bool,
    pub compact_mode: bool,
    pub always_on_top: bool,
    pub theme: String,
    pub custom_cli_path: Option<String>,
}

impl Default for AppSettings {
    fn default() -> Self {
        Self {
            only_accepted: false,
            notify_enabled: true,
            notify_threshold: "10m".to_string(),
            poll_interval_seconds: 60,
            minimize_to_tray: true,
            compact_mode: false,
            always_on_top: false,
            theme: "system".to_string(),
            custom_cli_path: None,
        }
    }
}

/// Raw JSON schema written by Go CLI `cache.Write()` in `next-meeting-cache.json`.
#[derive(Debug, Clone, Deserialize)]
pub struct RawCachedData {
    pub timestamp: String,
    #[serde(default)]
    pub events: Option<Vec<RawMeetingInfo>>,
}

/// Raw JSON event written by Go `calendar.MeetingInfo` (PascalCase fields).
#[derive(Debug, Clone, Deserialize)]
pub struct RawMeetingInfo {
    #[serde(rename = "Summary", default)]
    pub summary: String,
    #[serde(rename = "Start")]
    pub start: String,
    #[serde(rename = "End")]
    pub end: String,
    #[serde(rename = "IsAllDay", default)]
    pub is_all_day: bool,
    #[serde(rename = "Location", default)]
    pub location: String,
    #[serde(rename = "HangoutLink", default)]
    pub hangout_link: String,
    #[serde(rename = "Attendees", default)]
    pub attendees: i32,
    #[serde(rename = "SelfResponseStatus", default)]
    pub self_response_status: String,
}

#[derive(Debug, Clone)]
pub struct ParsedMeeting {
    pub raw: RawMeetingInfo,
    pub start_dt: DateTime<FixedOffset>,
    pub end_dt: DateTime<FixedOffset>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum AuthState {
    Authenticated,
    NotLoggedIn,
    MissingCredentials,
    Offline,
    Error,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum MeetingPhase {
    Past,
    Current,
    Upcoming,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct EnrichedMeeting {
    pub id: String,
    pub summary: String,
    pub start: String,
    pub end: String,
    pub start_epoch_ms: i64,
    pub end_epoch_ms: i64,
    pub duration_minutes: i64,
    pub is_all_day: bool,
    pub location: String,
    pub hangout_link: String,
    pub join_url: Option<String>,
    pub meeting_provider: Option<String>,
    pub attendees: i32,
    pub self_response_status: String,
    pub phase: MeetingPhase,
    pub notified: bool,
    pub is_overlapping: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DailyStats {
    pub total_meetings: usize,
    pub accepted_count: usize,
    pub remaining_count: usize,
    pub completed_count: usize,
    pub total_scheduled_minutes: i64,
    pub remaining_scheduled_minutes: i64,
    pub free_until_next_minutes: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CacheInfo {
    pub path: String,
    pub exists: bool,
    pub timestamp: Option<String>,
    pub age_seconds: Option<i64>,
    pub ttl_remaining_seconds: Option<i64>,
    pub is_expired: bool,
    pub notified_count: usize,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CredentialsInfo {
    pub path: String,
    pub exists: bool,
    pub valid: bool,
    pub client_id_preview: Option<String>,
    pub error: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct EngineInfo {
    pub binary_path: String,
    pub source: String,
    pub available: bool,
    pub binary_size_bytes: Option<u64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CliExecutionRecord {
    pub id: u64,
    pub timestamp: String,
    pub command: String,
    pub args: Vec<String>,
    pub stdout: String,
    pub stderr: String,
    pub exit_code: i32,
    pub duration_ms: u64,
    pub trigger: String,
    pub success: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DashboardSnapshot {
    pub raw_cli_output: String,
    pub auth_state: AuthState,
    pub error_message: Option<String>,
    pub current_meeting: Option<EnrichedMeeting>,
    pub next_meeting: Option<EnrichedMeeting>,
    pub events: Vec<EnrichedMeeting>,
    pub stats: DailyStats,
    pub cache_info: CacheInfo,
    pub credentials_info: CredentialsInfo,
    pub engine_info: EngineInfo,
    pub last_execution: Option<CliExecutionRecord>,
    pub synced_at: String,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct LoginProgressPayload {
    pub stage: String,
    pub url: Option<String>,
    pub message: String,
    pub success: Option<bool>,
}
