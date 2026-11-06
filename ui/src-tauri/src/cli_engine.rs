use std::collections::VecDeque;
use std::env;
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Stdio;
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{Duration, Instant};

use chrono::{DateTime, FixedOffset, SecondsFormat, Utc};
use sha2::{Digest, Sha256};
use tauri::{AppHandle, Emitter};
use tokio::io::{AsyncBufReadExt, BufReader};
use tokio::process::{Child, Command};
use tokio::sync::Mutex;
use tokio::time::timeout;

use crate::models::{
    AppSettings, AuthState, CacheInfo, CliExecutionRecord, DashboardSnapshot, DailyStats,
    EngineInfo, EnrichedMeeting, LoginProgressPayload, MeetingPhase, ParsedMeeting, RawCachedData,
};
use crate::paths::{cache_path, inspect_credentials, load_settings, notify_dir_path};

const MAX_HISTORY_ENTRIES: usize = 50;
const CACHE_TTL_SECONDS: i64 = 30 * 60;
const CLI_TIMEOUT: Duration = Duration::from_secs(25);

#[cfg(target_os = "windows")]
const CREATE_NO_WINDOW: u32 = 0x0800_0000;

pub struct EngineState {
    pub settings: Mutex<AppSettings>,
    pub sync_lock: Mutex<()>,
    pub history: Mutex<VecDeque<CliExecutionRecord>>,
    pub active_login: Mutex<Option<Child>>,
    pub last_snapshot: Mutex<Option<DashboardSnapshot>>,
    next_exec_id: AtomicU64,
}

impl Default for EngineState {
    fn default() -> Self {
        Self::new()
    }
}

impl EngineState {
    pub fn new() -> Self {
        Self {
            settings: Mutex::new(load_settings()),
            sync_lock: Mutex::new(()),
            history: Mutex::new(VecDeque::with_capacity(MAX_HISTORY_ENTRIES)),
            active_login: Mutex::new(None),
            last_snapshot: Mutex::new(None),
            next_exec_id: AtomicU64::new(1),
        }
    }

    pub async fn record_execution(&self, mut record: CliExecutionRecord) -> CliExecutionRecord {
        let id = self.next_exec_id.fetch_add(1, Ordering::Relaxed);
        record.id = id;
        let mut history = self.history.lock().await;
        if history.len() >= MAX_HISTORY_ENTRIES {
            history.pop_back();
        }
        history.push_front(record.clone());
        record
    }
}

fn cli_binary_filename() -> &'static str {
    if cfg!(target_os = "windows") {
        "next-meeting.exe"
    } else {
        "next-meeting"
    }
}

pub fn resolve_engine_info(custom_cli_path: Option<&str>) -> EngineInfo {
    // 1. Explicit user-configured path
    if let Some(custom) = custom_cli_path.map(str::trim).filter(|s| !s.is_empty()) {
        let candidate = PathBuf::from(custom);
        let meta = fs::metadata(&candidate).ok();
        let available = meta.as_ref().map(|m| m.is_file()).unwrap_or(false);
        return EngineInfo {
            binary_path: candidate.to_string_lossy().to_string(),
            source: "Custom Path".to_string(),
            available,
            binary_size_bytes: meta.map(|m| m.len()),
        };
    }

    // 2. Bundled Tauri sidecar adjacent to the running UI executable
    if let Ok(current_exe) = env::current_exe() {
        if let Some(exe_dir) = current_exe.parent() {
            let sibling = exe_dir.join(cli_binary_filename());
            if let Ok(meta) = fs::metadata(&sibling) {
                if meta.is_file() {
                    return EngineInfo {
                        binary_path: sibling.to_string_lossy().to_string(),
                        source: "Bundled Sidecar".to_string(),
                        available: true,
                        binary_size_bytes: Some(meta.len()),
                    };
                }
            }
        }
    }

    // 3. Development fallback inside src-tauri/binaries/ (e.g. during `cargo test`)
    let manifest_binaries = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("binaries");
    if let Ok(entries) = fs::read_dir(&manifest_binaries) {
        for entry in entries.flatten() {
            let path = entry.path();
            let name = entry.file_name().to_string_lossy().to_string();
            let matches_platform = if cfg!(target_os = "windows") {
                name.starts_with("next-meeting-") && name.ends_with(".exe")
            } else if cfg!(target_os = "macos") {
                name.starts_with("next-meeting-") && name.contains("apple-darwin")
            } else {
                name.starts_with("next-meeting-") && name.contains("linux")
            };
            if matches_platform {
                if let Ok(meta) = fs::metadata(&path) {
                    if meta.is_file() {
                        return EngineInfo {
                            binary_path: path.to_string_lossy().to_string(),
                            source: "Dev Sidecar".to_string(),
                            available: true,
                            binary_size_bytes: Some(meta.len()),
                        };
                    }
                }
            }
        }
    }

    // 4. System PATH lookup
    if let Some(path_var) = env::var_os("PATH") {
        for dir in env::split_paths(&path_var) {
            let candidate = dir.join(cli_binary_filename());
            if let Ok(meta) = fs::metadata(&candidate) {
                if meta.is_file() {
                    return EngineInfo {
                        binary_path: candidate.to_string_lossy().to_string(),
                        source: "System PATH".to_string(),
                        available: true,
                        binary_size_bytes: Some(meta.len()),
                    };
                }
            }
        }
    }

    EngineInfo {
        binary_path: cli_binary_filename().to_string(),
        source: "Not Found".to_string(),
        available: false,
        binary_size_bytes: None,
    }
}

