<script lang="ts">
	import { untrack } from 'svelte';
	import { fly } from 'svelte/transition';
	import type { Direction, GameState, LevelView, MoveEvent } from '$lib/types';
	import {
		DIRECTION_DELTA,
		directionToward,
		levelName,
		nextAngle,
		portalColor,
		portalLinks,
		tileKind,
		trailOnLevel
	} from '$lib/tower';
	import Cookie from './Cookie.svelte';

	let {
		view,
		remembered = false,
		game = null,
		lastEvent = null,
		bumpKey = 0,
		onmove
	}: {
		/** The level to draw: the miner's own (live) or a remembered one. */
		view: LevelView;
		/** True for a level remembered from earlier: drawn read-only with the miner hidden. */
		remembered?: boolean;
		game?: GameState | null;
		lastEvent?: MoveEvent | null;
		/** Changes on every server update so a repeated blocked move replays the bump. */
		bumpKey?: number;
		/** When set, the tiles next to the miner are clickable. */
		onmove?: (direction: Direction) => void;
	} = $props();

	const CELL = 40;

	const level = $derived(view.level);
	const rows = $derived(view.rows);
	const width = $derived(rows.reduce((w, r) => Math.max(w, r.length), 0));
	const height = $derived(rows.length);
	const pos = $derived(game?.pos ?? { level: 0, x: 0, y: 0 });
	const here = $derived(!remembered && !!game && pos.level === level);
	const links = $derived(portalLinks(rows, view.portals));
	const trail = $derived(trailOnLevel(game?.path ?? [], level));
	const blocked = $derived(here && !!lastEvent && lastEvent.blocked && lastEvent.move === game?.moves);
	const facing = $derived<Direction>(lastEvent?.direction ?? 'UP');
	const underMiner = $derived(here ? tileKind(rows[pos.y]?.[pos.x] ?? '.') : 'void');
	const onStairs = $derived(underMiner === 'up' || underMiner === 'down');
	const label = $derived(
		`${levelName(level)} of ${view.levelCount}, ${width} by ${height}. ` +
			(remembered
				? 'Remembered from earlier; the miner is elsewhere.'
				: here
					? `Miner at column ${pos.x}, row ${pos.y}${onStairs ? `, standing on the stairs ${underMiner}; step off and back on to take them` : ''}.`
					: 'Miner position unknown.')
	);

	let angle = $state(0);
	$effect.pre(() => {
		const f = facing;
		angle = nextAngle(untrack(() => angle), f);
	});

	let lastLevel = untrack(() => level);
	let levelDir = $state(1);
	$effect.pre(() => {
		const l = level;
		if (l !== lastLevel) {
			levelDir = l > lastLevel ? 1 : -1;
			lastLevel = l;
		}
	});

	const reduceMotion = () => typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches;

	let bumpEl = $state<SVGGElement | null>(null);
	let lastBump = 0;
	$effect(() => {
		const key = bumpKey;
		const hit = blocked;
		const dir = lastEvent?.direction;
		const replay = key !== lastBump;
		lastBump = key;
		if (!bumpEl || !hit || !dir || !replay || reduceMotion() || typeof bumpEl.animate !== 'function') return;
		const { dx, dy } = DIRECTION_DELTA[dir];
		bumpEl.animate(
			[{ transform: 'translate(0px, 0px)' }, { transform: `translate(${dx * 7}px, ${dy * 7}px)` }, { transform: 'translate(0px, 0px)' }],
			{ duration: 180, easing: 'ease-out' }
		);
	});

	const cx = (x: number) => x * CELL + CELL / 2;
	const cy = (y: number) => y * CELL + CELL / 2;

	/** Tiles next to the miner on this level that a click can step onto. */
	const taps = $derived(
		onmove && here
			? (['UP', 'RIGHT', 'DOWN', 'LEFT'] as Direction[])
					.map((d) => ({ d, x: pos.x + DIRECTION_DELTA[d].dx, y: pos.y + DIRECTION_DELTA[d].dy }))
					.filter((t) => t.x >= 0 && t.y >= 0 && t.x < width && t.y < height)
			: []
	);
</script>

