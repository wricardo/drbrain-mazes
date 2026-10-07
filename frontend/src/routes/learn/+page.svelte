<script lang="ts">
	import { GAME, pageTitle } from '$lib/brand';
	import AiPromptCard from '$lib/components/AiPromptCard.svelte';
	import Cookie from '$lib/components/Cookie.svelte';
	import { PORTAL_COLORS } from '$lib/tower';
	import { buildGenericPrompt } from '$lib/prompt';
	import { SHORTCUTS } from '$lib/shortcuts';

	const origin = typeof window !== 'undefined' ? window.location.origin : '';
	const genericPrompt = $derived(buildGenericPrompt(origin));

	const toc: [string, string][] = [
		['#game', 'The game'],
		['#tiles', 'Tiles'],
		['#rules', 'Rules'],
		['#controls', 'Controls'],
		['#ai', 'How an AI plays'],
		['#connect', 'Let an AI play'],
		['#tips', 'Tips']
	];

	const tiles: [string, string, string][] = [
		['o', 'Stone', 'Walkable. Everything that is not void is a stone-like tile you can stand on.'],
		['.', 'Void', 'Empty space: you cannot enter it, and neither can you leave the level at its edge.'],
		['S', 'Start', 'Where the miner begins, always on the bottom level (Level 1 on screen, level 0 in the API). Reset brings you back here.'],
		['F', 'Finish', 'The cookie, always on the top level. Stepping on it wins the game.'],
		['U', 'Stairs up', 'A blue ▲ stamp. Stepping on it lifts you to the same column and row on the level above, where you land on its paired ▼.'],
		['D', 'Stairs down', 'A red ▼ stamp. Stepping on it drops you to the same column and row on the level below, where you land on its paired ▲.'],
		['1–9', 'Portal', 'A coloured ring with its digit. Every digit used appears exactly twice in the tower; stepping on one teleports you to its twin, always on the same level. Only stairs change level.']
	];

	const aiSteps: [string, string][] = [
		['Read the catalog', 'Ask for the map list: size, number of levels, the first level preview, the move limit and the par. Create a session to get the view of the level the miner stands on.'],
		['Explore and keep notes', 'Every response carries the current level view. Record each level, stair and portal you find; blocked moves are free, so bumping around is cheap. Then treat what you know as a graph using level, x and y (y grows downward, level 0 is the bottom).'],
		['Play it', 'move sends one direction; bulkMove sends several and stops early at WON or LOST, and at the first blocked move so a wrong plan cannot run on from the wrong tile (pass stopOnBlocked: false to skip blocked moves instead).'],
		['Read the result', 'Check the status and lossReason. If it lost, reset and try again. The attempts counter and the best move count are kept on the session.']
	];

	const tips: [string, string][] = [
		['Probe for free', 'A blocked move is not counted, so bumping into a wall to see what is there never costs you.'],
		['Arrival is safe', 'Landing on paired stairs or a portal exit never triggers it. Step off and back on to use it again.'],
		['Stairs are two-way', 'Every ▲ has a ▼ at the same column and row one level up (and vice versa), so wherever you can climb you can come back down from the spot you landed on.'],
		['Follow the pipe', 'On a level, a dotted line joins a portal to its twin when both are on it; a “?” badge means the twin is on a level you cannot see yet.'],
		['Use the tower', 'The tower shows every level, START at the bottom, FINISH on top and a “You are here” marker on the miner’s level. Click a level you have already visited to look at it again without moving.'],
		['Beat par', 'Par is the shortest possible route. The result dialog compares your moves with it, and the session keeps your best.'],
		['Mind the budget', 'Reaching the move limit without finishing loses the game; reset and plan a shorter route.'],
		['Effects never chain', 'Landing on stairs or a portal exit never triggers another effect on the same move, so you can plan each hop separately. A stairs tile always triggers when you step on it, so you cannot walk across one like a corridor.']
	];

	const keys: [string, string][] = [...SHORTCUTS, ['Click a tile', 'Step onto a tile next to the miner']];
</script>

<svelte:head><title>{pageTitle('How it works')}</title></svelte:head>

