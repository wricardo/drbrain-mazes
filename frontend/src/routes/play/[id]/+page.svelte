<script lang="ts">
	import { pageTitle } from '$lib/brand';
	import { onDestroy, untrack } from 'svelte';
	import { page } from '$app/stores';
	import { getContextClient } from '@urql/svelte';
	import { unwrap } from '$lib/api';
	import AiPromptCard from '$lib/components/AiPromptCard.svelte';
	import Board from '$lib/components/Board.svelte';
	import ConnectionBadge from '$lib/components/ConnectionBadge.svelte';
	import Dpad from '$lib/components/Dpad.svelte';
	import ResultModal from '$lib/components/ResultModal.svelte';
	import Stats from '$lib/components/Stats.svelte';
	import Tower from '$lib/components/Tower.svelte';
	import { describeError, lossReasonText, type DescribedError } from '$lib/errors';
	import { MOVE_MUTATION, RESET_MUTATION } from '$lib/queries';
	import { buildSessionPrompt } from '$lib/prompt';
	import { isTypingTarget, keyToAction } from '$lib/shortcuts';
	import { createSessionStore } from '$lib/stores/session';
	import { createSessionSource } from '$lib/stores/source';
	import { knownLevels, rememberView, shownLevel, type SeenLevels } from '$lib/fog';
	import { describeEvent, levelName } from '$lib/tower';
	import type { Direction, Session } from '$lib/types';

	const sessionId = $page.params.id ?? '';
	const client = getContextClient();
	const store = createSessionStore(createSessionSource(client), sessionId);
	onDestroy(() => store.destroy());

	const origin = typeof window !== 'undefined' ? window.location.origin : '';
	const session = $derived($store.session);
	const connection = $derived($store.connection);

	let peek = $state<number | null>(null);
	let actionError = $state<DescribedError | null>(null);
	let dismissedAttempts = $state<number | null>(null);
	let initialised = false;

	$effect(() => {
		const s = session;
		if (!s || initialised) return;
		initialised = true;
		// Do not pop the result modal for an outcome that was already there on load.
		if (s.state.status === 'WON' || s.state.status === 'LOST') dismissedAttempts = s.attempts;
	});

	// Fog: only the miner's level is visible; levels already seen stay peekable (read-only, remembered).
	let seen = $state<SeenLevels>({});
	$effect(() => {
		const v = session?.view;
		if (v) seen = rememberView(untrack(() => seen), v);
	});

	// The board follows the miner across stairs and portals unless the player is peeking.
	let followedLevel = -1;
	$effect(() => {
		const l = session?.state.pos.level;
		if (l === undefined || l === followedLevel) return;
		followedLevel = l;
		peek = null;
	});

	const status = $derived(session?.state.status ?? 'READY');
	const terminal = $derived(status === 'WON' || status === 'LOST');
	const playerLevel = $derived(session?.state.pos.level ?? 0);
	const shown = $derived(session ? shownLevel(session.view, peek, seen) : null);
	const level = $derived(shown?.view.level ?? 0);
	const known = $derived(session ? knownLevels(session.view, seen) : []);
	const showModal = $derived(terminal && !!session && dismissedAttempts !== session.attempts);
	const resultText = $derived(
		!session || !terminal
			? ''
			: status === 'WON'
				? `You reached the finish in ${session.state.moves} moves.`
				: `Game lost. ${lossReasonText(session.state.lossReason)}`
	);
	const lastMove = $derived(describeEvent(session?.lastEvent ?? null));

	async function act(fn: () => Promise<Session | void>) {
		actionError = null;
		try {
			const s = await fn();
			if (s) store.apply(s);
		} catch (e) {
			actionError = describeError(e);
		}
	}

	function move(direction: Direction) {
		if (!session || terminal) return;
		peek = null;
		return act(async () => (await unwrap<{ move: Session }>(client.mutation(MOVE_MUTATION, { sessionID: sessionId, direction }))).move);
	}

	const reset = () =>
		act(async () => {
			const s = (await unwrap<{ reset: Session }>(client.mutation(RESET_MUTATION, { sessionID: sessionId }))).reset;
			dismissedAttempts = s.attempts;
			peek = null;
			return s;
		});

	function onKeyDown(e: KeyboardEvent) {
		if (showModal || e.defaultPrevented || isTypingTarget(e.target)) return;
		const action = keyToAction(e);
		if (!action) return;
		e.preventDefault();
		if (action.type === 'move') move(action.direction);
		else reset();
	}

	function onpeek(l: number) {
		peek = l === playerLevel ? null : l;
	}
</script>