<div class="w-full" style="max-width: {Math.max(280, width * 56)}px">
	{#key level}
		<svg
			viewBox="0 0 {width * CELL} {height * CELL}"
			class="block h-auto w-full rounded-xl border border-indigo-200 bg-indigo-50 shadow-sm {remembered ? 'opacity-70' : ''}"
			role="img"
			aria-label={label}
			data-testid="board"
			data-level={level}
			data-remembered={remembered}
			in:fly={{ y: reduceMotion() ? 0 : levelDir * -24, duration: reduceMotion() ? 0 : 220 }}
		>
			{#each { length: height } as _, y (y)}
				{#each { length: width } as __, x (x)}
					<rect x={x * CELL} y={y * CELL} width={CELL} height={CELL} fill={(x + y) % 2 === 0 ? '#eef2ff' : '#e0e7ff'} />
				{/each}
			{/each}

			{#each rows as row, y (y)}
				{#each { length: width } as _, x (x)}
					{@const glyph = row[x] ?? '.'}
					{@const kind = tileKind(glyph)}
					{#if kind !== 'void'}
						<g aria-hidden="true" data-testid="tile-{kind}">
							<rect x={x * CELL + 3} y={y * CELL + 3} width={CELL - 6} height={CELL - 6} rx="9" fill="#cbd5e1" stroke="#94a3b8" stroke-width="1.5" />
							<rect x={x * CELL + 7} y={y * CELL + 7} width={CELL - 22} height="6" rx="3" fill="#f1f5f9" opacity="0.8" />
							{#if kind === 'start'}
								<circle cx={cx(x)} cy={cy(y)} r="12" fill="none" stroke="#4f46e5" stroke-width="3" />
								<text x={cx(x)} y={cy(y) + 5.5} text-anchor="middle" font-size="15" font-weight="800" fill="#312e81">S</text>
							{:else if kind === 'finish'}
								<Cookie cx={cx(x)} cy={cy(y)} r={13} />
							{:else if kind === 'up'}
								<polygon
									points="{cx(x)},{cy(y) - 12} {cx(x) + 11},{cy(y) + 8} {cx(x) - 11},{cy(y) + 8}"
									fill="#2563eb"
									stroke="#fff"
									stroke-width="2"
									stroke-linejoin="round"
								/>
							{:else if kind === 'down'}
								<polygon
									points="{cx(x)},{cy(y) + 12} {cx(x) + 11},{cy(y) - 8} {cx(x) - 11},{cy(y) - 8}"
									fill="#e11d48"
									stroke="#fff"
									stroke-width="2"
									stroke-linejoin="round"
								/>
							{/if}
						</g>
					{/if}
				{/each}
			{/each}

			{#each links.pipes as p (p.glyph)}
				<line
					x1={cx(p.a.x)}
					y1={cy(p.a.y)}
					x2={cx(p.b.x)}
					y2={cy(p.b.y)}
					stroke={portalColor(p.glyph).ring}
					stroke-width="5"
					stroke-linecap="round"
					stroke-dasharray="1 9"
					opacity="0.9"
					aria-hidden="true"
					data-testid="pipe"
				/>
			{/each}

			{#each links.holes as h (`${h.x},${h.y}`)}
				{@const c = portalColor(h.glyph)}
				<g aria-hidden="true" data-testid="hole">
					<circle cx={cx(h.x)} cy={cy(h.y)} r="13" fill={c.hole} stroke={c.ring} stroke-width="3.5" />
					<text x={cx(h.x)} y={cy(h.y) + 4.5} text-anchor="middle" font-size="12" font-weight="800" fill={c.ring}>{h.glyph}</text>
					{#if h.twinElsewhere}
						<g data-testid="hidden-twin">
							<rect x={h.x * CELL + CELL - 17} y={h.y * CELL + 1} width="16" height="12" rx="6" fill="#fff" stroke={c.hole} stroke-width="1.5" />
							<text x={h.x * CELL + CELL - 9} y={h.y * CELL + 10.5} text-anchor="middle" font-size="9" font-weight="700" fill="#1e1b4b">?</text>
						</g>
					{/if}
				</g>
			{/each}

			{#each trail as p (`${p.x},${p.y}`)}
				<circle cx={cx(p.x)} cy={cy(p.y)} r="5" fill="#6366f1" opacity="0.35" aria-hidden="true" />
			{/each}

			{#each taps as t (t.d)}
				<rect
					role="presentation"
					x={t.x * CELL}
					y={t.y * CELL}
					width={CELL}
					height={CELL}
					rx="9"
					fill="#6366f1"
					fill-opacity="0.001"
					class="tap"
					data-testid="tap-{t.d.toLowerCase()}"
					onclick={() => onmove?.(t.d)}
				/>
			{/each}

			{#if here}
				{#if blocked}
					<rect
						x={pos.x * CELL + 2}
						y={pos.y * CELL + 2}
						width={CELL - 4}
						height={CELL - 4}
						rx="8"
						fill="none"
						stroke="#e11d48"
						stroke-width="3"
						aria-hidden="true"
						data-testid="bump"
					/>
				{/if}
				<g class="miner" style="transform: translate({cx(pos.x)}px, {cy(pos.y)}px)" aria-hidden="true" data-testid="miner">
					<g bind:this={bumpEl}>
						<g style="transform: rotate({angle}deg); transition: transform 120ms ease-out">
							<polygon
								points="0,-15 12,12 0,6 -12,12"
								fill="#4338ca"
								stroke="#fff"
								stroke-width="2"
								stroke-linejoin="round"
								transform={onStairs ? 'scale(0.7)' : undefined}
								data-testid={onStairs ? 'miner-on-stairs' : undefined}
							/>
						</g>
					</g>
				</g>
			{/if}
		</svg>
	{/key}
</div>

<style>
	.miner {
		transition: transform 120ms ease-out;
	}
	.tap {
		cursor: pointer;
	}
	.tap:hover {
		fill-opacity: 0.2;
	}
	@media (prefers-reduced-motion: reduce) {
		.miner {
			transition: none;
		}
	}
</style>