fn configure_command(binary_path: &str, args: &[String]) -> Command {
    let mut cmd = Command::new(binary_path);
    cmd.args(args)
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .kill_on_drop(true);

    #[cfg(target_os = "windows")]
    {
        cmd.creation_flags(CREATE_NO_WINDOW);
    }

    cmd
}

pub fn validate_notify_duration(raw: &str) -> Result<String, String> {
    let trimmed = raw.trim();
    if trimmed.is_empty() {
        return Err("Notification duration cannot be empty".to_string());
    }
    if trimmed.len() > 16
        || !trimmed
            .chars()
            .all(|c| c.is_ascii_digit() || matches!(c, 's' | 'm' | 'h' | '.'))
        || !trimmed.chars().any(|c| c.is_ascii_digit())
        || !trimmed.ends_with(['s', 'm', 'h'])
    {
        return Err(
            "Invalid duration format. Use Go duration syntax such as '5m', '10m', '30m', or '1h'."
                .to_string(),
        );
    }
    Ok(trimmed.to_string())
}

pub async fn execute_cli(
    state: &EngineState,
    binary_path: &str,
    args: Vec<String>,
    trigger: &str,
) -> Result<CliExecutionRecord, String> {
    let started = Instant::now();
    let display_cmd = if args.is_empty() {
        "next-meeting".to_string()
    } else {
        format!("next-meeting {}", args.join(" "))
    };

    let mut cmd = configure_command(binary_path, &args);
    let output = timeout(CLI_TIMEOUT, cmd.output())
        .await
        .map_err(|_| format!("CLI command `{display_cmd}` timed out after {}s", CLI_TIMEOUT.as_secs()))?
        .map_err(|e| format!("Failed to execute `{binary_path}`: {e}"))?;

    let duration_ms = started.elapsed().as_millis() as u64;
    let exit_code = output.status.code().unwrap_or(-1);
    let stdout = String::from_utf8_lossy(&output.stdout).trim().to_string();
    let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();

    let record = CliExecutionRecord {
        id: 0,
        timestamp: Utc::now().to_rfc3339_opts(SecondsFormat::Secs, true),
        command: display_cmd,
        args,
        stdout,
        stderr,
        exit_code,
        duration_ms,
        trigger: trigger.to_string(),
        success: output.status.success(),
    };

    Ok(state.record_execution(record).await)
}

/// Computes the exact 16-hex-character notification hash used by Go's `notify.getNotificationID`.
pub fn compute_notification_id(
    summary: &str,
    start: &DateTime<FixedOffset>,
    end: &DateTime<FixedOffset>,
) -> String {
    let start_rfc = start.to_rfc3339_opts(SecondsFormat::Secs, true);
    let end_rfc = end.to_rfc3339_opts(SecondsFormat::Secs, true);
    let payload = format!("{summary}|{start_rfc}|{end_rfc}");
    let digest = Sha256::digest(payload.as_bytes());
    let mut hex_out = String::with_capacity(16);
    for byte in &digest[..8] {
        use std::fmt::Write;
        let _ = write!(&mut hex_out, "{byte:02x}");
    }
    hex_out
}

fn has_been_notified_at(notify_dir: &Path, id: &str) -> bool {
    notify_dir.join(id).is_file()
}

fn count_notified_files(notify_dir: &Path) -> usize {
    fs::read_dir(notify_dir)
        .map(|entries| entries.flatten().filter(|e| e.path().is_file()).count())
        .unwrap_or(0)
}

pub fn extract_join_details(hangout_link: &str, location: &str) -> (Option<String>, Option<String>) {
    let candidate = if !hangout_link.trim().is_empty() {
        Some(hangout_link.trim().to_string())
    } else {
        location
            .split_whitespace()
            .find(|token| token.starts_with("https://") || token.starts_with("http://"))
            .map(|token| {
                token
                    .trim_end_matches([',', ';', ')', '>', '.', '"', '\''])
                    .to_string()
            })
    };

    let Some(url) = candidate else {
        return (None, None);
    };

    let provider = detect_meeting_provider(&url);
    (Some(url), Some(provider.to_string()))
}

/// Returns the lowercase hostname of an `http(s)://` URL, ignoring userinfo, port, path,
/// query, and fragment. Returns `None` for anything that is not an absolute http(s) URL.
fn url_hostname(url: &str) -> Option<String> {
    let (scheme, rest) = url.split_once("://")?;
    if !scheme.eq_ignore_ascii_case("https") && !scheme.eq_ignore_ascii_case("http") {
        return None;
    }
    let authority = rest.split(['/', '?', '#']).next()?;
    let host_port = authority.rsplit('@').next()?;
    let host = if let Some(ipv6) = host_port.strip_prefix('[') {
        ipv6.split(']').next()?
    } else {
        host_port.split(':').next()?
    };
    let host = host.trim_end_matches('.').to_ascii_lowercase();
    if host.is_empty() {
        None
    } else {
        Some(host)
    }
}

