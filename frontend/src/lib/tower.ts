import type { Direction, MoveEvent, Portal, Position } from './types';

export type TileKind = 'void' | 'stone' | 'start' | 'finish' | 'up' | 'down' | 'portal';

/** Classifies a layout glyph. Unknown glyphs read as void, like the server's blocked cells. */
export function tileKind(glyph: string): TileKind {
	switch (glyph) {
		case 'o':
			return 'stone';
		case 'S':
			return 'start';
		case 'F':
			return 'finish';
		case 'U':
			return 'up';
		case 'D':
			return 'down';
		default:
			return isPortalGlyph(glyph) ? 'portal' : 'void';
	}
}

/** True for the portal digits '1'..'9'. */
export function isPortalGlyph(glyph: string): boolean {
	return /^[1-9]$/.test(glyph);
}

export interface PortalColor {
	name: string;
	/** Ring / pipe colour. */
	ring: string;
	/** Dark inner colour of the hole. */
	hole: string;
}

/** One colour per portal digit, shared by the board, the mini previews and the editor. */
export const PORTAL_COLORS: Record<string, PortalColor> = {
	'1': { name: 'blue', ring: '#3b82f6', hole: '#0b1d4a' },
	'2': { name: 'yellow', ring: '#facc15', hole: '#4a3a02' },
	'3': { name: 'orange', ring: '#fb923c', hole: '#4a2204' },
	'4': { name: 'green', ring: '#22c55e', hole: '#04301a' },
	'5': { name: 'pink', ring: '#f472b6', hole: '#4a0a2c' },
	'6': { name: 'purple', ring: '#a855f7', hole: '#2b0a4a' },
	'7': { name: 'red', ring: '#ef4444', hole: '#4a0a0a' },
	'8': { name: 'cyan', ring: '#22d3ee', hole: '#04323b' },
	'9': { name: 'white', ring: '#f1f5f9', hole: '#2a2f3a' }
};

const NO_PORTAL_COLOR: PortalColor = { name: 'grey', ring: '#94a3b8', hole: '#1e293b' };

/** Colour of a portal digit; neutral grey for anything else. */
export function portalColor(glyph: string): PortalColor {
	return PORTAL_COLORS[glyph] ?? NO_PORTAL_COLOR;
}

/** Column / row change of a step; y grows downward. */
export const DIRECTION_DELTA: Record<Direction, { dx: number; dy: number }> = {
	UP: { dx: 0, dy: -1 },
	RIGHT: { dx: 1, dy: 0 },
	DOWN: { dx: 0, dy: 1 },
	LEFT: { dx: -1, dy: 0 }
};

/** Clockwise rotation (degrees) of an upward-pointing token for each direction. */
export function directionDegrees(d: Direction): number {
	return { UP: 0, RIGHT: 90, DOWN: 180, LEFT: 270 }[d];
}

/**
 * Returns an angle equivalent to the direction that is reachable from prev by the
 * shortest turn, so CSS transitions never spin the long way round (e.g. LEFT→UP).
 */
export function nextAngle(prev: number, d: Direction): number {
	let delta = (directionDegrees(d) - prev) % 360;
	if (delta > 180) delta -= 360;
	if (delta < -180) delta += 360;
	return prev + delta;
}

/** Arrow character drawn for a direction (d-pad, editor, event text). */
export const DIRECTION_ARROW: Record<Direction, string> = { UP: '▲', RIGHT: '▶', DOWN: '▼', LEFT: '◀' };

/** Glyph at p, or '.' (void) when p is outside the tower. Short or missing rows read as void. */
export function glyphAt(levels: string[][], p: Position): string {
	return levels[p.level]?.[p.y]?.[p.x] ?? '.';
}

/** The direction that moves from one tile to an orthogonally adjacent tile on the same level, else null. */
export function directionToward(from: Position, to: { level?: number; x: number; y: number }): Direction | null {
	if (to.level !== undefined && to.level !== from.level) return null;
	const dx = to.x - from.x;
	const dy = to.y - from.y;
	if (Math.abs(dx) + Math.abs(dy) !== 1) return null;
	return dx === 1 ? 'RIGHT' : dx === -1 ? 'LEFT' : dy === 1 ? 'DOWN' : 'UP';
}

