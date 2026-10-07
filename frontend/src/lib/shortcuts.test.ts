import { describe, expect, it } from 'vitest';
import { isTypingTarget, keyToAction } from './shortcuts';

describe('keyToAction', () => {
	it('maps arrows and WASD (any case) to moves', () => {
		expect(keyToAction({ key: 'ArrowUp' })).toEqual({ type: 'move', direction: 'UP' });
		expect(keyToAction({ key: 'ArrowLeft' })).toEqual({ type: 'move', direction: 'LEFT' });
		expect(keyToAction({ key: 'w' })).toEqual({ type: 'move', direction: 'UP' });
		expect(keyToAction({ key: 'D' })).toEqual({ type: 'move', direction: 'RIGHT' });
		expect(keyToAction({ key: 's' })).toEqual({ type: 'move', direction: 'DOWN' });
		expect(keyToAction({ key: 'A' })).toEqual({ type: 'move', direction: 'LEFT' });
	});

	it('maps R to reset, ? to help and ignores other keys', () => {
		expect(keyToAction({ key: 'r' })).toEqual({ type: 'reset' });
		expect(keyToAction({ key: 'R' })).toEqual({ type: 'reset' });
		expect(keyToAction({ key: '?' })).toEqual({ type: 'help' });
		expect(keyToAction({ key: 'Enter' })).toBeNull();
		expect(keyToAction({ key: 'x' })).toBeNull();
	});

	it('leaves browser shortcuts alone (Cmd+R reloads, Ctrl+W closes)', () => {
		expect(keyToAction({ key: 'r', metaKey: true })).toBeNull();
		expect(keyToAction({ key: 'w', ctrlKey: true })).toBeNull();
		expect(keyToAction({ key: 'ArrowLeft', altKey: true })).toBeNull();
	});
});

describe('isTypingTarget', () => {
	it('is true for text fields and selects only', () => {
		expect(isTypingTarget(document.createElement('input'))).toBe(true);
		expect(isTypingTarget(document.createElement('textarea'))).toBe(true);
		expect(isTypingTarget(document.createElement('select'))).toBe(true);
		expect(isTypingTarget(document.createElement('button'))).toBe(false);
		expect(isTypingTarget(null)).toBe(false);
	});
});
