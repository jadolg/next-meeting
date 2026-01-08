# next-meeting

A CLI tool that displays your current or upcoming Google Calendar meeting status. Designed for status bars (like i3blocks, polybar, etc.) or checking your schedule from the terminal.

## Features

- **Current Status**: Shows "🔴 [Meeting Name]" if you are currently in a meeting, with time remaining.
- **Upcoming Status**: Shows "🕐 [Meeting Name]" for the next meeting, with time until start.
- **Empty State**: Shows "📭 No meetings" if your schedule is clear.
- **Privacy**: Accesses your calendar in read-only mode. Credentials are stored securely in your system keyring.

## Prerequisites

- **Go**: You need Go installed to build the project.
- **Google Cloud Platform Project**: You need a project with the Google Calendar API enabled to generate your own `credentials.json`.

## Installation & Setup

### 1. Clone the Repository

```bash
git clone https://github.com/jadolg/next-meeting.git
cd next-meeting
```

### 2. Configure Google Credentials

To allow `next-meeting` to access your calendar, you need to create an OAuth 2.0 Client ID in the Google Cloud Console.

1.  Go to the [Google Cloud Console](https://console.cloud.google.com/).
2.  Create a new project or select an existing one.
3.  Enable the **Google Calendar API** for your project.
4.  Go to **APIs & Services** > **Credentials**.
5.  Click **Create Credentials** > **OAuth client ID**.
6.  Select **Desktop app** as the Application type.
7.  Name it (e.g., "Next Meeting CLI").
8.  Download the JSON file.
9.  Rename the downloaded file to `credentials.json`.
10. Move it to the `auth/` directory in this project:

    ```bash
    mv /path/to/downloaded-file.json auth/credentials.json
    ```

### 3. Build the Project

Run `go build` to compile the application. This will embed your `credentials.json` into the binary.

```bash
go build -o next-meeting
```

## Usage

### Authentication

First, you need to log in to authorize the application to access your calendar data.

```bash
./next-meeting --login
```

This will:
1. Print an authorization URL.
2. Attempt to open it in your default browser.
3. Once you approve access, the token will be securely stored in your system's keyring.

### Checking Status

Once logged in, simply run the binary:

```bash
./next-meeting
```

**Example Outputs:**
- `🔴 Weekly Sync (5m left) │ 🕐 Lunch in 1h 5m`
- `🕐 Team Standup in 10m`
- `📭 No meetings`

### Clearing Credentials

To remove the stored token from your system keyring:

```bash
./next-meeting --clear
```

## Troubleshooting

- **Authorization Server**: The `--login` command starts a local server on port `8085` to receive the strict callback from Google. Ensure this port is not in use.
- **Keyring Issues**: If you encounter issues saving the token, ensure you have a working system keyring (e.g., `gnome-keyring` on Linux, Keychain on macOS).