<svelte:head><title>{pageTitle(session?.map.name)}</title></svelte:head>
<svelte:window onkeydown={onKeyDown} />

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
				<h1 class="text-xl font-semibold text-indigo-950">{session.map.name}</h1>
				<p class="text-xs text-slate-600">
					{session.displayName || 'Anonymous'} ·
					<a class="underline" href="/watch/{session.id}">spectator link</a>
				</p>
			</div>
			<ConnectionBadge state={connection} />
		</div>

		{#if actionError}
			<p role="alert" class="mt-3 rounded-lg border border-rose-300 bg-rose-50 px-3 py-2 text-sm text-rose-900" data-testid="action-error">
				<strong>{actionError.title}</strong>{#if actionError.code}<span class="ml-1 font-mono text-xs">[{actionError.code}]</span>{/if}
				{#if actionError.message}— {actionError.message}{/if}
			</p>
		{/if}

		<div class="mt-4 grid gap-6 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)]">
			<section aria-label="Board" class="space-y-3">
				<div class="grid items-start gap-4 sm:grid-cols-[minmax(0,1fr)_auto]">
					<div class="space-y-2">
						<div class="flex items-center justify-between text-sm">
							<h2 class="font-semibold text-indigo-950" data-testid="level-title">{levelName(level)} <span class="font-normal text-slate-600">of {session.map.levelCount}</span>{#if shown?.remembered} <span class="ml-1 rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-900" data-testid="remembered">remembered</span>{/if}</h2>
							{#if level !== playerLevel}
								<button type="button" class="rounded-full border border-indigo-300 bg-white px-3 py-1 text-xs font-medium text-indigo-900 hover:bg-indigo-50" onclick={() => (peek = null)} data-testid="back-to-miner"
									>Back to the miner (level {playerLevel + 1})</button
								>
							{/if}
						</div>
						<Board
							view={shown?.view ?? session.view}
							remembered={shown?.remembered ?? false}
							game={session.state}
							lastEvent={session.lastEvent}
							bumpKey={session.seq}
							onmove={terminal ? undefined : move}
						/>
					</div>
					<div class="flex items-center justify-around gap-6 sm:flex-col sm:items-center sm:justify-start sm:gap-8">
						<Tower levelCount={session.map.levelCount} current={playerLevel} viewing={level} {known} {onpeek} />
						<Dpad onmove={move} disabled={terminal} />
					</div>
				</div>
				<p class="min-h-5 text-xs text-slate-600" data-testid="last-move">{lastMove}</p>
				<p class="sr-only" role="status" aria-live="polite">{lastMove}</p>
			</section>

			<section aria-label="Game" class="space-y-4">
				<Stats {session} />
				<p class="sr-only" role="status" aria-live="polite" data-testid="result-live">{resultText}</p>
				{#if terminal}
					<p
						class="rounded-lg px-3 py-2 text-sm font-medium {status === 'WON' ? 'bg-emerald-50 text-emerald-900' : 'bg-rose-50 text-rose-900'}"
						data-testid="result-banner"
					>
						{resultText}
					</p>
				{/if}
				<div class="rounded-xl border border-indigo-200 bg-white p-3 space-y-2">
					<div class="flex flex-wrap items-center gap-2">
						<button
							type="button"
							class="rounded-lg border border-indigo-300 px-4 py-2 text-sm font-semibold text-indigo-900 hover:bg-indigo-50"
							onclick={reset}
							data-testid="reset">Reset</button
						>
						<p class="text-xs text-slate-600">Back to the start; attempts and best are kept.</p>
					</div>
					<p class="text-xs text-slate-600">
						<kbd class="rounded border border-indigo-200 bg-white px-1">←↑↓→</kbd> or <kbd class="rounded border border-indigo-200 bg-white px-1">W A S D</kbd> move,
						<kbd class="rounded border border-indigo-200 bg-white px-1">R</kbd> resets. You can also click a tile next to the miner or press the d-pad. Blocked moves are free.
						Fog hides every level but the one you stand on; click a level you have already visited in the tower to look at it again.
					</p>
					<p class="text-xs text-slate-600">{session.map.description}</p>
				</div>
				<AiPromptCard
					prompt={buildSessionPrompt(origin, session)}
					blurb="Copy this prompt into an AI chat and it will steer the miner to the finish in session {session.id}."
				/>
			</section>
		</div>

		{#if showModal}
			<ResultModal
				{status}
				reason={session.state.lossReason}
				moves={session.state.moves}
				par={session.map.parMoves}
				onreset={reset}
				onclose={() => (dismissedAttempts = session.attempts)}
			/>
		{/if}
	{/if}
</div>
