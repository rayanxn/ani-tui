# ani-tui: Go TUI Anime Torrent Streamer

## Context

Build a polished terminal anime streaming app from scratch in `/home/rayan/ani-tui`. The app searches anime via AniList, finds torrents on nyaa.si, streams them via anacrolix/torrent, and plays through mpv — all within a Bubble Tea TUI with lipgloss styling. AniList OAuth2 enables list sync and progress tracking.

System has Go 1.25.7 and mpv 0.41.0 installed.

---

## Architecture

**State-machine router** in a root `AppModel` delegates to view-specific sub-models. Each view is a `tea.Model`. Navigation uses a view stack for back-navigation. All I/O happens in `tea.Cmd` functions.

```
main.go → AppModel (router)
             ├── SearchModel    (textinput + list)
             ├── DetailModel    (viewport + episode selector)
             ├── TorrentsModel  (nyaa results list)
             ├── PlayerModel    (torrent stats + playback status)
             ├── LibraryModel   (AniList list with tab filters)
             └── AuthModel      (OAuth token paste flow)
```

**Streaming pipeline**: Torrent Reader → `io.Copy` goroutine → mpv stdin pipe. mpv IPC socket for playback monitoring/control. `tea.Tick` polls stats every 500ms.

---

## Directory Structure

```
ani-tui/
├── main.go                          # Entry point
├── go.mod / go.sum / .gitignore
├── internal/
│   ├── anilist/
│   │   ├── client.go                # GraphQL HTTP client with doQuery helper
│   │   ├── queries.go               # Query/mutation string constants
│   │   ├── oauth.go                 # AuthURL() for implicit grant + pin redirect
│   │   └── types.go                 # Media, Title, MediaList, User structs
│   ├── nyaa/
│   │   ├── search.go                # RSS fetch, XML decode, sort by seeders
│   │   └── types.go                 # RSS/Channel/Item structs with XML namespace tags
│   ├── torrent/
│   │   ├── client.go                # anacrolix/torrent client wrapper + lifecycle
│   │   ├── stream.go                # AddMagnetAndStream: metadata wait → file select → Reader
│   │   └── stats.go                 # Stats struct + FormatBytes/FormatSpeed helpers
│   ├── player/
│   │   ├── mpv.go                   # Process launch, stdin pipe, lifecycle (Start/StreamFrom/Stop)
│   │   └── ipc.go                   # IPC socket connect with retry, GetPlaybackStatus, Seek, TogglePause
│   ├── config/
│   │   └── config.go                # XDG config (~/.config/ani-tui/config.json), Load/Save
│   └── ui/
│       ├── styles.go                # Lipgloss palette (purple/magenta), layout styles
│       ├── keys.go                  # key.Binding maps (global + per-view)
│       ├── components.go            # RenderHeader, RenderStatusBar, RenderError helpers
│       └── views/
│           ├── app.go               # AppModel: ViewState enum, view stack, router, navigation msgs
│           ├── search.go            # Text input → AniList search → list results
│           ├── detail.go            # Metadata viewport (left) + episode selector (right)
│           ├── torrents.go          # Nyaa search → sorted torrent list
│           ├── player.go            # Buffering spinner → progress bar + torrent stats
│           ├── library.go           # Tab bar (Watching/Completed/Planning/...) + list
│           └── auth.go              # Show URL → paste token → verify → save config
```

---

## Implementation Phases

### Phase 1: Foundation
**Files**: `go.mod`, `main.go`, `.gitignore`, `internal/config/config.go`, `internal/ui/styles.go`, `internal/ui/keys.go`, `internal/ui/components.go`, `internal/ui/views/app.go`, `internal/ui/views/search.go`

