# Cyeam CLI Technical Design

## Goal

Build a small Go CLI that exposes selected public cyeam.com capabilities. The CLI calls the production website by default and does not include a config command.

## Language

Use Go with Cobra. This matches `cyeam_web`, produces single-file binaries, handles HTTP/SSE/multipart cleanly, and supports GitHub Release based self-update.

## Project Structure

```text
cmd/cyeam/                 main package
internal/cli/              Cobra command construction
internal/client/           cyeam.com HTTP client
internal/output/           JSON, plain text, and file output helpers
internal/update/           GitHub Release self-update logic
internal/version/          build metadata
```

The CLI is an HTTP client for public cyeam.com routes. It should not import `cyeam_web` controllers because those handlers depend on web context, runtime resources, database state, and server initialization.

## Commands

```bash
cyeam version
cyeam update

cyeam ask "How should this system do rate limiting?"
cyeam ask "How should this system do rate limiting?" --mode fast|think|expert
cyeam ask search "golang optimization"

cyeam date slogan [YYYY-MM-DD]
cyeam date holiday [YYYY-MM-DD]

cyeam mo guwen "兰亭序"
cyeam mo guwen "兰亭序" --ai-compose
cyeam mo char detail "之"
cyeam mo char composition "曦"
cyeam mo char compose "曦" --out char.png
cyeam mo ocr image.png --out result.json

cyeam roadbook share roadbook.json
cyeam roadbook get <id>
```

## API Mapping

- `ask`: `GET /ai/architecture?q=...&mode=...`, streamed to stdout
- `ask search`: `GET /search/api?q=...`
- `date slogan`: new `GET /api/date/slogan?date=YYYY-MM-DD`
- `date holiday`: new `GET /api/date/holiday?date=YYYY-MM-DD`
- `mo guwen`: `GET /mo/api/guwen?text=...&font=行书`, with `compose=1` for `--ai-compose`
- `mo char detail`: `GET /mo/api/char/detail?char=...&font=行书`
- `mo char composition`: `GET /mo/api/char/composition?char=...&font=行书`
- `mo char compose`: `GET /mo/api/char/compose?char=...&font=行书`, PNG response
- `mo ocr`: multipart upload to `POST /mo/api/ocr`
- `roadbook share`: request body to `/api/roadbook/share`; CLI prints both `id` and `url`, where `url` is `https://www.cyeam.com/tool/roadbook?id=<id>`
- `roadbook get`: `GET /api/roadbook/get?id=...`

## HTTP Behavior

- Base URL is fixed to `https://www.cyeam.com`.
- Regular API responses are decoded as JSON and re-encoded to stdout.
- `ask` reads the architecture SSE/plain stream incrementally and writes chunks to stdout without buffering the full response.
- `mo ocr` sends multipart form field `image`.
- `mo char compose` requires `--out` and writes the PNG response to that path.
- Date commands default to the current local date when the date argument is omitted.

## Scope Exclusions

Do not support admin, login, cache, jobrunner, upload, glyph sync, push management, conversion tools, translation, search suggestions, QR code, barcode, roadbook CSV token flow, or Mo AI glyph save/write-db APIs.

## Output

Default output is JSON for API responses. `roadbook share` includes the generated share URL. Streaming `ask` architecture output is plain text. Binary image output requires `--out`.

## Error Handling

- Non-2xx HTTP responses should return a non-zero exit code and include response status plus a short response body excerpt.
- Invalid dates should fail before making a request.
- Missing files, empty input files, and unwritable output paths should fail before making a request.
- Unsupported modes for `ask` should fail locally; valid values are `fast`, `think`, and `expert`.

## Testing

- Unit test command parsing and URL construction.
- Unit test output handling for JSON, streaming text, and PNG writes.
- Use `httptest.Server` for HTTP client tests, including non-2xx responses and malformed JSON.
- Keep network-dependent end-to-end tests opt-in behind an environment variable.

## Update Flow

Use GitHub Releases as the binary distribution source. Users download the latest binary from the repository release page, and `cyeam update` uses the same release assets for self-update.

Release asset naming:

```text
cyeam_Darwin_arm64.tar.gz
cyeam_Darwin_x86_64.tar.gz
cyeam_Linux_arm64.tar.gz
cyeam_Linux_x86_64.tar.gz
cyeam_Windows_x86_64.zip
```

Initial install path:

- Users manually download the asset for their OS/architecture from GitHub Releases.
- The release page should include shell snippets for macOS/Linux installation and a Windows zip instruction.

Self-update path:

- `cyeam update` checks the latest GitHub Release.
- If the local version is current, it exits without changes.
- If a newer version exists, it downloads the matching asset for `runtime.GOOS` and `runtime.GOARCH`.
- It verifies the downloaded checksum when release checksums are available.
- It replaces the current executable atomically where possible and prints old/new versions.

`cyeam version` prints version, commit, build date, GOOS, and GOARCH.

## Release Flow

GitHub Actions builds release artifacts for macOS, Linux, and Windows on tags. Build metadata is injected with `-ldflags` for version, commit, and build date. Each release should also include a checksum file such as `checksums.txt`.

Implementation files:

- `.github/workflows/release.yml`
- `.goreleaser.yaml`

Release trigger:

```bash
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions runs tests, invokes GoReleaser, uploads Release assets, and then sends a Feishu notification.

Required repository secrets:

- `FEISHU_WEBHOOK_URL`: Feishu custom bot webhook URL. If unset, the notification step is skipped.

Release assets are downloaded from the GitHub Release page. The Release page is the canonical download location for new installs and the canonical source used by `cyeam update`.

## Skill

Include a repository-local skill at `skills/cyeam-cli/SKILL.md`. The skill teaches future agents when to use `cyeam`, lists supported commands, and explicitly blocks unsupported capabilities.
