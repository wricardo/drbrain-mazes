<script lang="ts">
	import { pageTitle } from '$lib/brand';
	import { onDestroy, untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { getContextClient } from '@urql/svelte';
	import { unwrap } from '$lib/api';
	import AiPromptCard from '$lib/components/AiPromptCard.svelte';
	import Board from '$lib/components/Board.svelte';
	import ConnectionBadge from '$lib/components/ConnectionBadge.svelte';
	import Dpad from '$lib/components/Dpad.svelte';
	import ResultModal from '$lib/components/ResultModal.svelte';
	import ShortcutsDialog from '$lib/components/ShortcutsDialog.svelte';
	import Stats from '$lib/components/Stats.svelte';
	import StatusLine from '$lib/components/StatusLine.svelte';
	import Tower from '$lib/components/Tower.svelte';
	import { describeError, lossReasonText, type DescribedError } from '$lib/errors';
	import { MAPS_QUERY, MOVE_MUTATION, RESET_MUTATION } from '$lib/queries';
	import { nextMap } from '$lib/maps';
	import { startSession } from '$lib/start';
	import { buildSessionPrompt } from '$lib/prompt';
	import { isTypingTarget, keyToAction } from '$lib/shortcuts';
	import { createSessionStore } from '$lib/stores/session';
	import { createSessionSource } from '$lib/stores/source';
	import { knownLevels, rememberView, shownLevel, type SeenLevels } from '$lib/fog';
	import { describeEvent, levelName } from '$lib/tower';
	import { createReplay, prefersReducedMotion, replayFrame } from '$lib/replay';
	import type { Direction, GameMap, Session } from '$lib/types';

	const sessionId = $page.params.id ?? '';
	const client = getContextClient();
	const store = createSessionStore(createSessionSource(client), sessionId);
	onDestroy(() => store.destroy());

	const origin = typeof window !== 'undefined' ? window.location.origin : '';
	const live = $derived($store.session);
	const connection = $derived($store.connection);

	let peek = $state<number | null>(null);
	let actionError = $state<DescribedError | null>(null);
	let dismissedAttempts = $state<number | null>(null);
	let showHelp = $state(false);
	let initialised = false;

	$effect(() => {
		const s = live;
		if (!s || initialised) return;
		initialised = true;
		// Do not pop the result modal for an outcome that was already there on load.
		if (s.state.status === 'WON' || s.state.status === 'LOST') dismissedAttempts = s.attempts;
	});

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
	// Input follows the server, not the replay: a finished game takes no more moves.
	const liveTerminal = $derived(live?.state.status === 'WON' || live?.state.status === 'LOST');
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

	// "Next map" in the result modal: the map after this one in the shared order, loaded on first need.
	let maps = $state<GameMap[] | null>(null);
	let nextBusy = $state(false);
	const next = $derived(maps && session ? nextMap(maps, session.mapId) : null);
	$effect(() => {
		if (!showModal || maps) return;
		maps = [];
		unwrap<{ maps: GameMap[] }>(client.query(MAPS_QUERY, {}))
			.then((d) => (maps = d.maps))
			// No "Next map" button this time; "All maps" still works.
			.catch(() => undefined);
	});

	async function playNext() {
		if (!next || !session) return;
		nextBusy = true;
		try {
			await goto(`/play/${await startSession(client, next.id, session.displayName)}`);
		} catch (e) {
			actionError = describeError(e);
			dismissedAttempts = session.attempts;
		} finally {
			nextBusy = false;
		}
	}

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
		if (!live || liveTerminal) return;
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
		if (showModal || showHelp || e.defaultPrevented || isTypingTarget(e.target)) return;
		const action = keyToAction(e);
		if (!action) return;
		e.preventDefault();
		if (action.type === 'move') move(action.direction);
		else if (action.type === 'help') showHelp = true;
		else reset();
	}

	function onpeek(l: number) {
		peek = l === playerLevel ? null : l;
	}
</script>

