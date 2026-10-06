import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import Board from './Board.svelte';
import type { GameState, LevelView } from '../types';

afterEach(cleanup);

const view = (level = 0): LevelView => ({
	level,
	levelCount: 3,
	rows: ['So..', 'oUo.', '2..2'],
	portals: [
		{ glyph: '2', a: { level, x: 0, y: 2 }, b: { level, x: 3, y: 2 } },
		{ glyph: '2', a: { level, x: 3, y: 2 }, b: { level, x: 0, y: 2 } }
	]
});

const game = (level: number, x: number, y: number, moves = 0): GameState => ({
	pos: { level, x, y }, moves, status: 'PLAYING', lossReason: null, path: [{ level: 0, x: 0, y: 0 }, { level, x, y }]
});

describe('Board', () => {
	it('draws the visible level: tiles and one pipe for a portal pair', () => {
		render(Board, { view: view(), game: game(0, 1, 1) });
		expect(screen.getAllByTestId('tile-stone')).toHaveLength(3);
		expect(screen.getAllByTestId('hole')).toHaveLength(2);
		expect(screen.getAllByTestId('pipe')).toHaveLength(1);
		expect(screen.getByTestId('board').getAttribute('aria-label')).toMatch(/Level 1 of 3.*column 1, row 1/);
	});

	it('keeps the miner visible on a stairs tile and says how to use the stairs again', () => {
		render(Board, { view: view(), game: game(0, 1, 1) });
		const b = screen.getByTestId('board');
		expect(screen.getByTestId('miner-on-stairs')).toBeTruthy();
		expect(b.getAttribute('aria-label')).toMatch(/standing on the stairs up; step off and back on/);
		expect(screen.getByTestId('miner')).toBeTruthy();
	});

	it('does not mention stairs when standing on a stone', () => {
		render(Board, { view: view(), game: game(0, 0, 1) });
		expect(screen.queryByTestId('miner-on-stairs')).toBeNull();
		expect(screen.getByTestId('board').getAttribute('aria-label')).not.toMatch(/stairs/);
	});

	it('draws a remembered level read-only: no miner, no taps, labelled as remembered', () => {
		render(Board, { view: view(1), remembered: true, game: game(2, 1, 1), onmove: vi.fn() });
		expect(screen.queryByTestId('miner')).toBeNull();
		expect(screen.queryAllByTestId(/^tap-/)).toHaveLength(0);
		const b = screen.getByTestId('board');
		expect(b.getAttribute('aria-label')).toMatch(/Remembered from earlier/);
		expect(b.getAttribute('data-remembered')).toBe('true');
	});

	it('steps toward a clicked neighbouring tile', async () => {
		const onmove = vi.fn();
		render(Board, { view: view(), game: game(0, 1, 1), onmove });
		expect(screen.getAllByTestId(/^tap-/)).toHaveLength(4);
		await fireEvent.click(screen.getByTestId('tap-right'));
		await fireEvent.click(screen.getByTestId('tap-up'));
		expect(onmove.mock.calls.map((c) => c[0])).toEqual(['RIGHT', 'UP']);
	});

	it('flashes the bump outline only when the latest move was blocked', () => {
		const ev = { move: 2, direction: 'LEFT', from: { level: 0, x: 0, y: 0 }, stepped: { level: 0, x: 0, y: 0 }, to: { level: 0, x: 0, y: 0 }, blocked: true, via: 'NONE', status: 'PLAYING', lossReason: null } as const;
		const { unmount } = render(Board, { view: view(), game: game(0, 0, 0, 2), lastEvent: ev });
		expect(screen.getByTestId('bump')).toBeTruthy();
		unmount();
		render(Board, { view: view(), game: game(0, 0, 0, 3), lastEvent: ev });
		expect(screen.queryByTestId('bump')).toBeNull();
	});
});