/// True when `host` is exactly `domain` or a subdomain of it (label boundary respected).
fn host_matches(host: &str, domain: &str) -> bool {
    host == domain
        || host
            .strip_suffix(domain)
            .is_some_and(|prefix| prefix.ends_with('.'))
}

/// Classifies a join URL by its hostname only. Substring matching on the whole URL would
/// label `https://zoom.us.evil.example/...` or `https://evil.example/?next=meet.google.com`
/// as a trusted provider even though "Join" opens an unrelated host.
pub fn detect_meeting_provider(url: &str) -> &'static str {
    const PROVIDERS: &[(&str, &[&str])] = &[
        ("google_meet", &["meet.google.com", "hangouts.google.com"]),
        ("zoom", &["zoom.us", "zoom.com"]),
        ("teams", &["teams.microsoft.com", "teams.live.com"]),
        ("webex", &["webex.com"]),
    ];

    let Some(host) = url_hostname(url) else {
        return "link";
    };
    PROVIDERS
        .iter()
        .find(|(_, domains)| domains.iter().any(|domain| host_matches(&host, domain)))
        .map(|(provider, _)| *provider)
        .unwrap_or("link")
}

pub fn filter_accepted_meetings(events: &[ParsedMeeting]) -> Vec<ParsedMeeting> {
    events
        .iter()
        .filter(|m| {
            matches!(
                m.raw.self_response_status.as_str(),
                "accepted" | "tentative"
            )
        })
        .cloned()
        .collect()
}

/// Exact Rust port of Go's `calendar.GetMeetingStatus` selection & tie-breaking rules.
pub fn select_current_and_next(
    events: &[ParsedMeeting],
    now: DateTime<Utc>,
) -> (Option<ParsedMeeting>, Option<ParsedMeeting>) {
    let mut current: Option<ParsedMeeting> = None;
    let mut next: Option<ParsedMeeting> = None;

    for meeting in events {
        let m_start = meeting.start_dt.with_timezone(&Utc);
        let m_end = meeting.end_dt.with_timezone(&Utc);

        // Current meeting: now in [Start, End)
        if now >= m_start && now < m_end {
            let replace = match &current {
                None => true,
                Some(curr) => {
                    let c_start = curr.start_dt.with_timezone(&Utc);
                    let c_end = curr.end_dt.with_timezone(&Utc);
                    m_start > c_start || (m_start == c_start && m_end < c_end)
                }
            };
            if replace {
                current = Some(meeting.clone());
            }
        }

        // Next meeting: earliest upcoming (Start > now), shorter wins ties
        if now < m_start {
            let replace = match &next {
                None => true,
                Some(nxt) => {
                    let n_start = nxt.start_dt.with_timezone(&Utc);
                    let n_end = nxt.end_dt.with_timezone(&Utc);
                    m_start < n_start || (m_start == n_start && m_end < n_end)
                }
            };
            if replace {
                next = Some(meeting.clone());
            }
        }
    }

    (current, next)
}

fn enrich_meeting(
    parsed: &ParsedMeeting,
    now: DateTime<Utc>,
    notify_dir: &Path,
    all_events: &[ParsedMeeting],
) -> EnrichedMeeting {
    let id = compute_notification_id(&parsed.raw.summary, &parsed.start_dt, &parsed.end_dt);
    let m_start = parsed.start_dt.with_timezone(&Utc);
    let m_end = parsed.end_dt.with_timezone(&Utc);

    let phase = if now >= m_end {
        MeetingPhase::Past
    } else if now >= m_start {
        MeetingPhase::Current
    } else {
        MeetingPhase::Upcoming
    };

    let duration_minutes = (m_end - m_start).num_minutes().max(0);
    let (join_url, meeting_provider) =
        extract_join_details(&parsed.raw.hangout_link, &parsed.raw.location);

    let is_overlapping = all_events.iter().any(|other| {
        let o_start = other.start_dt.with_timezone(&Utc);
        let o_end = other.end_dt.with_timezone(&Utc);
        let same_identity = other.raw.summary == parsed.raw.summary
            && o_start == m_start
            && o_end == m_end;
        !same_identity && m_start < o_end && o_start < m_end
    });

    EnrichedMeeting {
        id: id.clone(),
        summary: if parsed.raw.summary.trim().is_empty() {
            "(No title)".to_string()
        } else {
            parsed.raw.summary.clone()
        },
        start: parsed.raw.start.clone(),
        end: parsed.raw.end.clone(),
        start_epoch_ms: m_start.timestamp_millis(),
        end_epoch_ms: m_end.timestamp_millis(),
        duration_minutes,
        is_all_day: parsed.raw.is_all_day,
        location: parsed.raw.location.clone(),
        hangout_link: parsed.raw.hangout_link.clone(),
        join_url,
        meeting_provider,
        attendees: parsed.raw.attendees,
        self_response_status: parsed.raw.self_response_status.clone(),
        phase,
        notified: has_been_notified_at(notify_dir, &id),
        is_overlapping,
    }
}

