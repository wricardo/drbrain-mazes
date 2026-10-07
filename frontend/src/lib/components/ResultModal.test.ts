import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import ResultModal from './ResultModal.svelte';

afterEach(cleanup);

const base = { status: 'WON' as const, reason: null, moves: 12, maxMoves: 40, par: 10, best: 12 };

describe('ResultModal', () => {
	it('offers Try again · Next map · All maps in that order and shows the numbers', async () => {
		const onreset = vi.fn();
		const onnext = vi.fn();
		render(ResultModal, { ...base, nextName: 'Ladder', onreset, onnext, onclose: vi.fn() });
		const labels = screen.getAllByRole('button').concat(screen.getAllByRole('link')).map((el) => el.textContent?.trim());
		expect(labels.filter((l) => l !== '×')).toEqual(['Try again', 'Next map', 'All maps']);
		expect(screen.getByTestId('result-numbers').textContent).toMatch(/12\s*\/ 40/);
		await fireEvent.click(screen.getByTestId('result-retry'));
		await fireEvent.click(screen.getByTestId('result-next'));
		expect(onreset).toHaveBeenCalledOnce();
		expect(onnext).toHaveBeenCalledOnce();
	});

	it('hides Next map after the last map and closes on Escape', async () => {
		const onclose = vi.fn();
		render(ResultModal, { ...base, status: 'LOST', reason: 'MOVE_LIMIT', onreset: vi.fn(), onclose });
		expect(screen.queryByTestId('result-next')).toBeNull();
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(onclose).toHaveBeenCalledOnce();
	});
});
