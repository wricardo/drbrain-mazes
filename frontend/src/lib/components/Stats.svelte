<script lang="ts">
	import type { Session } from '$lib/types';

	let { session, compact = false }: { session: Pick<Session, 'state' | 'map' | 'bestMoves' | 'attempts'>; compact?: boolean } = $props();

	const movePct = $derived(Math.min(100, (session.state.moves / Math.max(1, session.map.maxMoves)) * 100));
	const cell = $derived(`rounded-lg border border-indigo-200 bg-white ${compact ? 'px-2 py-1' : 'p-2'}`);
</script>

<!-- Stats row: progress "value / limit", goal (par), best with attempts as a subline so the row stays at three. -->
<dl class="grid grid-cols-3 gap-2 {compact ? 'text-xs' : 'text-sm'}" data-testid="stats">
	<div class={cell}>
		<dt class="text-xs text-slate-600">Moves</dt>
		<dd class="font-semibold tabular-nums"><span data-testid="moves">{session.state.moves}</span> <span class="font-normal text-slate-500">/ <span data-testid="max-moves">{session.map.maxMoves}</span></span></dd>
		<div class="mt-1 h-1.5 overflow-hidden rounded-full bg-indigo-100" aria-hidden="true">
			<div class="h-full {movePct >= 80 ? 'bg-rose-500' : 'bg-indigo-500'}" style="width: {movePct}%"></div>
		</div>
	</div>
	<div class={cell}>
		<dt class="text-xs text-slate-600">Par</dt>
		<dd class="font-semibold tabular-nums" data-testid="par">{session.map.parMoves ?? '—'}</dd>
	</div>
	<div class={cell}>
		<dt class="text-xs text-slate-600">Best</dt>
		<dd class="font-semibold tabular-nums" data-testid="best">{session.bestMoves ?? '—'}</dd>
		<p class="whitespace-nowrap text-xs text-slate-500"><span data-testid="attempts">{session.attempts}</span> {session.attempts === 1 ? 'attempt' : 'attempts'}</p>
	</div>
</dl>