/** The tile one step from p in direction d on the same level (may be out of bounds). */
export function neighbor(p: Position, d: Direction): Position {
	const { dx, dy } = DIRECTION_DELTA[d];
	return { level: p.level, x: p.x + dx, y: p.y + dy };
}

export interface PortalPipe {
	glyph: string;
	a: { x: number; y: number };
	b: { x: number; y: number };
}

export interface PortalHole {
	glyph: string;
	x: number;
	y: number;
	/** True when the twin is on another level (its location is hidden by the fog). */
	twinElsewhere: boolean;
}

/**
 * Portal decorations of one visible level: every hole (read off the rows), a pipe for each
 * pair whose twin is on the same level, and a flag on holes whose twin is out of sight.
 */
export function portalLinks(rows: string[], portals: Pick<Portal, 'glyph' | 'b'>[]): { holes: PortalHole[]; pipes: PortalPipe[] } {
	const joined = new Set(portals.filter((p) => p.b).map((p) => p.glyph));
	const holes: PortalHole[] = [];
	const byGlyph = new Map<string, { x: number; y: number }[]>();
	rows.forEach((row, y) =>
		[...row].forEach((glyph, x) => {
			if (!isPortalGlyph(glyph)) return;
			holes.push({ glyph, x, y, twinElsewhere: !joined.has(glyph) });
			byGlyph.set(glyph, [...(byGlyph.get(glyph) ?? []), { x, y }]);
		})
	);
	const pipes: PortalPipe[] = [];
	for (const [glyph, ends] of byGlyph) if (joined.has(glyph) && ends.length === 2) pipes.push({ glyph, a: ends[0], b: ends[1] });
	return { holes, pipes };
}

/** The tiles of `path` that lie on `level`, in order, without duplicates. */
export function trailOnLevel(path: Position[], level: number): Position[] {
	const seen = new Set<string>();
	const out: Position[] = [];
	for (const p of path) {
		if (p.level !== level) continue;
		const key = `${p.x},${p.y}`;
		if (seen.has(key)) continue;
		seen.add(key);
		out.push(p);
	}
	return out;
}

/** Human "Level N" (1-based) for a 0-based level index. */
export function levelName(level: number): string {
	return `Level ${level + 1}`;
}

/**
 * Tower rows from the top level down to level 0. The start is always on level 0 and the
 * finish on the top level.
 */
export function towerRows(levelCount: number): { level: number; isStart: boolean; isFinish: boolean }[] {
	return Array.from({ length: levelCount }, (_, i) => {
		const level = levelCount - 1 - i;
		return { level, isStart: level === 0, isFinish: level === levelCount - 1 };
	});
}

const VIA_TEXT: Record<string, string> = {
	WALK: 'moved',
	STAIRS_UP: 'took the stairs up',
	STAIRS_DOWN: 'took the stairs down',
	PORTAL: 'went through a portal'
};

/**
 * One-line description of the last move for screen readers and the status line. Landing on paired
 * stairs does not trigger them: the miner has to step off and back on.
 */
export function describeEvent(ev: MoveEvent | null): string {
	if (!ev) return '';
	const dir = ev.direction.toLowerCase();
	if (ev.blocked) return `Blocked going ${dir}: nothing there (not counted).`;
	const dest = ev.to.level !== ev.from.level ? ` to level ${ev.to.level + 1}` : '';
	const text = VIA_TEXT[ev.via] ?? 'moved';
	const arrived =
		ev.via === 'STAIRS_UP'
			? ' and arrived on the stairs down; step off and back on to take them again'
			: ev.via === 'STAIRS_DOWN'
				? ' and arrived on the stairs up; step off and back on to take them again'
				: '';
	return `Move ${ev.move}: ${dir}, ${text}${ev.via === 'WALK' ? '' : dest}${arrived}.`;
}

/** Glyph counts of a map's levels, handy for summaries and editor validation. */
export function countGlyphs(levels: string[][]): Record<string, number> {
	const out: Record<string, number> = {};
	for (const rows of levels) for (const row of rows) for (const ch of row) out[ch] = (out[ch] ?? 0) + 1;
	return out;
}
