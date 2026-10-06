# Development

How Mazes is built, how it looks, how to maintain it, and how to build another game like it. Players should start at the [README](../README.md).

- [Design principles](#design-principles)
- [Run it](#run-it)
- [Architecture](#architecture)
- [Game rules and engine semantics](#game-rules-and-engine-semantics)
- [Fog](#fog)
- [Maps and reference solutions](#maps-and-reference-solutions)
- [API for humans and AI agents](#api-for-humans-and-ai-agents)
- [Frontend architecture](#frontend-architecture)
- [UI look and feel](#ui-look-and-feel)
- [Configuration](#configuration)
- [Deployment and hosting](#deployment-and-hosting)
- [Maintaining the project](#maintaining-the-project)
- [Building a game like this](#building-a-game-like-this)

## Design principles

These decisions shape everything else. Keep them when changing the game or starting a new one.

1. **One API for humans and AI.** The browser UI uses the same public GraphQL API an AI agent uses. There are no private endpoints for the UI. If a person can do something, an agent can too.
2. **The server is the source of truth.** Moves are applied on the server. Clients only display snapshots. Two browsers, or a browser and an agent, can play or watch the same session and always agree.
3. **Pure rules engine.** All game rules live in one package with no I/O, no clock and no dependencies. It can be tested exhaustively and reused by the validator and live sessions.
4. **Full snapshots, ordered by sequence number.** Every update carries the whole session state and a `seq` that only goes up. Clients never patch state; they replace it and drop anything older. Dropped or reordered messages can't corrupt the view.
5. **Files, not a database.** Maps are JSON files in `maps/`; each session is one JSON file in `sessions/`. Writes are atomic (temp file + fsync + rename). One binary plus two folders is the whole deployment.
6. **Agent-readable by design.** `/llms.txt` explains the rules and every API call for an AI. Each game page can copy a ready-made prompt. Answers are never exposed: reference solutions are not in the API, the prompts or the public repo.
7. **Players see one level at a time.** The fog is enforced by the server (see [Fog](#fog)), not by the UI, so an agent cannot read the whole tower from the API.
8. **Puzzles are verified, not hoped for.** Every map has a reference solution that must win, and its shortest route (par) is computed by the engine.

## Run it

```sh
cp .env.example .env      # optional
make run                  # builds and serves http://localhost:8000 (API + UI)
make dev                  # go run . -port 8000 -debug
```

Frontend development with hot reload, against a server running on :8000:

```sh
cd frontend && npm run dev:local     # http://localhost:5173
```

Server flags: `-port 8000 -host 0.0.0.0 -maps-dir maps -sessions-dir sessions -solutions-dir solutions -debug -public-url <url>` (`-public-url` sets the base URL written into `/llms.txt`; defaults to the request host).

Routes:

| Path | What it is |
|---|---|
| `/` | Landing page: what the game is, example map, map picker, recent sessions |
| `/play/<session>` | Board for the current level, tower diagram, d-pad, stats, AI prompt card |
| `/watch/<session>` | Read-only live view of any session that follows the player's level |
| `/sessions` | All sessions: filter by name/map/status, sort, "mine only" |
| `/maps` | All maps as cards with level-0 previews |
| `/editor` | Map editor with level tabs and server-side validation (loading and saving need the admin key) |
| `/learn` | Rules, fog, keyboard, how AI plays, a generic AI prompt, tips |
| `/graphql` | GraphQL over HTTP and WebSocket (`graphql-ws`); `/playground` when enabled |
| `/llms.txt` | Guide for AI agents, rendered from `llms.txt.tmpl` |

## Architecture

```
game/engine          pure rules, no deps/IO: MapConfig/NewMap, State.Move, Simulate, Shortest, Reachable, View
game/service         Session type, GameService facade + impl; interfaces MapStore, SessionStore, Broadcaster; coded errors
game/config          MapStore impl: maps/*.json, RWMutex, atomic writes
game/session         SessionStore impl: in-memory Manager + FilePersistence (sessions/<id>.json)
transport/websocket  Hub: per-session fan-out, latest-wins delivery
validate             map checks, finish reachability, reference-solution checks (+ cmd/validate CLI)
graph                gqlgen schema, resolvers, converters (fog), subscription forwarder, admin gate
api                  HTTP mux, /graphql, /llms.txt, SPA serving, hardening (limits, CORS, origin checks)
main.go              flags, .env, wiring, graceful shutdown
frontend/            SvelteKit (Svelte 5) + Tailwind v4, adapter-static → static/ (committed)
```

### Request and update flow

```
HTTP /graphql (api) → gqlgen resolver (graph) → GameService (game/service)
   → engine (pure rules)
   move: lock → State.Move → seq++ → Clone → unlock → Persist → Broadcast
   → websocket.Hub (8-slot queue per subscriber, drop oldest) → graph subscription forwarder (fogs the snapshot)
   → graphql-ws → frontend session store (keeps highest seq)
```

### Composition and dependencies

- `main.go` is the composition root. It builds the map store (`game/config`), the session store and file persistence (`game/session`) and the broadcaster (`transport/websocket`), passes them to `service.New(...)`, then starts `api.NewServer`. On shutdown it stops HTTP first, then persists every session.
- `game/service` defines the `MapStore`, `SessionStore` and `Broadcaster` interfaces it needs. Implementations import `service` and assert the interfaces at compile time; `service` never imports them. This keeps the core testable with fakes.
- `graph/` is a thin adapter: resolvers call `GameService` and convert results in `graph/convert.go`. `graph/generated` and `graph/model` are gqlgen output (committed, never hand-edited).

### Concurrency rules

- Each live `*service.Session` has its own lock. A mutating service method follows one pattern: lock → mutate → `touch` (seq++, last-action time) → `Clone` → unlock → persist → broadcast the clone.
- Never broadcast or do I/O or call `Persist` while holding a session lock.
- The hub and resolvers only ever see clones.

### Delivery guarantees

- Every broadcast is a full snapshot with a monotonic `seq`. The hub ignores anything with a `seq` at or below the last one it published for that session.
- Each subscriber has a small buffer. When it is full the hub drops the *oldest* queued update, so the final WON/LOST frame always arrives, even for a slow client.
- The subscription resolver subscribes first, then emits the current snapshot, then only newer updates. A new watcher sees the current state immediately and misses nothing.
- A blocked move (void or edge) bumps `seq`, sets `lastEvent.blocked` and is broadcast, so UIs can flash a bump, but it is not persisted: nothing durable changed.

### Persistence

- Sessions are written on create, every counted move (once per `bulkMove`), reset, rename and graceful shutdown.
- On load, each session is validated: the map snapshot is rebuilt with `engine.NewMap` and the stored `path` is replayed against it (adjacency, stairs and portal landings, position, move count, status).
- Each session embeds a copy of its map, so editing or deleting a map never affects existing sessions.
- Idle sessions older than `SESSION_TTL` are removed on startup. `MAX_SESSIONS` caps how many exist.

### Errors

`*service.Error{Code, Msg}` with codes `NOT_FOUND`, `INVALID_ARGUMENT`, `SESSION_TERMINAL`, `FORBIDDEN`, `LIMIT_REACHED`. Resolvers expose them as `extensions.code`; unknown errors are logged and returned as `INTERNAL`. Looking up a missing map or session returns `null`, not an error. Check codes with `errors.As`. Moving in a finished session is `SESSION_TERMINAL`; a blocked move is not an error.

### Security and hardening

- Map ids match `^[a-z0-9_-]{1,64}$` because they are file names. Session ids are 16 lowercase hex characters and are validated before touching disk.
- Admin operations (`createMap`, `updateMap`, `deleteMap`, `validateMap` and the `adminMap` query) require `X-Admin-Key` equal to `ADMIN_API_KEY`, compared through SHA-256 digests in constant time. `ALLOW_UNAUTHENTICATED_ADMIN=true` is honoured only when no key is set (local development). `adminMap` checks the key before looking the id up, so callers can't probe which maps exist.
- `max_moves ≤ 10000`. `bulkMove` accepts at most `service.MaxDirections` (20000) directions per call, because blocked moves are free and the request body cap alone would not bound the work.
- Request body capped at 1 MiB; query complexity capped at 1000 with weighted costs (`sessions` by `limit`). `api/hardening_test.go` keeps a copy of the frontend queries to make sure they stay under the cap.
- WebSocket: 10 s init timeout, frame-size guard (`api/wslimit.go`), origin check against `ALLOWED_ORIGINS`, which also drives CORS on `/graphql`.
- Map creation is atomic (`MapStore.Create`; a conflict is `INVALID_ARGUMENT`). Never check-then-save.

## Game rules and engine semantics

Moves: `UP`, `RIGHT`, `DOWN`, `LEFT`, one tile each.

- Coordinates: `x` = column, `y` = row, row 0 is the top, **y grows downward**. Level 0 is the bottom of the tower; `U` goes to level + 1.
- A move onto void or off the grid is **blocked**: nothing changes and it is **not counted**. Every other move is one counted move.
- Tiles: `.` void, `o` stone, `S` start, `F` finish, `U` stairs up, `D` stairs down, `1`-`9` portal holes.
- Stepping onto `U`/`D` moves you at once to the same x,y one level up/down (`via` = `STAIRS_UP`/`STAIRS_DOWN`). Stairs are **paired and two-way**: a `U` at level l (x,y) needs a `D` at level l+1 (x,y) and a `D` needs a `U` one level below (`NewMap` enforces it), so whoever can go up from a tile can go back down from the same x,y. You land on the paired stairs tile. Stepping onto a portal hole moves you to its twin (`via` = `PORTAL`), always on the same level (`NewMap` enforces it). Effects never chain: arriving on stairs or on a hole does not trigger it, so step off and back on to use it again. A stairs tile always triggers when stepped on, so it can't be walked across like a corridor; put stairs at dead ends.
- `State.Path` lists every tile entered, starting with the start tile. A stairs or portal move adds the tile stepped on and the landing tile.
- After each counted move: standing on `F` → **WON** (beats the move limit on the same move); otherwise `moves >= max_moves` → **LOST/MOVE_LIMIT**.
- Status: `READY` (no counted move yet), `PLAYING`, `WON`, `LOST`. Moving after `WON`/`LOST` is `ErrTerminal`.
- `Map.Par` is the shortest winning route by BFS over positions (every counted move costs 1; the move limit is ignored). `-1` means the finish is unreachable.
- `Map.ForcedDescents` is the fewest `D` stairs any winning route takes (BFS layered by descents; `-1` if unreachable). `MinLevelChanges()` is derived: every descent needs one extra climb, so it is `levels - 1 + 2*ForcedDescents`. Both are exposed on `Map`, `AdminMap` and `validateMap`, and printed by `make validate`. Every shipped map has 0 forced descents (asserted in `TestShippedMapsGolden`).

Engine rules for code: `State.Move` mutates in place; `Clone()` anything that leaves a lock. `Simulate(m, dirs, false)` allocates no events, so validating reference solutions stays cheap.

### Sessions

A session is one play of one map. `move` and `bulkMove` apply directions. `bulkMove` stops when the game ends and, by default (`stopOnBlocked: true`), at the first blocked attempt, so a misjudged plan cannot keep running from the wrong tile; `stopOnBlocked: false` skips blocked moves and carries on. `reset` returns to the start and keeps `attempts` (finished games) and `bestMoves` (fewest moves in a won game). Every update has a `seq`; clients drop updates with `seq <=` the last seen.

## Fog

Players only ever see the level they stand on. The server enforces it:

- The service keeps the full map in each session. Only `graph/convert.go` decides what leaves the server.
- `Session.map` is the public catalog entry: name, size, `levelCount`, `maxMoves`, `parMoves` and `preview` (the rows of level 0, which is where every game starts). It has no other level's layout, no start/finish coordinates and no portals. These fields do not exist on the type, so asking for them is a GraphQL validation error.
- `Session.view` is the current level: its rows and the portal holes on it, each with its twin's location (`b`) since both ends of a portal are always on the same level.
- `NewMap` guarantees start on level 0 and finish on the top level, so `levelCount - 1` is always the goal level and nothing needs hiding there.
- `adminMap` returns the full layout and is admin-only; the editor uses it.
- There is no dry-run query: players learn a map only through real moves in a session (`move`, `bulkMove`, `reset`).
- `state.path` and `lastEvent` carry positions on levels the player has already visited. That is the player's own history, not hidden information.

`api/fog_test.go` runs a session through every query, mutation and the subscription and asserts no other level's rows ever appear.

## Maps and reference solutions

```json
{
  "id": "two_floors", "name": "Two Floors", "difficulty": "easy",
  "levels": [
    ["Sooo..", "...o..", ".ooooU", ".o....", ".oo...", ".o...."],
    ["F.....", "o.....", "o..ooo", "o..o..", "oooo..", ".D...."]
  ],
  "max_moves": 100
}
```

`levels[0]` is the bottom level. All levels have the same size (3–20 per side), 1–12 levels. Glyphs: `.` void, `o` stone, `S` start (exactly one, on level 0), `F` finish (exactly one, on the top level), `U`/`D` stairs (paired: a `U` needs a `D` at the same x,y one level up and a `D` needs a `U` one level down), `1`-`9` portal holes (each digit used appears exactly twice, on the same level). `max_moves` defaults to 500 (max 10000). JSON Schema: `map-schema.json`.

Shipped maps (16), with their par:

| map | levels | idea | par |
|---|---|---|---|
| `first_steps` | 1 | a winding path with a dead-end branch | 9 |
| `ladder` | 3 | one short corridor and one staircase per level | 17 |
| `portal_twins` | 1 | three portal pairs chained across void | 11 |
| `pipe_dream` | 1 | portals hop across void | 9 |
| `two_floors` | 2 | two staircases up; one lands in a dead-end pocket | 18 |
| `lost_garden` | 1 | a 13×13 perfect maze, no tricks | 90 |
| `wormholes` | 1 | a 13×13 maze with three portal pairs between dead ends | 78 |
| `four_floors` | 4 | 7×7 corridor mazes, two staircase pairs per gap | 72 |
| `portal_climb` | 3 | one staircase per gap plus two portal pairs | 80 |
| `tower_maze` | 3 | corridor maze per level; staircases at dead ends, most of them lead somewhere you have already been | 62 |
| `miners_shaft` | 6 | level 2 (index 1) is an 8×8 board of islands joined only by arrows and pipes; the quick ways up are traps, so both pipes are required; the upper levels are corridors with trap pockets | 71 |
| `deep_tower` | 5 | five 9×9 corridor mazes, two staircase pairs per gap, two portal pairs | 93 |
| `grand_tower` | 6 | six 9×9 corridor mazes, two staircase pairs per gap, three portal pairs | 112 |
| `switchback` | 4 | 7×7 floors split into islands; two forced descents (7 level changes), a decoy up stair on every floor | 34 |
| `false_summit` | 5 | stairs by the start chain straight up to a pocket beside the visible finish; the real route circles level 0 and drops once | 66 |
| `portal_locks` | 4 | every floor is islands joined only by portals; the nearest stairs are decoys; one forced descent | 46 |

### Designing a good map

- Start from the trick you want to teach (stairs, drops that undo your progress, portals that skip walls or send you backward).
- Put dead ends where the stairs are and make the wrong stairs look attractive.
- The engine computes par; check that it matches your intended route length with `make validate`.
- Give the map a short `description`. It appears under the board.

### Reference solutions

`solutions/<id>.json` (`{"moves": ["UP", "RIGHT", …]}`). They are **not** in the public repo (`solutions/` is gitignored), never served by the API, and never put in prompts or `static/`. Without them `make validate` reports missing solutions and the solution-dependent tests skip (`requireSolutions` in `validate/validate_test.go`). A shortest route can be generated with `Map.Shortest()`.

### Adding a map

1. Add `maps/<id>.json` (id = file name).
2. Add `solutions/<id>.json` locally.
3. Run `make validate`: it checks structure, that the finish is reachable, and that the reference solution wins.
4. Add its golden par to `TestShippedMapsGolden` (`game/engine/engine_test.go`) and to the table in `validate/validate_test.go`.

Maps can also be created in `/editor` or with the admin mutations; both need the admin key.

## API for humans and AI agents

The schema lives in `graph/schema.graphqls`.

- **Queries:** `maps`, `map(id)`, `adminMap(id)` (admin), `session(id)`, `sessions(sort, limit, mapId)`. There is deliberately no `simulate`/dry-run query: agents test ideas with real moves and `reset`.
- **Mutations:** `createSession`, `renameSession`, `deleteSession`, `move`, `bulkMove`, `reset`, plus the admin map mutations.
- **Subscription:** `sessionUpdated(sessionID)` emits the current snapshot, then every change.

Agent support:

- `/llms.txt` (template `llms.txt.tmpl`, embedded in the binary) is the full guide for agents: rules, fog, endpoints and a valid example for every call. Keep it in sync with the schema; every example must run against the live API.
- The **Play with an AI** card on `/play` and `/watch` (`frontend/src/lib/prompt.ts`) builds a prompt for that exact session: id, the current level with an x/y ruler, portals on it, the fog explanation and ready-to-run GraphQL calls. Never a solution and never other levels.
- `/learn` has a generic prompt for any map.

## Frontend architecture

- SvelteKit with `adapter-static`, built as a single-page app: `ssr = false`, `prerender = false`, fallback `index.html`. The Go server serves `frontend/build` if present, otherwise `static/`, and falls back to `index.html` for the routes listed in `spaRoutes` (`api/server.go`). A new top-level route must be added there too.
- `static/` is committed. CI and deploys do not build the frontend.
- Svelte 5 runes only (`$state`, `$derived`, `$effect`, `$props`), `onclick`-style attributes, keyed `{#each}`.
- GraphQL: urql for queries and mutations (`network-only`, POST); one lazily created, shared `graphql-ws` client for subscriptions (`lib/graphql.ts`). `unwrap()` turns urql errors into exceptions; `lib/errors.ts` maps error codes to friendly titles and messages.
- Live sessions: `lib/stores/session.ts` fetches the snapshot, subscribes, and keeps only updates with a higher `seq`. The transport is injected (`lib/stores/source.ts`), so the store is tested with a fake source. Connection state (`connecting`, `live`, `reconnecting`, `offline`) drives the connection badge.
- Moves are sent straight to the server (`move`); the board renders whatever snapshot comes back, so the UI never disagrees with other viewers.
- Logic lives in plain TypeScript modules in `lib/` (`tower.ts`, `levels.ts`, `fog.ts`, `shortcuts.ts`, `prompt.ts`, `maps.ts`, `sessions.ts`), each with Vitest tests. Components (`Board`, `Tower`, `Dpad`, `MiniGrid`, `MapCard`, `Stats`, `ResultModal`, `AiPromptCard`, `ConnectionBadge`) stay thin.
- `lib/fog.ts` keeps an in-page memory of levels the player has already seen so the tower diagram can offer a read-only, clearly labelled peek at them. It is never sent to or from the server and is forgotten on reload.
- GraphQL documents and types are written by hand in `lib/queries.ts` and `lib/types.ts`. Update them whenever the schema changes.
- Brand names come from `lib/brand.ts` (`SITE`, `GAME`, `TITLE`, `pageTitle()`); don't hard-code them. `localStorage` keys use the `drbrain-mazes.` prefix; the admin key is kept in `sessionStorage` only.
- Gotcha: explicit `onchange`/`oninput` handlers run *before* `bind:value` updates. Read `event.currentTarget.value`, not the bound variable.

## UI look and feel

The goal: friendly, bright and calm, like a puzzle book. Readable for kids and adults, clear enough for a screenshot to explain the game. The Dr Brain games share one look.

### Palette and typography

- One accent family: **indigo**. Page background `#f5f6ff`, text `#1e1b4b` (indigo-950). Cards are white with `border-indigo-200`, `rounded-xl`, and at most a light `shadow-sm`.
- Primary actions: `bg-indigo-600` → `hover:bg-indigo-700`, white text. Secondary: `border-indigo-300`, indigo text, `hover:bg-indigo-50`. Success is emerald, failure is rose.
- Secondary text is `text-slate-600` at `text-xs`/`text-sm`. Section labels are small uppercase indigo text with wide tracking.
- System font stack. No web fonts. Navigation uses pills: `rounded-full`, the active page filled indigo.

### Layout

- Header: logo, **Dr Brain** with **Mazes** next to it (hidden on small screens), pill navigation. Footer: one-line tagline plus links to the rules, `/llms.txt` and the GraphQL endpoint.
- Content is centred at `max-w-6xl` with `px-4 sm:px-6`.
- Play page: two columns on large screens (board left; tower diagram, d-pad, stats and AI prompt card right), one column on mobile. Watch page: the same board, read-only, following the player's level.

### The board

- An SVG scaling to the width available, so it stays sharp at any size. Pale indigo checkerboard; void cells are empty.
- **Stone tiles:** rounded slate squares. **Start** and **finish** are marked (the finish is the cookie).
- **Stairs:** a blue up arrow (▲) or rose down arrow (▼) stamped on the tile. **Portals:** coloured rings, one colour per digit (1 blue, 2 yellow, 3 orange, 4 green, 5 pink, 6 purple, 7 red, 8 cyan, 9 white) with the digit as text. A dotted pipe joins a pair when both ends are on the level; a `?` badge marks a hole whose twin is out of sight.
- **Player:** an indigo arrowhead that glides (120 ms ease-out) and turns toward its last move.
- **Trail:** small translucent indigo dots on visited tiles.
- **Bump:** when the last move was blocked, a rose outline flashes on the player's tile.
- Moving between levels fades the board to the new level.

### The tower diagram

A white card of stacked ledges, one per level, with **FINISH** at the top and **START** at the bottom and a star on the player's level. Only levels the player has seen are buttons (read-only peek, labelled "remembered"); unexplored ledges are dashed and show `?`.

### Controls and feedback

- Arrow keys or WASD move; `R` resets; an on-screen d-pad serves touch; clicking a neighbouring tile steps there.
- Stats show moves, par, move limit, best and attempts. Errors appear inline as rose alert boxes with a friendly title and the error code in small monospace.
- On win or loss: a centred modal ("You won!" with moves versus par, or "Out of moves"), focus on **Reset & try again**, Escape closes it. A coloured banner stays under the board after the modal is closed.
- A connection badge shows whether live updates are connected.

### Accessibility and motion

- "Skip to content" link, `aria-current` on the active nav pill, labelled sections and button groups.
- The board has a text description of the player's position and level; results are announced through a polite live region.
- Visible focus everywhere: 2 px indigo outline with offset (`app.css`).
- `prefers-reduced-motion` turns off animations and transitions globally, including the glide.
- Colour is never the only signal: stairs have arrows, portals have digits, results have text.

## Configuration

`.env` (loaded at startup; see `.env.example`):

| Variable | Default | Purpose |
|---|---|---|
| `GRAPHQL_INTROSPECTION` | `true` | Allow schema introspection (keep on for agents) |
| `GRAPHQL_PLAYGROUND` | `true` | Serve `/playground` (off in production) |
| `ADMIN_API_KEY` | empty | Key for admin map operations |
| `ALLOW_UNAUTHENTICATED_ADMIN` | `false` | Local only: admin without a key, when none is set |
| `ALLOWED_ORIGINS` | empty (all) | Comma-separated origins for WebSocket upgrades and CORS |
| `MAX_SESSIONS` | `1000` | Cap on live sessions |
| `SESSION_TTL` | `168h` | Idle sessions older than this are removed on startup |

Frontend build-time variables: `PUBLIC_GRAPHQL_URL`, `PUBLIC_WS_URL`. When unset, the UI uses the page's own host and picks `ws`/`wss` from the page protocol.

## Deployment and hosting

**Continuous deployment.** Every push to `main` runs `.github/workflows/deploy.yml`: vet and test, build a static `linux/arm64` binary, upload it with `maps/` and `static/`, unpack into `/opt/drbrain-mazes` (keeping `sessions/` and `.env`), restart the service and smoke-test GraphQL on the host. Repository secrets: `EC2_HOST`, `EC2_SSH_KEY` (a deploy-only key), `EC2_KNOWN_HOSTS`.

**One-time host setup** (not in the repo; repeat on a new host):

1. Create `/opt/drbrain-mazes/` owned by `ubuntu`, with `sessions/` and a `.env` (production: playground off, a random `ADMIN_API_KEY`, `ALLOWED_ORIGINS` set to the site's http and https origins).
2. Add a systemd unit `drbrain-mazes.service`: `WorkingDirectory=/opt/drbrain-mazes`, `ExecStart=… -host 127.0.0.1 -port 8087 -maps-dir maps -sessions-dir sessions -public-url https://<your-domain>`, `Restart=always`.
3. Add an nginx `server` block for the domain that proxies to `127.0.0.1:8087` with WebSocket upgrade headers and a long `proxy_read_timeout`.
4. Point DNS at the host and issue a TLS certificate (certbot).
5. Add the deploy key's public half to `~/.ssh/authorized_keys` and set the three repository secrets.

## Maintaining the project

### Common changes

- **Schema change:** edit `graph/schema.graphqls` → `make generate` → update `graph/convert.go` and resolvers → mirror in `frontend/src/lib/{queries,types}.ts` → update `llms.txt.tmpl` and `prompt.ts` if agents are affected → update the query copies in `api/hardening_test.go`. Anything that reaches a `Session` goes through `toSession`; never add layout data there.
- **Engine change:** update the golden pars in `game/engine/engine_test.go` and `validate/validate_test.go` on purpose, never just to make a failure go away.
- **UI change:** `cd frontend && npm run check && npm test && npm run build:static` (or `make build-frontend`), then commit `static/`. If you skip the build, production keeps the old UI.
- **New page:** add the route under `frontend/src/routes/`, add it to `spaRoutes` in `api/server.go`, and to the nav in `+layout.svelte` if needed.
- **Dependencies:** don't run `go mod tidy` casually. `gqlgen generate` rewrites `go.mod`; re-`go get` anything it drops.

### Testing conventions

- Go: standard `testing`, table tests with `t.Run`, `t.Helper`/`t.TempDir`/`t.Cleanup`. No `t.Parallel()` (tests use `t.Setenv`). Always run with `-race`.
- Service tests use `newEnv` in `game/service/helpers_test.go` (fake map store, spy persistence, recording broadcaster). API tests use `httptest.Server`.
- Frontend: Vitest + jsdom + Testing Library.

```sh
make test                                        # go test -race ./...
go test -race -run TestName ./game/engine        # one Go test
make validate                                    # maps + reference solutions
make lint                                        # golangci-lint
make verify                                      # gofmt, vet, lint, test
cd frontend && npm run check && npm test
cd frontend && npx vitest run src/lib/tower.test.ts   # one frontend test file
scripts/smoke.sh 9191                            # end-to-end HTTP smoke (starts its own server)
```

### Before you push

`make verify && make validate`, then `go run . -port 9191` and `scripts/smoke.sh 9191`. For UI changes, open `/play/<id>` and `/watch/<id>` in a browser and walk a map to the end. Pushing to `main` deploys.

### Public repo rules

- `README.md` is for players. Technical content goes here.
- Never commit `solutions/`, and never put solution content in the API, prompts or `static/`.
- Docs and code comments describe the game on its own terms, without references to other private projects.

## Building a game like this

A checklist for the next game in the collection, based on what worked here:

1. **Write the rules as a pure engine first.** State in, step, state out. No I/O, no time. Add `Simulate` (run to the end) early; the validator and the tests use it.
2. **Pin the rules with golden tests** before building anything else: known move lists and their exact lengths and outcomes.
3. **Define the data as files with a JSON Schema** (maps, puzzles, levels). Validate on load and in CI. Keep reference solutions private and check them with a CLI.
4. **Put one service facade in front of the engine.** It owns sessions and locking. Storage and broadcasting are interfaces it defines.
5. **Make every update a full snapshot with a monotonic `seq`**, and make subscriptions emit the current snapshot first. This removes a whole class of sync bugs.
6. **Decide what players may see and enforce it on the server**, in one conversion layer, with end-to-end tests that probe every read path.
7. **Expose everything through one GraphQL schema** (queries, mutations, one subscription per live object). Add coded errors, complexity limits, body limits and origin checks from day one.
8. **Write `/llms.txt` alongside the schema**, plus a per-session "Play with an AI" prompt. Test the examples against the real server.
9. **Build the UI as a static SPA served by the same binary.** Keep logic in tested TypeScript modules, reuse the brand constants, and follow the look and feel above so games in the collection feel related.
10. **Ship as one binary + data folders** behind nginx with TLS, deployed by a GitHub Action on push to `main`.
11. **Make the README about playing**, with a section on letting an AI play. Keep developer material in documents like this one.
