import { describe, expect, it } from 'vitest';
import {
	PORTAL_COLORS,
	countGlyphs,
	describeEvent,
	directionToward,
	glyphAt,
	neighbor,
	nextAngle,
	portalColor,
	portalLinks,
	tileKind,
	towerRows,
	trailOnLevel
} from './tower';
import type { MoveEvent, Position } from './types';

const p = (level: number, x: number, y: number): Position => ({ level, x, y });

describe('tileKind', () => {
	it('classifies every glyph; unknown glyphs are void', () => {
		expect(['.', 'o', 'S', 'F', 'U', 'D', '1', '9', '0', 'x', ''].map(tileKind)).toEqual([
			'void', 'stone', 'start', 'finish', 'up', 'down', 'portal', 'portal', 'void', 'void', 'void'
		]);
	});
});

describe('portalColor', () => {
	it('gives each digit 1-9 its own colour and greys out anything else', () => {
		const rings = Object.keys(PORTAL_COLORS).map((d) => portalColor(d).ring);
		expect(rings).toHaveLength(9);
		expect(new Set(rings).size).toBe(9);
		expect(portalColor('1').name).toBe('blue');
		expect(portalColor('x').name).toBe('grey');
	});
});

describe('glyphAt', () => {
	const levels = [['So.', 'o'], ['F..', '...']];
	it('reads cells and treats out-of-range / short rows as void', () => {
		expect(glyphAt(levels, p(0, 0, 0))).toBe('S');
		expect(glyphAt(levels, p(1, 0, 0))).toBe('F');
		expect(glyphAt(levels, p(0, 2, 1))).toBe('.');
		expect(glyphAt(levels, p(5, 0, 0))).toBe('.');
		expect(glyphAt(levels, p(0, -1, 0))).toBe('.');
	});
});

describe('directionToward / neighbor', () => {
	it('maps adjacent same-level tiles to a direction (y grows downward)', () => {
		const from = p(0, 2, 2);
		expect(directionToward(from, { x: 2, y: 1 })).toBe('UP');
		expect(directionToward(from, { x: 3, y: 2 })).toBe('RIGHT');
		expect(directionToward(from, { x: 2, y: 3 })).toBe('DOWN');
		expect(directionToward(from, { x: 1, y: 2 })).toBe('LEFT');
	});
	it('rejects the same tile, diagonals, far tiles and other levels', () => {
		const from = p(0, 2, 2);
		expect(directionToward(from, { x: 2, y: 2 })).toBeNull();
		expect(directionToward(from, { x: 3, y: 3 })).toBeNull();
		expect(directionToward(from, { x: 4, y: 2 })).toBeNull();
		expect(directionToward(from, { level: 1, x: 2, y: 1 })).toBeNull();
	});
	it('neighbor is the inverse of directionToward', () => {
		for (const d of ['UP', 'RIGHT', 'DOWN', 'LEFT'] as const) {
			expect(directionToward(p(0, 4, 4), neighbor(p(0, 4, 4), d))).toBe(d);
		}
	});
});

describe('nextAngle', () => {
	it('turns the short way round', () => {
		expect(nextAngle(270, 'UP')).toBe(360);
		expect(nextAngle(0, 'LEFT')).toBe(-90);
		expect(nextAngle(90, 'DOWN')).toBe(180);
		expect(nextAngle(0, 'UP')).toBe(0);
	});
});

describe('portalLinks', () => {
	const rows = ['1.2.', '..1.', '3...'];
	it('finds every hole and pipes the same-level pairs', () => {
		const { holes, pipes } = portalLinks(rows, [
			{ glyph: '1', b: p(0, 2, 1) },
			{ glyph: '2', b: null },
			{ glyph: '3', b: null }
		]);
		expect(holes.map((h) => `${h.glyph}@${h.x},${h.y}`)).toEqual(['1@0,0', '2@2,0', '1@2,1', '3@0,2']);
		expect(pipes).toEqual([{ glyph: '1', a: { x: 0, y: 0 }, b: { x: 2, y: 1 } }]);
	});
	it('draws no pipe when a hole has no listed twin and ignores non-portal glyphs', () => {
		expect(portalLinks(['So.', 'oFo'], [])).toEqual({ holes: [], pipes: [] });
		expect(portalLinks(['1..'], [{ glyph: '1', b: null }]).pipes).toEqual([]);
	});
});

describe('trailOnLevel', () => {
	it('keeps only that level, in order, without repeats', () => {
		const path = [p(0, 0, 0), p(0, 1, 0), p(1, 1, 0), p(0, 1, 0), p(0, 2, 0)];
		expect(trailOnLevel(path, 0)).toEqual([p(0, 0, 0), p(0, 1, 0), p(0, 2, 0)]);
		expect(trailOnLevel(path, 1)).toEqual([p(1, 1, 0)]);
		expect(trailOnLevel(path, 3)).toEqual([]);
	});
});

describe('towerRows', () => {
	it('lists levels top first with FINISH on top and START at the bottom', () => {
		const rows = towerRows(3);
		expect(rows.map((r) => r.level)).toEqual([2, 1, 0]);
		expect(rows.map((r) => [r.isFinish, r.isStart])).toEqual([[true, false], [false, false], [false, true]]);
		expect(towerRows(1)).toEqual([{ level: 0, isStart: true, isFinish: true }]);
	});
});

describe('describeEvent', () => {
	const base: MoveEvent = {
		move: 3, direction: 'UP', from: p(0, 1, 1), stepped: p(0, 1, 0), to: p(0, 1, 0), blocked: false, via: 'WALK', status: 'PLAYING', lossReason: null
	};
	it('describes walks, level changes and free blocked attempts', () => {
		expect(describeEvent(null)).toBe('');
		expect(describeEvent(base)).toBe('Move 3: up, moved.');
		expect(describeEvent({ ...base, via: 'STAIRS_UP', to: p(1, 1, 0) })).toBe('Move 3: up, took the stairs up to level 2 and arrived on the stairs down; step off and back on to take them again.');
		expect(describeEvent({ ...base, via: 'STAIRS_DOWN', from: p(1, 1, 1), to: p(0, 1, 0) })).toMatch(/took the stairs down to level 1 and arrived on the stairs up/);
		expect(describeEvent({ ...base, via: 'PORTAL', to: p(0, 4, 4) })).toBe('Move 3: up, went through a portal.');
		expect(describeEvent({ ...base, blocked: true, via: 'NONE' })).toMatch(/Blocked going up.*not counted/);
	});
});

describe('countGlyphs', () => {
	it('counts glyphs across all levels', () => {
		expect(countGlyphs([['So'], ['oF']])).toEqual({ S: 1, o: 2, F: 1 });
	});
});
