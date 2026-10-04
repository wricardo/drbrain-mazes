<script lang="ts">
	import { levelName, towerRows } from '$lib/tower';

	let {
		levelCount,
		current,
		viewing = current,
		known,
		onpeek
	}: {
		levelCount: number;
		/** The level the miner is on (marked with a star). */
		current: number;
		/** The level drawn on the board (highlighted). */
		viewing?: number;
		/**
		 * Levels that may be opened (the miner's own plus remembered ones). Other ledges are
		 * unexplored and disabled. Omit to allow every level (editor).
		 */
		known?: number[];
		/** When set, ledges are buttons that open a level; the miner's own level means "follow". */
		onpeek?: (level: number) => void;
	} = $props();

	const rows = $derived(towerRows(levelCount));
	const ledge = 'flex h-7 w-14 items-center justify-center rounded-md border-2 text-xs font-semibold transition-colors';

	function roleText(r: { isStart: boolean; isFinish: boolean }): string {
		return [r.isFinish ? 'finish' : '', r.isStart ? 'start' : ''].filter(Boolean).join(' and ');
	}
	const isKnown = (level: number) => !known || known.includes(level);
</script>

{#snippet face(level: number, isHere: boolean, explored: boolean)}
	<span aria-hidden="true">{explored ? level + 1 : '?'}</span>
	{#if isHere}<span class="ml-1 text-amber-400" aria-hidden="true" data-testid="tower-star">★</span>{/if}
{/snippet}

<nav aria-label="Tower, {levelCount} {levelCount === 1 ? 'level' : 'levels'}" class="select-none rounded-xl border border-indigo-200 bg-white p-3" data-testid="tower">
	<ol class="flex flex-col">
		{#each rows as r, i (r.level)}
			{@const isHere = r.level === current}
			{@const isViewing = r.level === viewing}
			{@const explored = isKnown(r.level)}
			{@const state = isHere ? ', miner is here' : explored ? '' : ', unexplored'}
			{@const name = `${levelName(r.level)}${roleText(r) ? `, ${roleText(r)}` : ''}${state}`}
			<li>
				{#if i > 0}
					<div class="flex w-14 justify-between px-2.5" aria-hidden="true">
						<span class="h-3 w-1.5 rounded-sm bg-indigo-300"></span>
						<span class="h-3 w-1.5 rounded-sm bg-indigo-300"></span>
					</div>
				{/if}
				<div class="flex items-center gap-2">
					{#if onpeek && explored}
						<button
							type="button"
							aria-label={name}
							aria-current={isViewing ? 'true' : undefined}
							onclick={() => onpeek(r.level)}
							data-testid="tower-level-{r.level}"
							class="{ledge} {isViewing ? 'border-indigo-700 bg-indigo-600 text-white hover:bg-indigo-700' : 'border-indigo-300 bg-indigo-100 text-indigo-900 hover:bg-indigo-200'} cursor-pointer"
						>
							{@render face(r.level, isHere, explored)}
						</button>
					{:else}
						<div
							role="img"
							aria-label={name}
							data-testid="tower-level-{r.level}"
							class="{ledge} {!explored ? 'border-dashed border-indigo-200 bg-white text-indigo-300' : isViewing ? 'border-indigo-700 bg-indigo-600 text-white' : 'border-indigo-300 bg-indigo-100 text-indigo-900'}"
						>
							{@render face(r.level, isHere, explored)}
						</div>
					{/if}
					{#if r.isStart || r.isFinish}
						<span
							class="rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide {r.isFinish
								? 'bg-amber-100 text-amber-900'
								: 'bg-emerald-100 text-emerald-900'}"
							aria-hidden="true">{r.isFinish && r.isStart ? 'Start · Finish' : r.isFinish ? 'Finish' : 'Start'}</span
						>
					{/if}
				</div>
			</li>
		{/each}
	</ol>
</nav>