- `go mod init github.com/rayanxn/ani-tui` + fetch deps (bubbletea, bubbles, lipgloss, anacrolix/torrent, mpvipc)
- **Config**: XDG path via `os.UserConfigDir()`, JSON load/save with atomic write, fields: `AniListToken`, `AniListUserID`, `DownloadDir`, `MpvPath`, `PreferredQuality`
- **Styles**: Adaptive purple/magenta palette, header/status bar/title/error/help styles, `CenterHorizontal` helper
- **Keys**: `GlobalKeyMap` (q/ctrl+c quit, ? help, esc back, tab switch, enter select), per-view key maps
- **AppModel**: `ViewState` enum (Search/Detail/Torrents/Player/Library/Auth), `viewHistory []ViewState` stack, `pushView`/`navigateBack`, `WindowSizeMsg` propagation, delegates Update/View to current sub-model
- **SearchModel**: `bubbles/textinput` (focused initially) + `bubbles/list` (empty). `/` refocuses input. Enter on input triggers search. Enter on list item emits `NavigateToDetailMsg`.
- **main.go**: Load config, create `AppModel`, run with `tea.WithAltScreen()` + `tea.WithMouseCellMotion()`

### Phase 2: AniList Integration
**Files**: `internal/anilist/types.go`, `internal/anilist/queries.go`, `internal/anilist/client.go`, update `internal/ui/views/search.go`, `internal/ui/views/detail.go`

- **Types**: `Media` (id, title, description, episodes, duration, status, format, genres, averageScore, studios, nextAiringEpisode), `Title` with `DisplayTitle()` (English → Romaji fallback), `MediaList`, `User`
- **Queries**: `searchAnimeQuery` (Page + media with search/ANIME/SEARCH_MATCH sort), `getAnimeDetailsQuery`, `getUserListQuery` (MediaListCollection), `updateProgressMutation` (SaveMediaListEntry), `viewerQuery`
- **Client**: `NewClient(token)`, generic `doQuery(ctx, query, vars, target)` using net/http + encoding/json. Methods: `SearchAnime`, `GetAnimeDetails`, `GetUserList`, `UpdateProgress`, `GetViewer`
- **Search wiring**: `searchAniListCmd(query)` returns `SearchResultsMsg`. `AnimeListItem` implements `list.Item` (Title=DisplayTitle, Description=format|eps|score|status)
- **DetailModel**: Split layout — left 2/3: `bubbles/viewport` with formatted metadata (title, native title, format, status, episodes, duration, score, genres, studio, description). Right 1/3: episode selector (j/k navigate, enter selects → `NavigateToTorrentsMsg`). If width < 80, stack vertically.

### Phase 3: Nyaa + Torrent Engine
**Files**: `internal/nyaa/types.go`, `internal/nyaa/search.go`, `internal/torrent/client.go`, `internal/torrent/stream.go`, `internal/torrent/stats.go`, `internal/ui/views/torrents.go`

- **Nyaa types**: RSS/Channel/Item XML structs. Namespace `https://nyaa.si/xmlns/nyaa` for seeders/leechers/infoHash/size. `Item.MagnetURI()` builds magnet from infoHash with common anime trackers.
- **Nyaa search**: `Search(ctx, query) → []Item`. GET `https://nyaa.si/?page=rss&q=QUERY&c=1_2&f=0`. Sort results by seeders descending. `BuildSearchQuery(title, ep, quality)` formats as `"Title 02 1080p"`.
- **Torrent client**: Wrap `anacrolix/torrent`. `NewClient(downloadDir)` configures with `Seed=false`. `Close()` drops torrent + closes client.
- **Stream**: `AddMagnetAndStream(ctx, magnetURI) → (Reader, error)`. Waits for `GotInfo()` with context cancellation. Finds largest file. Sets other files to `PiecePriorityNone`. Creates Reader with `SetReadahead(10MB)` + `SetResponsive()`.
- **Stats**: `GetStats(torrent) → Stats{BytesCompleted, BytesTotal, Peers, Seeders}`. `FormatBytes`, `FormatSpeed` helpers.
- **TorrentsModel**: Spinner during search. `searchNyaaCmd` fetches results. List with `TorrentListItem` (Title, "S:N | L:N | Size | Trusted"). Enter on item emits `NavigateToPlayerMsg{MagnetURI}`.

### Phase 4: mpv Playback
**Files**: `internal/player/mpv.go`, `internal/player/ipc.go`, `internal/ui/views/player.go`

