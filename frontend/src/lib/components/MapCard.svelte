<script lang="ts">
	import type { GameMap } from '$lib/types';
	import MiniGrid from './MiniGrid.svelte';

	let {
		map,
		busy = false,
		onplay,
		editHref = null
	}: { map: GameMap; busy?: boolean; onplay?: (map: GameMap) => void; editHref?: string | null } = $props();

	const badge: Record<string, string> = {
		easy: 'bg-emerald-100 text-emerald-800',
		medium: 'bg-amber-100 text-amber-800',
		hard: 'bg-rose-100 text-rose-800'
	};
</script>

<article class="flex flex-col gap-3 rounded-xl border border-indigo-200 bg-white p-4 shadow-sm" data-testid="map-card">
	<div class="flex justify-center pb-2"><MiniGrid rows={map.preview} levelCount={map.levelCount} label="Preview of {map.name}, {map.levelCount} {map.levelCount === 1 ? 'level' : 'levels'}" /></div>
	<div>
		<h3 class="font-semibold text-indigo-950">{map.name}</h3>
		<p class="text-sm text-slate-600 line-clamp-3">{map.description}</p>
	</div>
	<div class="flex flex-wrap gap-1.5 text-xs">
		<span class="rounded-full px-2 py-0.5 {badge[map.difficulty] ?? 'bg-slate-100 text-slate-700'}">{map.difficulty}</span>
		<span class="rounded-full bg-indigo-100 text-indigo-800 px-2 py-0.5">{map.levelCount} {map.levelCount === 1 ? 'level' : 'levels'}</span>
		<span class="rounded-full bg-indigo-50 text-indigo-800 px-2 py-0.5">{map.width}×{map.height}</span>
		{#if map.parMoves !== null}
			<span class="rounded-full bg-indigo-50 text-indigo-800 px-2 py-0.5">par {map.parMoves}</span>
		{/if}
		<span class="rounded-full bg-indigo-50 text-indigo-800 px-2 py-0.5">≤ {map.maxMoves} moves</span>
	</div>
	<div class="mt-auto flex gap-2">
		{#if onplay}
			<button
				type="button"
				class="flex-1 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
				disabled={busy}
				onclick={() => onplay?.(map)}>Play</button
			>
		{/if}
		{#if editHref}
			<a href={editHref} class="rounded-lg border border-indigo-300 px-3 py-2 text-sm text-indigo-800 hover:bg-indigo-50">Edit</a>
			<a href="{editHref}&duplicate=1" class="rounded-lg border border-indigo-300 px-3 py-2 text-sm text-indigo-800 hover:bg-indigo-50">Duplicate</a>
		{/if}
	</div>
</article>
