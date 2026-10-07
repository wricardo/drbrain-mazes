import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';
import { createReplay, facingAt, MAX_STEP_MS, MIN_STEP_MS, replayFrame, sharesPrefix, stepDelay } from './replay';
import type { LevelView, Position, Session } from './types';

const p = (x: number, y: number, level = 0): Position => ({ level, x, y });
const all = () => true;

describe('stepDelay', () => {
	it('uses the slide length for short bursts and speeds up long ones', () => {
		expect(stepDelay(1)).toBe(MAX_STEP_MS);
		expect(stepDelay(10)).toBe(MAX_STEP_MS);
		expect(stepDelay(50)).toBe(60);
		expect(stepDelay(5000)).toBe(MIN_STEP_MS);
	});
});

describe('sharesPrefix', () => {
	it('compares the first n tiles', () => {
		expect(sharesPrefix([p(0, 0), p(1, 0)], [p(0, 0), p(1, 0), p(2, 0)], 2)).toBe(true);
		expect(sharesPrefix([p(0, 0), p(1, 0)], [p(0, 0), p(0, 1)], 2)).toBe(false);
		expect(sharesPrefix([p(0, 0)], [p(0, 0)], 2)).toBe(false);
	});
});

describe('facingAt', () => {
	it('is the direction of the last walking step, skipping stairs landings', () => {
		const path = [p(0, 0), p(1, 0), p(1, 0, 1)];
		expect(facingAt(path, 2)).toBe('RIGHT');
		expect(facingAt(path, 3)).toBe('RIGHT');
		expect(facingAt(path, 1)).toBeNull();
	});
});

describe('createReplay', () => {
	beforeEach(() => vi.useFakeTimers());
	afterEach(() => vi.useRealTimers());

	it('shows the first snapshot at once', () => {
		const r = createReplay();
		r.sync([p(0, 0), p(1, 0), p(2, 0)], all);
		expect(get(r)).toBe(3);
		r.destroy();
	});

	it('plays an extended path one tile at a time, the first one immediately', () => {
		const r = createReplay();
		r.sync([p(0, 0)], all);
		r.sync([p(0, 0), p(1, 0), p(2, 0), p(3, 0)], all);
		expect(get(r)).toBe(2);
		vi.advanceTimersByTime(MAX_STEP_MS);
		expect(get(r)).toBe(3);
		vi.advanceTimersByTime(MAX_STEP_MS);
		expect(get(r)).toBe(4);
		vi.advanceTimersByTime(MAX_STEP_MS * 5);
		expect(get(r)).toBe(4);
		r.destroy();
	});

	it('keeps playing when more moves arrive mid-replay', () => {
		const r = createReplay();
		r.sync([p(0, 0)], all);
		r.sync([p(0, 0), p(1, 0), p(2, 0)], all);
		r.sync([p(0, 0), p(1, 0), p(2, 0), p(3, 0)], all);
		expect(get(r)).toBe(2);
		vi.advanceTimersByTime(MAX_STEP_MS * 2);
		expect(get(r)).toBe(4);
		r.destroy();
	});

	it('jumps when the path is not an extension (reset)', () => {
		const r = createReplay();
		r.sync([p(0, 0), p(1, 0), p(2, 0)], all);
		r.sync([p(0, 0)], all);
		expect(get(r)).toBe(1);
		r.sync([p(0, 0), p(0, 1)], all);
		expect(get(r)).toBe(2);
		r.destroy();
	});

	it('skips tiles on levels the board cannot draw', () => {
		const r = createReplay();
		r.sync([p(0, 0)], all);
		const path = [p(0, 0), p(1, 0), p(1, 0, 1), p(2, 0, 1), p(2, 0, 2), p(3, 0, 2)];
		r.sync(path, (q) => q.level !== 1);
		expect(get(r)).toBe(2);
		vi.advanceTimersByTime(MAX_STEP_MS);
		expect(get(r)).toBe(5);
		vi.advanceTimersByTime(MAX_STEP_MS);
		expect(get(r)).toBe(6);
		r.destroy();
	});

	it('is instant when asked to be', () => {
		const r = createReplay({ instant: () => true });
		r.sync([p(0, 0)], all);
		r.sync([p(0, 0), p(1, 0), p(2, 0)], all);
		expect(get(r)).toBe(3);
		r.destroy();
	});
});

describe('replayFrame', () => {
	const view = (level: number): LevelView => ({ level, levelCount: 2, rows: [`L${level}`], portals: [] });
	const path = [p(0, 0), p(1, 0), p(1, 0, 1), p(2, 0, 1)];
	const live = {
		view: view(1),
		lastEvent: { blocked: false },
		seq: 9,
		state: { pos: p(2, 0, 1), moves: 3, status: 'WON', lossReason: null, path }
	} as unknown as Session;

	it('is the live session once caught up', () => {
		expect(replayFrame(live, 4, {})).toEqual({ session: live, replaying: false, facing: null });
	});

	it('holds back the outcome and counts only the moves played', () => {
		const f = replayFrame(live, 2, { 0: view(0) });
		expect(f.replaying).toBe(true);
		expect(f.facing).toBe('RIGHT');
		expect(f.session.state.pos).toEqual(p(1, 0));
		expect(f.session.state.moves).toBe(2);
		expect(f.session.state.status).toBe('PLAYING');
		expect(f.session.state.path).toHaveLength(2);
		expect(f.session.lastEvent).toBeNull();
		expect(f.session.view.level).toBe(0);
	});
});