<svelte:head><title>{pageTitle(session?.map.name)}</title></svelte:head>
<svelte:window onkeydown={onKeyDown} />

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
				<h1 class="text-2xl font-semibold text-indigo-950" title={session.map.description}>{session.map.name}</h1>
				<p class="text-sm text-slate-600">
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

		<div class="mt-4 grid gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
			<section aria-label="Board" class="space-y-2">
				<div class="flex flex-wrap items-center justify-between gap-2 text-sm">
					<h2 class="font-semibold text-indigo-950" data-testid="level-title">
						{levelName(level)} <span class="font-normal text-slate-600">of {session.map.levelCount}</span>
						<span class="ml-1 text-xs font-normal text-slate-500" title="AI agents and the API count levels from 0">· API level {level}</span>
						{#if shown?.remembered} <span class="ml-1 rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-900" data-testid="remembered">remembered</span>{/if}
					</h2>
					{#if level !== playerLevel}
						<button type="button" class="rounded-full border border-indigo-300 bg-white px-3 py-1 text-xs font-medium text-indigo-900 hover:bg-indigo-50" onclick={() => (peek = null)} data-testid="back-to-miner"
							>Back to the miner (level {playerLevel + 1})</button
						>
					{/if}
				</div>
				<div class="flex flex-col items-start gap-4 sm:flex-row">
					<div class="w-full min-w-0 flex-1">
					<Board
						view={shown?.view ?? session.view}
						remembered={shown?.remembered ?? false}
						game={session.state}
						lastEvent={session.lastEvent}
						bumpKey={frame?.replaying ? 0 : session.seq}
						facing={frame?.facing}
						onmove={liveTerminal ? undefined : move}
					/>
					</div>
					<Tower levelCount={session.map.levelCount} current={playerLevel} viewing={level} {known} {onpeek} />
				</div>
				<p class="min-h-5 text-xs text-slate-600" data-testid="last-move">{lastMove}</p>
				<p class="sr-only" role="status" aria-live="polite">{lastMove}</p>
			</section>

			<section aria-label="Game" class="space-y-4">
				<StatusLine {session} />
				<p class="sr-only" role="status" aria-live="polite" data-testid="result-live">{resultText}</p>
				<Stats {session} />
				<div class="rounded-xl border border-indigo-200 bg-white p-3" data-testid="controls">
					<h2 class="text-xs font-semibold uppercase tracking-widest text-indigo-900">Controls</h2>
					<div class="mt-2 flex flex-col items-center gap-3">
						<Dpad onmove={move} disabled={liveTerminal} />
						<div class="flex w-full items-center gap-3 border-t border-indigo-100 pt-3">
							<button
								type="button"
								class="rounded-lg border border-indigo-300 px-4 py-2 text-sm font-semibold text-indigo-900 hover:bg-indigo-50"
								onclick={reset}
								data-testid="reset">Reset</button
							>
							<p class="text-xs text-slate-600">Back to the start; attempts and best are kept.</p>
						</div>
					</div>
				</div>
				<div class="rounded-xl border border-indigo-200 bg-white p-3" data-testid="help">
					<h2 class="text-xs font-semibold uppercase tracking-widest text-indigo-900">Help</h2>
					<ul class="mt-2 list-disc space-y-1 pl-5 text-xs text-slate-700">
						<li><kbd class="rounded border border-indigo-200 bg-white px-1">← ↑ ↓ →</kbd> or <kbd class="rounded border border-indigo-200 bg-white px-1">W A S D</kbd> move one tile</li>
						<li><kbd class="rounded border border-indigo-200 bg-white px-1">R</kbd> resets to the start</li>
						<li>Click a tile next to the miner to step onto it; blocked moves are free</li>
						<li>Click a visited level in the tower to look at it again</li>
						<li>Press <kbd class="rounded border border-indigo-200 bg-white px-1">?</kbd> for shortcuts</li>
					</ul>
					<a href="/learn" class="mt-2 inline-block text-xs font-medium text-indigo-700 underline">All rules →</a>
				</div>
				<AiPromptCard
					prompt={buildSessionPrompt(origin, session)}
					blurb="Copy this prompt into an AI chat and it will steer the miner to the finish in this session."
				/>
			</section>
		</div>

		{#if showModal}
			<ResultModal
				{status}
				reason={session.state.lossReason}
				moves={session.state.moves}
				maxMoves={session.map.maxMoves}
				par={session.map.parMoves}
				best={session.bestMoves}
				nextName={next?.name ?? null}
				{nextBusy}
				onreset={reset}
				onnext={playNext}
				onclose={() => (dismissedAttempts = session.attempts)}
			/>
		{/if}
		{#if showHelp}
			<ShortcutsDialog onclose={() => (showHelp = false)} />
		{/if}
	{/if}
</div>
