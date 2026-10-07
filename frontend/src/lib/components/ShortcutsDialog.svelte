<script lang="ts">
	import { SHORTCUTS } from '$lib/shortcuts';

	let { onclose }: { onclose: () => void } = $props();

	let closeBtn: HTMLButtonElement | undefined = $state();
	$effect(() => {
		closeBtn?.focus();
	});

	function onKeyDown(e: KeyboardEvent) {
		if (e.key === 'Escape' || e.key === '?') {
			e.preventDefault();
			onclose();
		}
	}
</script>

<svelte:window onkeydown={onKeyDown} />

<div class="fixed inset-0 z-40 flex items-center justify-center bg-indigo-950/40 p-4" role="presentation" onclick={(e) => e.target === e.currentTarget && onclose()}>
	<div role="dialog" aria-modal="true" aria-labelledby="shortcuts-title" class="w-full max-w-sm rounded-2xl bg-white p-5 shadow-xl" data-testid="shortcuts-dialog">
		<div class="flex items-center justify-between">
			<h2 id="shortcuts-title" class="text-lg font-semibold text-indigo-950">Keyboard shortcuts</h2>
			<button bind:this={closeBtn} type="button" class="rounded-full px-2 text-xl leading-none text-slate-500 hover:bg-indigo-50" aria-label="Close" onclick={onclose}>×</button>
		</div>
		<dl class="mt-3 grid grid-cols-[max-content_1fr] gap-x-4 gap-y-2 text-sm">
			{#each SHORTCUTS as [k, d] (k)}
				<dt><kbd class="rounded border border-indigo-200 bg-indigo-50 px-1.5 font-mono text-xs">{k}</kbd></dt>
				<dd class="text-slate-700">{d}</dd>
			{/each}
		</dl>
		<p class="mt-3 text-xs text-slate-600">You can also click a tile next to the miner or use the d-pad.</p>
	</div>
</div>
