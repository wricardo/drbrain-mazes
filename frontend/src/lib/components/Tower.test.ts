import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import Tower from './Tower.svelte';

afterEach(cleanup);

describe('Tower', () => {
	it('lists levels top first: FINISH on top, START at the bottom, a star on the miner level', () => {
		render(Tower, { levelCount: 3, current: 1 });
		const names = screen.getAllByTestId(/^tower-level-/).map((el) => el.getAttribute('aria-label'));
		expect(names).toEqual(['Level 3, finish', 'Level 2, miner is here', 'Level 1, start']);
		expect(screen.getAllByTestId('tower-star')).toHaveLength(1);
		expect(screen.getByText('Finish')).toBeTruthy();
		expect(screen.getByText('Start')).toBeTruthy();
	});

	it('only lets the user open levels already seen; the rest are unexplored and not buttons', async () => {
		const onpeek = vi.fn();
		render(Tower, { levelCount: 3, current: 1, known: [0, 1], onpeek });
		expect(screen.getByTestId('tower-level-2').getAttribute('aria-label')).toBe('Level 3, finish, unexplored');
		expect(screen.getByTestId('tower-level-2').tagName).not.toBe('BUTTON');
		await fireEvent.click(screen.getByTestId('tower-level-2'));
		await fireEvent.click(screen.getByTestId('tower-level-0'));
		expect(onpeek.mock.calls.map((c) => c[0])).toEqual([0]);
	});

	it('is read-only without onpeek', () => {
		render(Tower, { levelCount: 2, current: 0 });
		expect(screen.queryAllByRole('button')).toHaveLength(0);
	});
});
