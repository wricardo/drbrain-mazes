import{$ as e,A as t,F as n,I as r,J as i,M as a,N as o,O as s,R as c,U as l,X as u,Z as d,a as f,ct as p,d as m,et as h,h as g,k as _,m as v,nt as y,u as b,y as x}from"./B_GccA4l.js";import"./xihTtKlq.js";import{r as S}from"./BHZg_cxL.js";var C=a(`<p role="alert" class="mt-2 text-xs text-amber-800">Could not access the clipboard — the prompt is selected below, press Ctrl/Cmd+C.</p>`),w=a(`<div class="rounded-xl border border-indigo-200 bg-white p-4" data-testid="ai-prompt-card"><div class="flex items-start justify-between gap-3"><div class="min-w-0"><h2 class="text-xs font-semibold uppercase tracking-widest text-indigo-900"> </h2> <p class="mt-1 text-sm text-slate-700"> </p></div> <button type="button" data-testid="ai-prompt-copy"> </button></div> <p class="sr-only" role="status" aria-live="polite"> </p> <!> <details class="mt-3"><summary class="cursor-pointer select-none text-xs font-medium text-slate-700 hover:text-slate-900">Show full prompt</summary> <textarea readonly="" rows="14" aria-label="Prompt for an AI" class="mt-2 w-full resize-y rounded-lg border border-indigo-100 bg-slate-50 p-3 font-mono text-xs leading-relaxed text-slate-700"></textarea></details></div>`);function T(n,a){let o=f(a,`title`,3,`Play with an AI`),v=h(!1),S=h(!1),T=h(!1),E=h(null),D;async function O(){e(S,!1);try{await navigator.clipboard.writeText(a.prompt),e(v,!0),clearTimeout(D),D=setTimeout(()=>e(v,!1),2e3)}catch{e(S,!0),e(T,!0),queueMicrotask(()=>c(E)?.select())}}var k=w(),A=i(k),j=i(A),M=i(j),N=u(M,!0),P=d(M,2),F=u(P,!0);p(j);var I=d(j,2),L=u(I,!0);p(A);var R=d(A,2),z=u(R,!0),B=d(R,2),V=e=>{var n=C();t(e,n)};s(B,e=>{c(S)&&e(V)});var H=d(B,2),U=d(i(H),2);y(U),m(U,t=>e(E,t),()=>c(E)),p(H),p(k),l(()=>{_(N,o()),_(F,a.blurb),x(I,1,`shrink-0 rounded-full border px-4 py-2 text-sm font-semibold transition-colors ${c(v)?`border-emerald-300 bg-emerald-50 text-emerald-800`:`border-indigo-300 text-indigo-800 hover:bg-indigo-50`}`),_(L,c(v)?`Copied!`:`Copy prompt`),_(z,c(v)?`Prompt copied to clipboard`:``),g(U,a.prompt)}),r(`click`,I,O),r(`click`,U,e=>e.currentTarget.select()),b(`open`,`toggle`,H,t=>e(T,t),()=>c(T)),t(n,k)}n([`click`]);var E=o(`<g aria-hidden="true"><circle r="15" fill="#d9a066" stroke="#8a5a2b" stroke-width="1.5"></circle><circle cx="-5" cy="-6" r="2.4" fill="#4a2c12"></circle><circle cx="6" cy="-4" r="2.2" fill="#4a2c12"></circle><circle cx="-7" cy="4" r="2" fill="#4a2c12"></circle><circle cx="3" cy="7" r="2.5" fill="#4a2c12"></circle><circle cx="9" cy="5" r="1.6" fill="#4a2c12"></circle><circle cx="-1" cy="-1" r="1.6" fill="#4a2c12"></circle></g>`);function D(e,n){let r=f(n,`r`,3,15);var i=E();l(()=>v(i,`transform`,`translate(${n.cx??``} ${n.cy??``}) scale(${r()/15})`)),t(e,i)}function O(e){let t=e.reduce((e,t)=>Math.max(e,t.length),0),n=String(Math.max(0,e.length-1)).length;return[` `.repeat(n+3)+Array.from({length:t},(e,t)=>String(t%10)).join(``),...e.map((e,t)=>`y${String(t).padStart(n,` `)}  ${e}`)].join(`
`)}var k=e=>`(${e.x},${e.y})`;function A(e){return e.portals.length===0?`(none on this level)`:e.portals.map(e=>e.b?`${e.glyph}: ${k(e.a)} <-> ${k(e.b)}`:`${e.glyph}: ${k(e.a)} -> twin is on another level (location hidden)`).join(`
`)}function j(e,t){let n=t.map,r=t.view,i=n.levelCount-1,a=n.parMoves===null?`unknown`:`${n.parMoves} moves`;return`Use this GraphQL API to play an existing ${S} session and WIN it.

${S} is a tower-of-mazes puzzle played in fog. You steer a miner one tile at a time (UP, RIGHT, DOWN, LEFT) from START on level 0 (the bottom) to FINISH on level ${i} (the top) of a ${n.levelCount}-level tower, using stairs and portals to change level. There is no program to write: you send moves, and you win the moment you step on FINISH.

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
UP = y-1, RIGHT = x+1, DOWN = y+1, LEFT = x-1 (always on the same level; levels change only by stairs and portals).

The miner is on level ${t.state.pos.level} at (${t.state.pos.x},${t.state.pos.y}) right now. This is level ${r.level} as seen when the prompt was copied:
${O(r.rows)}

Glyphs: '.' void (blocked), 'o' stone (walkable), 'S' start, 'F' finish, 'U' stairs up, 'D' stairs down, '1'-'9' portal holes.
Portals on this level (each digit appears exactly twice in the whole tower):
${A(r)}

Limits: at most ${n.maxMoves} counted moves. Par (shortest route) = ${a}.

## Rules that matter
- Stepping onto a stone, S or F just moves you. Void tiles and the edge of a level block the move: nothing happens and it is NOT counted (blocked: true, via NONE), so probing is free.
- Stairs come in pairs at the same x,y: a 'U' on level l has a 'D' on level l+1. Stepping on 'U' lifts you to level+1 and you land ON that paired 'D'; stepping on 'D' drops you to level-1 and you land on the paired 'U'. Arriving does not trigger the stairs you land on (same as portals): step off and back on to use them. So wherever you can go up you can come back down from the same x,y. Effects never chain.
- Portals: stepping on a hole teleports you to its twin (possibly on another level, which you cannot see until you get there). Arriving does not trigger the twin; step off and back on to teleport again.
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
4. Once you know a route, test it for free: simulate replays directions from a fresh start without touching the session:
query { simulate(mapID: "${n.id}", moves: [UP, RIGHT, DOWN, LEFT], includeEvents: false) { status lossReason moves finalState { pos { level x y } moves status } } }
5. If the status is LOST (or you want a better score), call reset, replay the best route you know and shorten it. Do not stop until state.status is WON, then report the winning direction list and its move count.

Other mutations: reset, renameSession, deleteSession. Do not try to guess other sessions' ids and do not delete this one.`}function M(e){return`Play ${S}, a tower-of-mazes puzzle played in fog, through its GraphQL API and win it.

You steer a miner one tile at a time (UP, RIGHT, DOWN, LEFT) from START (level 0, the bottom) to FINISH (the top level) of a tower. You only ever see the level you stand on, so you map the tower by exploring it. Stairs lift or drop you one level, numbered portal holes teleport you to their twin, and blocked moves are free. You win when you step on FINISH, before the move limit runs out.

1. Read ${e}/llms.txt first: it has the complete rules, the coordinate convention (level 0 is the bottom, row 0 is the top, y grows downward) and working examples for every step.
2. GraphQL endpoint: ${e}/graphql (Playground: ${e}/playground).
3. List the maps: query { maps { id name difficulty levelCount preview maxMoves parMoves } } and pick one (start with an easy one). Only level 0 (\`preview\`) is public.
4. Create your own session: mutation { createSession(mapID: "MAP_ID", displayName: "my agent") { id view { level rows portals { glyph a { x y } b { x y } } } } }
5. Explore with move / bulkMove, reading \`view\` after each move and keeping notes on every level; once you know a route, test it for free with the simulate query. If the game is LOST, reset and try again with what you learned.
6. Tell me the session id so I can watch it live at ${e}/watch/SESSION_ID, and report the winning direction list and its move count.`}export{T as i,j as n,D as r,M as t};