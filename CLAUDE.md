# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
go build -o ani-tui .        # Build the binary
go test ./...                 # Run all tests
go test ./internal/nyaa/      # Run tests for a specific package
go test -run TestClientSearch ./internal/nyaa/  # Run a single test
```

No Makefile, CI, or linter config exists — standard Go tooling only.

## Architecture

Terminal anime streaming app: search AniList → pick episode → find torrents on Nyaa → stream via torrent client → play in mpv. Built on Bubble Tea (Elm architecture).

### State Machine Router

`internal/ui/views/app.go` is the root `tea.Model`. It uses a `ViewState` enum and `viewHistory []ViewState` stack for back-navigation.

- Navigation happens via typed messages (`NavigateToDetailMsg`, `NavigateToTorrentsMsg`, etc.) emitted by sub-views and caught in `AppModel.Update()`
- `pushView()` appends current view to history; `navigateBack()` pops it
- All non-navigation messages are forwarded to the active sub-view via `propagateMsg()`
- Sub-views implement `View(width, contentHeight int) string` (not the standard `View() string`)

### View Flow

`SearchModel` → `DetailModel` → `TorrentsModel` → (PlayerModel planned)

- **SearchModel**: textinput + list + spinner; `/` focuses input, Enter on list emits `NavigateToDetailMsg`
- **DetailModel**: viewport + manual episode list; responsive layout (vertical if width < 80, horizontal otherwise); Enter emits `NavigateToTorrentsMsg`
- **TorrentsModel**: list + spinner; results sorted by seeders; trusted torrents highlighted green; Enter emits `NavigateToPlayerMsg`

### Package Responsibilities

- `internal/anilist/` — Hand-rolled GraphQL client (`doQuery` posts JSON to `graphql.anilist.co`). Token optional for public queries. Query strings in `queries.go`, response types in `types.go`.
- `internal/nyaa/` — RSS-based torrent search. XML parsing with `nyaa` namespace. `MagnetURI()` constructs magnets from info hash + hardcoded trackers. `BuildSearchQuery` zero-pads episode numbers.
- `internal/config/` — XDG config at `~/.config/ani-tui/config.json`. Atomic writes via tmp file + rename. Returns zero-value Config if file missing.
- `internal/ui/` — Shared styles (`styles.go` with `AdaptiveColor` for dark/light), key bindings (`keys.go`), header/status bar components (`components.go`).

### Testing Patterns

Tests exist only in `internal/nyaa/`. They use `httptest.NewServer` with injectable `BaseURL`/`HTTPClient` on the `Client` struct. All tests call `t.Parallel()`.

## PR and Commit Guidelines

- Do not add "Generated with Claude Code" or any AI attribution lines to PR descriptions.
