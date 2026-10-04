<script lang="ts">
	import type { Direction } from '$lib/types';
	import { DIRECTION_ARROW } from '$lib/tower';

	let { onmove, disabled = false }: { onmove: (direction: Direction) => void; disabled?: boolean } = $props();

	const cell: Record<Direction, string> = {
		UP: 'col-start-2 row-start-1',
		LEFT: 'col-start-1 row-start-2',
		RIGHT: 'col-start-3 row-start-2',
		DOWN: 'col-start-2 row-start-3'
	};
	const order: Direction[] = ['UP', 'LEFT', 'RIGHT', 'DOWN'];
</script>

<div class="inline-grid grid-cols-3 grid-rows-3 gap-1.5" role="group" aria-label="Move the miner" data-testid="dpad">
	{#each order as d (d)}
		<button
			type="button"
			{disabled}
			aria-label="Move {d.toLowerCase()}"
			data-testid="dpad-{d.toLowerCase()}"
			class="{cell[d]} flex h-14 w-14 items-center justify-center rounded-xl border-2 border-indigo-300 bg-white text-xl text-indigo-900 shadow-sm hover:bg-indigo-50 active:bg-indigo-100 disabled:cursor-not-allowed disabled:opacity-50"
			onclick={() => onmove(d)}
		>
			<span aria-hidden="true">{DIRECTION_ARROW[d]}</span>
		</button>
	{/each}
</div>
