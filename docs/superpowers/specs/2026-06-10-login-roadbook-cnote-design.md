# cyeam-cli: Login + Roadbook Upgrade + CNote Integration

## Problem

cyeam-cli lacks authentication and cannot access OneDrive-stored data,
while the web frontend (cyeam.com/note, cyeam.com/tool/roadbook) already
uses Microsoft login + OneDrive Graph API for Roadbook and CNote. The
CLI needs to match this capability so AI agents and users can manage
roadbooks and notes from the terminal.

## Scope

Three capabilities, delivered together:

1. **Microsoft login** via OAuth Device Code Flow
2. **Roadbook upgrade** — list/share from OneDrive (existing `share`/
   `get` reworked)
3. **CNote integration** — list/new/append from OneDrive

## Architecture

```
┌──────────────────┐   Device Code Flow     ┌─────────────────────┐
│   cyeam CLI      │ ◄─────────────────────► │ Microsoft Identity  │
│                  │                         │ Platform + Graph    │
│  ┌────────────┐  │                         └─────────────────────┘
│  │ Keychain   │  │
│  │ (token)    │  │
│  └────────────┘  │
│                  │   POST /api/roadbook/share
│                  │◄────────────────────────► cyeam.com backend
└──────────────────┘
```

The CLI authenticates directly against Microsoft Identity Platform
(just like the frontend), then calls OneDrive Graph API directly for
data operations. Roadbook share uses cyeam.com backend only for
short-ID generation (reuses existing `/api/roadbook/share`).

## Authentication

- **Flow**: OAuth Device Authorization Grant (Device Code Flow)
- **Client ID**: `e1f582c1-1568-4347-8c5b-1906164e637f` (same as frontend)
- **Authority**: `https://login.microsoftonline.com/consumers`
- **Scopes**: `Files.ReadWrite`, `User.Read`
- **Token storage**: System keychain via `zalando/go-keyring`
  - macOS: Keychain
  - Linux: Secret Service (libsecret)
  - Windows: Credential Manager
- **Token refresh**: Automatic on 401 or expiry, using refresh token

### Command: `cyeam login`

1. Call Device Code endpoint (`https://login.microsoftonline.com/consumers/oauth2/v2.0/devicecode`)
2. Print URL + code to stdout (user opens in browser)
3. Poll for token (`https://login.microsoftonline.com/consumers/oauth2/v2.0/token`)
4. Store access_token + refresh_token + expiry in keychain
5. Verify by calling `User.Read`

### Command: `cyeam logout`

1. Clear token from keychain

### Token lifecycle

- Every command that needs auth reads tokens from keychain
- If access_token is expired, use refresh_token to get a new one
- If refresh fails, print "Please run `cyeam login`" and exit

## Command Tree

```
cyeam
├── login                          # Device Code Flow auth
├── logout                         # Clear stored token
├── roadbook
│   ├── list                       # List roadbooks from OneDrive 路书/
│   ├── share <filename>           # Share a roadbook by filename
│   └── get <id>                   # (existing, unchanged)
└── cnote
    ├── list                       # List notes from OneDrive Notes/
    ├── new <title>                # Create note (content from stdin)
    └── append <title>             # Append to note (content from stdin)
```

## OneDrive Graph API Integration

### Shared auth header

All requests include:
```
Authorization: Bearer {access_token}
```

### Roadbook: `路书/` folder

| Operation | Graph API Call |
|-----------|---------------|
| list | `GET /me/drive/root:/路书/children?$select=id,name,lastModifiedDateTime&$top=50` |
| share | 1. `POST /me/drive/root:/路书/{filename}:/createLink` with `{type:"view",scope:"anonymous"}` → get `link.webUrl` |
|        | 2. `POST /api/roadbook/share` with `{url: webUrl}` → get `{id}` |
|        | 3. Output `https://www.cyeam.com/tool/roadbook?id={id}` |

`share` accepts filename WITHOUT `.json` suffix (auto-appends internally).

`roadbook list` output format:
```
文件名                     修改时间
p_a1e4022426ed92.json  2026-06-10 14:30
p_...                   ...
```

`roadbook share p_a1e4022426ed92` output:
```
https://www.cyeam.com/tool/roadbook?id=abc123
```

### CNote: `Notes/` folder

| Operation | Graph API Call |
|-----------|---------------|
| list | `GET /me/drive/root:/Notes/children?$select=id,name,lastModifiedDateTime&$top=50` |
| new | `PUT /me/drive/root:/Notes/{title}.html:/content` with `Content-Type: text/html` and stdin as body |
| append | 1. `GET /me/drive/root:/Notes/{title}.html:/content` → read existing |
|        | 2. Read stdin → append to content |
|        | 3. `PUT ...:/content` with combined content |

- Content type is `text/html` (matching frontend)
- If title already exists for `new`, overwrite (same as frontend save behavior)
- `list` strips `.html` extension in output for cleaner display

## Internal Package Structure

```
internal/auth/
├── auth.go              # Device Code Flow, token refresh
├── keychain.go          # Keychain store/load/delete
├── auth_test.go

internal/onedrive/
├── onedrive.go          # Graph API: list, read, write, createLink
├── onedrive_test.go

internal/cli/
├── root.go              # Add login, logout, cnote commands; update roadbook
└── root_test.go
```

No changes to `internal/client/` or `internal/cyeam/service.go` (roadbook
share reworked but can go in a new path or be replaced).

## Dependencies

```
go get github.com/zalando/go-keyring
go get golang.org/x/oauth2
```

These are the only new Go module deps. MSAL is not used directly - we
use raw OAuth2 endpoints since the CLI only needs Device Code Flow.

## Error Handling

- Network errors: print message + exit code 1
- Auth errors (401): attempt refresh; if refresh fails, advise re-login
- OneDrive 404 for share/append: "File not found in OneDrive"
- Empty stdin for new/append: "No content provided (stdin is empty)"
- Keychain errors: fallback to error message (no silent fallback to insecure storage)

## Testing

- Auth module: test device code URL parsing, token refresh logic with
  mock HTTP
- OneDrive module: test with recorded Graph API responses
- CLI integration: table-driven tests for command parsing/output
- Manual: `cyeam login` end-to-end