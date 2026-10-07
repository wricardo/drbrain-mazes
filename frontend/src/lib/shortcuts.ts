import type { Direction } from './types';

export type KeyAction = { type: 'move'; direction: Direction } | { type: 'reset' } | { type: 'help' };

/** Every key of the play page, for the help list and the "?" shortcuts dialog. */
export const SHORTCUTS: [string, string][] = [
	['← ↑ ↓ →', 'Move one tile'],
	['W A S D', 'Move one tile'],
	['R', 'Reset to the start (attempts and best are kept)'],
	['?', 'Show these shortcuts'],
	['Esc', 'Close a dialog']
];

const KEY_DIRECTIONS: Record<string, Direction> = {
	ArrowUp: 'UP',
	ArrowRight: 'RIGHT',
	ArrowDown: 'DOWN',
	ArrowLeft: 'LEFT',
	w: 'UP',
	d: 'RIGHT',
	s: 'DOWN',
	a: 'LEFT'
};

export interface KeyLike {
	key: string;
	ctrlKey?: boolean;
	metaKey?: boolean;
	altKey?: boolean;
}

/**
 * Maps a key press to a game action: arrows or WASD move, R resets, ? opens the shortcuts. Keys combined with
 * Ctrl/Cmd/Alt (browser and OS shortcuts) are never actions.
 */
export function keyToAction(e: KeyLike): KeyAction | null {
	if (e.ctrlKey || e.metaKey || e.altKey) return null;
	const key = e.key.length === 1 ? e.key.toLowerCase() : e.key;
	if (key === 'r') return { type: 'reset' };
	if (key === '?') return { type: 'help' };
	const direction = KEY_DIRECTIONS[key];
	return direction ? { type: 'move', direction } : null;
}

/** True when keystrokes target a text field / select / button-like control and must not steer the game. */
export function isTypingTarget(target: EventTarget | null): boolean {
	const el = target as { tagName?: string; isContentEditable?: boolean } | null;
	if (!el?.tagName) return false;
	return el.isContentEditable === true || ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName);
}
