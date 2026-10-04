#!/usr/bin/env bash
# End-to-end smoke test: starts its own server, plays a session to WON over
# GraphQL (blocked move, single move, bulkMove, terminal refusal, reset),
# checks the fog (view of the current level, no layout in the public types,
# adminMap behind the admin key), checks /llms.txt, then shuts down. Needs solutions/first_steps.json.
# Usage: scripts/smoke.sh [port]
set -euo pipefail

PORT="${1:-9191}"
URL="http://127.0.0.1:${PORT}"
cd "$(dirname "$0")/.."

TMP="$(mktemp -d)"
go build -o "${TMP}/drbrain-mazes" .
ADMIN_API_KEY=smoke-key ALLOW_UNAUTHENTICATED_ADMIN=false "${TMP}/drbrain-mazes" -port "${PORT}" -host 127.0.0.1 -sessions-dir "${TMP}/sessions" >"${TMP}/server.log" 2>&1 &
PID=$!
trap 'kill "${PID}" 2>/dev/null || true; wait "${PID}" 2>/dev/null || true; rm -rf "${TMP}"' EXIT

for _ in $(seq 1 50); do
  curl -sf "${URL}/healthz" >/dev/null && break
  sleep 0.1
done

gql() { # gql '<query>'
  curl -sf -H 'Content-Type: application/json' -d "$(jq -nc --arg q "$1" '{query:$q}')" "${URL}/graphql"
}
gqlraw() { # like gql, but keeps the body of non-2xx (validation error) responses
  curl -s -H 'Content-Type: application/json' -d "$(jq -nc --arg q "$1" '{query:$q}')" "${URL}/graphql"
}
fail() { echo "FAIL: $*" >&2; cat "${TMP}/server.log" >&2; exit 1; }

SOLUTION=solutions/first_steps.json
[ -f "${SOLUTION}" ] || fail "${SOLUTION} missing (reference solutions are local-only)"
FIRST=$(jq -er '.moves[0]' "${SOLUTION}")
REST=$(jq -er '.moves[1:] | join(",")' "${SOLUTION}")
ALL=$(jq -er '.moves | join(",")' "${SOLUTION}")
N=$(jq -er '.moves | length' "${SOLUTION}")

echo "maps:"; gql '{ maps { id parMoves } }' | tee /dev/stderr | jq -e '.data.maps | length > 0' >/dev/null || fail "no maps"
echo

SIM=$(gql "query { simulate(mapID:\"first_steps\", moves:[${ALL}]) { status moves } }")
echo "simulate: ${SIM}"
echo "${SIM}" | jq -e '.data.simulate.status == "WON"' >/dev/null || fail "simulate not WON"

R=$(gql 'mutation { createSession(mapID:"first_steps") { id view { level levelCount rows } map { levelCount preview } } }')
SID=$(echo "${R}" | jq -er '.data.createSession.id') || fail "createSession"
echo "session: ${SID}"
echo "${R}" | jq -e '.data.createSession.view.level == 0 and (.data.createSession.view.rows | length) > 0 and .data.createSession.view.rows == .data.createSession.map.preview' >/dev/null || fail "start view must be the level-0 preview"

# Fog: the public map has no layout fields; adminMap needs the admin key.
gqlraw '{ maps { levels } }' | jq -e '.errors[0].extensions.code == "GRAPHQL_VALIDATION_FAILED"' >/dev/null || fail "public Map must not expose levels"
gql '{ adminMap(id:"first_steps") { id } }' | jq -e '.errors[0].extensions.code == "FORBIDDEN"' >/dev/null || fail "adminMap without key must be FORBIDDEN"
curl -sf -H 'Content-Type: application/json' -H 'X-Admin-Key: smoke-key' -d '{"query":"{ adminMap(id:\"first_steps\") { levelCount levels } }"}' "${URL}/graphql" \
  | jq -e '.data.adminMap.levelCount >= 1 and (.data.adminMap.levels | length) >= 1' >/dev/null || fail "adminMap with key"

# The start tile sits in the top-left corner: UP runs off the map.
R=$(gql "mutation { move(sessionID:\"${SID}\", direction:UP) { state { moves status } lastEvent { blocked via } } }")
echo "blocked move: ${R}"
echo "${R}" | jq -e '.data.move.lastEvent.blocked == true and .data.move.state.moves == 0 and .data.move.state.status == "READY"' >/dev/null || fail "blocked move"

R=$(gql "mutation { move(sessionID:\"${SID}\", direction:${FIRST}) { state { moves status } lastEvent { blocked via } } }")
echo "first move: ${R}"
echo "${R}" | jq -e '.data.move.lastEvent.blocked == false and .data.move.state.moves == 1 and .data.move.state.status == "PLAYING"' >/dev/null || fail "first move"

R=$(gql "mutation { bulkMove(sessionID:\"${SID}\", directions:[${REST}]) { state { moves status } bestMoves attempts } }")
echo "bulkMove: ${R}"
echo "${R}" | jq -e ".data.bulkMove.state.status == \"WON\" and .data.bulkMove.state.moves == ${N} and .data.bulkMove.bestMoves == ${N} and .data.bulkMove.attempts == 1" >/dev/null || fail "bulkMove did not win"

gql "mutation { move(sessionID:\"${SID}\", direction:UP) { id } }" | jq -e '.errors[0].extensions.code == "SESSION_TERMINAL"' >/dev/null || fail "move after WON must be SESSION_TERMINAL"

R=$(gql "mutation { reset(sessionID:\"${SID}\") { state { moves status } bestMoves attempts } }")
echo "reset: ${R}"
echo "${R}" | jq -e ".data.reset.state.status == \"READY\" and .data.reset.state.moves == 0 and .data.reset.bestMoves == ${N} and .data.reset.attempts == 1" >/dev/null || fail "reset"

curl -sf "${URL}/llms.txt" | grep -q "Dr Brain" || fail "/llms.txt"
echo "llms.txt ok"
echo "SMOKE OK"
