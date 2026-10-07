<script lang="ts">
	import { page } from '$app/stores';
	import { getContextClient } from '@urql/svelte';
	import { pageTitle } from '$lib/brand';
	import WatchCell from '$lib/components/WatchCell.svelte';

	/** Session ids from ?ids=a,b,c (deduplicated, at most 12 boards). */
	const ids = [...new Set(($page.url.searchParams.get('ids') ?? '').split(',').map((s) => s.trim()).filter(Boolean))].slice(0, 12);
	const client = getContextClient();
</script>

<svelte:head><title>{pageTitle('Watch multiple')}</title></svelte:head>

<div class="mx-auto max-w-7xl px-4 sm:px-6 py-6">
	<div class="flex flex-wrap items-baseline justify-between gap-2">
		<div>
			<h1 class="text-2xl font-semibold text-indigo-950">Watching {ids.length} {ids.length === 1 ? 'session' : 'sessions'}</h1>
			<p class="text-sm text-slate-600">Read-only live boards. Open one for the full spectator view.</p>
		</div>
		<a href="/sessions" class="text-sm text-indigo-700 underline">Change selection</a>
	</div>
	{#if ids.length === 0}
		<p class="mt-6 rounded-xl border border-indigo-200 bg-white px-4 py-6 text-sm text-slate-600" data-testid="multi-empty">
			Pick sessions on the Sessions page.
			<a href="/sessions" class="ml-2 inline-block rounded-lg bg-indigo-600 px-3 py-1.5 font-semibold text-white hover:bg-indigo-700">Go to Sessions</a>
		</p>
	{:else}
		<div class="mt-5 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
			{#each ids as id (id)}
				<WatchCell {id} {client} />
			{/each}
		</div>
	{/if}
</div>
