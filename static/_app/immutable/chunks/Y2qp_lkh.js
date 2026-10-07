import{A as e,I as t,L as n,N as r,Q as i,W as a,Y as o,Z as s,a as c,b as l,d as u,et as d,g as f,j as p,k as m,lt as h,rt as g,tt as _,u as v,z as y}from"./DMzxm-I7.js";import"./xihTtKlq.js";import{r as b}from"./BHZg_cxL.js";var x=r(`<p role="alert" class="mt-2 text-xs text-amber-800">Could not access the clipboard — the prompt is selected below, press Ctrl/Cmd+C.</p>`),S=r(`<div class="rounded-xl border border-indigo-200 bg-white p-4" data-testid="ai-prompt-card"><div class="flex items-start justify-between gap-3"><div class="min-w-0"><h2 class="text-xs font-semibold uppercase tracking-widest text-indigo-900"> </h2> <p class="mt-1 text-sm text-slate-700"> </p></div> <button type="button" data-testid="ai-prompt-copy"> </button></div> <p class="sr-only" role="status" aria-live="polite"> </p> <!> <details class="mt-3"><summary class="cursor-pointer select-none text-xs font-medium text-slate-700 hover:text-slate-900">Show full prompt</summary> <textarea readonly="" rows="14" aria-label="Prompt for an AI" class="mt-2 w-full resize-y rounded-lg border border-indigo-100 bg-slate-50 p-3 font-mono text-xs leading-relaxed text-slate-700"></textarea></details></div>`);function C(t,r){let b=c(r,`title`,3,`Play with an AI`),C=_(!1),w=_(!1),T=_(!1),E=_(null),D;async function O(){d(w,!1);try{await navigator.clipboard.writeText(r.prompt),d(C,!0),clearTimeout(D),D=setTimeout(()=>d(C,!1),2e3)}catch{d(w,!0),d(T,!0),queueMicrotask(()=>y(E)?.select())}}var k=S(),A=o(k),j=o(A),M=o(j),N=s(M,!0),P=i(M,2),F=s(P,!0);h(j);var I=i(j,2),L=s(I,!0);h(A);var R=i(A,2),z=s(R,!0),B=i(R,2),V=e=>{var t=x();p(e,t)};m(B,e=>{y(w)&&e(V)});var H=i(B,2),U=i(o(H),2);g(U),u(U,e=>d(E,e),()=>y(E)),h(H),h(k),a(()=>{e(N,b()),e(F,r.blurb),l(I,1,`shrink-0 rounded-full border px-4 py-2 text-sm font-semibold transition-colors ${y(C)?`border-emerald-300 bg-emerald-50 text-emerald-800`:`border-indigo-300 text-indigo-800 hover:bg-indigo-50`}`),e(L,y(C)?`Copied!`:`Copy prompt`),e(z,y(C)?`Prompt copied to clipboard`:``),f(U,r.prompt)}),n(`click`,I,O),n(`click`,U,e=>e.currentTarget.select()),v(`open`,`toggle`,H,e=>d(T,e),()=>y(T)),p(t,k)}t([`click`]);function w(e){let t=e.reduce((e,t)=>Math.max(e,t.length),0),n=String(Math.max(0,e.length-1)).length;return[` `.repeat(n+3)+Array.from({length:t},(e,t)=>String(t%10)).join(``),...e.map((e,t)=>`y${String(t).padStart(n,` `)}  ${e}`)].join(`
`)}var T=e=>`(${e.x},${e.y})`;function E(e){return e.portals.length===0?`(none on this level)`:e.portals.map(e=>e.b?`${e.glyph}: ${T(e.a)} <-> ${T(e.b)}`:`${e.glyph}: ${T(e.a)} -> twin is on another level (location hidden)`).join(`
`)}function D(e,t){let n=t.map,r=t.view,i=n.levelCount-1,a=n.parMoves===null?`unknown`:`${n.parMoves} moves`;return`Use this GraphQL API to play an existing ${b} session and WIN it.

${b} is a tower-of-mazes puzzle played in fog. You steer a miner one tile at a time (UP, RIGHT, DOWN, LEFT) from START on level 0 (the bottom) to FINISH on level ${i} (the top) of a ${n.levelCount}-level tower, using stairs to change level and portals to hop within a level. There is no program to write: you send moves, and you win the moment you step on FINISH.

Session ID: ${t.id}
GraphQL endpoint: ${e}/graphql
Playground: ${e}/playground
Full rules and every field: ${e}/llms.txt
Spectators can watch live at: ${e}/watch/${t.id}

GraphQL introspection is usually enabled; use the Playground or __schema/__type if you need to discover fields.

## This map: ${n.name} (${n.width}x${n.height}, ${n.levelCount} ${n.levelCount===1?`level`:`levels`}, ${n.difficulty})
${n.description}

## Fog: you only see the level you stand on
The server never reveals other levels. You learn the tower by exploring it: every response to move / bulkMove / reset / session carries \`view\`, the layout of the level the miner is on right now. Keep your own notes about each level you have seen. START is always on level 0 and FINISH always on the top level (${i}), but their exact positions are not given until you see them.

Coordinates: level 0 is the BOTTOM; x = column, y = row; row 0 is the TOP of a level and y grows DOWNWARD.
UP = y-1, RIGHT = x+1, DOWN = y+1, LEFT = x-1 (always on the same level; levels change only by stairs).

The miner is on level ${t.state.pos.level} at (${t.state.pos.x},${t.state.pos.y}) right now. This is level ${r.level} as seen when the prompt was copied:
${w(r.rows)}

Glyphs: '.' void (blocked), 'o' stone (walkable), 'S' start, 'F' finish, 'U' stairs up, 'D' stairs down, '1'-'9' portal holes.
Portals on this level (each digit appears exactly twice, both ends on this level):
${E(r)}

Limits: at most ${n.maxMoves} counted moves. Par (shortest route) = ${a}.

## Rules that matter
- Stepping onto a stone, S or F just moves you. Void tiles and the edge of a level block the move: nothing happens and it is NOT counted (blocked: true, via NONE), so probing is free.
- Stairs come in pairs at the same x,y: a 'U' on level l has a 'D' on level l+1. Stepping on 'U' lifts you to level+1 and you land ON that paired 'D'; stepping on 'D' drops you to level-1 and you land on the paired 'U'. Arriving does not trigger the stairs you land on (same as portals): step off and back on to use them. So wherever you can go up you can come back down from the same x,y. Effects never chain.
- Portals: stepping on a hole teleports you to its twin, always on this same level. Arriving does not trigger the twin; step off and back on to teleport again.
- Every successful move counts. If moves reach ${n.maxMoves} without finishing you lose (MOVE_LIMIT). Winning on the last allowed move still wins.
- A finished game (WON or LOST) refuses moves until reset (SESSION_TERMINAL). reset keeps attempts and bestMoves.

## How to win
1. Explore: move, read \`view\` after each move (or after a short bulkMove), and build a map of every level in your notes. Probing walls is free, so bump around to learn a level.
2. Move one step at a time or in short bursts:
mutation { move(sessionID: "${t.id}", direction: UP) { view { level rows portals { glyph a { x y } b { x y } } } state { status moves pos { level x y } } lastEvent { blocked via } } }
mutation { bulkMove(sessionID: "${t.id}", directions: [UP, RIGHT, RIGHT]) { view { level rows } state { status moves pos { level x y } } lastEvent { blocked direction } } }
bulkMove stops at the first blocked move (or when the game ends), so a misread tile cannot send the rest of the batch from the wrong spot: check lastEvent.blocked and state.pos, then send a corrected plan. Add stopOnBlocked: false to skip blocked moves and carry on.
3. Read the session any time:
query { session(id: "${t.id}") { attempts bestMoves view { level levelCount rows } state { status lossReason moves pos { level x y } } } }
4. If the status is LOST (or you want a better score), call reset, replay the best route you know and shorten it. Do not stop until state.status is WON, then report the winning direction list and its move count.

Other mutations: reset, renameSession, deleteSession. Do not try to guess other sessions' ids and do not delete this one.`}function O(e){return`Play ${b}, a tower-of-mazes puzzle played in fog, through its GraphQL API and win it.

You steer a miner one tile at a time (UP, RIGHT, DOWN, LEFT) from START (level 0, the bottom) to FINISH (the top level) of a tower. You only ever see the level you stand on, so you map the tower by exploring it. Stairs lift or drop you one level, numbered portal holes teleport you to their twin, and blocked moves are free. You win when you step on FINISH, before the move limit runs out.

1. Read ${e}/llms.txt first: it has the complete rules, the coordinate convention (level 0 is the bottom, row 0 is the top, y grows downward) and working examples for every step.
2. GraphQL endpoint: ${e}/graphql (Playground: ${e}/playground).
3. List the maps: query { maps { id name difficulty levelCount preview maxMoves parMoves } } and pick one (start with an easy one). Only level 0 (\`preview\`) is public.
4. Create your own session: mutation { createSession(mapID: "MAP_ID", displayName: "my agent") { id view { level rows portals { glyph a { x y } b { x y } } } } }
5. Explore with move / bulkMove, reading \`view\` after each move and keeping notes on every level. If the game is LOST, reset and try again with what you learned.
6. Tell me the session id so I can watch it live at ${e}/watch/SESSION_ID, and report the winning direction list and its move count.`}export{D as n,C as r,O as t};