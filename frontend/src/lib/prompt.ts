import { TITLE } from './brand';
import type { GameMap, LevelView, Position } from './types';

/** The slice of a session the prompt needs: the catalog map, what the miner sees now, and where it stands. */
export interface PromptSession {
	id: string;
	map: GameMap;
	view: LevelView;
	state: { pos: Position };
}

/** Renders one level with an x ruler and a y prefix so an agent can read coordinates off it directly. */
export function layoutBlock(rows: string[]): string {
	const width = rows.reduce((w, r) => Math.max(w, r.length), 0);
	const pad = String(Math.max(0, rows.length - 1)).length;
	const ruler = ' '.repeat(pad + 3) + Array.from({ length: width }, (_, x) => String(x % 10)).join('');
	const body = rows.map((row, y) => `y${String(y).padStart(pad, ' ')}  ${row}`);
	return [ruler, ...body].join('\n');
}

const at = (p: { x: number; y: number }): string => `(${p.x},${p.y})`;

/** One line per portal visible on the current level. */
function portalLines(view: LevelView): string {
	if (view.portals.length === 0) return '(none on this level)';
	return view.portals
		.map((p) => (p.b ? `${p.glyph}: ${at(p.a)} <-> ${at(p.b)}` : `${p.glyph}: ${at(p.a)} -> twin is on another level (location hidden)`))
		.join('\n');
}

/**
 * Prompt that makes an AI agent play (and win) one specific, existing session through the GraphQL API.
 * It contains only what a player may see: the level the miner is on when the prompt is copied,
 * never the other levels and never a solution.
 */
export function buildSessionPrompt(origin: string, s: PromptSession): string {
	const m = s.map;
	const v = s.view;
	const top = m.levelCount - 1;
	const par = m.parMoves === null ? 'unknown' : `${m.parMoves} moves`;

	return `Use this GraphQL API to play an existing ${TITLE} session and WIN it.

${TITLE} is a tower-of-mazes puzzle played in fog. You steer a miner one tile at a time (UP, RIGHT, DOWN, LEFT) from START on level 0 (the bottom) to FINISH on level ${top} (the top) of a ${m.levelCount}-level tower, using stairs and portals to change level. There is no program to write: you send moves, and you win the moment you step on FINISH.

Session ID: ${s.id}
GraphQL endpoint: ${origin}/graphql
Playground: ${origin}/playground
Full rules and every field: ${origin}/llms.txt
Spectators can watch live at: ${origin}/watch/${s.id}

GraphQL introspection is usually enabled; use the Playground or __schema/__type if you need to discover fields.

## This map: ${m.name} (${m.width}x${m.height}, ${m.levelCount} ${m.levelCount === 1 ? 'level' : 'levels'}, ${m.difficulty})
${m.description}

## Fog: you only see the level you stand on
The server never reveals other levels. You learn the tower by exploring it: every response to move / bulkMove / reset / session carries \`view\`, the layout of the level the miner is on right now. Keep your own notes about each level you have seen. START is always on level 0 and FINISH always on the top level (${top}), but their exact positions are not given until you see them.

Coordinates: level 0 is the BOTTOM; x = column, y = row; row 0 is the TOP of a level and y grows DOWNWARD.
UP = y-1, RIGHT = x+1, DOWN = y+1, LEFT = x-1 (always on the same level; levels change only by stairs and portals).

The miner is on level ${s.state.pos.level} at (${s.state.pos.x},${s.state.pos.y}) right now. This is level ${v.level} as seen when the prompt was copied:
${layoutBlock(v.rows)}

Glyphs: '.' void (blocked), 'o' stone (walkable), 'S' start, 'F' finish, 'U' stairs up, 'D' stairs down, '1'-'9' portal holes.
Portals on this level (each digit appears exactly twice in the whole tower):
${portalLines(v)}

Limits: at most ${m.maxMoves} counted moves. Par (shortest route) = ${par}.

## Rules that matter
- Stepping onto a stone, S or F just moves you. Void tiles and the edge of a level block the move: nothing happens and it is NOT counted (blocked: true, via NONE), so probing is free.
- Stairs come in pairs at the same x,y: a 'U' on level l has a 'D' on level l+1. Stepping on 'U' lifts you to level+1 and you land ON that paired 'D'; stepping on 'D' drops you to level-1 and you land on the paired 'U'. Arriving does not trigger the stairs you land on (same as portals): step off and back on to use them. So wherever you can go up you can come back down from the same x,y. Effects never chain.
- Portals: stepping on a hole teleports you to its twin (possibly on another level, which you cannot see until you get there). Arriving does not trigger the twin; step off and back on to teleport again.
- Every successful move counts. If moves reach ${m.maxMoves} without finishing you lose (MOVE_LIMIT). Winning on the last allowed move still wins.
- A finished game (WON or LOST) refuses moves until reset (SESSION_TERMINAL). reset keeps attempts and bestMoves.

## How to win
1. Explore: move, read \`view\` after each move (or after a short bulkMove), and build a map of every level in your notes. Probing walls is free, so bump around to learn a level.
2. Move one step at a time or in short bursts:
mutation { move(sessionID: "${s.id}", direction: UP) { view { level rows portals { glyph a { x y } b { x y } } } state { status moves pos { level x y } } lastEvent { blocked via } } }
mutation { bulkMove(sessionID: "${s.id}", directions: [UP, RIGHT, RIGHT]) { view { level rows } state { status moves pos { level x y } } } }
3. Read the session any time:
query { session(id: "${s.id}") { attempts bestMoves view { level levelCount rows } state { status lossReason moves pos { level x y } } } }
4. Once you know a route, test it for free: simulate replays directions from a fresh start without touching the session:
query { simulate(mapID: "${m.id}", moves: [UP, RIGHT, DOWN, LEFT], includeEvents: false) { status lossReason moves finalState { pos { level x y } moves status } } }
5. If the status is LOST (or you want a better score), call reset, replay the best route you know and shorten it. Do not stop until state.status is WON, then report the winning direction list and its move count.

Other mutations: reset, renameSession, deleteSession. Do not try to guess other sessions' ids and do not delete this one.`;
}

/**
 * Generic prompt (no session yet): the agent reads llms.txt, creates its own session on a map it picks and wins it.
 */
export function buildGenericPrompt(origin: string): string {
	return `Play ${TITLE}, a tower-of-mazes puzzle played in fog, through its GraphQL API and win it.

You steer a miner one tile at a time (UP, RIGHT, DOWN, LEFT) from START (level 0, the bottom) to FINISH (the top level) of a tower. You only ever see the level you stand on, so you map the tower by exploring it. Stairs lift or drop you one level, numbered portal holes teleport you to their twin, and blocked moves are free. You win when you step on FINISH, before the move limit runs out.

1. Read ${origin}/llms.txt first: it has the complete rules, the coordinate convention (level 0 is the bottom, row 0 is the top, y grows downward) and working examples for every step.
2. GraphQL endpoint: ${origin}/graphql (Playground: ${origin}/playground).
3. List the maps: query { maps { id name difficulty levelCount preview maxMoves parMoves } } and pick one (start with an easy one). Only level 0 (\`preview\`) is public.
4. Create your own session: mutation { createSession(mapID: "MAP_ID", displayName: "my agent") { id view { level rows portals { glyph a { x y } b { x y } } } } }
5. Explore with move / bulkMove, reading \`view\` after each move and keeping notes on every level; once you know a route, test it for free with the simulate query. If the game is LOST, reset and try again with what you learned.
6. Tell me the session id so I can watch it live at ${origin}/watch/SESSION_ID, and report the winning direction list and its move count.`;
}