pub fn read_cache_and_meetings(
    cache_file: &Path,
    notify_dir: &Path,
    now: DateTime<Utc>,
) -> (CacheInfo, Vec<ParsedMeeting>) {
    let notified_count = count_notified_files(notify_dir);
    let path_str = cache_file.to_string_lossy().to_string();

    let Ok(raw_text) = fs::read_to_string(cache_file) else {
        return (
            CacheInfo {
                path: path_str,
                exists: false,
                timestamp: None,
                age_seconds: None,
                ttl_remaining_seconds: None,
                is_expired: true,
                notified_count,
            },
            Vec::new(),
        );
    };

    let Ok(cached) = serde_json::from_str::<RawCachedData>(&raw_text) else {
        return (
            CacheInfo {
                path: path_str,
                exists: true,
                timestamp: None,
                age_seconds: None,
                ttl_remaining_seconds: None,
                is_expired: true,
                notified_count,
            },
            Vec::new(),
        );
    };

    let (age_seconds, ttl_remaining_seconds, is_expired) =
        match DateTime::parse_from_rfc3339(&cached.timestamp) {
            Ok(ts) => {
                let age = (now - ts.with_timezone(&Utc)).num_seconds().max(0);
                let rem = (CACHE_TTL_SECONDS - age).max(0);
                (Some(age), Some(rem), age > CACHE_TTL_SECONDS)
            }
            Err(_) => (None, None, true),
        };

    let mut parsed_events = Vec::new();
    for item in cached.events.unwrap_or_default() {
        if item.is_all_day {
            continue;
        }
        if let (Ok(start_dt), Ok(end_dt)) = (
            DateTime::parse_from_rfc3339(&item.start),
            DateTime::parse_from_rfc3339(&item.end),
        ) {
            parsed_events.push(ParsedMeeting {
                raw: item,
                start_dt,
                end_dt,
            });
        }
    }

    parsed_events.sort_by(|a, b| {
        a.start_dt
            .with_timezone(&Utc)
            .cmp(&b.start_dt.with_timezone(&Utc))
            .then_with(|| {
                a.end_dt
                    .with_timezone(&Utc)
                    .cmp(&b.end_dt.with_timezone(&Utc))
            })
    });

    (
        CacheInfo {
            path: path_str,
            exists: true,
            timestamp: Some(cached.timestamp),
            age_seconds,
            ttl_remaining_seconds,
            is_expired,
            notified_count,
        },
        parsed_events,
    )
}

fn compute_daily_stats(
    events: &[EnrichedMeeting],
    current: Option<&EnrichedMeeting>,
    next: Option<&EnrichedMeeting>,
    now_ms: i64,
) -> DailyStats {
    let total_meetings = events.len();
    let accepted_count = events
        .iter()
        .filter(|e| matches!(e.self_response_status.as_str(), "accepted" | "tentative"))
        .count();
    let completed_count = events
        .iter()
        .filter(|e| e.phase == MeetingPhase::Past)
        .count();
    let remaining_count = events
        .iter()
        .filter(|e| e.phase != MeetingPhase::Past)
        .count();

    let total_scheduled_minutes: i64 = events.iter().map(|e| e.duration_minutes).sum();
    let remaining_scheduled_minutes: i64 = events
        .iter()
        .map(|e| match e.phase {
            MeetingPhase::Past => 0,
            MeetingPhase::Current => ((e.end_epoch_ms - now_ms).max(0)) / 60_000,
            MeetingPhase::Upcoming => e.duration_minutes,
        })
        .sum();

    let free_until_next_minutes = if current.is_some() {
        Some(0)
    } else {
        next.map(|n| ((n.start_epoch_ms - now_ms).max(0)) / 60_000)
    };

    DailyStats {
        total_meetings,
        accepted_count,
        remaining_count,
        completed_count,
        total_scheduled_minutes,
        remaining_scheduled_minutes,
        free_until_next_minutes,
    }
}

/// Exact stdout printed by the Go CLI (`main.go`) when no keyring token is present.
const CLI_SIGNED_OUT_OUTPUT: &str = "🔒 Not logged in";
/// Exact stderr line printed by the Go CLI when the Calendar API is unreachable.
const CLI_OFFLINE_OUTPUT: &str = "📡 Calendar Offline";

/// The signed-out message is the CLI's *entire* stdout. Meeting titles are also echoed to
/// stdout, so substring checks would misclassify a title such as "🔒 Security review".
fn cli_reports_signed_out(stdout: &str) -> bool {
    stdout.trim() == CLI_SIGNED_OUT_OUTPUT
}

fn cli_reports_offline(stderr: &str) -> bool {
    stderr.lines().any(|line| line.trim() == CLI_OFFLINE_OUTPUT)
}

fn classify_cli_result(
    record: &CliExecutionRecord,
    creds_exists: bool,
    creds_valid: bool,
) -> (AuthState, Option<String>) {
    if cli_reports_signed_out(&record.stdout) {
        if !creds_exists || !creds_valid {
            return (
                AuthState::MissingCredentials,
                Some(
                    "Google OAuth credentials.json is not configured yet. Import or paste your Desktop OAuth credentials to sign in."
                        .to_string(),
                ),
            );
        }
        return (AuthState::NotLoggedIn, None);
    }

    if cli_reports_offline(&record.stderr) {
        return (
            AuthState::Offline,
            Some("Calendar is currently offline. Showing cached schedule if available.".to_string()),
        );
    }

    if record.stderr.contains("credentials file not found") {
        return (
            AuthState::MissingCredentials,
            Some(record.stderr.clone()),
        );
    }

    if !record.success {
        let msg = if !record.stderr.is_empty() {
            record.stderr.clone()
        } else if !record.stdout.is_empty() {
            record.stdout.clone()
        } else {
            format!("CLI exited with status code {}", record.exit_code)
        };
        return (AuthState::Error, Some(msg));
    }

    (AuthState::Authenticated, None)
}

