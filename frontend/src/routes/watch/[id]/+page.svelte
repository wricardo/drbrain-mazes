<script lang="ts">
	import { pageTitle } from '$lib/brand';
	import { onDestroy, untrack } from 'svelte';
	import { page } from '$app/stores';
	import { getContextClient } from '@urql/svelte';
	import AiPromptCard from '$lib/components/AiPromptCard.svelte';
	import Board from '$lib/components/Board.svelte';
	import ConnectionBadge from '$lib/components/ConnectionBadge.svelte';
	import Stats from '$lib/components/Stats.svelte';
	import Tower from '$lib/components/Tower.svelte';
	import { lossReasonText } from '$lib/errors';
	import { buildSessionPrompt } from '$lib/prompt';
	import { createSessionStore } from '$lib/stores/session';
	import { createSessionSource } from '$lib/stores/source';
	import { knownLevels, rememberView, shownLevel, type SeenLevels } from '$lib/fog';
	import { describeEvent, levelName } from '$lib/tower';

	const sessionId = $page.params.id ?? '';
	const store = createSessionStore(createSessionSource(getContextClient()), sessionId);
	onDestroy(() => store.destroy());

	const origin = typeof window !== 'undefined' ? window.location.origin : '';
	const session = $derived($store.session);
	const status = $derived(session?.state.status ?? 'READY');

	let peek = $state<number | null>(null);
	// Fog: only the miner's level is visible; levels already seen stay peekable (read-only, remembered).
	let seen = $state<SeenLevels>({});
	$effect(() => {
		const v = session?.view;
		if (v) seen = rememberView(untrack(() => seen), v);
	});

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
	const label = $derived(
		!session
			? ''
			: status === 'WON'
				? `Won in ${session.state.moves} moves`
				: status === 'LOST'
					? `Lost: ${lossReasonText(session.state.lossReason)}`
					: status === 'READY'
						? 'Waiting for the first move'
						: 'Playing'
	);
</script>

<svelte:head><title>{pageTitle(session ? `Watching ${session.map.name}` : 'Watch')}</title></svelte:head>

<div class="mx-auto max-w-6xl px-4 sm:px-6 py-6">
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
				<h1 class="text-xl font-semibold text-indigo-950">Watching: {session.map.name}</h1>
				<p class="text-xs text-slate-600">{session.displayName || 'Anonymous'} · read-only spectator view</p>
			</div>
			<ConnectionBadge state={$store.connection} />
		</div>
		{#if $store.error && $store.connection !== 'live'}
			<p role="status" class="mt-2 text-xs text-amber-800">{$store.error.title}: retrying…</p>
		{/if}

		<div class="mt-4 grid gap-6 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)]">
			<section aria-label="Board" class="space-y-3">
				<div class="grid items-start gap-4 sm:grid-cols-[minmax(0,1fr)_auto]">
					<div class="space-y-2">
						<div class="flex items-center justify-between text-sm">
							<h2 class="font-semibold text-indigo-950" data-testid="level-title">
								{levelName(level)} <span class="font-normal text-slate-600">of {session.map.levelCount}</span>{#if shown?.remembered} <span class="ml-1 rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-900" data-testid="remembered">remembered</span>{/if}{#if shown?.remembered} <span class="ml-1 rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-900" data-testid="remembered">remembered</span>{/if}
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
						<Board view={shown?.view ?? session.view} remembered={shown?.remembered ?? false} game={session.state} lastEvent={session.lastEvent} bumpKey={session.seq} />
					</div>
					<Tower levelCount={session.map.levelCount} current={playerLevel} viewing={level} {known} onpeek={(l) => (peek = l === playerLevel ? null : l)} />
				</div>
				<p class="min-h-5 text-xs text-slate-600" data-testid="last-move">{lastMove}</p>
			</section>
			<section aria-label="Game" class="space-y-4">
				<Stats {session} />
				<p
					class="rounded-lg px-3 py-2 text-sm font-medium {status === 'WON'
						? 'bg-emerald-50 text-emerald-900'
						: status === 'LOST'
							? 'bg-rose-50 text-rose-900'
							: 'bg-indigo-50 text-indigo-900'}"
					role="status"
					aria-live="polite"
					data-testid="watch-status"
				>
					{label}
				</p>
				<AiPromptCard
					prompt={buildSessionPrompt(origin, session)}
					blurb="Copy this prompt into an AI chat and it will steer the miner to the finish in session {session.id}."
				/>
				<a href="/" class="inline-block text-sm text-indigo-700 underline">Pick your own map</a>
			</section>
		</div>
	{/if}
</div>
