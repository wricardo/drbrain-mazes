import { describe, expect, it } from 'vitest';
import { knownLevels, rememberView, shownLevel } from './fog';
import type { LevelView } from './types';

const view = (level: number, rows = ['So.']): LevelView => ({ level, levelCount: 3, rows, portals: [] });

describe('rememberView', () => {
	it('stores the current level by index and keeps older ones', () => {
		const a = rememberView({}, view(0));
		const b = rememberView(a, view(1, ['oF.']));
		expect(Object.keys(b)).toEqual(['0', '1']);
		expect(b[0].rows).toEqual(['So.']);
	});
	it('returns the same object when the level is unchanged', () => {
		const a = rememberView({}, view(0));
		expect(rememberView(a, view(0))).toBe(a);
		expect(rememberView(a, view(0, ['oo.']))).not.toBe(a);
	});
});

describe('shownLevel', () => {
	const seen = { 0: view(0, ['old']), 1: view(1) };
	it('shows the live view unless the peek is a different, remembered level', () => {
		const live = view(1, ['live']);
		expect(shownLevel(live, null, seen)).toEqual({ view: live, remembered: false });
		expect(shownLevel(live, 1, seen)).toEqual({ view: live, remembered: false });
		expect(shownLevel(live, 0, seen)).toEqual({ view: seen[0], remembered: true });
		expect(shownLevel(live, 2, seen)).toEqual({ view: live, remembered: false });
	});
});

describe('knownLevels', () => {
	it('lists the live level plus remembered ones, ascending', () => {
		expect(knownLevels(view(2), { 0: view(0) })).toEqual([0, 2]);
		expect(knownLevels(view(0), {})).toEqual([0]);
	});
});