pub async fn sync_dashboard(
    state: &EngineState,
    force_refresh: bool,
    trigger: &str,
) -> Result<DashboardSnapshot, String> {
    let _guard = state.sync_lock.lock().await;
    let settings = state.settings.lock().await.clone();
    let engine_info = resolve_engine_info(settings.custom_cli_path.as_deref());
    let creds_info = inspect_credentials();

    if !engine_info.available {
        let now = Utc::now();
        let snapshot = DashboardSnapshot {
            raw_cli_output: "CLI engine binary not found".to_string(),
            auth_state: AuthState::Error,
            error_message: Some(format!(
                "Could not locate `next-meeting` binary (checked `{}`). Verify your build or custom CLI path.",
                engine_info.binary_path
            )),
            current_meeting: None,
            next_meeting: None,
            events: Vec::new(),
            stats: DailyStats {
                total_meetings: 0,
                accepted_count: 0,
                remaining_count: 0,
                completed_count: 0,
                total_scheduled_minutes: 0,
                remaining_scheduled_minutes: 0,
                free_until_next_minutes: None,
            },
            cache_info: CacheInfo {
                path: cache_path().to_string_lossy().to_string(),
                exists: false,
                timestamp: None,
                age_seconds: None,
                ttl_remaining_seconds: None,
                is_expired: true,
                notified_count: 0,
            },
            credentials_info: creds_info,
            engine_info,
            last_execution: None,
            synced_at: now.to_rfc3339_opts(SecondsFormat::Secs, true),
        };
        *state.last_snapshot.lock().await = Some(snapshot.clone());
        return Ok(snapshot);
    }

    if force_refresh {
        let _ = execute_cli(
            state,
            &engine_info.binary_path,
            vec!["--clear-cache".to_string()],
            trigger,
        )
        .await?;
    }

    let mut args = Vec::new();
    if settings.only_accepted {
        args.push("--only-accepted".to_string());
    }
    if settings.notify_enabled {
        if let Ok(valid_dur) = validate_notify_duration(&settings.notify_threshold) {
            args.push("--notify".to_string());
            args.push(valid_dur);
        }
    }

    let exec_record = execute_cli(state, &engine_info.binary_path, args, trigger).await?;
    let (auth_state, error_message) =
        classify_cli_result(&exec_record, creds_info.exists, creds_info.valid);

    let now = Utc::now();
    let c_path = cache_path();
    let n_dir = notify_dir_path();
    let (cache_info, all_parsed) = read_cache_and_meetings(&c_path, &n_dir, now);

    let active_meetings = match auth_state {
        AuthState::Authenticated | AuthState::Offline => {
            if settings.only_accepted {
                filter_accepted_meetings(&all_parsed)
            } else {
                all_parsed.clone()
            }
        }
        _ => Vec::new(),
    };

    let (curr_parsed, next_parsed) = select_current_and_next(&active_meetings, now);

    let enriched_all: Vec<EnrichedMeeting> = match auth_state {
        AuthState::Authenticated | AuthState::Offline => active_meetings
            .iter()
            .map(|m| enrich_meeting(m, now, &n_dir, &active_meetings))
            .collect(),
        _ => Vec::new(),
    };

    let current_meeting = curr_parsed
        .as_ref()
        .map(|m| enrich_meeting(m, now, &n_dir, &active_meetings));
    let next_meeting = next_parsed
        .as_ref()
        .map(|m| enrich_meeting(m, now, &n_dir, &active_meetings));

    let stats = compute_daily_stats(
        &enriched_all,
        current_meeting.as_ref(),
        next_meeting.as_ref(),
        now.timestamp_millis(),
    );

    let raw_cli_output = if !exec_record.stdout.is_empty() {
        exec_record.stdout.clone()
    } else if !exec_record.stderr.is_empty() {
        exec_record.stderr.clone()
    } else {
        "📭 No meetings".to_string()
    };

    let snapshot = DashboardSnapshot {
        raw_cli_output,
        auth_state,
        error_message,
        current_meeting,
        next_meeting,
        events: enriched_all,
        stats,
        cache_info,
        credentials_info: creds_info,
        engine_info,
        last_execution: Some(exec_record),
        synced_at: now.to_rfc3339_opts(SecondsFormat::Secs, true),
    };

    *state.last_snapshot.lock().await = Some(snapshot.clone());
    Ok(snapshot)
}

