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
	import { MAPS_QUERY, SESSIONS_QUERY } from '$lib/queries';
	import { createSessionSource } from '$lib/stores/source';
	import { forgetSession, isFinished, recalledSession, recentStarted, statusText } from '$lib/sessions';
	import { savedName, startSession } from '$lib/start';
	import { absoluteTime, relativeTime } from '$lib/time';
	import type { GameMap, Session, SessionSummary } from '$lib/types';

	/** Maps shown on the home page; the rest are one click away on /maps. */
	const HOME_MAPS = 3;

	const client = getContextClient();

	let maps = $state<GameMap[]>([]);
	let sessions = $state<SessionSummary[]>([]);
	let resume = $state<Session | null>(null);
	let loading = $state(true);
	let creating = $state(false);
	let error = $state<DescribedError | null>(null);
	const starter = $derived(maps[0] ?? null);
	const recent = $derived(recentStarted(sessions));
	let displayName = $state(savedName());

	onMount(async () => {
		void loadResume();
		try {
			const [m, s] = await Promise.all([
				unwrap<{ maps: GameMap[] }>(client.query(MAPS_QUERY, {})),
				unwrap<{ sessions: SessionSummary[] }>(client.query(SESSIONS_QUERY, { limit: 40 }, { requestPolicy: 'network-only' }))
			]);
			maps = sortMapsByDifficulty(m.maps);
			sessions = s.sessions;
		} catch (e) {
			error = describeError(e);
		} finally {
			loading = false;
		}
	});

	/** The "Continue" card: the last session this browser created, while it is unfinished. */
	async function loadResume() {
		const id = recalledSession();
		if (!id) return;
		try {
			const s = await createSessionSource(client).fetch(id);
			if (s && !isFinished(s)) resume = s;
			else forgetSession();
		} catch (e) {
			// Gone from the server: forget it. Anything else (offline): no card this time.
			if (describeError(e).code === 'NOT_FOUND') forgetSession();
		}
	}

	async function play(map: GameMap) {
		creating = true;
		error = null;
		try {
			await goto(`/play/${await startSession(client, map.id, displayName)}`);
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
				Steer the miner one tile at a time from START at the bottom of a fog-wrapped tower to the cookie at the top, using stairs and portals. Play with the keyboard, hand the
				session to an AI agent, or watch any game live — <a href="/learn" class="font-medium text-indigo-700 underline">learn how it works</a>.
			</p>
			<div class="mt-6 flex flex-wrap gap-3">
				<button
					type="button"
					class="rounded-full bg-indigo-600 px-6 py-3 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
					disabled={!starter || creating}
					onclick={() => starter && play(starter)}
					data-testid="play-easiest">{creating ? 'Creating…' : starter ? `Start with “${starter.name}”` : 'Loading…'}</button
				>
				<a href="/maps" class="rounded-full border border-indigo-300 bg-white px-6 py-3 text-sm font-semibold text-indigo-900 hover:bg-indigo-50">Browse all maps</a>
				<a href="/sessions" class="rounded-full border border-indigo-300 bg-white px-6 py-3 text-sm font-semibold text-indigo-900 hover:bg-indigo-50">Watch sessions</a>
			</div>
		</div>
		{#if resume}
			<aside class="mt-10 lg:mt-0 rounded-2xl border-2 border-indigo-400 bg-indigo-50 p-5" aria-label="Continue your game" data-testid="continue">
				<p class="text-xs font-semibold uppercase tracking-wide text-indigo-700">Continue where you left off</p>
				<p class="mt-1 text-lg font-semibold text-indigo-950">{resume.map.name}</p>
				<p class="mt-0.5 text-sm text-slate-700">
					{resume.displayName || 'Anonymous'} · {resume.state.moves} / {resume.map.maxMoves} moves ·
					<span title={absoluteTime(resume.lastActionAt)}>{relativeTime(resume.lastActionAt)}</span>
				</p>
				<div class="mt-3 flex justify-center pb-2"><MiniGrid rows={resume.map.preview} levelCount={resume.map.levelCount} label="Preview of {resume.map.name}'s first level" /></div>
				<a href="/play/{resume.id}" class="mt-3 block rounded-lg bg-indigo-600 px-4 py-2 text-center text-sm font-semibold text-white hover:bg-indigo-700">Continue</a>
			</aside>
		{:else if starter}
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
		<div class="flex flex-wrap items-baseline justify-between gap-2">
			<h2 class="text-2xl font-semibold text-indigo-950">Pick a map</h2>
			{#if maps.length > 0}
				<a href="/maps" class="text-sm font-medium text-indigo-700 underline" data-testid="browse-all">Browse all {maps.length} maps →</a>
			{/if}
		</div>
		<p class="mt-1 text-sm text-slate-700">The easiest maps first. New here? Read <a href="/learn" class="underline text-indigo-700">how it works</a>.</p>
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
			<p class="mt-6 rounded-xl border border-indigo-200 bg-white px-4 py-6 text-sm text-slate-600">
				No maps available yet.
				<a href="/editor" class="ml-2 inline-block rounded-lg bg-indigo-600 px-3 py-1.5 font-semibold text-white hover:bg-indigo-700">Create one in the editor</a>
			</p>
		{:else}
			<div class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each maps.slice(0, HOME_MAPS) as map (map.id)}
					<MapCard {map} busy={creating} onplay={play} />
				{/each}
			</div>
			{#if maps.length > HOME_MAPS}
				<p class="mt-4 text-center">
					<a href="/maps" class="inline-block rounded-full border border-indigo-300 bg-white px-5 py-2 text-sm font-semibold text-indigo-900 hover:bg-indigo-50"
						>Browse all {maps.length} maps →</a
					>
				</p>
			{/if}
		{/if}
	</section>

	<section aria-labelledby="recent-h">
		<div class="flex items-baseline justify-between gap-2">
			<h2 id="recent-h" class="text-xl font-semibold text-indigo-950">Recent sessions</h2>
			<a href="/sessions" class="text-sm text-indigo-700 underline">View all sessions</a>
		</div>
		{#if loading}
			<p class="mt-2 text-sm text-slate-600" role="status">Loading sessions…</p>
		{:else if recent.length === 0}
			<p class="mt-3 rounded-xl border border-indigo-200 bg-white px-4 py-6 text-sm text-slate-600">
				Nobody has made a move yet.
				<button
					type="button"
					class="ml-2 rounded-lg bg-indigo-600 px-3 py-1.5 font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
					disabled={!starter || creating}
					onclick={() => starter && play(starter)}>Be the first</button
				>
			</p>
		{:else}
			<ul class="mt-3 divide-y divide-indigo-100 rounded-xl border border-indigo-200 bg-white">
				{#each recent as s (s.id)}
					{@const st = statusText(s)}
					<li class="flex flex-wrap items-center justify-between gap-2 px-4 py-3 text-sm">
						<div>
							<span class="font-medium">{s.displayName || 'Anonymous'}</span>
							<span class="text-slate-600"> · {s.map.name}</span>
							<span class="ml-2 rounded-full bg-indigo-50 px-2 py-0.5 text-xs text-indigo-800">{st}</span>
						</div>
						<div class="flex items-center gap-4 text-xs text-slate-600">
							<span class="tabular-nums">{s.state.moves} / {s.map.maxMoves} moves{s.map.parMoves !== null ? ` · par ${s.map.parMoves}` : ''}</span>
							<span title={absoluteTime(s.lastActionAt)}>{relativeTime(s.lastActionAt)}</span>
							{#if !isFinished(s)}<a class="font-semibold text-indigo-700 underline" href="/play/{s.id}">Play</a>{/if}
							<a class="text-indigo-700 underline" href="/watch/{s.id}">Watch</a>
						</div>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>
