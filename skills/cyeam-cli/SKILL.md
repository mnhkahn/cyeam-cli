---
name: cyeam-cli
description: cyeam.com public site CLI — ask/search architecture, date holiday, Mo calligraphy/OCR, roadbook sharing, cnote cloud notes, TV schedule (NBA/World Cup/China football), geek news & AI news, Microsoft login, self-update. All commands support --json for structured output.
---

# Cyeam CLI

## Overview

Use the `cyeam` command for supported public cyeam.com capabilities. Do not expose or call admin, cache, jobrunner, upload, push-management, conversion-tool, translation, qrcode, barcode, or search-suggestion features.

## Before Running

- Prefer the local `cyeam` binary if it exists in `PATH`.
- If `cyeam` is not installed, install it before running cyeam commands. Detect the OS and CPU architecture, then use the matching command:
  - macOS Apple Silicon:
    ```bash
    curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Darwin_arm64.tar.gz | tar xz
    chmod +x cyeam
    sudo mv cyeam /usr/local/bin/
    ```
  - macOS Intel:
    ```bash
    curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Darwin_x86_64.tar.gz | tar xz
    chmod +x cyeam
    sudo mv cyeam /usr/local/bin/
    ```
  - Linux amd64:
    ```bash
    curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Linux_x86_64.tar.gz | tar xz
    chmod +x cyeam
    sudo mv cyeam /usr/local/bin/
    ```
  - Windows: Download `cyeam_Windows_x86_64.zip`, unzip it, and add `cyeam.exe` to `PATH`.
- Run `cyeam version` first when the user asks about installation state, updates, or compatibility.
- Use `cyeam update` only when the user asks to update the CLI. It updates from GitHub Release assets and auto-syncs this skill file.
- Commands call the production cyeam.com service by default; there is no config command.
- For multi-threaded downloads, use `aria2c` directly. Install via `brew install aria2` if missing. Default output to `~/Downloads/`. Supports `--all-proxy` for proxy.
- All commands support `--json` flag. When set, output is a JSON envelope: `{"ok": true, "data": ..., "_notice": {...}}`. The `_notice` field may contain update/skill sync reminders.
- Set `CYEAM_CLI_NO_UPDATE_NOTIFIER=1` to suppress update/skill notices.
- In CI environments (`CI=true`), update checks are skipped automatically.

## Supported Commands

```bash
cyeam version
cyeam update
cyeam login
cyeam logout
cyeam whoami

cyeam ask "How should this system do rate limiting?"
cyeam ask "How should this system do rate limiting?" --mode fast
cyeam ask "How should this system do rate limiting?" --mode think
cyeam ask "How should this system do rate limiting?" --mode expert
cyeam ask search "golang optimization"

cyeam date holiday
cyeam date holiday 2026-06-09

cyeam mo guwen "兰亭序"
cyeam mo guwen "兰亭序" --ai-compose
cyeam mo char detail "之"
cyeam mo char composition "曦"
cyeam mo char compose "曦" --out char.png
cyeam mo ocr image.png

cyeam roadbook list
cyeam roadbook share roadbook.json
cyeam roadbook get <id>

cyeam cnote list
cyeam cnote get "note-title"
cyeam cnote get "note-title" --format text
cyeam cnote new "note-title" < note.html
cyeam cnote append "note-title" < more.html

cyeam tv list
cyeam tv list --league nba --days 7
cyeam tv list --league worldcup,cn-football
cyeam tv list --team 湖人
cyeam tv list --source CCTV5
cyeam tv list --from 2026-06-15 --to 2026-06-20
cyeam tv list --json
cyeam tv today
cyeam tv next --league nba

cyeam news list
cyeam news list --from 2026-06-10 --to 2026-06-14
cyeam news get --date 2026-06-14

cyeam update --help
```

## Behavior Notes

- `date` subcommands accept an optional `YYYY-MM-DD`; omit it for today.
- `ask` streams architecture output to stdout. Default mode is `fast`.
- `ask search` returns normal site search JSON.
- `login` uses Microsoft Device Code Flow, requests offline access for refresh tokens, and stores tokens in the system keychain. `logout` clears stored credentials. `whoami` prints login status, access-token expiry, auto-refresh availability, and user info when available.
- `mo` uses xingshu only. Do not add or request a font option.
- `mo char compose` writes a PNG file. Require `--out` when the user needs a saved image.
- `mo ocr` uploads an image and writes JSON to stdout.
- `roadbook list` reads OneDrive folder `路书` and requires login.
- `roadbook share` reads a local JSON file and returns both the share id and `https://www.cyeam.com/tool/roadbook?id=<id>`.
- `cnote list`, `cnote get`, `cnote new`, and `cnote append` read/write OneDrive folder `Notes` and require login. `cnote list` includes a clickable terminal hyperlink when OneDrive returns `webUrl`. `cnote get` reads `Notes/<title>.html` and outputs Markdown by default or plain text with `--format text`. New and append read HTML content from stdin.
- `tv list` shows upcoming NBA, World Cup, and China men's/women's national football matches in `Asia/Shanghai` by default. Defaults to the next 3 days (max 14 via `--days`).
- `--league` accepts `nba`, `worldcup`, `cn-football` (comma-separated or repeatable). `--from`/`--to` override `--days` start. `--include-finished` to also show finished matches; `--source` filters by broadcaster (e.g. CCTV5); `--team` filters by team name or abbreviation.
- Broadcasters (CCTV5, 央视频, 腾讯体育, 咪咕视频, etc.) are read-only viewing hints. The CLI does not capture streams, decode m3u8, or bypass paywalls. Names render as clickable terminal hyperlinks when supported.
- `tv` does not require login or call cyeam.com. If a data source (cdn.nba.com, ESPN site API) is unreachable, that league is skipped with a warning instead of failing the whole command.
- Team/country names and league stages (e.g. "Finals", "Regular", "Playoffs") are in English. Translate them to Chinese and add flag emojis when presenting to the user.

## Unsupported Requests

If the user asks for an unsupported capability, say it is intentionally not part of the CLI and do not invent a command.

- Admin, cache, jobrunner, upload, glyph sync, push management
- Developer conversion tools: curl2go, json2go, json2ddl, ddl2go, XML, SQL, msgpack, base encoders
- Translation
- Search suggestions
- Top-level `search` or `architecture` commands; use `ask search` or `ask`
- QR code or barcode generation
- Mo AI glyph save/write-db API
- Roadbook CSV token flow unless explicitly added later
- Live stream capture, m3u8/flv URL extraction, paywall or geo-restriction bypass for any TV/streaming source
- Real-time score push or post-game highlight downloads
- Sports leagues outside NBA / FIFA World Cup / China national football (not in scope yet)
