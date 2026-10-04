<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { getContextClient } from '@urql/svelte';
	import { unwrap } from '$lib/api';
	import MapCard from '$lib/components/MapCard.svelte';
	import { GAME, SITE, pageTitle } from '$lib/brand';
	import MiniGrid from '$lib/components/MiniGrid.svelte';
	import { sortMapsByDifficulty } from '$lib/maps';
	import { describeError, type DescribedError } from '$lib/errors';
	import { CREATE_SESSION_MUTATION, MAPS_QUERY, SESSIONS_QUERY } from '$lib/queries';
	import { statusText } from '$lib/sessions';
	import type { GameMap, Session, SessionSummary } from '$lib/types';

	const client = getContextClient();
	const NAME_KEY = 'drbrain-mazes.displayName';

	let maps = $state<GameMap[]>([]);
	let sessions = $state<SessionSummary[]>([]);
	let loading = $state(true);
	let creating = $state(false);
	let error = $state<DescribedError | null>(null);
	const starter = $derived(maps[0] ?? null);
	let displayName = $state(typeof localStorage !== 'undefined' ? (localStorage.getItem(NAME_KEY) ?? '') : '');

	onMount(async () => {
		try {
			const [m, s] = await Promise.all([
				unwrap<{ maps: GameMap[] }>(client.query(MAPS_QUERY, {})),
				unwrap<{ sessions: SessionSummary[] }>(client.query(SESSIONS_QUERY, { limit: 12 }))
			]);
			maps = sortMapsByDifficulty(m.maps);
			sessions = s.sessions;
		} catch (e) {
			error = describeError(e);
		} finally {
			loading = false;
		}
	});

	async function play(map: GameMap) {
		creating = true;
		error = null;
		try {
			const name = displayName.trim();
			localStorage.setItem(NAME_KEY, name);
			const data = await unwrap<{ createSession: Session }>(
				client.mutation(CREATE_SESSION_MUTATION, { mapID: map.id, displayName: name || null })
			);
			await goto(`/play/${data.createSession.id}`);
		} catch (e) {
			error = describeError(e);
		} finally {
			creating = false;
		}
	}
</script>

<svelte:head><title>{pageTitle('Pick a map')}</title></svelte:head>

