import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import Dpad from './Dpad.svelte';

afterEach(cleanup);

describe('Dpad', () => {
	it('reports the pressed direction and does nothing when disabled', async () => {
		const onmove = vi.fn();
		const { unmount } = render(Dpad, { onmove });
		await fireEvent.click(screen.getByLabelText('Move up'));
		await fireEvent.click(screen.getByLabelText('Move left'));
		expect(onmove.mock.calls.map((c) => c[0])).toEqual(['UP', 'LEFT']);
		unmount();
		render(Dpad, { onmove, disabled: true });
		for (const b of screen.getAllByRole('button')) expect((b as HTMLButtonElement).disabled).toBe(true);
	});
});
