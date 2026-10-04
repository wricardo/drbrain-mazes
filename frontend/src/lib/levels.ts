import { countGlyphs, tileKind } from './tower';

export const MIN_SIDE = 3;
export const MAX_SIDE = 20;
export const MAX_LEVELS = 12;

/** Every glyph the editor can paint, in palette order. */
export const PAINT_GLYPHS = ['.', 'o', 'S', 'F', 'U', 'D', '1', '2', '3', '4', '5', '6', '7', '8', '9'] as const;

export const GLYPH_LABELS: Record<string, string> = {
	'.': 'void',
	o: 'stone',
	S: 'start',
	F: 'finish',
	U: 'stairs up',
	D: 'stairs down'
};

/** Accessible name of a glyph. */
export function glyphLabel(glyph: string): string {
	return GLYPH_LABELS[glyph] ?? (tileKind(glyph) === 'portal' ? `portal ${glyph}` : 'void');
}

/** A grid of single-glyph cells for one level, row 0 on top. */
export type Cells = string[][];

/** Creates a width x height level of void. */
export function blankCells(width: number, height: number): Cells {
	return Array.from({ length: height }, () => Array.from({ length: width }, () => '.'));
}

/** Parses level rows into cells; short rows are padded with void. */
export function parseLevel(rows: string[], width: number, height: number): Cells {
	return Array.from({ length: height }, (_, y) => Array.from({ length: width }, (_, x) => rows[y]?.[x] ?? '.'));
}

/** Serialises cells back to the level rows the API expects. */
export function formatLevel(cells: Cells): string[] {
	return cells.map((row) => row.join(''));
}

/** Resizes every level to width x height, keeping existing content and padding with void. */
export function resizeLevels(levels: Cells[], width: number, height: number): Cells[] {
	return levels.map((cells) => Array.from({ length: height }, (_, y) => Array.from({ length: width }, (_, x) => cells[y]?.[x] ?? '.')));
}

/** Clamps a side length to the allowed range; non-numbers fall back to the minimum. */
export function clampSide(n: number): number {
	return Math.max(MIN_SIDE, Math.min(MAX_SIDE, Math.floor(n) || MIN_SIDE));
}

/** Cheap client-side checks mirroring the server's structural rules, for instant feedback. */
export function localIssues(levels: Cells[]): string[] {
	const issues: string[] = [];
	const rows = levels.map(formatLevel);
	const width = levels[0]?.[0]?.length ?? 0;
	const height = levels[0]?.length ?? 0;
	if (levels.length < 1 || levels.length > MAX_LEVELS) issues.push(`A map needs 1–${MAX_LEVELS} levels.`);
	if (width < MIN_SIDE || width > MAX_SIDE || height < MIN_SIDE || height > MAX_SIDE) {
		issues.push(`Levels must be ${MIN_SIDE}–${MAX_SIDE} cells per side.`);
	}
	const counts = countGlyphs(rows);
	if ((counts.S ?? 0) !== 1) issues.push('Place exactly one start (S).');
	if ((counts.F ?? 0) !== 1) issues.push('Place exactly one finish (F).');
	if ((counts.S ?? 0) === 1 && levelOfGlyph(levels, 'S') !== 0) issues.push('The start (S) must be on level 1, the bottom.');
	if ((counts.F ?? 0) === 1 && levelOfGlyph(levels, 'F') !== levels.length - 1) issues.push('The finish (F) must be on the top level.');
	for (let g = 1; g <= 9; g++) {
		const n = counts[String(g)] ?? 0;
		if (n !== 0 && n !== 2) issues.push(`Portal ${g} must appear exactly twice (found ${n}).`);
	}
	levels.forEach((cells, level) =>
		cells.forEach((row, y) =>
			row.forEach((glyph, x) => {
				const kind = tileKind(glyph);
				if (kind !== 'up' && kind !== 'down') return;
				const up = kind === 'up';
				const target = level + (up ? 1 : -1);
				const what = `stairs ${glyph} at level ${level + 1} (${x},${y})`;
				if (target < 0 || target >= levels.length) issues.push(`${what} lead off the tower`);
				else if (levels[target][y]?.[x] !== (up ? 'D' : 'U')) issues.push(`${what} need a matching ${up ? 'D' : 'U'} on level ${target + 1}`);
			})
		)
	);
	return issues;
}

/** Level of the first cell holding `glyph`, or null when absent. */
export function levelOfGlyph(levels: Cells[], glyph: string): number | null {
	const i = levels.findIndex((cells) => cells.some((row) => row.includes(glyph)));
	return i < 0 ? null : i;
}

/** The glyph a stairs tile must be paired with on the adjacent level, or null for any other glyph. */
function stairsPartner(glyph: string, level: number): { level: number; glyph: string } | null {
	if (glyph === 'U') return { level: level + 1, glyph: 'D' };
	if (glyph === 'D') return { level: level - 1, glyph: 'U' };
	return null;
}

/**
 * Paints one cell (mutating `levels`) and keeps stairs paired, as the server requires: a U needs a D at
 * the same x,y one level up and a D needs a U one level down. Painting a stairs glyph places its partner
 * on the adjacent level when that level exists (never over S or F); replacing a stairs tile turns its
 * partner into stone. Returns false when nothing changed.
 */
export function paintCell(levels: Cells[], level: number, x: number, y: number, glyph: string): boolean {
	const old = levels[level]?.[y]?.[x];
	if (old === undefined || old === glyph) return false;
	levels[level][y][x] = glyph;

	const oldPartner = stairsPartner(old, level);
	if (oldPartner && levels[oldPartner.level]?.[y]?.[x] === oldPartner.glyph) levels[oldPartner.level][y][x] = 'o';

	const partner = stairsPartner(glyph, level);
	const cells = partner && levels[partner.level];
	if (partner && cells) {
		const there = cells[y][x];
		if (there !== 'S' && there !== 'F') {
			const displaced = stairsPartner(there, partner.level);
			if (displaced && displaced.level !== level && levels[displaced.level]?.[y]?.[x] === displaced.glyph) levels[displaced.level][y][x] = 'o';
			cells[y][x] = partner.glyph;
		}
	}
	return true;
}
