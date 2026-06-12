---
name: cyeam-cli
description: Use when a user asks to use cyeam.com public site capabilities from the command line, including ask/search, date holiday lookup, Mo calligraphy/OCR, roadbook sharing, OneDrive-backed roadbook/note workflows, Microsoft login, CLI version checks, or CLI self-update.
---

# Cyeam CLI

## Overview

Use the `cyeam` command for supported public cyeam.com capabilities. Do not expose or call admin, cache, jobrunner, upload, push-management, conversion-tool, translation, qrcode, barcode, or search-suggestion features.

## Before Running

- Prefer the local `cyeam` binary if it exists in `PATH`.
- Run `cyeam version` first when the user asks about installation state, updates, or compatibility.
- Use `cyeam update` only when the user asks to update the CLI. It updates from GitHub Release assets.
- Commands call the production cyeam.com service by default; there is no config command.

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
cyeam cnote new "note-title" < note.html
cyeam cnote append "note-title" < more.html
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
- `cnote list`, `cnote new`, and `cnote append` read/write OneDrive folder `Notes` and require login. `cnote list` includes a clickable terminal hyperlink when OneDrive returns `webUrl`. New and append read HTML content from stdin.

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