- **mpv.go**: `Player` struct manages exec.Cmd. `Start(ctx)` removes stale socket, launches mpv with `--no-terminal --input-ipc-server=/tmp/ani-tui-mpv.sock --cache=yes --cache-secs=30 --demuxer-max-bytes=50MiB --demuxer-readahead-secs=20 -` (stdin). `StreamFrom(reader)` copies via `io.Copy` in caller's goroutine. `Stop()` cancels context, waits 3s, kills if needed, removes socket.
- **ipc.go**: `ConnectIPC(socketPath)` retries 20x at 250ms intervals. `GetPlaybackStatus() → PlaybackStatus{Position, Duration, Paused, EOFReached, Percentage}`. `TogglePause()`, `Seek(seconds)`. Uses `github.com/dexterlb/mpvipc`.
- **PlayerModel**: States: buffering → playing → done. `Init()` starts spinner + `startTorrentCmd`. On `torrentReadyMsg`: launch mpv, pipe reader, `connectIPCCmd`. On `ipcConnectedMsg`: start `tea.Tick` polls (500ms) for torrent stats + playback status. View shows: progress bar (`bubbles/progress`), position/duration, download speed, peers/seeders. Keys: space/p pause, h/l seek 10s. On EOF or mpv exit → `PlayerDoneMsg`.

### Phase 5: AniList Sync
**Files**: `internal/anilist/oauth.go`, `internal/ui/views/auth.go`, `internal/ui/views/library.go`, update `internal/ui/views/app.go`

- **OAuth**: Implicit grant flow. `AuthURL(clientID)` returns `https://anilist.co/api/v2/oauth/authorize?client_id=ID&response_type=token`. User visits URL, gets token, pastes into TUI. Token valid 1 year, no refresh.
- **AuthModel**: Step 0: show URL + instructions. Step 1: `textinput` with `EchoPassword` for token. Step 2: `verifyTokenCmd` calls `GetViewer`. Step 3: save to config, emit `AuthCompleteMsg`.
- **LibraryModel**: If not authenticated → redirect to auth. Loads via `GetUserList`. Tab bar for status categories (Watching/Completed/Planning/Dropped/Paused), tab/shift+tab cycles. List shows entries with progress. Enter navigates to detail.
- **Progress update**: On `PlayerDoneMsg` in `app.go`, if authenticated, call `UpdateProgress(animeID, episode, "CURRENT")`. Navigate back to detail view.
- **Tab in app.go**: Global tab key toggles between Search and Library views.

### Phase 6: Polish
- Error handling: typed error messages per view, dismissible with esc/enter
- Help overlay: `?` toggles centered bordered box with context-sensitive keybindings using `lipgloss.Place`
- Responsive layout: detail view stacks vertically below 80 columns
- Loading states: consistent `bubbles/spinner` across all async views
- Graceful shutdown: cleanup torrent client, mpv process, IPC connection before `tea.Quit`
- Torrent timeout: 2-minute deadline on metadata wait with cancel via esc

---

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| mpv input | stdin pipe | Simpler than file-watching; Reader auto-manages piece priorities |
| AniList auth | Implicit grant + paste token | No local HTTP server needed; ideal for CLI |
| Nyaa scraping | RSS feed | Cleaner than HTML scraping; includes structured metadata via XML namespace |
| GraphQL client | net/http + encoding/json | No external GraphQL library needed for simple query/mutation pattern |
| View routing | Enum state machine + view stack | Simple, no external router library, supports back-navigation |
| Torrent readahead | 10MB | Balances startup latency vs smooth playback |
| mpv IPC | dexterlb/mpvipc | Battle-tested Go library, handles JSON IPC protocol |

---

## Verification

1. **Build**: `go build -o ani-tui .` should produce binary
2. **Search**: Launch, type anime name, press enter → see AniList results
3. **Detail**: Select anime → see metadata + episode list
4. **Torrents**: Select episode → see nyaa.si results sorted by seeders
5. **Playback**: Select torrent → spinner during buffering → mpv opens with video → TUI shows stats
6. **Library**: Tab to library → see "not authenticated" → auth flow → see anime list
7. **Progress**: After episode finishes, AniList progress should update
8. **Navigation**: Esc goes back through view stack, q quits cleanly
9. **Resize**: Shrink terminal → layout adapts (detail view stacks, progress bar resizes)
