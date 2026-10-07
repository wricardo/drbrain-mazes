import { describe, expect, it } from 'vitest';
import {
	LAST_SESSION_KEY,
	countByStatus,
	filterSessions,
	forgetSession,
	isFinished,
	recalledSession,
	recentStarted,
	rememberSession,
	sortSessions,
	statusText
} from './sessions';
import type { SessionSummary } from './types';

function sess(id: string, over: Partial<SessionSummary> & { status?: string; moves?: number } = {}): SessionSummary {
	const { status = 'READY', moves = 0, ...rest } = over;
	return {
		id,
		displayName: '',
		mapId: 'm',
		createdAt: '2026-01-01T00:00:00Z',
		lastActionAt: '2026-01-01T00:00:00Z',
		map: { name: 'M', parMoves: 5, maxMoves: 50 },
		state: { status: status as SessionSummary['state']['status'], moves },
		...rest
	};
}

describe('statusText', () => {
	it('maps every game status to its label', () => {
		expect(statusText(sess('a', { status: 'WON' }))).toBe('won');
		expect(statusText(sess('a', { status: 'LOST' }))).toBe('lost');
		expect(statusText(sess('a', { status: 'PLAYING' }))).toBe('playing');
		expect(statusText(sess('a', { status: 'READY' }))).toBe('ready');
	});

	it('knows which sessions are finished', () => {
		expect(isFinished(sess('a', { status: 'WON' }))).toBe(true);
		expect(isFinished(sess('a', { status: 'LOST' }))).toBe(true);
		expect(isFinished(sess('a', { status: 'PLAYING' }))).toBe(false);
		expect(isFinished(sess('a'))).toBe(false);
	});
});

describe('filterSessions', () => {
	const list = [
		sess('1', { displayName: 'Alice', status: 'WON' }),
		sess('2', { displayName: 'bob', status: 'PLAYING', map: { name: 'Ladder', parMoves: 3, maxMoves: 30 } }),
		sess('abc3', { status: 'LOST' }),
		sess('4', { displayName: 'Dee' })
	];
	const base = { status: 'all' as const, text: '' };

	it('filters by status; Playing includes sessions that have not moved yet', () => {
		expect(filterSessions(list, base).map((s) => s.id)).toEqual(['1', '2', 'abc3', '4']);
		expect(filterSessions(list, { ...base, status: 'won' }).map((s) => s.id)).toEqual(['1']);
		expect(filterSessions(list, { ...base, status: 'playing' }).map((s) => s.id)).toEqual(['2', '4']);
	});

	it('searches name (anonymous fallback), id and map name, case-insensitive', () => {
		expect(filterSessions(list, { ...base, text: ' ALI ' }).map((s) => s.id)).toEqual(['1']);
		expect(filterSessions(list, { ...base, text: 'anon' }).map((s) => s.id)).toEqual(['abc3']);
		expect(filterSessions(list, { ...base, text: 'ABC' }).map((s) => s.id)).toEqual(['abc3']);
		expect(filterSessions(list, { ...base, text: 'ladder' }).map((s) => s.id)).toEqual(['2']);
		expect(filterSessions(list, { status: 'lost', text: 'bob' })).toEqual([]);
	});
});

describe('countByStatus', () => {
	it('counts every chip and the total', () => {
		expect(countByStatus([sess('1', { status: 'WON' }), sess('2', { status: 'WON' }), sess('3'), sess('4', { status: 'PLAYING' })])).toEqual({
			all: 4, playing: 2, won: 2, lost: 0
		});
	});
});

describe('sortSessions', () => {
	const list = [sess('a', { moves: 2 }), sess('b', { moves: 9 }), sess('c', { moves: 5 })];
	it('orders by most moves without mutating, and keeps the server order otherwise', () => {
		expect(sortSessions(list, 'MOVES').map((s) => s.id)).toEqual(['b', 'c', 'a']);
		expect(list.map((s) => s.id)).toEqual(['a', 'b', 'c']);
		expect(sortSessions(list, 'RECENT')).toBe(list);
	});
});

describe('recentStarted', () => {
	it('hides sessions without moves and caps the list', () => {
		const list = Array.from({ length: 12 }, (_, i) => sess(String(i), { moves: i % 3 === 0 ? 0 : i }));
		const out = recentStarted(list);
		expect(out).toHaveLength(8);
		expect(out.every((s) => s.state.moves > 0)).toBe(true);
		expect(recentStarted(list, 2).map((s) => s.id)).toEqual(['1', '2']);
	});
});

describe('last session memory', () => {
	it('remembers, recalls and forgets the id under the repo-prefixed key', () => {
		const data = new Map<string, string>();
		const store = {
			getItem: (k: string) => data.get(k) ?? null,
			setItem: (k: string, v: string) => void data.set(k, v),
			removeItem: (k: string) => void data.delete(k)
		};
		expect(LAST_SESSION_KEY.startsWith('drbrain-mazes.')).toBe(true);
		expect(recalledSession(store)).toBeNull();
		rememberSession('s1', store);
		expect(recalledSession(store)).toBe('s1');
		forgetSession(store);
		expect(recalledSession(store)).toBeNull();
		expect(recalledSession(null)).toBeNull();
	});
});
