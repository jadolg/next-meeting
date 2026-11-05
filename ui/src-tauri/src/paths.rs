use std::env;
use std::fs;
use std::path::{Path, PathBuf};

use serde::Deserialize;

use crate::models::{AppSettings, CredentialsInfo};

const APP_DIR_NAME: &str = "next-meeting";
const CREDENTIALS_FILE_NAME: &str = "credentials.json";
const SETTINGS_FILE_NAME: &str = "ui-settings.json";
const CACHE_FILE_NAME: &str = "next-meeting-cache.json";
const NOTIFY_DIR_NAME: &str = "next-meeting-notify";

/// Matches Go's `os.UserConfigDir()` behavior across Windows, macOS, and Linux.
pub fn user_config_dir() -> Result<PathBuf, String> {
    #[cfg(target_os = "windows")]
    {
        env::var_os("APPDATA")
            .filter(|v| !v.is_empty())
            .map(PathBuf::from)
            .ok_or_else(|| "%APPDATA% is not defined".to_string())
    }

    #[cfg(target_os = "macos")]
    {
        let home = env::var_os("HOME")
            .filter(|v| !v.is_empty())
            .map(PathBuf::from)
            .ok_or_else(|| "$HOME is not defined".to_string())?;
        Ok(home.join("Library").join("Application Support"))
    }

    #[cfg(not(any(target_os = "windows", target_os = "macos")))]
    {
        if let Some(xdg) = env::var_os("XDG_CONFIG_HOME").filter(|v| !v.is_empty()) {
            return Ok(PathBuf::from(xdg));
        }
        let home = env::var_os("HOME")
            .filter(|v| !v.is_empty())
            .map(PathBuf::from)
            .ok_or_else(|| "$HOME is not defined".to_string())?;
        Ok(home.join(".config"))
    }
}

pub fn app_config_dir() -> Result<PathBuf, String> {
    Ok(user_config_dir()?.join(APP_DIR_NAME))
}

pub fn credentials_path() -> Result<PathBuf, String> {
    Ok(app_config_dir()?.join(CREDENTIALS_FILE_NAME))
}

pub fn settings_path() -> Result<PathBuf, String> {
    Ok(app_config_dir()?.join(SETTINGS_FILE_NAME))
}

pub fn cache_path() -> PathBuf {
    env::temp_dir().join(CACHE_FILE_NAME)
}

pub fn notify_dir_path() -> PathBuf {
    env::temp_dir().join(NOTIFY_DIR_NAME)
}

#[derive(Debug, Deserialize)]
struct InstalledCredsEnvelope {
    installed: Option<InstalledCredsBody>,
    web: Option<serde_json::Value>,
}

#[derive(Debug, Deserialize)]
struct InstalledCredsBody {
    client_id: Option<String>,
    client_secret: Option<String>,
}

fn mask_client_id(client_id: &str) -> String {
    let trimmed = client_id.trim();
    if trimmed.len() <= 16 {
        return trimmed.to_string();
    }
    let prefix = &trimmed[..10];
    let suffix = &trimmed[trimmed.len().saturating_sub(14)..];
    format!("{prefix}...{suffix}")
}

pub fn validate_credentials_content(raw: &str) -> Result<String, String> {
    let parsed: InstalledCredsEnvelope = serde_json::from_str(raw)
        .map_err(|e| format!("Invalid JSON in credentials.json: {e}"))?;

    if parsed.installed.is_none() && parsed.web.is_some() {
        return Err(
            "Detected a 'Web application' OAuth client ('web' key). Please create a 'Desktop app' OAuth client ID in Google Cloud Console so it contains the 'installed' section required by next-meeting."
                .to_string(),
        );
    }

    let installed = parsed.installed.ok_or_else(|| {
        "Missing 'installed' OAuth configuration in credentials.json. Make sure you selected 'Desktop app' in Google Cloud Console."
            .to_string()
    })?;

    let client_id = installed
        .client_id
        .unwrap_or_default()
        .trim()
        .to_string();
    let client_secret = installed
        .client_secret
        .unwrap_or_default()
        .trim()
        .to_string();

    if client_id.is_empty() || client_secret.is_empty() {
        return Err(
            "Both 'installed.client_id' and 'installed.client_secret' must be non-empty."
                .to_string(),
        );
    }

    Ok(mask_client_id(&client_id))
}