<div class="mx-auto max-w-3xl px-4 sm:px-6 py-8 space-y-10">
	<section id="game" class="scroll-mt-6">
		<h1 class="text-2xl font-semibold text-indigo-950">How {GAME} works</h1>
		<p class="mt-3 text-slate-800">
			You steer the miner through a tower of floors, one tile per move, from START to the cookie
			<svg viewBox="-16 -16 32 32" class="inline h-5 w-5 align-text-bottom" role="img" aria-label="cookie"><Cookie cx={0} cy={0} /></svg>
			at FINISH. The tower is wrapped in <strong>fog</strong>: you only ever see the level you stand on, so you map it by exploring. Stairs and portals are how you get between levels. It is a puzzle about finding a short route:
			the move limit is finite and <em>par</em> tells you the best possible score. People play with the keyboard, the d-pad or clicks; AI agents play through an API, and any game can be watched live.
		</p>
		<nav aria-label="On this page" class="mt-4 flex flex-wrap gap-2 text-sm">
			{#each toc as [href, label]}
				<a {href} class="rounded-full border border-indigo-200 bg-white px-3 py-1 text-indigo-800 hover:bg-indigo-50">{label}</a>
			{/each}
		</nav>
	</section>



	<section id="tiles" class="scroll-mt-6" aria-labelledby="tiles-h">
		<h2 id="tiles-h" class="text-xl font-semibold text-indigo-950">Tiles</h2>
		<ul class="mt-3 space-y-2">
			{#each tiles as [glyph, name, text] (glyph)}
				<li class="flex items-start gap-3 rounded-lg border border-indigo-100 bg-white p-3">
					<span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-md border-2 border-indigo-200 bg-indigo-50 font-bold text-indigo-900" aria-hidden="true">{glyph}</span>
					<div>
						<p class="font-semibold">{name}</p>
						<p class="text-sm text-slate-700">{text}</p>
					</div>
				</li>
			{/each}
		</ul>
		<p class="mt-3 flex flex-wrap items-center gap-2 text-sm text-slate-800">
			Portal colours:
			{#each Object.entries(PORTAL_COLORS) as [digit, c] (digit)}
				<span
					class="inline-flex h-7 w-7 items-center justify-center rounded-full border-[3px] text-xs font-bold"
					style="border-color: {c.ring}; background: {c.hole}; color: {c.ring}"
					title="{digit}: {c.name}">{digit}</span
				>
			{/each}
		</p>
	</section>

	<section id="rules" class="scroll-mt-6" aria-labelledby="rules-h">
		<h2 id="rules-h" class="text-xl font-semibold text-indigo-950">Rules</h2>
		<h3 class="mt-3 font-semibold text-indigo-950">Fog of the tower</h3>
		<ul class="mt-2 list-disc space-y-1 pl-5 text-slate-800">
			<li>The board shows <strong>only the level you stand on</strong>. Nothing about the other levels is sent to your browser — not even to spectators — until the miner gets there.</li>
			<li>The START is always on the <strong>bottom</strong> level and the FINISH on the <strong>top</strong> level, as the tower on the side shows. The map card on the home page previews only the first level.</li>
			<li>This page remembers the levels the miner has already visited. Click one in the tower to look at it again; it is dimmed and labelled <em>remembered</em>, and the miner is not on it. Levels you have not visited yet show a “?”. Reloading the page forgets what was remembered.</li>
		</ul>
		<h3 class="mt-5 font-semibold text-indigo-950">Moves, stairs, portals and limits</h3>
		<ul class="mt-2 list-disc space-y-1 pl-5 text-slate-800">
			<li>
				<strong>Coordinates:</strong> <code>level</code> 0 is the bottom, <code>x</code> is the column and <code>y</code> the row; row 0 is the <em>top</em>, so
				<strong>y grows downward</strong>. Up means <code>y − 1</code>, right <code>x + 1</code>, down <code>y + 1</code>, left <code>x − 1</code>. The board numbers levels from 1 for people (“Level 2 of 4”) and shows the API index next to it (“API level 1”), which is what AI agents and the API use.
			</li>
			<li>Every successful move is counted. A move into void or off the edge is <strong>blocked</strong>: nothing changes and it is <strong>not counted</strong>.</li>
			<li>Taking stairs or a portal is still one move. The tile you stepped on and the tile you land on are both added to your trail; the effect never chains into another one, and arriving on a stairs or portal tile does not trigger it — step off and back on to use it again.</li>
			<li>Arriving by portal does not trigger the twin. Step off and back on to teleport again.</li>
			<li>
				You <strong>win</strong> when you step on FINISH — even on the very last allowed move. If you reach the move limit without finishing you <strong>lose</strong>
				(<code>MOVE_LIMIT</code>). Reset puts you back at the start and keeps your attempts and best score.
			</li>
			<li><strong>Par</strong> is the length of the shortest winning route, computed by the server. Your best result so far is shown next to it.</li>
		</ul>
	</section>

	<section id="controls" class="scroll-mt-6" aria-labelledby="keys-h">
		<h2 id="keys-h" class="text-xl font-semibold text-indigo-950">Controls</h2>
		<p class="mt-2 text-sm text-slate-800">Keyboard play works anywhere on the page except while typing in a text field. On touch screens use the d-pad in the Controls panel. Press <kbd class="rounded border border-indigo-200 bg-white px-1">?</kbd> on the play page for the list of shortcuts.</p>
		<dl class="mt-3 grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1 text-sm">
			{#each keys as [k, d]}
				<dt><kbd class="rounded border border-indigo-200 bg-white px-1.5">{k}</kbd></dt>
				<dd>{d}</dd>
			{/each}
		</dl>
		<p class="mt-2 text-sm text-slate-800">
			The tower beside the board lists every level with a “You are here” marker on the miner’s level and the START (bottom) and FINISH (top) labels. Click a level you have already visited to look at it again (read-only, marked remembered); the board jumps back to the miner on its next move.
		</p>
	</section>

	<section id="ai" class="scroll-mt-6" aria-labelledby="ai-h">
		<h2 id="ai-h" class="text-xl font-semibold text-indigo-950">How an AI plays</h2>
		<ol class="mt-3 flex flex-col gap-3">
			{#each aiSteps as [title, desc], i}
				<li class="flex items-start gap-4 rounded-xl border border-indigo-100 bg-white p-4">
					<span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-indigo-600 text-sm font-medium text-white" aria-hidden="true">{i + 1}</span>
					<div>
						<p class="text-sm font-semibold text-indigo-950">{title}</p>
						<p class="mt-0.5 text-sm leading-relaxed text-slate-700">{desc}</p>
					</div>
				</li>
			{/each}
		</ol>
	</section>

	<section id="connect" class="scroll-mt-6" aria-labelledby="connect-h">
		<h2 id="connect-h" class="text-xl font-semibold text-indigo-950">Let an AI play</h2>
		<p class="mt-3 text-sm leading-relaxed text-slate-800">
			Everything an agent needs is in one page: <a href="/llms.txt" target="_blank" rel="noreferrer" class="font-medium underline">llms.txt</a>
			(rules, connection details, and working GraphQL examples). To hand a <em>specific</em> session to an AI, open it and use the
			<strong>“Play with an AI”</strong> card — the copied prompt includes the session id, the level the miner is on (the only one it may see), the fog rules and the exact calls to make.
			To let an agent pick a map and create its own session, copy this prompt:
		</p>
		<div class="mt-3">
			<AiPromptCard
				prompt={genericPrompt}
				title="Prompt for any AI agent"
				blurb="Paste it into an AI chat that can make HTTP requests; it will read llms.txt, create a session and play to win."
			/>
		</div>
		<p class="mt-3 text-sm"><a href="/llms.txt" target="_blank" rel="noreferrer" class="inline-block rounded-full bg-indigo-600 px-5 py-2 font-semibold text-white hover:bg-indigo-700">Open llms.txt</a></p>
	</section>

	<section id="tips" class="scroll-mt-6" aria-labelledby="tips-h">
		<h2 id="tips-h" class="text-xl font-semibold text-indigo-950">Tips</h2>
		<div class="mt-3 grid gap-3 sm:grid-cols-2">
			{#each tips as [title, desc]}
				<div class="rounded-xl border border-indigo-100 bg-white p-4">
					<p class="text-sm font-semibold text-indigo-950">{title}</p>
					<p class="mt-1 text-sm leading-relaxed text-slate-700">{desc}</p>
				</div>
			{/each}
		</div>
	</section>

	<p><a href="/" class="inline-block rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700">Pick a map</a></p>
</div>
