<p align="center">
  <img src=".github/assets/logo.png" alt="NOTIF Logo" width="250"/>
</p>

<h1 align="center">Notif</h1>

<p align="center">
  <a href="https://github.com/tanq16/notif/actions/workflows/release.yml"><img src="https://github.com/tanq16/notif/actions/workflows/release.yml/badge.svg" alt="Release Build"></a>&nbsp;<a href="https://github.com/tanq16/notif/releases/latest"><img src="https://img.shields.io/github/v/release/tanq16/notif" alt="Latest Release"></a>&nbsp;<a href="https://goreportcard.com/report/github.com/tanq16/notif"><img src="https://goreportcard.com/badge/github.com/tanq16/notif" alt="Go Report Card"></a>
</p><br>

---

`Notif` is a command-line tool that sends Discord webhook notifications for command execution and custom messages. It's designed to monitor long-running commands and receive immediate notifications about their completion and output.

## Features

Notif integrates with Discord webhooks to deliver notifications about command execution and custom messages. The tool supports command execution tracking with duration measurement, output capturing, and output splitting for large results. It can handle interactive shell commands and maintains environment variables during execution.

## Installation

Download the latest release binary for your platform from the [releases page](https://github.com/tanq16/notif/releases). The tool is available for Linux, macOS, and Windows on both AMD64 and ARM64 architectures.

For persistent webhook configuration, create a `.notif.webhook` file in either your home directory (`~/.notif.webhook`) or `/persist/.notif.webhook` containing your Discord webhook URL.

## Usage

Notif can be used in two primary modes: command execution monitoring and raw message sending.

For command execution:

```bash
# Send notification without command output
notif -c "sleep 10 && echo 'Task completed'"

# Send notification with command output
notif -t out -c "ls -la"
```

For sending custom messages:

```bash
# Send a custom message
notif -m "Deployment completed successfully!"
```

> [!TIP]
> Notif sends the message as text, but Discord interprets it as Markdown. So you can get creative with custom messages! Just be mindful that Discord Markdown has a limited syntax.

You can specify the webhook URL directly using the `-w` flag or configure it in the `.notif.webhook` file:

```bash
notif -w "https://discord.com/api/webhooks/..." -c "make build"
```

## Configuration

The webhook URL can be configured in two ways:

1. Environment configuration file at `~/.notif.webhook`
2. System-wide configuration at `/persist/.notif.webhook`

The configuration file should contain only the webhook URL as plain text.

## Example Screenshots

Below are some examples of how Notif notifications appear in Discord:

| Description | Screenshot |
|-------------|------------|
| Command Execution | `[Screenshot placeholder]` |
| Command Output | `[Screenshot placeholder]` |
| Custom Message | `[Screenshot placeholder]` |
| Large Output Split | `[Screenshot placeholder]` |

## Technical Details

- Built with Go 1.23
- Uses Cobra for CLI command handling
- Supports interactive shell commands (bash/zsh with environment loading)
- Automatically splits large outputs into multiple Discord messages (witholds output when more than 5 splits)
- Maximum field length of 1024 characters
