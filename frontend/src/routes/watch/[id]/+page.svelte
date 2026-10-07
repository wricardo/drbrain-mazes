<script lang="ts">
	import { pageTitle } from '$lib/brand';
	import { onDestroy, untrack } from 'svelte';
	import { page } from '$app/stores';
	import { getContextClient } from '@urql/svelte';
	import AiPromptCard from '$lib/components/AiPromptCard.svelte';
	import Board from '$lib/components/Board.svelte';
	import ConnectionBadge from '$lib/components/ConnectionBadge.svelte';
	import Stats from '$lib/components/Stats.svelte';
	import StatusLine from '$lib/components/StatusLine.svelte';
	import Tower from '$lib/components/Tower.svelte';
	import { buildSessionPrompt } from '$lib/prompt';
	import { createSessionStore } from '$lib/stores/session';
	import { createSessionSource } from '$lib/stores/source';
	import { knownLevels, rememberView, shownLevel, type SeenLevels } from '$lib/fog';
	import { describeEvent, levelName } from '$lib/tower';
	import { createReplay, prefersReducedMotion, replayFrame } from '$lib/replay';

	const sessionId = $page.params.id ?? '';
	const store = createSessionStore(createSessionSource(getContextClient()), sessionId);
	onDestroy(() => store.destroy());

	const origin = typeof window !== 'undefined' ? window.location.origin : '';
	const live = $derived($store.session);

	let peek = $state<number | null>(null);
	// Fog: only the miner's level is visible; levels already seen stay peekable (read-only, remembered).
	let seen = $state<SeenLevels>({});
	$effect(() => {
		const v = live?.view;
		if (v) seen = rememberView(untrack(() => seen), v);
	});

	// Replay: a bulkMove arrives as one snapshot; walk the miner through the new path tile by tile.
	const replay = createReplay({ instant: prefersReducedMotion });
	onDestroy(() => replay.destroy());
	$effect(() => {
		const s = live;
		if (!s) return;
		const liveLevel = s.view.level;
		replay.sync(s.state.path, (p) => p.level === liveLevel || untrack(() => seen)[p.level] !== undefined);
	});
	const frame = $derived(live ? replayFrame(live, $replay, seen) : null);
	const session = $derived(frame?.session ?? null);

	// Follow the miner across stairs and portals: a level change drops any peek.
	let followedLevel = -1;
	$effect(() => {
		const l = session?.state.pos.level;
		if (l === undefined || l === followedLevel) return;
		followedLevel = l;
		peek = null;
	});

	const playerLevel = $derived(session?.state.pos.level ?? 0);
	const shown = $derived(session ? shownLevel(session.view, peek, seen) : null);
	const level = $derived(shown?.view.level ?? 0);
	const known = $derived(session ? knownLevels(session.view, seen) : []);
	const lastMove = $derived(describeEvent(session?.lastEvent ?? null));
</script>

<svelte:head><title>{pageTitle(session ? `Watching ${session.map.name}` : 'Watch')}</title></svelte:head>

<div class="mx-auto max-w-7xl px-4 sm:px-6 py-6">
	{#if $store.loading}
		<p role="status" class="text-sm text-slate-600">Loading session…</p>
	{:else if !session}
		<div role="alert" class="rounded-lg border border-rose-300 bg-rose-50 p-4 text-sm text-rose-900">
			<strong>{$store.error?.title ?? 'Session unavailable'}.</strong>
			{$store.error?.message}
			<a href="/" class="ml-2 underline">Back to maps</a>
		</div>
	{:else}
		<div class="flex flex-wrap items-center justify-between gap-2">
			<div>
				<h1 class="text-2xl font-semibold text-indigo-950">Watching: {session.map.name}</h1>
				<p class="text-sm text-slate-600">{session.displayName || 'Anonymous'} · read-only spectator view</p>
			</div>
			<ConnectionBadge state={$store.connection} />
		</div>
		{#if $store.error && $store.connection !== 'live'}
			<p role="status" class="mt-2 text-xs text-amber-800">{$store.error.title}: retrying…</p>
		{/if}

		<div class="mt-4 grid gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
			<section aria-label="Board" class="space-y-2">
				<div class="flex flex-wrap items-center justify-between gap-2 text-sm">
					<h2 class="font-semibold text-indigo-950" data-testid="level-title">
						{levelName(level)} <span class="font-normal text-slate-600">of {session.map.levelCount}</span>
						<span class="ml-1 text-xs font-normal text-slate-500" title="AI agents and the API count levels from 0">· API level {level}</span>
						{#if shown?.remembered} <span class="ml-1 rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-900" data-testid="remembered">remembered</span>{/if}
					</h2>
					{#if level !== playerLevel}
						<button
							type="button"
							class="rounded-full border border-indigo-300 bg-white px-3 py-1 text-xs font-medium text-indigo-900 hover:bg-indigo-50"
							onclick={() => (peek = null)}
							data-testid="back-to-miner">Follow the miner (level {playerLevel + 1})</button
						>
					{/if}
				</div>
				<div class="flex flex-col items-start gap-4 sm:flex-row">
					<div class="w-full min-w-0 flex-1"><Board view={shown?.view ?? session.view} remembered={shown?.remembered ?? false} game={session.state} lastEvent={session.lastEvent} bumpKey={frame?.replaying ? 0 : session.seq} facing={frame?.facing} /></div>
					<Tower levelCount={session.map.levelCount} current={playerLevel} viewing={level} {known} onpeek={(l) => (peek = l === playerLevel ? null : l)} />
				</div>
				<p class="min-h-5 text-xs text-slate-600" data-testid="last-move">{lastMove}</p>
			</section>
			<section aria-label="Game" class="space-y-4">
				<StatusLine {session} spectator />
				<Stats {session} />
				<div class="flex flex-wrap items-center gap-3">
					<a href="/play/{session.id}" class="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700" data-testid="take-over">Take over</a>
					<a href="/maps" class="text-sm text-indigo-700 underline">Pick your own map</a>
				</div>
				<AiPromptCard
					prompt={buildSessionPrompt(origin, session)}
					blurb="Copy this prompt into an AI chat and it will steer the miner to the finish in this session."
				/>
			</section>
		</div>
	{/if}
</div>