pub async fn start_oauth_login(app: AppHandle, state: &EngineState) -> Result<(), String> {
    let creds = inspect_credentials();
    if !creds.exists || !creds.valid {
        return Err(creds.error.unwrap_or_else(|| {
            format!(
                "Missing valid OAuth credentials at {}. Import or paste your Desktop OAuth credentials.json first.",
                creds.path
            )
        }));
    }

    // Cancel any stale login process before starting a new one
    cancel_oauth_login(state).await;

    let settings = state.settings.lock().await.clone();
    let engine = resolve_engine_info(settings.custom_cli_path.as_deref());
    if !engine.available {
        return Err("CLI engine binary is not available.".to_string());
    }

    let args = vec!["--login".to_string()];
    let mut cmd = configure_command(&engine.binary_path, &args);
    let mut child = cmd
        .spawn()
        .map_err(|e| format!("Failed to start `next-meeting --login`: {e}"))?;

    let stdout = child.stdout.take();
    let stderr = child.stderr.take();

    {
        let mut active = state.active_login.lock().await;
        *active = Some(child);
    }

    let _ = app.emit(
        "cli://login-progress",
        LoginProgressPayload {
            stage: "starting".to_string(),
            url: None,
            message: "Starting local OAuth callback listener on http://localhost:8085/callback..."
                .to_string(),
            success: None,
        },
    );

    let app_stdout = app.clone();
    if let Some(out) = stdout {
        tokio::spawn(async move {
            let mut lines = BufReader::new(out).lines();
            while let Ok(Some(line)) = lines.next_line().await {
                let trimmed = line.trim();
                if trimmed.is_empty() {
                    continue;
                }
                if trimmed.starts_with("https://accounts.google.com")
                    || trimmed.starts_with("http://")
                    || trimmed.starts_with("https://")
                {
                    let _ = app_stdout.emit(
                        "cli://login-progress",
                        LoginProgressPayload {
                            stage: "waiting_browser".to_string(),
                            url: Some(trimmed.to_string()),
                            message: "Browser authorization opened. Approve access in your browser to continue."
                                .to_string(),
                            success: None,
                        },
                    );
                } else {
                    let _ = app_stdout.emit(
                        "cli://login-progress",
                        LoginProgressPayload {
                            stage: "log".to_string(),
                            url: None,
                            message: trimmed.to_string(),
                            success: None,
                        },
                    );
                }
            }
        });
    }

    let stderr_buf = std::sync::Arc::new(Mutex::new(String::new()));
    let stderr_buf_clone = stderr_buf.clone();
    let app_stderr = app.clone();
    if let Some(err_stream) = stderr {
        tokio::spawn(async move {
            let mut lines = BufReader::new(err_stream).lines();
            while let Ok(Some(line)) = lines.next_line().await {
                let trimmed = line.trim();
                if trimmed.is_empty() {
                    continue;
                }
                {
                    let mut buf = stderr_buf_clone.lock().await;
                    if !buf.is_empty() {
                        buf.push('\n');
                    }
                    buf.push_str(trimmed);
                }
                let _ = app_stderr.emit(
                    "cli://login-progress",
                    LoginProgressPayload {
                        stage: "stderr".to_string(),
                        url: None,
                        message: trimmed.to_string(),
                        success: None,
                    },
                );
            }
        });
    }

    let app_wait = app.clone();
    tokio::spawn(async move {
        let started = Instant::now();
        // Poll wait cleanly without holding lock across await so `cancel_oauth_login` can kill immediately
        let status_opt = loop {
            let mut guard = {
                use tauri::Manager;
                let st = app_wait.state::<EngineState>();
                let mut active = st.active_login.lock().await;
                let Some(child_ref) = active.as_mut() else {
                    break None;
                };
                match child_ref.try_wait() {
                    Ok(Some(status)) => {
                        *active = None;
                        Some(Ok(status))
                    }
                    Ok(None) => None,
                    Err(e) => {
                        *active = None;
                        Some(Err(e))
                    }
                }
            };

            if let Some(done) = guard.take() {
                break Some(done);
            }

            if started.elapsed() > Duration::from_secs(180) {
                use tauri::Manager;
                let st = app_wait.state::<EngineState>();
                cancel_oauth_login(&st).await;
                let _ = app_wait.emit(
                    "cli://login-progress",
                    LoginProgressPayload {
                        stage: "finished".to_string(),
                        url: None,
                        message: "Authorization timed out after 3 minutes.".to_string(),
                        success: Some(false),
                    },
                );
                return;
            }

            tokio::time::sleep(Duration::from_millis(200)).await;
        };

        let Some(wait_res) = status_opt else {
            // Cancelled by user
            let _ = app_wait.emit(
                "cli://login-progress",
                LoginProgressPayload {
                    stage: "cancelled".to_string(),
                    url: None,
                    message: "Sign-in cancelled.".to_string(),
                    success: Some(false),
                },
            );
            return;
        };

        let stderr_collected = stderr_buf.lock().await.clone();
        match wait_res {
            Ok(status) if status.success() => {
                {
                    use tauri::Manager;
                    let st = app_wait.state::<EngineState>();
                    let _ = st
                        .record_execution(CliExecutionRecord {
                            id: 0,
                            timestamp: Utc::now().to_rfc3339_opts(SecondsFormat::Secs, true),
                            command: "next-meeting --login".to_string(),
                            args: vec!["--login".to_string()],
                            stdout: "✓ Logged in successfully".to_string(),
                            stderr: stderr_collected,
                            exit_code: status.code().unwrap_or(0),
                            duration_ms: started.elapsed().as_millis() as u64,
                            trigger: "auth".to_string(),
                            success: true,
                        })
                        .await;
                }
                let _ = app_wait.emit(
                    "cli://login-progress",
                    LoginProgressPayload {
                        stage: "finished".to_string(),
                        url: None,
                        message: "Logged in to Google Calendar successfully!".to_string(),
                        success: Some(true),
                    },
                );
            }
            Ok(status) => {
                let err_msg = if stderr_collected.is_empty() {
                    format!(
                        "Authorization failed (exit code {})",
                        status.code().unwrap_or(-1)
                    )
                } else {
                    stderr_collected.clone()
                };
                {
                    use tauri::Manager;
                    let st = app_wait.state::<EngineState>();
                    let _ = st
                        .record_execution(CliExecutionRecord {
                            id: 0,
                            timestamp: Utc::now().to_rfc3339_opts(SecondsFormat::Secs, true),
                            command: "next-meeting --login".to_string(),
                            args: vec!["--login".to_string()],
                            stdout: String::new(),
                            stderr: err_msg.clone(),
                            exit_code: status.code().unwrap_or(1),
                            duration_ms: started.elapsed().as_millis() as u64,
                            trigger: "auth".to_string(),
                            success: false,
                        })
                        .await;
                }
                let _ = app_wait.emit(
                    "cli://login-progress",
                    LoginProgressPayload {
                        stage: "finished".to_string(),
                        url: None,
                        message: err_msg,
                        success: Some(false),
                    },
                );
            }
            Err(err) => {
                let _ = app_wait.emit(
                    "cli://login-progress",
                    LoginProgressPayload {
                        stage: "finished".to_string(),
                        url: None,
                        message: format!("Error waiting for login process: {err}"),
                        success: Some(false),
                    },
                );
            }
        }
    });

    Ok(())
}

