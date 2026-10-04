import { describe, expect, it } from 'vitest';
import { buildGenericPrompt, buildSessionPrompt, layoutBlock, type PromptSession } from './prompt';
import type { GameMap } from './types';

const map: GameMap = {
	id: 'tiny',
	name: 'Tiny',
	description: 'A tiny tower.',
	difficulty: 'easy',
	width: 3,
	height: 3,
	levelCount: 3,
	preview: ['SoU', '.o.', '1.2'],
	maxMoves: 40,
	parMoves: 4
};

function session(over: Partial<PromptSession> = {}): PromptSession {
	return {
		id: 'abc123',
		map,
		view: {
			level: 0,
			levelCount: 3,
			rows: ['SoU', '.o.', '1.2'],
			portals: [
				{ glyph: '1', a: { level: 0, x: 0, y: 2 }, b: null },
				{ glyph: '2', a: { level: 0, x: 2, y: 2 }, b: { level: 0, x: 2, y: 0 } }
			]
		},
		state: { pos: { level: 0, x: 0, y: 0 } },
		...over
	};
}

describe('layoutBlock', () => {
	it('adds an x ruler and y prefixes aligned with the cells', () => {
		expect(layoutBlock(['So.', '...'])).toBe(['    012', 'y0  So.', 'y1  ...'].join('\n'));
	});
	it('keeps columns aligned when there are 10+ rows', () => {
		const rows = Array.from({ length: 11 }, () => '...');
		const lines = layoutBlock(rows).split('\n');
		expect(lines[0].indexOf('0')).toBe(lines[1].indexOf('.'));
		expect(lines[11].indexOf('.')).toBe(lines[1].indexOf('.'));
	});
});

describe('buildSessionPrompt', () => {
	const p = buildSessionPrompt('http://h:1', session());

	it('carries the session id, endpoints and map facts', () => {
		expect(p).toContain('Session ID: abc123');
		expect(p).toContain('http://h:1/graphql');
		expect(p).toContain('http://h:1/llms.txt');
		expect(p).toContain('http://h:1/watch/abc123');
		expect(p).toContain('at most 40 counted moves');
		expect(p).toContain('Par (shortest route) = 4 moves');
	});

	it('explains the fog and where START and FINISH are', () => {
		expect(p).toContain('Fog: you only see the level you stand on');
		expect(p).toContain('START on level 0 (the bottom) to FINISH on level 2 (the top)');
	});

	it('contains only the current level, with an x/y ruler, position and portals', () => {
		expect(p).toContain('The miner is on level 0 at (0,0)');
		expect(p).toContain('    012\ny0  SoU');
		expect(p).toContain('1: (0,2) -> twin is on another level (location hidden)');
		expect(p).toContain('2: (2,2) <-> (2,0)');
		expect(p).not.toMatch(/Level 1:|Level 2:/);
	});

	it('shows the level the miner is on, not level 0, once it has moved', () => {
		const up = buildSessionPrompt('http://h', session({ view: { level: 1, levelCount: 3, rows: ['ooo', 'oDo', 'ooo'], portals: [] }, state: { pos: { level: 1, x: 1, y: 1 } } }));
		expect(up).toContain('The miner is on level 1 at (1,1)');
		expect(up).toContain('y1  oDo');
		expect(up).not.toContain('SoU');
		expect(up).toContain('(none on this level)');
	});

	it('explains paired two-way stairs', () => {
		expect(p).toContain("a 'U' on level l has a 'D' on level l+1");
		expect(p).toContain('land ON that paired');
		expect(p).toContain('step off and back on');
		expect(p).not.toContain('plain stone');
	});

	it('uses the real Mazes API calls with this session and map', () => {
		expect(p).toContain('simulate(mapID: "tiny"');
		expect(p).toContain('move(sessionID: "abc123"');
		expect(p).toContain('bulkMove(sessionID: "abc123"');
		expect(p).toContain('view {');
		expect(p).toContain('reset');
	});

	it('handles an unknown par', () => {
		expect(buildSessionPrompt('http://h', session({ map: { ...map, parMoves: null } }))).toContain('Par (shortest route) = unknown.');
	});

	it('is stable for the same snapshot', () => {
		expect(buildSessionPrompt('http://h:1', session())).toBe(p);
	});
});

describe('buildGenericPrompt', () => {
	it('points the agent at llms.txt, mentions the fog and the create-session flow', () => {
		const g = buildGenericPrompt('http://h:1');
		expect(g).toContain('http://h:1/llms.txt');
		expect(g).toContain('only ever see the level you stand on');
		expect(g).toContain('createSession');
		expect(g).toContain('http://h:1/watch/SESSION_ID');
	});
});
