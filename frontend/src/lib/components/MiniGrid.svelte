<script lang="ts">
	import { portalColor, tileKind } from '$lib/tower';

	let { rows, levelCount = 1, label = '' }: { rows: string[]; levelCount?: number; label?: string } = $props();

	const width = $derived(rows.reduce((w, r) => Math.max(w, r.length), 0));
	const height = $derived(rows.length);
</script>

<div class="relative w-full max-w-[200px]">
	<div
		class="grid gap-px overflow-hidden rounded-md bg-indigo-200 p-px"
		style="aspect-ratio: {width || 1} / {height || 1}; grid-template-columns: repeat({width}, minmax(0, 1fr)); grid-template-rows: repeat({height}, minmax(0, 1fr))"
		role="img"
		aria-label={label || `Map preview of level 1: ${width} by ${height}, ${levelCount} ${levelCount === 1 ? 'level' : 'levels'}`}
	>
		{#each rows as row, y (y)}
			{#each { length: width } as _, x (x)}
				{@const g = row[x] ?? '.'}
				{@const kind = tileKind(g)}
				<div
					class="flex items-center justify-center rounded-sm text-[7px] font-bold leading-none {kind === 'void'
						? 'bg-indigo-50'
						: kind === 'start'
							? 'bg-indigo-600 text-white'
							: kind === 'finish'
								? 'bg-amber-400 text-amber-950'
								: kind === 'up'
									? 'bg-slate-300 text-blue-600'
									: kind === 'down'
										? 'bg-slate-300 text-rose-600'
										: kind === 'portal'
											? 'bg-slate-700 ring-1 ring-inset'
											: 'bg-slate-300'}"
					style={kind === 'portal' ? `color: ${portalColor(g).ring}; --tw-ring-color: ${portalColor(g).ring}` : undefined}
					aria-hidden="true"
				>
					{kind === 'up' ? '▲' : kind === 'down' ? '▼' : kind === 'portal' ? g : ''}
				</div>
			{/each}
		{/each}
	</div>
	{#if levelCount > 1}
		<span class="absolute -bottom-2 right-1 rounded-full bg-indigo-600 px-1.5 py-0.5 text-[10px] font-semibold text-white" data-testid="level-count"
			>{levelCount} levels</span
		>
	{/if}
</div>
