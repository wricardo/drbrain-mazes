import type { SessionSummary } from './types';

/** Row status label. "ready" = created but no move yet. */
export type SessionStatus = 'ready' | 'playing' | 'won' | 'lost';

/** Sessions page chips. "playing" covers every unfinished session (ready or playing). */
export type StatusFilter = 'all' | 'playing' | 'won' | 'lost';

export const STATUS_FILTERS: StatusFilter[] = ['all', 'playing', 'won', 'lost'];

export const STATUS_FILTER_LABEL: Record<StatusFilter, string> = { all: 'All', playing: 'Playing', won: 'Won', lost: 'Lost' };

/** Human status for a session row. */
export function statusText(s: Pick<SessionSummary, 'state'>): SessionStatus {
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

/** True once the game is over (WON or LOST): such a session can only be watched or reset. */
export function isFinished(s: Pick<SessionSummary, 'state'>): boolean {
	return s.state.status === 'WON' || s.state.status === 'LOST';
}

function statusGroup(s: Pick<SessionSummary, 'state'>): Exclude<StatusFilter, 'all'> {
	const st = statusText(s);
	return st === 'ready' ? 'playing' : st;
}

export interface SessionFilters {
	status: StatusFilter;
	/** Case-insensitive substring match on display name, session id or map name (empty = no filter). */
	text: string;
}

/** Applies the client-side filters; keeps the given order. */
export function filterSessions(list: SessionSummary[], f: SessionFilters): SessionSummary[] {
	const needle = f.text.trim().toLowerCase();
	return list.filter((s) => {
		if (f.status !== 'all' && statusGroup(s) !== f.status) return false;
		if (!needle) return true;
		return [s.displayName || 'anonymous', s.id, s.map.name].some((v) => v.toLowerCase().includes(needle));
	});
}

/** Counts sessions per status chip. */
export function countByStatus(list: SessionSummary[]): Record<StatusFilter, number> {
	const out: Record<StatusFilter, number> = { all: list.length, playing: 0, won: 0, lost: 0 };
	for (const s of list) out[statusGroup(s)]++;
	return out;
}

/** RECENT and CREATED are server orders; MOVES is applied on the client. */
export type SessionSort = 'RECENT' | 'CREATED' | 'MOVES';

/** Orders by move count (most first) for MOVES; other sorts keep the server order. Does not mutate. */
export function sortSessions(list: SessionSummary[], sort: SessionSort): SessionSummary[] {
	return sort === 'MOVES' ? [...list].sort((a, b) => b.state.moves - a.state.moves) : list;
}

/** Recent-sessions block: drops sessions that never started (0 moves) and keeps at most `max`. */
export function recentStarted(list: SessionSummary[], max = 8): SessionSummary[] {
	return list.filter((s) => s.state.moves > 0).slice(0, max);
}

/** localStorage key holding the id of the last session this browser created (home "Continue" card). */
export const LAST_SESSION_KEY = 'drbrain-mazes.lastSession';

type KeyStore = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>;

function storage(): KeyStore | null {
	try {
		return typeof localStorage === 'undefined' ? null : localStorage;
	} catch {
		return null;
	}
}

/** Records the session this browser just created. */
export function rememberSession(id: string, store: KeyStore | null = storage()): void {
	try {
		store?.setItem(LAST_SESSION_KEY, id);
	} catch {
		// Storage full or blocked: the Continue card is a convenience only.
	}
}

/** The last session id this browser created, if any. */
export function recalledSession(store: KeyStore | null = storage()): string | null {
	try {
		return store?.getItem(LAST_SESSION_KEY) || null;
	} catch {
		return null;
	}
}

/** Forgets the recorded session (it finished or no longer exists). */
export function forgetSession(store: KeyStore | null = storage()): void {
	try {
		store?.removeItem(LAST_SESSION_KEY);
	} catch {
		// ignore
	}
}
