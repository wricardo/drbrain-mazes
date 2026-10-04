import { describe, expect, it } from 'vitest';
import { blankCells, clampSide, formatLevel, levelOfGlyph, localIssues, paintCell, parseLevel, resizeLevels, type Cells } from './levels';

const level = (rows: string[]): Cells => parseLevel(rows, rows[0].length, rows.length);

describe('level cells', () => {
	it('round-trips rows and pads short ones with void', () => {
		expect(formatLevel(parseLevel(['So', 'o'], 3, 2))).toEqual(['So.', 'o..']);
		expect(formatLevel(blankCells(3, 2))).toEqual(['...', '...']);
	});
	it('resizes every level keeping content', () => {
		const out = resizeLevels([level(['So.', 'oo.', '..F'])], 4, 2);
		expect(out.map(formatLevel)).toEqual([['So..', 'oo..']]);
	});
	it('clamps sides to 3..20 and falls back to 3 for NaN', () => {
		expect([clampSide(1), clampSide(8.9), clampSide(99), clampSide(NaN)]).toEqual([3, 8, 20, 3]);
	});
	it('finds the level holding a glyph', () => {
		expect(levelOfGlyph([level(['ooo']), level(['oFo'])], 'F')).toBe(1);
		expect(levelOfGlyph([level(['ooo'])], 'F')).toBeNull();
	});
});

describe('localIssues', () => {
	const ok = [level(['So.', 'oUo', '...']), level(['...', 'oDF', '...'])];

	it('accepts a well-formed tower', () => {
		expect(localIssues(ok)).toEqual([]);
	});
	it('requires exactly one start and one finish', () => {
		const issues = localIssues([level(['SS.', '...', '...'])]);
		expect(issues).toContain('Place exactly one start (S).');
		expect(issues).toContain('Place exactly one finish (F).');
	});
	it('requires the start on the bottom level and the finish on the top level', () => {
		expect(localIssues([level(['oSo', '...', '...']), level(['...', 'oFo', '...'])]).length).toBe(0);
		const flipped = localIssues([level(['oFo', '...', '...']), level(['...', 'oSo', '...'])]);
		expect(flipped).toEqual(['The start (S) must be on level 1, the bottom.', 'The finish (F) must be on the top level.']);
	});
	it('requires portal digits to appear exactly twice', () => {
		const issues = localIssues([level(['S1F', '1.1', '...'])]);
		expect(issues).toEqual(['Portal 1 must appear exactly twice (found 3).']);
	});
	it('requires every stairs tile to have its twin on the adjacent level at the same x,y', () => {
		expect(localIssues([level(['SUF', '...', '...'])])).toEqual(['stairs U at level 1 (1,0) lead off the tower']);
		const unmatched = [level(['SUo', '...', '...']), level(['F..', '...', '...'])];
		expect(localIssues(unmatched)).toEqual(['stairs U at level 1 (1,0) need a matching D on level 2']);
		const paired = [level(['SUo', '...', '...']), level(['FD.', '...', '...'])];
		expect(localIssues(paired)).toEqual([]);
		const orphanDown = [level(['Soo', '...', '...']), level(['FD.', '...', '...'])];
		expect(localIssues(orphanDown)).toEqual(['stairs D at level 2 (1,0) need a matching U on level 1']);
	});
});

describe('paintCell', () => {
	const tower = () => [level(['So.', '...', '...']), level(['.o.', '...', '...']), level(['.o.', '..F', '...'])];
	const rows = (t: Cells[]) => t.map(formatLevel);

	it('painting U places the matching D one level up, and D places U one level down', () => {
		const t = tower();
		expect(paintCell(t, 0, 1, 0, 'U')).toBe(true);
		expect(rows(t)[0][0]).toBe('SU.');
		expect(rows(t)[1][0]).toBe('.D.');
		paintCell(t, 2, 0, 2, 'D');
		expect(rows(t)[2][2]).toBe('D..');
		expect(rows(t)[1][2]).toBe('U..');
		expect(localIssues(t).filter((i) => i.startsWith('stairs'))).toEqual([]);
	});

	it('removing or replacing one end turns its partner into stone', () => {
		const t = tower();
		paintCell(t, 0, 1, 0, 'U');
		paintCell(t, 0, 1, 0, '.');
		expect(rows(t)[1][0]).toBe('.o.');
		paintCell(t, 1, 0, 2, 'D');
		expect(rows(t)[0][2]).toBe('U..');
		paintCell(t, 1, 0, 2, 'o');
		expect(rows(t)[0][2]).toBe('o..');
	});

	it('does nothing for an unchanged cell and never overwrites S or F with a partner', () => {
		const t = tower();
		expect(paintCell(t, 0, 0, 0, 'S')).toBe(false);
		paintCell(t, 1, 0, 1, 'D');
		expect(rows(t)[0][1]).toBe('U..');
		const t2 = [level(['...', '...', '...']), level(['F..', '...', '...'])];
		paintCell(t2, 0, 0, 0, 'U');
		expect(rows(t2)[1][0]).toBe('F..');
	});

	it('keeps other stairs paired when a partner displaces them', () => {
		const t = [level(['...', '...', '...']), level(['...', '...', '...']), level(['...', '...', '...'])];
		paintCell(t, 1, 0, 0, 'U'); // level1 U, level2 D
		paintCell(t, 0, 0, 0, 'U'); // level0 U overwrites level1's U with D: old U's D at level 2 must go
		expect(rows(t).map((r) => r[0])).toEqual(['U..', 'D..', 'o..']);
	});

	it('leaves a lone stairs when the adjacent level does not exist', () => {
		const t = [level(['...', '...', '...'])];
		paintCell(t, 0, 0, 0, 'U');
		expect(rows(t)[0][0]).toBe('U..');
		expect(localIssues(t)).toContain('stairs U at level 1 (0,0) lead off the tower');
	});
});