pub async fn cancel_oauth_login(state: &EngineState) {
    let mut active = state.active_login.lock().await;
    if let Some(mut child) = active.take() {
        let _ = child.kill().await;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::RawMeetingInfo;

    fn make_parsed(summary: &str, start: &str, end: &str, status: &str) -> ParsedMeeting {
        let start_dt = DateTime::parse_from_rfc3339(start).unwrap();
        let end_dt = DateTime::parse_from_rfc3339(end).unwrap();
        ParsedMeeting {
            raw: RawMeetingInfo {
                summary: summary.to_string(),
                start: start.to_string(),
                end: end.to_string(),
                is_all_day: false,
                location: "https://zoom.us/j/99999999".to_string(),
                hangout_link: String::new(),
                attendees: 4,
                self_response_status: status.to_string(),
            },
            start_dt,
            end_dt,
        }
    }

    #[test]
    fn matches_go_notification_id_sha256_prefix() {
        let start = DateTime::parse_from_rfc3339("2026-11-05T17:00:00-06:00").unwrap();
        let end = DateTime::parse_from_rfc3339("2026-11-05T17:30:00-06:00").unwrap();
        let id = compute_notification_id("Weekly Sync", &start, &end);
        assert_eq!(id.len(), 16);

        let expected_hash =
            Sha256::digest(b"Weekly Sync|2026-11-05T17:00:00-06:00|2026-11-05T17:30:00-06:00");
        let expected_hex: String = expected_hash[..8].iter().map(|b| format!("{b:02x}")).collect();
        assert_eq!(id, expected_hex);
    }

    #[test]
    fn selects_current_and_next_matching_go_tiebreakers() {
        let now = DateTime::parse_from_rfc3339("2026-01-09T14:30:00Z")
            .unwrap()
            .with_timezone(&Utc);

        let events = vec![
            make_parsed(
                "Earlier Long Current",
                "2026-01-09T13:30:00Z",
                "2026-01-09T15:30:00Z",
                "accepted",
            ),
            make_parsed(
                "Later Current",
                "2026-01-09T14:15:00Z",
                "2026-01-09T15:00:00Z",
                "accepted",
            ),
            make_parsed(
                "Long Future",
                "2026-01-09T15:30:00Z",
                "2026-01-09T17:30:00Z",
                "accepted",
            ),
            make_parsed(
                "Short Future",
                "2026-01-09T15:30:00Z",
                "2026-01-09T16:00:00Z",
                "tentative",
            ),
        ];

        let (curr, next) = select_current_and_next(&events, now);
        assert_eq!(curr.unwrap().raw.summary, "Later Current");
        assert_eq!(next.unwrap().raw.summary, "Short Future");
    }

    #[test]
    fn filters_accepted_and_tentative_only() {
        let events = vec![
            make_parsed("A", "2026-01-09T10:00:00Z", "2026-01-09T11:00:00Z", "accepted"),
            make_parsed("B", "2026-01-09T11:00:00Z", "2026-01-09T12:00:00Z", "declined"),
            make_parsed("C", "2026-01-09T12:00:00Z", "2026-01-09T13:00:00Z", "tentative"),
            make_parsed("D", "2026-01-09T13:00:00Z", "2026-01-09T14:00:00Z", "needsAction"),
        ];
        let filtered = filter_accepted_meetings(&events);
        let titles: Vec<&str> = filtered.iter().map(|e| e.raw.summary.as_str()).collect();
        assert_eq!(titles, vec!["A", "C"]);
    }

    #[test]
    fn parses_cache_with_null_events_slice() {
        let temp_file = env::temp_dir().join("next-meeting-test-null-cache.json");
        let notify_dir = env::temp_dir().join("next-meeting-test-notify-empty");
        fs::write(
            &temp_file,
            r#"{"timestamp":"2026-11-05T17:15:00Z","events":null}"#,
        )
        .unwrap();
        let now = DateTime::parse_from_rfc3339("2026-11-05T17:20:00Z")
            .unwrap()
            .with_timezone(&Utc);

        let (info, meetings) = read_cache_and_meetings(&temp_file, &notify_dir, now);
        let _ = fs::remove_file(&temp_file);

        assert!(info.exists);
        assert!(!info.is_expired);
        assert_eq!(info.age_seconds, Some(300));
        assert!(meetings.is_empty());
    }

    #[test]
    fn extracts_join_links_and_providers() {
        let (url, provider) =
            extract_join_details("https://meet.google.com/xyz-abcd-efg", "Room 302");
        assert_eq!(url.as_deref(), Some("https://meet.google.com/xyz-abcd-efg"));
        assert_eq!(provider.as_deref(), Some("google_meet"));

        let (z_url, z_prov) =
            extract_join_details("", "Join via Zoom: https://company.zoom.us/j/123456789.");
        assert_eq!(z_url.as_deref(), Some("https://company.zoom.us/j/123456789"));
        assert_eq!(z_prov.as_deref(), Some("zoom"));
    }

    #[test]
    fn provider_detection_respects_hostname_boundaries() {
        // Untrusted hosts that merely contain a provider name must not get a provider label.
        assert_eq!(detect_meeting_provider("https://zoom.us.evil.example/j/123"), "link");
        assert_eq!(
            detect_meeting_provider("https://evil.example/?next=meet.google.com"),
            "link"
        );
        assert_eq!(detect_meeting_provider("https://notzoom.us/j/1"), "link");
        assert_eq!(detect_meeting_provider("https://evil.example/#meet.google.com"), "link");
        assert_eq!(detect_meeting_provider("https://meet.google.com@evil.example/x"), "link");
        assert_eq!(detect_meeting_provider("not a url"), "link");

        // Exact hosts and legitimate subdomains, regardless of case, port, or userinfo.
        assert_eq!(detect_meeting_provider("https://zoom.us/j/1"), "zoom");
        assert_eq!(detect_meeting_provider("https://company.zoom.us/j/123"), "zoom");
        assert_eq!(detect_meeting_provider("https://user:pw@zoom.us:443/j/1"), "zoom");
        assert_eq!(
            detect_meeting_provider("HTTPS://MEET.GOOGLE.COM/abc-defg-hij"),
            "google_meet"
        );
        assert_eq!(
            detect_meeting_provider("https://teams.microsoft.com/l/meetup-join/x"),
            "teams"
        );
        assert_eq!(detect_meeting_provider("https://example.webex.com/meet/x"), "webex");
    }

    fn make_record(stdout: &str, stderr: &str, success: bool) -> CliExecutionRecord {
        CliExecutionRecord {
            id: 0,
            timestamp: String::new(),
            command: "next-meeting".to_string(),
            args: Vec::new(),
            stdout: stdout.to_string(),
            stderr: stderr.to_string(),
            exit_code: if success { 0 } else { 1 },
            duration_ms: 0,
            trigger: "test".to_string(),
            success,
        }
    }

    #[test]
    fn classifies_signed_out_only_on_exact_cli_output() {
        let signed_out = make_record("🔒 Not logged in", "", true);
        assert_eq!(
            classify_cli_result(&signed_out, true, true).0,
            AuthState::NotLoggedIn
        );
        assert_eq!(
            classify_cli_result(&signed_out, false, false).0,
            AuthState::MissingCredentials
        );

        // Meeting titles are part of stdout and must not trigger the signed-out screen.
        let titles = make_record(
            "🔴 🔒 Security review (12m left) │ 🕐 Not logged in troubleshooting in 40m",
            "",
            true,
        );
        assert_eq!(
            classify_cli_result(&titles, true, true).0,
            AuthState::Authenticated
        );

        let offline = make_record("", "Warning: something\n📡 Calendar Offline", false);
        assert_eq!(classify_cli_result(&offline, true, true).0, AuthState::Offline);
    }

    #[tokio::test]
    async fn resolves_and_executes_sidecar_cli() {
        let info = resolve_engine_info(None);
        assert!(
            info.available,
            "Expected compiled sidecar CLI to be available at {}",
            info.binary_path
        );

        let state = EngineState::new();
        // `--help` exits cleanly without modifying user keyring or cache!
        let cmd_res = configure_command(&info.binary_path, &[String::from("-h")])
            .output()
            .await
            .expect("should execute sidecar binary");
        let combined = format!(
            "{}{}",
            String::from_utf8_lossy(&cmd_res.stdout),
            String::from_utf8_lossy(&cmd_res.stderr)
        );
        assert!(combined.contains("only-accepted"));
        assert!(combined.contains("clear-cache"));
        assert!(combined.contains("notify"));
        assert!(state.history.lock().await.is_empty());
    }
}
