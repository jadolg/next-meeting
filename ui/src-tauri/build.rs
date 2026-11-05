use std::env;
use std::fs;
use std::path::PathBuf;
use std::process::Command;

fn map_target_to_go(target: &str) -> (&'static str, &'static str, &'static str) {
    let (goos, ext) = if target.contains("windows") {
        ("windows", ".exe")
    } else if target.contains("darwin") || target.contains("apple") {
        ("darwin", "")
    } else if target.contains("linux") {
        ("linux", "")
    } else {
        panic!("Unsupported target OS in triple: {target}");
    };

    let goarch = if target.starts_with("x86_64") {
        "amd64"
    } else if target.starts_with("aarch64") {
        "arm64"
    } else if target.starts_with("i686") {
        "386"
    } else {
        panic!("Unsupported target architecture in triple: {target}");
    };

    (goos, goarch, ext)
}

fn build_go_sidecar() {
    let manifest_dir = PathBuf::from(env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR missing"));
    let repo_root = manifest_dir
        .parent()
        .and_then(|p| p.parent())
        .expect("Failed to locate repository root from src-tauri");

    for tracked in [
        "main.go",
        "go.mod",
        "go.sum",
        "auth",
        "cache",
        "calendar",
        "keyring",
        "notify",
    ] {
        println!("cargo:rerun-if-changed={}", repo_root.join(tracked).display());
    }

    let target = env::var("TARGET").expect("TARGET env var missing");
    let (goos, goarch, ext) = map_target_to_go(&target);

    let binaries_dir = manifest_dir.join("binaries");
    fs::create_dir_all(&binaries_dir).expect("Failed to create src-tauri/binaries directory");

    let output_bin = binaries_dir.join(format!("next-meeting-{target}{ext}"));

    let status = Command::new("go")
        .current_dir(repo_root)
        .env("CGO_ENABLED", "0")
        .env("GOOS", goos)
        .env("GOARCH", goarch)
        .args([
            "build",
            "-trimpath",
            "-ldflags=-s -w",
            "-o",
            output_bin.to_str().expect("Invalid output binary path"),
            ".",
        ])
        .status();

    match status {
        Ok(s) if s.success() => {}
        Ok(s) => {
            if !output_bin.exists() {
                panic!("Failed to build Go sidecar binary for {target} (exit status: {s})");
            }
        }
        Err(err) => {
            if !output_bin.exists() {
                panic!("Failed to invoke `go build` for sidecar ({target}): {err}");
            }
        }
    }
}

fn main() {
    build_go_sidecar();
    tauri_build::build();
}
