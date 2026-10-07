<script lang="ts">
	import { onDestroy, untrack } from 'svelte';
	import type { Client } from '@urql/svelte';
	import Board from './Board.svelte';
	import ConnectionBadge from './ConnectionBadge.svelte';
	import Stats from './Stats.svelte';
	import StatusLine from './StatusLine.svelte';
	import { createSessionStore } from '$lib/stores/session';
	import { createSessionSource } from '$lib/stores/source';
	import { rememberView, type SeenLevels } from '$lib/fog';
	import { createReplay, prefersReducedMotion, replayFrame } from '$lib/replay';
	import { levelName } from '$lib/tower';

	/** One compact live board on the multi-watch page; same store and replay as /watch/<id>. */
	let { id, client }: { id: string; client: Client } = $props();

	const store = createSessionStore(createSessionSource(untrack(() => client)), untrack(() => id));
	onDestroy(() => store.destroy());
	const live = $derived($store.session);

	let seen = $state<SeenLevels>({});
	$effect(() => {
		const v = live?.view;
		if (v) seen = rememberView(untrack(() => seen), v);
	});

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
</script>

<article class="flex flex-col gap-2 rounded-xl border border-indigo-200 bg-white p-3" data-testid="watch-cell">
	{#if $store.loading}
		<p role="status" class="text-sm text-slate-600">Loading {id}…</p>
	{:else if !session}
		<p role="alert" class="text-sm text-rose-900"><strong>{$store.error?.title ?? 'Session unavailable'}.</strong> <span class="font-mono text-xs">{id}</span></p>
	{:else}
		<div class="flex items-start justify-between gap-2">
			<div class="min-w-0">
				<h2 class="truncate font-semibold text-indigo-950">{session.map.name}</h2>
				<p class="truncate text-xs text-slate-600">{session.displayName || 'Anonymous'} · {levelName(session.state.pos.level)} of {session.map.levelCount}</p>
			</div>
			<ConnectionBadge state={$store.connection} />
		</div>
		<StatusLine {session} spectator compact />
		<Stats {session} compact />
		<div class="flex justify-center" style="--board-max-h: 300px">
			<Board view={session.view} game={session.state} lastEvent={session.lastEvent} bumpKey={frame?.replaying ? 0 : session.seq} facing={frame?.facing} />
		</div>
		<a href="/watch/{session.id}" class="mt-auto text-sm font-medium text-indigo-700 underline">Open full watch page →</a>
	{/if}
</article>
