# Dr Brain - Mazes

A maze puzzle through a tower of levels, playable by **humans and AI alike**.

You walk a little explorer through a cave. Stone tiles are the floor, black is void. Reach the **FINISH** on the top level, starting from the bottom.

Humans play in the browser with the arrow keys, WASD, the on-screen pad or by clicking a neighbouring tile. AI agents play the same game through a public **GraphQL API**: they read the level they are standing on, move, and see what changes. A human and an AI can even share a session, and anyone can watch it live.

## How it works

You only ever see **the level you are standing on**. The other levels stay dark until you get there, so you have to explore.

| Tile | What it does |
|---|---|
| Stone | Walk on it. |
| Void (black) | Blocked. Walking into it, or off the edge, does nothing and **costs no move**. |
| Blue **up arrow** | Stepping on it lifts you one level, onto the matching red arrow at the same x,y spot. Step off and back on to use it again. |
| Red **down arrow** | Stepping on it drops you one level, onto the matching blue arrow at the same x,y spot. Whenever you can climb, you can come back down from the same spot. |
| Glowing **hole** (digits 1-9) | Stepping on it sends you through the pipe to the hole with the same number, which can be on a different level. You can step off and on again to go back. |
| **START** | Always on the bottom level. |
| **FINISH** | Always on the top level. Reaching it wins. |

Every successful move counts. Each map has a move limit; run out of moves and you lose. Every map also shows its **par**: the shortest possible route. Reset and try again to beat your best.

A small tower diagram next to the board shows how many levels there are and which one you are on.

## Maps

There are 16 maps, from easy to hard:

- **First Steps**, **Ladder**, **Portal Twins**, **Pipe Dream** and **Two Floors** teach walking, stairs and portals.
- **Lost Garden** is one big level and no tricks, just a long way out.
- **Wormholes** is a big maze with three pipes: a pipe can save you a long walk or send you back.
- **Four Floors** and **Tower Maze** are corridor mazes across several levels, with staircases that only take you somewhere you have already been.
- **Portal Climb** mixes stairs with pipes.
- **Deep Tower** and **Grand Tower** are five- and six-level mazes with pipes and staircases everywhere.
- **Miner's Shaft** is a six-level tower whose second level is a patchwork of islands that only the pipes and arrows connect. The shortcut up is a trap.
- **Switchback** is four small floors where the only way up often starts by going down. Twice.
- **False Summit** has stairs by the door that race to the top and leave you one tile from the finish, with no way across.
- **Portal Locks** splits every floor into islands that only pipes connect, and the nearest stairs are rarely the right ones.

Every map can be solved. Each one has been checked against a known solution.

## Playing with AI

There are two ways to get an AI to play.

**1. Copy the prompt from the game (easiest).** Start a game in the browser and click **Play with an AI**. It copies a ready-made prompt with the session and the API calls the AI needs. Paste it into an AI assistant that can make web requests (for example a coding agent). Then open the game's watch page and see if the AI can win.

**2. Point an agent at the API.** The game publishes a guide for AI agents at `/llms.txt`. It has the rules, the endpoints and examples for every call. Give your agent that link and ask it to beat a map.

The API is plain GraphQL at `/graphql`. A game takes three calls:

```graphql
query    { maps { id name difficulty levelCount parMoves maxMoves } }                     # 1. pick a map
mutation { createSession(mapID: "first_steps") { id view { level rows } } }               # 2. start a game
mutation { move(sessionID: "…", direction: RIGHT) { state { status moves } view { level rows } } }   # 3. move
```

Moves are `UP`, `RIGHT`, `DOWN` and `LEFT`. The response after each move shows the level you are on now. Read the session (or subscribe to it) until the status is `WON` or `LOST`.

The server never shows an agent more than the current level, just like a person. Every test of an idea is a real move in a session; `reset` starts over.

Some questions worth exploring:

- Can the AI map out a level it can only see one slice at a time?
- Does it notice that every staircase works both ways, and plan around the matching arrow it lands on?
- How many moves does it need compared to par?

## About

Mazes is part of **Dr Brain**, a collection of browser brain games. It is free to play.

Run it yourself:

```sh
make run          # builds and serves http://localhost:8000
```

Want to hack on it or add a map? See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).
