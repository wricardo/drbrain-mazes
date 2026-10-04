<script lang="ts">
	import type { Session } from '$lib/types';

	let { session }: { session: Session } = $props();

	const movePct = $derived(Math.min(100, (session.state.moves / Math.max(1, session.map.maxMoves)) * 100));
</script>

<dl class="grid grid-cols-2 gap-2 text-sm sm:grid-cols-3" data-testid="stats">
	<div class="col-span-2 rounded-lg border border-indigo-200 bg-white p-2 sm:col-span-1">
		<dt class="text-xs text-slate-600">Moves</dt>
		<dd class="font-semibold" data-testid="moves">{session.state.moves}</dd>
		<div class="mt-1 h-1.5 overflow-hidden rounded-full bg-indigo-100" aria-hidden="true">
			<div class="h-full {movePct >= 80 ? 'bg-rose-500' : 'bg-indigo-500'}" style="width: {movePct}%"></div>
		</div>
	</div>
	<div class="rounded-lg border border-indigo-200 bg-white p-2">
		<dt class="text-xs text-slate-600">Par</dt>
		<dd class="font-semibold" data-testid="par">{session.map.parMoves ?? '—'}</dd>
	</div>
	<div class="rounded-lg border border-indigo-200 bg-white p-2">
		<dt class="text-xs text-slate-600">Max moves</dt>
		<dd class="font-semibold" data-testid="max-moves">{session.map.maxMoves}</dd>
	</div>
	<div class="rounded-lg border border-indigo-200 bg-white p-2">
		<dt class="text-xs text-slate-600">Best</dt>
		<dd class="font-semibold" data-testid="best">{session.bestMoves ?? '—'}</dd>
	</div>
	<div class="rounded-lg border border-indigo-200 bg-white p-2">
		<dt class="text-xs text-slate-600">Attempts</dt>
		<dd class="font-semibold" data-testid="attempts">{session.attempts}</dd>
	</div>
</dl>
