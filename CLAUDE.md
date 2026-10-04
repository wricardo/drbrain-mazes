# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# Dr Brain - Mazes (`drbrain-mazes`) — agent guide

Brand: **Dr Brain** is the family of reimplemented brain games and **Mazes** is this game. Every game lives in its own repo; this repo, its Go module path (`github.com/wricardo/drbrain-mazes`) and the server binary share the name `drbrain-mazes`. `localStorage` keys use that prefix too. User-facing names live in `frontend/src/lib/brand.ts`, `llms.txt.tmpl` and the README (player-facing).

Maze puzzle server + SvelteKit UI: a tower of levels, the player walks one tile per move from START (bottom level) to FINISH (top level), using stairs and portals. Players only ever see the level they stand on (fog). Read `docs/DEVELOPMENT.md` for rules, map format and semantics (`README.md` is player-facing).

## Commands

```sh
make dev                                   # go run . -port 8000 -debug (serves static/ UI too)
make test                                  # go test -race ./...
go test -race -run TestName ./game/engine  # single Go test
make validate                              # maps + reference solutions (needs local solutions/)
make lint                                  # golangci-lint
make generate                              # gqlgen after schema edits
cd frontend && npm run dev:local           # Vite UI against a local server on :8000
cd frontend && npx vitest run src/lib/tower.test.ts   # single frontend test
```

## Public repo, private solutions, auto-deploy

- The repo is public. `README.md` is for players (humans and AI agents), not developers; technical docs go in `docs/DEVELOPMENT.md`. Never mention other local repos (sibling games, rule references) in any tracked file.
- `solutions/` is gitignored and exists only locally (`{"moves":["UP","RIGHT",…]}` per map). Never commit it or add solution content anywhere tracked. Solution-dependent tests in `validate/validate_test.go` skip via `requireSolutions` when it is absent (CI, fresh clones).
- Every push to `main` runs `.github/workflows/deploy.yml` (vet, test, linux/arm64 build, ship binary + `maps/` + `static/`). The UI is not built in CI, so a stale `static/` ships stale UI. Host-side setup (systemd unit `drbrain-mazes` on `127.0.0.1:8087`, nginx vhost, TLS, `/opt/drbrain-mazes/.env` with `ADMIN_API_KEY`) is manual and not in the repo.

## Package map (dependency direction →)

```
game/engine      pure rules, no deps/IO: MapConfig/NewMap, State.Move, Simulate, Shortest/Reachable, View (fog)
game/service     Session type, GameService facade + impl; interfaces MapStore, SessionStore, Broadcaster; coded errors
game/config      MapStore impl: maps/*.json, RWMutex, atomic writes
game/session     SessionStore impl: in-memory Manager + FilePersistence (sessions/<id>.json)
transport/websocket  Hub: SubscribeSession fanout, latest-wins (no raw WS clients)
validate         map checks, finish reachability, reference-solution check (+ cmd/validate)
graph            gqlgen schema/resolvers, converters (fog lives here), subscription forwarder, admin gate
api              HTTP mux, /graphql, /llms.txt, SPA serving (frontend → static/)
main.go          flags, .env, wiring, graceful shutdown
frontend/        SvelteKit (Svelte 5) + Tailwind v4, adapter-static → static/ (committed)
maps/ solutions/ map-schema.json   data; solutions are never exposed by the API
```

`service` defines `Session`; `session` and `websocket` import `service`, never the reverse (the service talks to them through interfaces).

## Invariants — do not break

- **Engine**: `State.Move` mutates in place; `Clone()` anything leaving the lock. A blocked move (void/edge) changes nothing and is not counted. Stairs are paired and two-way: `U` at level l (x,y) needs `D` at level l+1 (x,y) and vice versa (`NewMap` enforces it), so whoever can climb can climb back from the same x,y. You land on the paired stairs tile; effects never chain, so arriving on stairs or a portal hole does not trigger it (step off and on again). A stairs tile always triggers when stepped on, so it can't be crossed like a corridor: put stairs at dead ends. Start is on level 0, finish on the top level (`NewMap` enforces both). `Simulate` with `withEvents=false` allocates no events.
- **Fog**: players never receive more than the current level. The service keeps the full map snapshot in `Session.Map`; only `graph/convert.go` turns it into the public `Map` (level-0 preview) and `LevelView` (current level). No resolver, subscription or mutation may serialize `AdminMap` data into a `Session`. `adminMap` is admin-gated.
- **Locking**: live `*service.Session` is guarded by `Lock/Unlock`. Never broadcast or do I/O while holding it. Hub/resolvers only see `Clone()` snapshots. Don't hold the lock when calling `SessionStore.Persist`.
- **Delivery**: every broadcast carries a full snapshot + monotonically increasing `Seq`; hub drops the *oldest* queued update on a full buffer so terminal states are never lost. Subscription resolver emits the current snapshot first. A blocked move still bumps `Seq` and broadcasts (so UIs can flash a bump) but is not persisted.
- **Map snapshot**: sessions embed their own copy of the map; map edits/deletes must never touch existing sessions.
- **Security**: map ids match `^[a-z0-9_-]{1,64}$` (they are filenames); session ids are validated before touching disk; admin mutations and `adminMap` need `X-Admin-Key` == `ADMIN_API_KEY` (constant-time) unless `ALLOW_UNAUTHENTICATED_ADMIN=true`; `max_moves ≤ 10000`; `simulate`/`bulkMove` direction lists are capped (`service.MaxDirections`). Never add solutions to the GraphQL schema.

## Workflows

- Edit `graph/schema.graphqls` → `make generate` → update `graph/convert.go` / resolvers → mirror in `frontend/src/lib/{queries,types}.ts` → update `llms.txt.tmpl`, `prompt.ts` and the query copies in `api/hardening_test.go`.
- New map: see `docs/DEVELOPMENT.md` "Adding a map"; `make validate` must pass.
- Engine change: update the golden shortest-route lengths in `game/engine/engine_test.go` (`TestShippedMapsGolden`) and `validate/validate_test.go` (the table in `docs/DEVELOPMENT.md` lists them) deliberately, never to make a failure disappear.
- UI change: `cd frontend && npm run check && npm test && npm run build:static` (or `make build-frontend`), commit `static/`.
- Don't run `go mod tidy` casually — `gqlgen generate` rewrites go.mod; re-`go get` any dropped deps.

## Verification checklist

`make verify` (or `go build ./... && go vet ./... && go test -race ./... && make validate`), then run `go run . -port 9191` and `scripts/smoke.sh 9191`; for UI changes load `/play/<id>` and `/watch/<id>` in a browser and walk a map to the end.
