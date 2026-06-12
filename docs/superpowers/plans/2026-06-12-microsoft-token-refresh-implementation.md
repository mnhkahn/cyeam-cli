# Microsoft Token Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure Microsoft login requests refresh-token capability and the CLI keeps users signed in by refreshing access tokens automatically.

**Architecture:** Keep the existing device-code flow and keychain storage. Add `offline_access` to requested scopes, preserve the previous refresh token when Microsoft does not rotate it, and make `whoami` text explain that the displayed expiry is only for the access token.

**Tech Stack:** Go 1.23, `net/http/httptest`, Cobra command tests.

---

## Tasks

- [ ] Add auth tests proving device-code and refresh requests include `offline_access`.
- [ ] Add auth test proving refresh preserves the old refresh token if the response omits a new one.
- [ ] Add CLI test proving `whoami` describes automatic refresh when a refresh token exists.
- [ ] Implement the minimal code changes in `internal/auth/auth.go` and `internal/cli/root.go`.
- [ ] Run `gofmt`, targeted tests, and `go test -count=1 ./...`.
