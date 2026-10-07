<script lang="ts">
	import { lossReasonText } from '$lib/errors';
	import type { Session } from '$lib/types';

	let {
		session,
		spectator = false,
		compact = false
	}: {
		session: Pick<Session, 'state' | 'map'>;
		/** Spectator wording ("waiting for the first move") instead of the player's. */
		spectator?: boolean;
		compact?: boolean;
	} = $props();

	const status = $derived(session.state.status);
	const text = $derived.by(() => {
		const moves = session.state.moves;
		switch (status) {
			case 'WON':
				return `Won in ${moves} ${moves === 1 ? 'move' : 'moves'}${session.map.parMoves !== null ? ` (par ${session.map.parMoves})` : ''}`;
			case 'LOST':
				return `Lost: ${lossReasonText(session.state.lossReason)}`;
			case 'PLAYING':
				return 'Playing';
			default:
				return spectator ? 'Ready · waiting for the first move' : 'Ready';
		}
	});
	const tone = $derived(
		status === 'WON'
			? 'bg-emerald-50 text-emerald-900 border-emerald-200'
			: status === 'LOST'
				? 'bg-rose-50 text-rose-900 border-rose-200'
				: status === 'PLAYING'
					? 'bg-indigo-600 text-white border-indigo-600'
					: 'bg-indigo-50 text-indigo-900 border-indigo-200'
	);
</script>

<p class="rounded-lg border font-medium {compact ? 'px-2 py-1 text-xs' : 'px-3 py-2 text-sm'} {tone}" role="status" aria-live="polite" data-testid="status-line">
	{text}
</p>
