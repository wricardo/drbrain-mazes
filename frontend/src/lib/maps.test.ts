import { describe, expect, it } from 'vitest';
import { countByDifficulty, filterMaps, nextMap, sortMapsByDifficulty } from './maps';

describe('sortMapsByDifficulty', () => {
	it('orders easy -> medium -> hard, then by name, unknown last, without mutating', () => {
		const input = [
			{ name: 'B', difficulty: 'hard' },
			{ name: 'Z', difficulty: 'weird' },
			{ name: 'C', difficulty: 'easy' },
			{ name: 'A', difficulty: 'hard' },
			{ name: 'M', difficulty: 'medium' },
			{ name: 'A2', difficulty: 'easy' }
		];
		const copy = structuredClone(input);
		expect(sortMapsByDifficulty(input).map((m) => m.name)).toEqual(['A2', 'C', 'M', 'A', 'B', 'Z']);
		expect(input).toEqual(copy);
	});
});

const maps = [
	{ id: 'h', name: 'Deep Tower', description: 'Many floors', difficulty: 'hard' },
	{ id: 'e', name: 'Ladder', description: 'A first climb', difficulty: 'easy' },
	{ id: 'm', name: 'Wormholes', description: 'Portals everywhere', difficulty: 'medium' }
];

describe('filterMaps', () => {
	it('filters by difficulty and by name or description text', () => {
		expect(filterMaps(maps, { difficulty: 'all', text: '' }).map((m) => m.id)).toEqual(['h', 'e', 'm']);
		expect(filterMaps(maps, { difficulty: 'easy', text: '' }).map((m) => m.id)).toEqual(['e']);
		expect(filterMaps(maps, { difficulty: 'all', text: ' PORTAL ' }).map((m) => m.id)).toEqual(['m']);
		expect(filterMaps(maps, { difficulty: 'all', text: 'tower' }).map((m) => m.id)).toEqual(['h']);
		expect(filterMaps(maps, { difficulty: 'hard', text: 'ladder' })).toEqual([]);
	});
});

describe('countByDifficulty', () => {
	it('counts each difficulty and the total', () => {
		expect(countByDifficulty([...maps, { difficulty: 'easy' }])).toEqual({ all: 4, easy: 2, medium: 1, hard: 1 });
	});
});

describe('nextMap', () => {
	it('follows the shared order and stops after the last map', () => {
		expect(nextMap(maps, 'e')?.id).toBe('m');
		expect(nextMap(maps, 'm')?.id).toBe('h');
		expect(nextMap(maps, 'h')).toBeNull();
		expect(nextMap(maps, 'nope')).toBeNull();
	});
});
