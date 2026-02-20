# PROJECT_BRAIN.md — ani-tui

Last updated: 2026-02-20

## 1) Project Snapshot

**Project:** `ani-tui` (Go terminal anime streaming app)

**Core flow:**
AniList search → anime detail + episode pick → Nyaa torrents → torrent stream → mpv playback

**Current status (from codebase):**
- Search view ✅
- Detail view ✅
- Torrents view ✅
- Player view ✅ (basic stream + stats + mpv lifecycle)
- AniList auth/library sync ⏳ not implemented yet (planned Phase 5)

---

## 2) What Exists Right Now (Code Reality)

### Entry + Root App
- `main.go` loads config, starts Bubble Tea app with alt-screen.
- `internal/ui/views/app.go` routes across view states with a back-stack.
- Implemented states in enum: Search, Detail, Torrents, Player (+ Library/Auth placeholders).

### API + Data Layers
- `internal/anilist/`:
  - GraphQL client + query constants + media types.
  - Public search/details flow wired.
- `internal/nyaa/`:
  - RSS search + XML parsing + seeders sorting.
  - query builder and tests exist.
- `internal/torrent/`:
  - anacrolix torrent client wrapper.
  - magnet add, metadata wait, largest-file selection, prioritized streaming.
  - transfer stats helpers.

### Player
- `internal/player/player.go` runs mpv and serves torrent reader via local HTTP endpoint (`/video`) using `http.ServeContent`.
- `internal/ui/views/player.go`:
  - starts stream,
  - shows spinner/loading/errors,
  - polls torrent stats,
  - exits back when mpv closes.

### UI
- Shared styles, key maps, components under `internal/ui/`.
- Search/detail/torrents/player views exist.

---

## 3) Important Gap vs PLAN.md

`PLAN.md` Phase 4 describes **mpv stdin pipe + explicit IPC module** (`mpv.go` + `ipc.go`).

Current implementation instead uses:
- local HTTP stream to mpv URL (`internal/player/player.go`),
- no dedicated `ipc.go` file,
- no playback-status polling from mpv IPC yet.

This is not necessarily wrong — just a **design divergence** to decide on intentionally.

---

## 4) Recommended Next Milestones

1. **Phase 4 hardening (short term):**
   - Decide canonical player architecture:
     - keep HTTP stream approach, or
     - switch to stdin+IPC as in plan.
   - Add player-focused tests where possible (unit-level around state transitions and cleanup behavior).

2. **Phase 5 delivery (high value):**
   - AniList OAuth token flow.
   - Library view + tabs.
   - Progress update after episode completion.

3. **Polish/ops:**
   - better error surface in all views,
   - timeout/cancel controls for metadata wait,
   - graceful shutdown verification.

---

## 5) Working Agreement (You + Me + Claude Code)

### Claude Code role
- Executes implementation tasks quickly (phase work).

### My role (Ray)
- Keep this file up to date.
- Track architecture decisions and drift from plan.
- Review runs and suggest next feature/cleanup work.
- Help convert vague goals into concrete phase contracts.

### Your role (Rayan)
- Approve direction and tradeoffs.
- Greenlight architecture decisions.

---

## 6) Phase Contract Template (Use Before Every Run)

```md
## Task

## Goal (1 sentence)

## Files expected to change
- 

## Risks
- 

## Validation
- go build ./...
- go test ./...

## Done criteria
- 
```

---

## 7) Run Log Template (Append After Every Run)

```md
### YYYY-MM-DD HH:MM
- Task:
- Files changed:
- What passed:
- What failed:
- Notes/risk:
- Next step:
```

---

## 8) Questions for You (to lock direction)

1. For Phase 4: do you want to **keep current HTTP-to-mpv design** or align to PLAN’s **stdin+IPC** architecture?
2. Should I maintain this file as the single source of truth, and keep `PLAN.md` as roadmap only?
3. Do you want a second file (`RUN_LOG.md`) for chronological execution history, or keep logs in this file?