pub fn inspect_credentials_at(path: &Path) -> CredentialsInfo {
    let path_str = path.to_string_lossy().to_string();
    if !path.exists() {
        return CredentialsInfo {
            path: path_str,
            exists: false,
            valid: false,
            client_id_preview: None,
            error: None,
        };
    }

    match fs::read_to_string(path) {
        Ok(content) => match validate_credentials_content(&content) {
            Ok(preview) => CredentialsInfo {
                path: path_str,
                exists: true,
                valid: true,
                client_id_preview: Some(preview),
                error: None,
            },
            Err(err) => CredentialsInfo {
                path: path_str,
                exists: true,
                valid: false,
                client_id_preview: None,
                error: Some(err),
            },
        },
        Err(err) => CredentialsInfo {
            path: path_str,
            exists: true,
            valid: false,
            client_id_preview: None,
            error: Some(format!("Unable to read credentials file: {err}")),
        },
    }
}

pub fn inspect_credentials() -> CredentialsInfo {
    match credentials_path() {
        Ok(path) => inspect_credentials_at(&path),
        Err(err) => CredentialsInfo {
            path: String::new(),
            exists: false,
            valid: false,
            client_id_preview: None,
            error: Some(err),
        },
    }
}

pub fn write_credentials_content(raw: &str) -> Result<CredentialsInfo, String> {
    let _preview = validate_credentials_content(raw)?;
    let path = credentials_path()?;
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)
            .map_err(|e| format!("Failed to create config directory: {e}"))?;
    }

    // Pretty-format normalized JSON while keeping original structure intact
    let value: serde_json::Value =
        serde_json::from_str(raw).map_err(|e| format!("Invalid JSON: {e}"))?;
    let formatted = serde_json::to_string_pretty(&value)
        .map_err(|e| format!("Failed to serialize credentials: {e}"))?;

    write_private_file(&path, formatted.as_bytes())?;
    Ok(inspect_credentials_at(&path))
}

pub fn delete_credentials_file() -> Result<CredentialsInfo, String> {
    let path = credentials_path()?;
    if path.exists() {
        fs::remove_file(&path)
            .map_err(|e| format!("Failed to delete credentials.json: {e}"))?;
    }
    Ok(inspect_credentials_at(&path))
}

fn write_private_file(path: &Path, bytes: &[u8]) -> Result<(), String> {
    #[cfg(unix)]
    {
        use std::io::Write;
        use std::os::unix::fs::OpenOptionsExt;
        let mut file = fs::OpenOptions::new()
            .create(true)
            .write(true)
            .truncate(true)
            .mode(0o600)
            .open(path)
            .map_err(|e| format!("Failed to open {} for writing: {e}", path.display()))?;
        file.write_all(bytes)
            .map_err(|e| format!("Failed to write {}: {e}", path.display()))?;
        Ok(())
    }

    #[cfg(not(unix))]
    {
        fs::write(path, bytes)
            .map_err(|e| format!("Failed to write {}: {e}", path.display()))
    }
}

pub fn load_settings() -> AppSettings {
    let Ok(path) = settings_path() else {
        return AppSettings::default();
    };
    let Ok(content) = fs::read_to_string(path) else {
        return AppSettings::default();
    };
    serde_json::from_str::<AppSettings>(&content).unwrap_or_default()
}

pub fn save_settings(settings: &AppSettings) -> Result<(), String> {
    let path = settings_path()?;
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)
            .map_err(|e| format!("Failed to create config directory: {e}"))?;
    }
    let json = serde_json::to_string_pretty(settings)
        .map_err(|e| format!("Failed to serialize settings: {e}"))?;
    write_private_file(&path, json.as_bytes())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn validates_desktop_installed_credentials() {
        let valid = r#"{
            "installed": {
                "client_id": "123456789012-abcdefg.apps.googleusercontent.com",
                "client_secret": "GOCSPX-secretValue"
            }
        }"#;
        let preview = validate_credentials_content(valid).expect("should be valid");
        assert!(preview.contains("1234567890"));
        assert!(preview.contains("..."));
    }

    #[test]
    fn rejects_web_oauth_credentials_with_helpful_guidance() {
        let web_creds = r#"{
            "web": {
                "client_id": "123.apps.googleusercontent.com",
                "client_secret": "secret"
            }
        }"#;
        let err = validate_credentials_content(web_creds).unwrap_err();
        assert!(err.contains("Desktop app"));
    }

    #[test]
    fn rejects_empty_fields() {
        let empty = r#"{"installed": {"client_id": "  ", "client_secret": ""}}"#;
        assert!(validate_credentials_content(empty).is_err());
    }
}