<div class="border-b border-indigo-200 bg-white">
	<div class="mx-auto max-w-6xl px-4 sm:px-6 py-10 lg:py-12 lg:grid lg:grid-cols-[1fr_340px] lg:gap-12 lg:items-start">
		<div>
			<p class="text-xs font-bold uppercase tracking-widest text-indigo-600">🧠 {SITE} · {GAME}</p>
			<h1 class="mt-3 text-4xl lg:text-5xl font-light leading-tight tracking-tight text-indigo-950">
				Climb the tower.<br />Find the way out.<br />Mind the portals.
			</h1>
			<p class="mt-5 max-w-2xl text-lg leading-relaxed text-slate-600">
				{GAME} is the second game in the {SITE} collection: a small puzzle about finding a route. A tower of stone floors wrapped in fog: you only ever see the floor you stand on. You steer the
				miner one tile per move from START at the bottom to the cookie at FINISH at the top, mapping the tower as you go. Stairs lift or drop you a level, numbered portal holes teleport you to their twin, and bumping into a
				void costs nothing — but the move limit is real, and par is the shortest route.
				People play with the keyboard, the d-pad or a click; AI agents play through a GraphQL API, and anyone can watch a game live.
				<a href="/learn" class="font-medium text-indigo-700 underline">Learn how it works</a>
			</p>
			<div class="mt-6 flex flex-wrap gap-3">
				<button
					type="button"
					class="rounded-full bg-indigo-600 px-6 py-3 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
					disabled={!starter || creating}
					onclick={() => starter && play(starter)}
					data-testid="play-easiest">{creating ? 'Creating…' : starter ? `Start with “${starter.name}”` : 'Loading…'}</button
				>
				<a href="#maps" class="rounded-full border border-indigo-300 bg-white px-6 py-3 text-sm font-semibold text-indigo-900 hover:bg-indigo-50">Browse all maps</a>
				<a href="/sessions" class="rounded-full border border-indigo-300 bg-white px-6 py-3 text-sm font-semibold text-indigo-900 hover:bg-indigo-50">Watch sessions</a>
			</div>
		</div>
		{#if starter}
			<aside class="mt-10 lg:mt-0 rounded-2xl border border-indigo-200 bg-indigo-50/60 p-5" aria-label="Example map">
				<p class="text-xs font-semibold uppercase tracking-wide text-indigo-900">Example map</p>
				<p class="mt-1 text-sm font-medium text-slate-800">{starter.name}</p>
				<div class="mt-3 flex justify-center pb-2"><MiniGrid rows={starter.preview} levelCount={starter.levelCount} label="Preview of {starter.name}'s first level" /></div>
				<p class="mt-3 text-xs text-slate-600">
					<span class="inline-block h-3 w-3 rounded-sm bg-indigo-600 align-middle"></span> start ·
					<span class="inline-block h-3 w-3 rounded-sm bg-amber-400 align-middle"></span> finish ·
					<span class="text-blue-600" aria-hidden="true">▲</span> stairs up ·
					<span class="text-rose-600" aria-hidden="true">▼</span> stairs down · ringed digits are portals
				</p>
				<p class="mt-1 text-xs text-slate-600">{starter.levelCount} {starter.levelCount === 1 ? 'level' : 'levels'} · {starter.width}×{starter.height}{starter.parMoves !== null ? ` · par ${starter.parMoves}` : ''} · at most {starter.maxMoves} moves</p>
			</aside>
		{/if}
	</div>
</div>

<div class="mx-auto max-w-6xl px-4 sm:px-6 py-10 space-y-10">
	<section aria-label="What you will do" class="grid gap-4 sm:grid-cols-3">
		<div class="rounded-2xl border border-indigo-200 bg-white p-5">
			<div class="text-2xl" aria-hidden="true">🧭</div>
			<h2 class="mt-2 font-medium text-indigo-950">One tile at a time</h2>
			<p class="mt-1 text-sm leading-relaxed text-slate-600">Move with the arrow keys, WASD, the d-pad or by clicking a neighbouring tile. Bumping into the void is free.</p>
		</div>
		<div class="rounded-2xl border border-indigo-200 bg-white p-5">
			<div class="text-2xl" aria-hidden="true">🌀</div>
			<h2 class="mt-2 font-medium text-indigo-950">Stairs and portals</h2>
			<p class="mt-1 text-sm leading-relaxed text-slate-600">Stairs change level, portal pairs jump across the tower. Only your own floor is visible; the tower map shows which floor you are on and where FINISH is — always on top.</p>
		</div>
		<div class="rounded-2xl border border-indigo-200 bg-white p-5">
			<div class="text-2xl" aria-hidden="true">🤖</div>
			<h2 class="mt-2 font-medium text-indigo-950">Try it with an AI</h2>
			<p class="mt-1 text-sm leading-relaxed text-slate-600">Every session has a “Play with an AI” prompt you can paste into a chat to have an agent steer the miner. <a href="/learn#ai" class="underline text-indigo-700">How agents play</a></p>
		</div>
	</section>

	<section id="maps" class="scroll-mt-6">
		<h2 class="text-2xl font-semibold text-indigo-950">Pick a map</h2>
		<p class="mt-1 text-sm text-slate-700">Maps are ordered from easy to hard. New here? Read <a href="/learn" class="underline text-indigo-700">how it works</a>.</p>
		<div class="mt-4 max-w-sm">
			<label for="display-name" class="block text-sm font-medium text-indigo-900">Your name (optional)</label>
			<input
				id="display-name"
				type="text"
				maxlength="40"
				bind:value={displayName}
				placeholder="Anonymous"
				class="mt-1 w-full rounded-lg border border-indigo-300 bg-white px-3 py-2 text-sm"
			/>
		</div>

		{#if error}
			<p role="alert" class="mt-4 rounded-lg border border-rose-300 bg-rose-50 px-3 py-2 text-sm text-rose-900">
				<strong>{error.title}.</strong> {error.message}
			</p>
		{/if}

		{#if loading}
			<p class="mt-6 text-sm text-slate-600" role="status">Loading maps…</p>
		{:else if maps.length === 0 && !error}
			<p class="mt-6 text-sm text-slate-600">No maps available yet. Create one in the <a href="/editor" class="underline">editor</a>.</p>
		{:else}
			<div class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each maps as map (map.id)}
					<MapCard {map} busy={creating} onplay={play} />
				{/each}
			</div>
		{/if}
	</section>

	<section aria-labelledby="recent-h">
		<div class="flex items-baseline justify-between gap-2">
			<h2 id="recent-h" class="text-xl font-semibold text-indigo-950">Recent sessions</h2>
			<a href="/sessions" class="text-sm text-indigo-700 underline">View all sessions</a>
		</div>
		{#if sessions.length === 0}
			<p class="mt-2 text-sm text-slate-600">Nothing yet — start a map above.</p>
		{:else}
			<ul class="mt-3 divide-y divide-indigo-100 rounded-xl border border-indigo-200 bg-white">
				{#each sessions as s (s.id)}
					<li class="flex flex-wrap items-center justify-between gap-2 px-4 py-3 text-sm">
						<div>
							<span class="font-medium">{s.displayName || 'Anonymous'}</span>
							<span class="text-slate-600"> · {s.map.name}</span>
							<span class="ml-2 rounded-full bg-indigo-50 px-2 py-0.5 text-xs text-indigo-800">{statusText(s)}</span>
						</div>
						<div class="flex items-center gap-4 text-xs text-slate-600">
							<span>{s.state.moves} {s.state.moves === 1 ? 'move' : 'moves'}{s.map.parMoves !== null ? ` · par ${s.map.parMoves}` : ''}</span>
							<a class="font-semibold text-indigo-700 underline" href="/play/{s.id}">Play</a>
							<a class="text-indigo-700 underline" href="/watch/{s.id}">Watch</a>
						</div>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>
