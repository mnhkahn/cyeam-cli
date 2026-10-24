---
name: calligraphy-composition
metadata:
  version: "0.1.18"
description: Assemble Chinese running-script calligraphy from original glyphs, look up character forms and sources, inspect radicals, and synthesize missing glyphs. Use for calligraphy composition, character images, or copybook references. Read this skill for exact CLI syntax before running commands.
---

# Calligraphy Composition

Retrieve running-script calligraphy for text or individual Chinese characters, inspect radicals and components, and synthesize missing glyphs when needed. For seal-script image recognition, use the separate `seal-script-ocr` skill.

## Commands

```bash
cyeam mo guwen "兰亭序"                       # Retrieve glyphs for text
cyeam mo guwen "兰亭序" --ai-compose          # Synthesize missing glyphs only
cyeam mo char detail "之"                     # Original glyph candidates and sources
cyeam mo char composition "曦"               # Radicals and components
cyeam mo char compose "曦" --out char.png     # Force component-based synthesis
```

## Glyph selection rules

When asked to write, return, or display a character, **look up an original glyph first; do not default to `char compose`**.

1. For one character, run `cyeam mo char detail <char>` and use an original glyph when candidates exist.
2. For a passage, run `cyeam mo guwen <text>`. By default, it returns only original glyphs from the library; missing glyphs are empty.
3. Use synthesis only when original candidates are absent, the passage contains missing glyphs, or the user explicitly requests component-based synthesis. Use `guwen --ai-compose` for text and `char compose` for a single character.

`char compose` always forces component-based synthesis, even when a complete original glyph exists. For example, look up `char detail "无"` first; use `char compose "无"` only when component-based synthesis is appropriate.

## Image delivery

When the user wants to see a glyph or calligraphy image, embed the selected candidate's `image` URL as a Markdown image and include its `source`. Do not return only a URL.

```md
「无」 — 王羲之《临锺繇千字文》

![无](https://res.cloudinary.com/.../word.webp)
```

The URLs from `char detail` point to original copybook images and can be embedded directly. Do not switch to `char compose` just to obtain a displayable image. Use an image link as a fallback only when the client cannot render Markdown images.

## Output and scope

- `guwen`, `char detail`, and `char composition` write JSON to stdout.
- `char detail` candidates include original glyph images and their sources.
- `char compose` writes a PNG to the required `--out` path. It always synthesizes components and does not represent an original copybook glyph.
- This workflow covers running script (行书), not regular script (楷书) or clerical script (隶书).
- `--ai-compose` fills only missing glyphs; preserve existing original glyphs.
- No login is required.

## Installation


Review the commands before installing. These examples pin **v1.1.4**, rather than latest, and download an executable archive.
Install only when the CLI is needed; do not reinstall for every command.

### macOS / Linux: verify and install in the user directory

Run in Bash or Zsh with `curl`, `tar`, `install`, and either `shasum` or `sha256sum`.
The script selects macOS / Linux on arm64 or x86_64 and stops on other platforms.

```bash
(
  set -eu
  version='v1.1.4'
  case "$(uname -s)/$(uname -m)" in
    Darwin/arm64)
      asset='cyeam_Darwin_arm64.tar.gz'
      sha256='06b5af004ab4f0f05d5829ea6bd18b97c4449c262af7df14c67175f7d3cb29db' ;;
    Darwin/x86_64)
      asset='cyeam_Darwin_x86_64.tar.gz'
      sha256='107342bb9d2fa444b96155c468e71b8cc63e0f5ad0acc784698afe221548efa6' ;;
    Linux/aarch64|Linux/arm64)
      asset='cyeam_Linux_arm64.tar.gz'
      sha256='ed87697f3948bead67b585e9881c3bef20e8d2d903beb5b125587281a5091c5b' ;;
    Linux/x86_64)
      asset='cyeam_Linux_x86_64.tar.gz'
      sha256='873d6805fab8ab56540bb034aaf2f57f12eb672fcabd322ebe401e8250a5966d' ;;
    *) echo 'Unsupported platform; see the pinned release.' >&2; exit 1 ;;
  esac
  workdir=$(mktemp -d)
  trap 'rm -rf "$workdir"' EXIT
  cd "$workdir"
  curl --fail --show-error --silent --location --proto '=https' --proto-redir '=https' \
    --output "$asset" "https://github.com/mnhkahn/cyeam-cli/releases/download/$version/$asset"
  printf '%s  %s\n' "$sha256" "$asset" > expected.sha256
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c expected.sha256
  else
    shasum -a 256 -c expected.sha256
  fi
  # set -e prevents extraction or installation if download or verification fails.
  tar -xzf "$asset" cyeam
  mkdir -p "$HOME/.local/bin"
  install -m 0755 cyeam "$HOME/.local/bin/cyeam"
  CYEAM_CLI_NO_UPDATE_NOTIFIER=1 "$HOME/.local/bin/cyeam" --no-analytics version
)
```

This writes only to a temporary directory and `~/.local/bin/cyeam` (replacing an existing binary).
No administrator privileges are needed; system directories, services, and shell configuration are unchanged. For the current shell:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

For persistent access, add that line to your shell configuration.
Check the [v1.1.4 release page](https://github.com/mnhkahn/cyeam-cli/releases/tag/v1.1.4) and
[checksums.txt](https://github.com/mnhkahn/cyeam-cli/releases/download/v1.1.4/checksums.txt)
for checksums. When changing the pinned version, verify and replace both the version and its SHA-256 values.
Checksums confirm release-file consistency; they are not an independent signature or security audit.

### External requests and local state

- **Installation:** Downloads the selected archive from a pinned GitHub release, which may redirect to GitHub's asset CDN. Verifies the pinned SHA-256 before extracting only `cyeam`. Installation verification disables analytics and update checks.
- **Calligraphy queries:** `mo guwen` and `mo char detail/composition/compose` send the selected text, font, and synthesis options to `https://www.cyeam.com/mo/api/*`. Synthesized images are written to `--out`, which may overwrite an existing file. Original images may come from Cloudinary or another image CDN; displaying them requests their URLs.
- **Analytics:** By default, sends command names, CLI version, system information, and a random installation ID to `https://us.i.posthog.com`. The server may infer location from the outgoing IP. Command arguments, image contents, and credentials are not included. The ID is stored in `~/.cyeam/analytics-id`; disable with `--no-analytics` or `CYEAM_CLI_NO_ANALYTICS=1`.
- **Update notices:** Checks GitHub for the latest release at most once every 24 hours and caches state in `~/.cyeam/update-state.json`. Notices do not replace the executable. Set `CYEAM_CLI_NO_UPDATE_NOTIFIER=1` to disable. The explicit `cyeam update` command downloads and verifies the latest release, replaces the executable, and then attempts to sync skills through `npx skills add`. To remain on a pinned version, use the installation steps above instead.

To keep only the network requests required for this skill in the current shell:

```bash
export CYEAM_CLI_NO_ANALYTICS=1
export CYEAM_CLI_NO_UPDATE_NOTIFIER=1
```
