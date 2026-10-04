import type { LevelView } from './types';

/** Levels the player has stood on in this page session, by level index. */
export type SeenLevels = Record<number, LevelView>;

/** Remembers the level currently in view; returns the same object when nothing changed. */
export function rememberView(seen: SeenLevels, view: LevelView): SeenLevels {
	const prev = seen[view.level];
	if (prev && prev.rows.join('\n') === view.rows.join('\n') && prev.portals.length === view.portals.length) return seen;
	return { ...seen, [view.level]: view };
}

export interface ShownLevel {
	view: LevelView;
	/** True when this is a remembered level, not the one the miner stands on. */
	remembered: boolean;
}

/**
 * Chooses what the board shows: the live view, or a remembered level the user peeked at.
 * A peek at a level that was never seen (or at the current one) shows the live view.
 */
export function shownLevel(live: LevelView, peek: number | null, seen: SeenLevels): ShownLevel {
	if (peek !== null && peek !== live.level) {
		const remembered = seen[peek];
		if (remembered) return { view: remembered, remembered: true };
	}
	return { view: live, remembered: false };
}

/** Levels that can be opened in the tower: the current one and every remembered one, ascending. */
export function knownLevels(live: LevelView, seen: SeenLevels): number[] {
	return [...new Set([live.level, ...Object.keys(seen).map(Number)])].sort((a, b) => a - b);
}
