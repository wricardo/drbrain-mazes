<script lang="ts">
	import { lossReasonText } from '$lib/errors';
	import type { LossReason, Status } from '$lib/types';

	let {
		status,
		reason,
		moves,
		maxMoves,
		par,
		best,
		nextName = null,
		nextBusy = false,
		onreset,
		onnext,
		onclose
	}: {
		status: Status;
		reason: LossReason | null;
		moves: number;
		maxMoves: number;
		par: number | null;
		best: number | null;
		/** Name of the next map in the shared order; null hides "Next map" (last map or not loaded). */
		nextName?: string | null;
		nextBusy?: boolean;
		onreset: () => void;
		onnext?: () => void;
		onclose: () => void;
	} = $props();

	let primary: HTMLButtonElement | undefined = $state();
	$effect(() => {
		primary?.focus();
	});

	function onKeyDown(e: KeyboardEvent) {
		if (e.key === 'Escape') onclose();
	}

	const btn = 'rounded-lg px-4 py-2 text-sm font-semibold';
</script>

<svelte:window onkeydown={onKeyDown} />

<div class="fixed inset-0 z-40 flex items-center justify-center bg-indigo-950/50 p-4">
	<div
		role="dialog"
		aria-modal="true"
		aria-labelledby="result-title"
		class="relative w-full max-w-md rounded-2xl bg-white p-6 shadow-xl text-center"
		data-testid="result-modal"
	>
		<button type="button" class="absolute right-3 top-2 rounded-full px-2 text-2xl leading-none text-slate-400 hover:bg-indigo-50 hover:text-slate-700" aria-label="Close" onclick={onclose}
			>×</button
		>
		<h2 id="result-title" class="text-2xl font-bold {status === 'WON' ? 'text-emerald-700' : 'text-rose-700'}">
			{status === 'WON' ? 'You made it!' : 'Out of moves'}
		</h2>
		<p class="mt-2 text-sm text-slate-700">
			{#if status === 'WON'}
				You reached the finish{#if par !== null && moves === par}{' '}with a perfect run{/if}.
			{:else}
				{lossReasonText(reason)}
			{/if}
		</p>
		<dl class="mt-4 grid grid-cols-3 gap-2 text-sm" data-testid="result-numbers">
			<div class="rounded-lg bg-indigo-50 p-2">
				<dt class="text-xs text-slate-600">Moves</dt>
				<dd class="font-semibold tabular-nums">{moves} <span class="font-normal text-slate-500">/ {maxMoves}</span></dd>
			</div>
			<div class="rounded-lg bg-indigo-50 p-2">
				<dt class="text-xs text-slate-600">Par</dt>
				<dd class="font-semibold tabular-nums">{par ?? '—'}</dd>
			</div>
			<div class="rounded-lg bg-indigo-50 p-2">
				<dt class="text-xs text-slate-600">Best</dt>
				<dd class="font-semibold tabular-nums">{best ?? '—'}</dd>
			</div>
		</dl>
		<div class="mt-5 flex flex-wrap justify-center gap-2">
			<button bind:this={primary} type="button" class="{btn} bg-indigo-600 text-white hover:bg-indigo-700" onclick={onreset} data-testid="result-retry">Try again</button>
			{#if nextName && onnext}
				<button
					type="button"
					class="{btn} border border-indigo-300 text-indigo-900 hover:bg-indigo-50 disabled:opacity-50"
					disabled={nextBusy}
					title="Next map: {nextName}"
					onclick={onnext}
					data-testid="result-next">{nextBusy ? 'Creating…' : 'Next map'}</button
				>
			{/if}
			<a href="/maps" class="{btn} border border-indigo-300 text-indigo-900 hover:bg-indigo-50" data-testid="result-all-maps">All maps</a>
		</div>
		{#if nextName && onnext}
			<p class="mt-2 text-xs text-slate-500">Next up: {nextName}</p>
		{/if}
	</div>
</div>
