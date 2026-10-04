# Repository Guidelines

## Project Overview
drbrain-mazes (product name: **Dr Brain - Mazes**; each Dr Brain game is its own repo): maze puzzle server (Go 1.25) + SvelteKit (Svelte 5) UI. A player walks one tile per move through a tower of levels (stone floor, void, stairs up/down, paired portals) from START on the bottom level to FINISH on the top level, seeing only the level they stand on (fog). Exposes GraphQL (queries, mutations, subscriptions over graphql-ws), `/llms.txt` (agent-facing rules, rendered from `llms.txt.tmpl`), and serves the built SPA. Root `CLAUDE.md` holds the invariants; `docs/DEVELOPMENT.md` documents rules, map format and semantics (`README.md` is player-facing); brand constants live in `frontend/src/lib/brand.ts`.

## Architecture & Data Flow
```
HTTP /graphql (api/server.go) → gqlgen resolver (graph/) → service.GameService (game/service)
   → engine (pure rules) ; Move → advance → Seq++ → Clone → unlock → Persist → Broadcast
   → websocket.Hub (8-slot per-subscriber queue, drop-oldest) → graph/subscription_forward.go → graphql-ws → frontend store
```
- **Composition root**: `main.go:run` wires `config.Manager` (MapStore) + `session.Manager`/`FilePersistence` (SessionStore) + `websocket.Hub` (Broadcaster) into `service.New(...)`, then `api.NewServer`. Shutdown: stop HTTP, then persist sessions.
- **DI via interfaces** defined in `game/service` (`MapStore`, `SessionStore`, `Broadcaster`); implementations assert them at compile time. `service` never imports `session`/`websocket`.
- **Engine** (`game/engine`): no IO/deps. `NewMap`, `NewState`, `State.Move`, `Simulate`, `Map.Shortest`, `Map.Reachable`, `Map.View`. `Move` mutates in place; `Clone()` before leaving a lock.
- **Fog**: the service stores the full map snapshot; `graph/convert.go` is the only place that shapes what clients see (`Map` = catalog + level-0 preview, `LevelView` = current level, `AdminMap` = full, admin only).
- **Locking**: live `*service.Session` guarded by `Lock/Unlock`. Never broadcast, do I/O, or `Persist` while holding the session lock. Hub/resolvers only see clones.
- **Delivery**: every broadcast = full snapshot + monotonic `Seq`. Subscription resolver subscribes first, then emits current snapshot, then only increasing `Seq`. Frontend `lib/stores/session.ts` also merges by `seq`.
- **Map snapshot**: sessions embed their own map copy; map edit/delete never touches sessions.
- **Errors**: `*service.Error{Code,Msg}` (`NOT_FOUND`, `INVALID_ARGUMENT`, `SESSION_TERMINAL`, `FORBIDDEN`, `LIMIT_REACHED`); resolvers surface as `extensions.code`; unknown errors log and return `INTERNAL`. Missing map/session query → GraphQL `null`. Check with `errors.As`.

