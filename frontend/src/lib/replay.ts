import { writable, type Readable } from 'svelte/store';
import { directionToward } from './tower';
import type { Direction, LevelView, Position, Session } from './types';
import type { SeenLevels } from './fog';

/** Slowest replay step; matches the miner's 120ms slide so single moves look the same as before. */
export const MAX_STEP_MS = 120;
/** Fastest replay step, used when a long bulkMove has to be caught up on. */
export const MIN_STEP_MS = 30;
/** Rough upper bound for replaying one burst of moves. */
export const BURST_MS = 3000;

/** Delay before showing the next path entry when `remaining` entries are still queued. */
export function stepDelay(remaining: number): number {
	return Math.round(Math.max(MIN_STEP_MS, Math.min(MAX_STEP_MS, BURST_MS / Math.max(1, remaining))));
}

const samePos = (a: Position, b: Position) => a.level === b.level && a.x === b.x && a.y === b.y;

/** True when the first n entries of a and b are the same tiles. */
export function sharesPrefix(a: Position[], b: Position[], n: number): boolean {
	if (a.length < n || b.length < n) return false;
	for (let i = 0; i < n; i++) if (!samePos(a[i], b[i])) return false;
	return true;
}

/**
 * A path entry is a move when it is one step from the entry before it on the same level.
 * Stairs and portal landings are extra entries without a move of their own.
 */
const isMoveEntry = (path: Position[], i: number) => i > 0 && directionToward(path[i - 1], path[i]) !== null;

/** The direction of the last walking step within the first n path entries. */
export function facingAt(path: Position[], n: number): Direction | null {
	for (let i = Math.min(n, path.length) - 1; i > 0; i--) {
		const d = directionToward(path[i - 1], path[i]);
		if (d) return d;
	}
	return null;
}

export interface Replay extends Readable<number> {
	/**
	 * Feeds the latest server path. Entries that extend what is already shown are
	 * played one at a time; anything else (first load, reset, a different path)
	 * is shown at once. canShow says whether the board can draw a tile's level
	 * (fog: only levels the page has seen); tiles it cannot draw are skipped.
	 */
	sync(path: Position[], canShow: (p: Position) => boolean): void;
	destroy(): void;
}

export const prefersReducedMotion = () =>
	typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches;

export interface ReplayOptions {
	/** When true every update is shown at once (prefers-reduced-motion). */
	instant?: () => boolean;
}

/** Plays a growing session path one tile at a time; the store value is how many entries to show. */
export function createReplay(opts: ReplayOptions = {}): Replay {
	const shown = writable(0);
	let n = 0;
	let path: Position[] = [];
	let canShow: (p: Position) => boolean = () => true;
	let timer: ReturnType<typeof setTimeout> | null = null;
	let started = false;

	const set = (v: number) => {
		n = v;
		shown.set(v);
	};
	const stop = () => {
		if (timer !== null) clearTimeout(timer);
		timer = null;
	};
	// Moves n one entry forward, skipping tiles on levels the board cannot draw.
	const advance = () => {
		let next = n + 1;
		while (next < path.length && !canShow(path[next - 1])) next++;
		set(next);
	};
	const schedule = () => {
		stop();
		if (n >= path.length) return;
		timer = setTimeout(() => {
			timer = null;
			advance();
			schedule();
		}, stepDelay(path.length - n));
	};

	return {
		subscribe: shown.subscribe,
		sync(next, show) {
			const prev = path;
			path = next;
			canShow = show;
			const extends_ = started && next.length > n && sharesPrefix(prev, next, n);
			started = true;
			if (!extends_ || opts.instant?.()) {
				stop();
				set(next.length);
				return;
			}
			// Idle: show the first new tile right away so a human's own move feels instant.
			if (timer === null) advance();
			schedule();
		},
		destroy: stop
	};
}

export interface ReplayFrame {
	/** The session as it should be drawn right now. */
	session: Session;
	replaying: boolean;
	/** Miner facing while replaying; null when the live lastEvent decides. */
	facing: Direction | null;
}

/**
 * Builds the session to draw when only the first `shown` path entries have been played.
 * While replaying, the outcome (WON/LOST), the last event and the move count wait for the
 * miner to catch up, and the view is the remembered level the miner is on.
 */
export function replayFrame(live: Session, shown: number, seen: SeenLevels): ReplayFrame {
	const path = live.state.path;
	if (shown >= path.length || shown < 1) return { session: live, replaying: false, facing: null };
	const pos = path[shown - 1];
	let pending = 0;
	for (let i = shown; i < path.length; i++) if (isMoveEntry(path, i)) pending++;
	const view: LevelView = pos.level === live.view.level ? live.view : (seen[pos.level] ?? live.view);
	return {
		session: {
			...live,
			view,
			lastEvent: null,
			state: {
				...live.state,
				pos,
				path: path.slice(0, shown),
				moves: Math.max(0, live.state.moves - pending),
				status: 'PLAYING',
				lossReason: null
			}
		},
		replaying: true,
		facing: facingAt(path, shown)
	};
}
