import type { SessionSummary } from './types';

export type StatusFilter = 'all' | 'playing' | 'ready' | 'won' | 'lost';

export const STATUS_FILTERS: StatusFilter[] = ['all', 'playing', 'ready', 'won', 'lost'];

/** Human status for a session row. */
export function statusText(s: Pick<SessionSummary, 'state'>): Exclude<StatusFilter, 'all'> {
	switch (s.state.status) {
		case 'WON':
			return 'won';
		case 'LOST':
			return 'lost';
		case 'PLAYING':
			return 'playing';
		default:
			return 'ready';
	}
}

export interface SessionFilters {
	status: StatusFilter;
	/** Case-insensitive substring match on display name (empty = no filter). */
	text: string;
}

/** Applies the client-side filters; keeps the server-provided order. */
export function filterSessions(list: SessionSummary[], f: SessionFilters): SessionSummary[] {
	const needle = f.text.trim().toLowerCase();
	return list.filter((s) => {
		if (f.status !== 'all' && statusText(s) !== f.status) return false;
		if (needle && !(s.displayName || 'anonymous').toLowerCase().includes(needle)) return false;
		return true;
	});
}

/** Counts sessions per status (for filter badges). */
export function countByStatus(list: SessionSummary[]): Record<StatusFilter, number> {
	const out: Record<StatusFilter, number> = { all: list.length, playing: 0, ready: 0, won: 0, lost: 0 };
	for (const s of list) out[statusText(s)]++;
	return out;
}