## Key Directories
| Path | Purpose |
|---|---|
| `game/engine` | rules, moves, BFS solver, fog view (`types.go`, `engine.go`, `view.go`) |
| `game/service` | `Session`, `GameService` (`impl.go`), `errors.go` |
| `game/config` | map store: `maps/*.json`, RWMutex, atomic temp+rename writes |
| `game/session` | in-memory registry + `sessions/<id>.json` versioned persistence |
| `transport/websocket` | `Hub` (Broadcaster impl; no raw WS clients) |
| `graph/` | `schema.graphqls`, resolvers, `convert.go`, admin gate in `resolver.go`; `generated/`, `model/` are gqlgen output (don't hand-edit) |
| `api/` | mux, body cap (1 MiB), complexity cap (1000), WS origin check, SPA fallback (`spaRoutes`) |
| `validate/`, `cmd/validate` | map + finish reachability + reference-solution checks |
| `maps/`, `solutions/`, `map-schema.json` | data; `sessions/` is runtime, gitignored |
| `frontend/` | SvelteKit + Tailwind v4; builds into `static/` (committed) |

## Development Commands
```bash
make build | run PORT=8000 | dev        # dev = go run . -port 8000 -debug
make test                                # go test -race ./...
go test ./game/engine -run '^TestShippedMapsGolden$'   # single test
make vet | fmt | lint                    # lint needs golangci-lint
make validate                            # all maps + reference solutions
make generate                            # gqlgen, after editing graph/schema.graphqls
make verify                              # gofmt check, vet, lint, test
make build-frontend                      # npm ci if needed + build:static → static/
cd frontend && npm run check && npm test && npm run build:static
go run . -port 9191 && scripts/smoke.sh 9191   # smoke.sh starts its own temp server
```
Flags: `-port 8000 -host 0.0.0.0 -maps-dir -sessions-dir -solutions-dir -debug -public-url`. Env (`.env` via godotenv): `ADMIN_API_KEY`, `ALLOW_UNAUTHENTICATED_ADMIN`, `ALLOWED_ORIGINS`, `GRAPHQL_INTROSPECTION`, `GRAPHQL_PLAYGROUND`, `MAX_SESSIONS=1000`, `SESSION_TTL=168h`. Frontend env: `PUBLIC_GRAPHQL_URL`, `PUBLIC_WS_URL`.

## Code Conventions & Common Patterns
- Standard `gofmt`; domain-noun types (`GameService`, `MapStore`), concrete impls usually `Manager`; GraphQL camelCase, enums/codes `UPPER_SNAKE`.
- Mutating service method pattern: lock session → mutate → `touch` (Seq++, LastActionAt) → `Clone` → unlock → persist → broadcast the clone. A blocked move bumps Seq and broadcasts but does not persist.
- Persistence: temp file + rename (`game/session/file_persistence.go`, `game/config/manager.go`); load-time validation replays the stored path against the map snapshot.
- Schema change flow: edit `graph/schema.graphqls` → `make generate` → update `graph/convert.go` + resolvers → mirror by hand in `frontend/src/lib/{queries,types}.ts` → `llms.txt.tmpl`, `prompt.ts`, `api/hardening_test.go`.
- Frontend: Svelte 5 runes only (`$state/$derived/$effect/$props`), `onclick` attributes, keyed `{#each}`; urql (network-only, POST) + lazy shared `graphql-ws`; `unwrap()` turns urql failures into exceptions. **Gotcha**: explicit `onchange/oninput` handlers run *before* `bind:value` updates — read `event.currentTarget.value`, not the bound var.
- New top-level route → also add to `spaRoutes` in `api/server.go`.
- Don't run `go mod tidy` casually (gqlgen generate rewrites go.mod; re-`go get` dropped deps).

## Important Files
`main.go`, `api/server.go`, `graph/schema.graphqls`, `graph/resolver.go` (admin gate), `graph/schema.resolvers.go`, `graph/convert.go` (fog), `game/engine/{types,engine,view}.go`, `game/service/{impl,errors}.go`, `transport/websocket/hub.go`, `llms.txt.tmpl`, `map-schema.json`, `gqlgen.yml`, `Makefile`, `frontend/src/lib/{queries,types,graphql}.ts`, `frontend/src/lib/stores/{session,source}.ts`, `frontend/src/routes/{play,watch,editor,sessions,maps}`.

Adding a map: create `maps/<id>.json` (levels 1–12, each 3–20 × 3–20, identical size; glyphs `. o S F U D 1-9`; S on level 0, F on the top level; stairs are paired: `U` at level l (x,y) needs `D` at level l+1 (x,y) and vice versa; each portal digit exactly twice) + `solutions/<id>.json` (`{"moves":[…]}`), then `make validate` and add its golden par.

## Runtime/Tooling Preferences
Go 1.25; gqlgen 0.17.73, gorilla/mux, gorilla/websocket, godotenv. Frontend: **npm** (lockfile v3, `.npmrc engine-strict=true`), Node `^20.19 || >=22.12`, Vite, adapter-static (`fallback: index.html`). Go serves `frontend/build` first, then `static/`. `static/` is committed — rebuild and commit after UI changes. `graph/generated` is committed.

## Testing & QA
- Go: stdlib `testing`, `t.Helper/TempDir/Cleanup`, table + `t.Run`, no `t.Parallel()` (tests use `t.Setenv`). Service tests use `newEnv` in `game/service/helpers_test.go` (fake MapStore, spy persistence, broadcaster recorder). API tests use `httptest.Server`. Always run with `-race`.
- Golden shortest-route lengths of all 13 shipped maps (deliberate changes only): `game/engine/engine_test.go` (`TestShippedMapsGolden`) and `validate/validate_test.go`; the table is in `docs/DEVELOPMENT.md`.
- Frontend: Vitest + jsdom + Testing Library (`cd frontend && npm test`, single file `npm test -- src/lib/tower.test.ts`).
- UI changes: load `/play/<id>` and `/watch/<id>` in a browser.

## Known Issues
- None tracked. Add items here only when verified.
