# AGENTS.md

## Project: Sharknado

Napster-inspired music streaming platform with Go backend + React frontend.
Integrates with Tidal, Qobuz, and Deezer via CLI tools (streamrip, tidal-dl-ng).
No user accounts — private service.

## Workflow Rules

**Work in parallel batches as much as possible.** When multiple tasks are independent, dispatch them simultaneously. When researching, gather information from all sources at once. When writing files, write independent files in the same turn. Batch related work into single parallel operations rather than sequential ones.

## Architecture

- Backend: Go (net/http, modernc.org/sqlite)
- Frontend: React + Vite + howler.js
- Providers: streamrip CLI (Qobuz/Deezer), tidal-dl-ng CLI (Tidal)
- Auth: mounted config files, no user management
- Audio: FFmpeg transcode for streaming
- Docker: multi-stage build, Alpine + distroless

## Key Paths

- `backend/` — Go server
- `frontend/` — React UI
- `data/` — runtime data, configs, DB
- `data/streamrip-config/config.toml` — Qobuz/Deezer auth
- `data/tidal-config/` — Tidal auth (settings.json, token.json)

## CLI Tool Details

### streamrip (`rip`)
- `rip search <source> track '<query>' -n <count> -o <file>` — search (outputs JSON)
- `rip url <url>` — download
- Source: `qobuz`, `deezer`
- Auth: config.toml (qobuz token, deezer arl)
- Note: deezer auth may fail if arl cookie expired

### tidal-dl-ng
- `tidal-dl-ng dl <url>` — download (no search command)
- Search: use tidalapi Python library directly or write wrapper script
- Auth: settings.json + token.json

### tidalapi (Python library)
- `tidalapi.Session()` for search
- Auth via OAuth tokens stored in token.json
