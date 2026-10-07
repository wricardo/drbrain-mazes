<script lang="ts">
	import { pageTitle } from '$lib/brand';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { getContextClient } from '@urql/svelte';
	import { unwrap } from '$lib/api';
	import MapCard from '$lib/components/MapCard.svelte';
	import { describeError, type DescribedError } from '$lib/errors';
	import { DIFFICULTY_FILTERS, countByDifficulty, filterMaps, sortMapsByDifficulty, type DifficultyFilter } from '$lib/maps';
	import { MAPS_QUERY } from '$lib/queries';
	import { savedName, startSession } from '$lib/start';
	import type { GameMap } from '$lib/types';

	const client = getContextClient();
	let maps = $state<GameMap[]>([]);
	let loading = $state(true);
	let busy = $state(false);
	let error = $state<DescribedError | null>(null);
	let text = $state('');
	let difficulty = $state<DifficultyFilter>('all');

	const counts = $derived(countByDifficulty(maps));
	const shown = $derived(filterMaps(maps, { difficulty, text }));
	const chipLabel: Record<DifficultyFilter, string> = { all: 'All', easy: 'Easy', medium: 'Medium', hard: 'Hard' };

	onMount(async () => {
		try {
			maps = sortMapsByDifficulty((await unwrap<{ maps: GameMap[] }>(client.query(MAPS_QUERY, {}))).maps);
		} catch (e) {
			error = describeError(e);
		} finally {
			loading = false;
		}
	});

	async function play(map: GameMap) {
		busy = true;
		error = null;
		try {
			await goto(`/play/${await startSession(client, map.id, savedName())}`);
		} catch (e) {
			error = describeError(e);
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>{pageTitle('Maps')}</title></svelte:head>

<div class="mx-auto max-w-6xl px-4 sm:px-6 py-8">
	<div class="flex items-center justify-between gap-3">
		<div>
			<h1 class="text-2xl font-semibold text-indigo-950">Maps</h1>
			<p class="mt-1 text-sm text-slate-700">Ordered from easy to hard. Play starts a new session on the map.</p>
		</div>
		<a href="/editor" class="rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700">New map</a>
	</div>

	<div class="mt-5 flex flex-wrap items-center gap-3">
		<label for="map-search" class="sr-only">Search maps</label>
		<input
			id="map-search"
			type="search"
			bind:value={text}
			placeholder="Search name or description"
			class="w-full max-w-xs rounded-lg border border-indigo-300 bg-white px-3 py-2 text-sm"
		/>
		<div class="flex flex-wrap gap-1.5" role="group" aria-label="Difficulty">
			{#each DIFFICULTY_FILTERS as d (d)}
				<button
					type="button"
					aria-pressed={difficulty === d}
					onclick={() => (difficulty = d)}
					data-testid="difficulty-{d}"
					class="rounded-full border px-3 py-1.5 text-sm transition-colors {difficulty === d
						? 'border-indigo-600 bg-indigo-600 text-white'
						: 'border-indigo-200 bg-white text-indigo-900 hover:bg-indigo-50'}">{chipLabel[d]} <span class="opacity-75">({counts[d]})</span></button
				>
			{/each}
		</div>
	</div>

	{#if error}
		<p role="alert" class="mt-4 rounded-lg border border-rose-300 bg-rose-50 px-3 py-2 text-sm text-rose-900">
			<strong>{error.title}.</strong> {error.message}
		</p>
	{/if}
	{#if loading}
		<p class="mt-6 text-sm text-slate-600" role="status">Loading maps…</p>
	{:else if shown.length === 0}
		<p class="mt-6 rounded-xl border border-indigo-200 bg-white px-4 py-6 text-sm text-slate-600">
			{#if maps.length === 0}
				No maps yet.
				<a href="/editor" class="ml-2 inline-block rounded-lg bg-indigo-600 px-3 py-1.5 font-semibold text-white hover:bg-indigo-700">Create one</a>
			{:else}
				No maps match these filters.
				<button
					type="button"
					class="ml-2 rounded-lg bg-indigo-600 px-3 py-1.5 font-semibold text-white hover:bg-indigo-700"
					onclick={() => {
						text = '';
						difficulty = 'all';
					}}>Show all maps</button
				>
			{/if}
		</p>
	{:else}
		<div class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each shown as map (map.id)}
				<MapCard {map} {busy} onplay={play} editHref="/editor?map={encodeURIComponent(map.id)}" />
			{/each}
		</div>
	{/if}
</div>
