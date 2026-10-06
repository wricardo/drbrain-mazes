// Hand-written types mirroring graph/schema.graphqls. Keep in sync with the schema.

export type Direction = 'UP' | 'RIGHT' | 'DOWN' | 'LEFT';
export type Status = 'READY' | 'PLAYING' | 'WON' | 'LOST';
export type LossReason = 'MOVE_LIMIT';
/** How a move changed the position. NONE = blocked (free, nothing happened). */
export type Via = 'NONE' | 'WALK' | 'STAIRS_UP' | 'STAIRS_DOWN' | 'PORTAL';

/** A tile in the tower. level 0 is the bottom; x = column, y = row (row 0 on top). */
export interface Position {
	level: number;
	x: number;
	y: number;
}

/**
 * Two linked holes sharing a glyph digit ('1'..'9'), always on the same level. In a LevelView
 * `a` is on the viewed level and `b` is its twin.
 */
export interface Portal {
	glyph: string;
	a: Position;
	b: Position | null;
}

/** Public catalog entry: nothing beyond the start level (level 0) is revealed. */
export interface GameMap {
	id: string;
	name: string;
	description: string;
	difficulty: string;
	width: number;
	height: number;
	levelCount: number;
	/** Glyph rows of level 0, the level every game starts on. */
	preview: string[];
	maxMoves: number;
	/** Shortest winning route; null when the finish is unreachable. */
	parMoves: number | null;
	/** Fewest stairs any winning route takes (levelCount - 1 + 2 * forcedDescents); null when unreachable. */
	minLevelChanges: number | null;
	/** Fewest down stairs any winning route takes (0 = climb straight up); null when unreachable. */
	forcedDescents: number | null;
}

/** Full map for the editor (admin only). Start is always on level 0, finish on the top level. */
export interface AdminMap {
	id: string;
	name: string;
	description: string;
	difficulty: string;
	width: number;
	height: number;
	levelCount: number;
	/** Layout glyphs per level (index 0 = bottom), each a list of rows. */
	levels: string[][];
	start: Position;
	finish: Position;
	portals: Portal[];
	maxMoves: number;
	parMoves: number | null;
	minLevelChanges: number | null;
	forcedDescents: number | null;
}

/** Fog: the only part of the map a player can see, the level they stand on. */
export interface LevelView {
	level: number;
	levelCount: number;
	rows: string[];
	portals: Portal[];
}

export interface GameState {
	pos: Position;
	moves: number;
	status: Status;
	lossReason: LossReason | null;
	/** Every tile entered, starting with the start tile; stairs/portal moves add the stepped-on and landing tiles. */
	path: Position[];
}

export interface MoveEvent {
	move: number;
	direction: Direction;
	from: Position;
	stepped: Position;
	to: Position;
	blocked: boolean;
	via: Via;
	status: Status;
	lossReason: LossReason | null;
}

export interface Session {
	id: string;
	displayName: string;
	mapId: string;
	map: GameMap;
	view: LevelView;
	state: GameState;
	seq: number;
	createdAt: string;
	lastActionAt: string;
	attempts: number;
	bestMoves: number | null;
	lastEvent: MoveEvent | null;
}

export interface SessionUpdate {
	seq: number;
	session: Session;
	event: MoveEvent | null;
}

/** Lightweight session row used by the recent-sessions list. */
export interface SessionSummary {
	id: string;
	displayName: string;
	mapId: string;
	createdAt: string;
	lastActionAt: string;
	map: { name: string; parMoves: number | null };
	state: { status: Status; moves: number };
}

export interface MapValidationResult {
	valid: boolean;
	issues: { message: string }[];
	finishReachable: boolean;
	parMoves: number | null;
	minLevelChanges: number | null;
	forcedDescents: number | null;
}

export interface MapInput {
	id: string;
	name: string;
	description?: string;
	difficulty?: string;
	levels: string[][];
	maxMoves?: number;
}

/** WebSocket connection indicator for spectator/play views. */
export type ConnectionState = 'connecting' | 'live' | 'reconnecting' | 'offline';
