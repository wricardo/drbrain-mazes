<script lang="ts">
	import { pageTitle } from '$lib/brand';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { getContextClient } from '@urql/svelte';
	import { unwrap } from '$lib/api';
	import { getAdminKey, setAdminKey } from '$lib/admin';
	import Tower from '$lib/components/Tower.svelte';
	import { describeError, type DescribedError } from '$lib/errors';
	import {
		MAX_LEVELS,
		MAX_SIDE,
		MIN_SIDE,
		PAINT_GLYPHS,
		blankCells,
		clampSide,
		formatLevel,
		glyphLabel,
			localIssues,
	paintCell,
		parseLevel,
		resizeLevels,
		type Cells
	} from '$lib/levels';
	import {
		CREATE_MAP_MUTATION,
		DELETE_MAP_MUTATION,
		ADMIN_MAP_QUERY,
		UPDATE_MAP_MUTATION,
		VALIDATE_MAP_MUTATION
	} from '$lib/queries';
	import { DIRECTION_ARROW, portalColor, tileKind } from '$lib/tower';
	import type { AdminMap, MapInput, MapValidationResult } from '$lib/types';

	const client = getContextClient();

	let id = $state('');
	let name = $state('');
	let description = $state('');
	let difficulty = $state('easy');
	let maxMoves = $state(200);
	let width = $state(8);
	let height = $state(8);
	let levels = $state<Cells[]>([blankCells(8, 8)]);
	let active = $state(0);

	let brush = $state('o');
	let painting = $state(false);
	let existing = $state(false);
	let adminKey = $state(getAdminKey());
	let busy = $state(false);
	let error = $state<DescribedError | null>(null);
	let notice = $state('');
	let validation = $state<MapValidationResult | null>(null);

	const issues = $derived(localIssues(levels));

	onMount(async () => {
		const params = $page.url.searchParams;
		const mapId = params.get('map');
		if (!mapId) return;
		try {
			const data = await unwrap<{ adminMap: AdminMap | null }>(client.query(ADMIN_MAP_QUERY, { id: mapId }));
			if (!data.adminMap) throw new Error(`Map "${mapId}" not found`);
			load(data.adminMap, ['1', 'true', 'yes'].includes((params.get('duplicate') ?? '').toLowerCase()));
		} catch (e) {
			error = describeError(e);
		}
	});

	function load(m: AdminMap, duplicate: boolean) {
		width = m.width;
		height = m.height;
		levels = m.levels.map((rows) => parseLevel(rows, m.width, m.height));
		active = 0;
		id = duplicate ? `${m.id}_copy` : m.id;
		name = duplicate ? `${m.name} (copy)` : m.name;
		description = m.description;
		difficulty = m.difficulty;
		maxMoves = m.maxMoves;
		existing = !duplicate;
		validation = null;
	}

	function invalidate() {
		validation = null;
		notice = '';
	}

	function paint(x: number, y: number) {
		invalidate();
		// S and F are unique: painting one elsewhere moves it.
		if (brush === 'S' || brush === 'F') {
			for (const cells of levels) for (const row of cells) for (let i = 0; i < row.length; i++) if (row[i] === brush) row[i] = 'o';
		}
		// Stairs stay paired: painting U/D also places or removes the partner on the adjacent level.
		paintCell(levels, active, x, y, brush);
	}

	function resize(nextWidth: number, nextHeight: number) {
		width = clampSide(nextWidth);
		height = clampSide(nextHeight);
		levels = resizeLevels(levels, width, height);
		invalidate();
	}

	function addLevel() {
		if (levels.length >= MAX_LEVELS) return;
		levels = [...levels, blankCells(width, height)];
		active = levels.length - 1;
		invalidate();
	}

	function removeLevel() {
		if (levels.length <= 1) return;
		if (!confirm(`Remove level ${active + 1}?`)) return;
		levels = levels.filter((_, i) => i !== active);
		active = Math.min(active, levels.length - 1);
		invalidate();
	}

	function clearLevel() {
		if (!confirm(`Clear level ${active + 1}?`)) return;
		levels[active] = blankCells(width, height);
		invalidate();
	}

	function toInput(): MapInput {
		return {
			id: id.trim(),
			name: name.trim(),
			description,
			difficulty,
			levels: levels.map(formatLevel),
			maxMoves: Math.floor(maxMoves)
		};
	}

	async function run(fn: () => Promise<void>) {
		busy = true;
		error = null;
		notice = '';
		try {
			await fn();
		} catch (e) {
			error = describeError(e);
		} finally {
			busy = false;
		}
	}

	const validate = () =>
		run(async () => {
			const data = await unwrap<{ validateMap: MapValidationResult }>(client.mutation(VALIDATE_MAP_MUTATION, { map: toInput() }));
			validation = data.validateMap;
		});

	const save = () =>
		run(async () => {
			if (existing) await unwrap(client.mutation(UPDATE_MAP_MUTATION, { map: toInput() }));
			else await unwrap(client.mutation(CREATE_MAP_MUTATION, { map: toInput() }));
			existing = true;
			notice = `Saved map "${id.trim()}".`;
		});

	const remove = () =>
		run(async () => {
			if (!confirm(`Delete map "${id}"? Existing sessions keep their own copy.`)) return;
			await unwrap(client.mutation(DELETE_MAP_MUTATION, { id: id.trim() }));
			await goto('/maps');
		});

	function cellClass(glyph: string): string {
		switch (tileKind(glyph)) {
			case 'void':
				return 'bg-indigo-50 hover:bg-indigo-100';
			case 'start':
				return 'bg-indigo-600 text-white';
			case 'finish':
				return 'bg-amber-400 text-amber-950';
			case 'up':
				return 'bg-slate-300 text-blue-600';
			case 'down':
				return 'bg-slate-300 text-rose-600';
			case 'portal':
				return 'bg-slate-700';
			default:
				return 'bg-slate-300 hover:bg-slate-200';
		}
	}

	const field = 'mt-1 w-full rounded-lg border border-indigo-300 bg-white px-3 py-2 text-sm';
	const btn = 'rounded-lg px-4 py-2 text-sm font-semibold disabled:opacity-50';
</script>

<svelte:head><title>{pageTitle('Map editor')}</title></svelte:head>
<svelte:window onpointerup={() => (painting = false)} />

<div class="mx-auto max-w-6xl px-4 sm:px-6 py-6">
	<h1 class="text-2xl font-semibold text-indigo-950">Map editor</h1>
	<p class="mt-1 text-sm text-slate-700">
		Pick a glyph, paint each level, validate to see the par, then save. Saving needs the admin key.
	</p>

	{#if error}
		<p role="alert" class="mt-3 rounded-lg border border-rose-300 bg-rose-50 px-3 py-2 text-sm text-rose-900" data-testid="editor-error">
			<strong>{error.title}</strong>{#if error.code}<span class="ml-1 font-mono text-xs">[{error.code}]</span>{/if}
			{#if error.message}— {error.message}{/if}
		</p>
	{/if}
	{#if notice}
		<p role="status" class="mt-3 rounded-lg border border-emerald-300 bg-emerald-50 px-3 py-2 text-sm text-emerald-900">{notice}</p>
	{/if}

	<div class="mt-5 grid gap-6 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)]">
		<section aria-label="Level painter" class="space-y-3">
			<div class="flex flex-wrap items-center gap-2" role="radiogroup" aria-label="Glyph to paint">
				{#each PAINT_GLYPHS as g (g)}
					{@const kind = tileKind(g)}
					<button
						type="button"
						role="radio"
						aria-checked={brush === g}
						aria-label={glyphLabel(g)}
						title="{glyphLabel(g)} ({g})"
						data-testid="brush-{g}"
						class="flex h-9 w-9 items-center justify-center rounded-lg border-2 text-sm font-bold {brush === g ? 'border-indigo-600 ring-2 ring-indigo-300' : 'border-indigo-200'} {cellClass(g)}"
						style={kind === 'portal' ? `color: ${portalColor(g).ring}` : undefined}
						onclick={() => (brush = g)}
					>
						{kind === 'up' ? DIRECTION_ARROW.UP : kind === 'down' ? DIRECTION_ARROW.DOWN : kind === 'void' ? '·' : kind === 'stone' ? '' : g}
					</button>
				{/each}
			</div>

			<div class="flex flex-wrap items-center gap-2" role="tablist" aria-label="Levels">
				{#each levels as _, i (i)}
					<button
						type="button"
						role="tab"
						aria-selected={active === i}
						data-testid="level-tab-{i}"
						class="rounded-lg border-2 px-3 py-1.5 text-sm {active === i ? 'border-indigo-600 bg-indigo-100 font-semibold' : 'border-indigo-200 bg-white'}"
						onclick={() => (active = i)}>Level {i + 1}</button
					>
				{/each}
				<button type="button" class="{btn} border border-indigo-300 text-indigo-900 hover:bg-indigo-50" disabled={levels.length >= MAX_LEVELS} onclick={addLevel} data-testid="add-level">Add level</button>
				<button type="button" class="{btn} border border-indigo-300 text-indigo-900 hover:bg-indigo-50" disabled={levels.length <= 1} onclick={removeLevel} data-testid="remove-level">Remove level</button>
			</div>

			<div class="grid items-start gap-4 sm:grid-cols-[minmax(0,1fr)_auto]">
				<div class="overflow-auto rounded-xl border border-indigo-200 bg-white p-2" role="tabpanel" aria-label="Level {active + 1}">
					<div
						class="inline-grid gap-px bg-indigo-200 p-px select-none touch-none"
						style="grid-template-columns: repeat({width}, 1.75rem)"
						role="grid"
						aria-label="Level {active + 1} cells, {width} by {height}"
					>
						{#each levels[active] as row, y}
							{#each row as glyph, x}
								<button
									type="button"
									class="flex h-7 w-7 items-center justify-center text-xs font-bold {cellClass(glyph)}"
									style={tileKind(glyph) === 'portal' ? `color: ${portalColor(glyph).ring}` : undefined}
									aria-label="Column {x}, row {y}: {glyphLabel(glyph)}"
									data-testid="cell-{x}-{y}"
									onpointerdown={() => {
										painting = true;
										paint(x, y);
									}}
									onpointerenter={() => painting && paint(x, y)}
									onkeydown={(e) => {
										if (e.key === 'Enter' || e.key === ' ') {
											e.preventDefault();
											paint(x, y);
										}
									}}
								>
									<span aria-hidden="true"
										>{tileKind(glyph) === 'up' ? DIRECTION_ARROW.UP : tileKind(glyph) === 'down' ? DIRECTION_ARROW.DOWN : tileKind(glyph) === 'void' || tileKind(glyph) === 'stone' ? '' : glyph}</span
									>
								</button>
							{/each}
						{/each}
					</div>
				</div>
				<Tower levelCount={levels.length} current={active} onpeek={(l) => (active = l)} />
			</div>

			<div class="flex flex-wrap items-end gap-3 text-sm">
				<label class="block">Width
					<input type="number" min={MIN_SIDE} max={MAX_SIDE} value={width} onchange={(e) => resize(e.currentTarget.valueAsNumber, height)} class="mt-1 block w-20 rounded-lg border border-indigo-300 px-2 py-1" />
				</label>
				<label class="block">Height
					<input type="number" min={MIN_SIDE} max={MAX_SIDE} value={height} onchange={(e) => resize(width, e.currentTarget.valueAsNumber)} class="mt-1 block w-20 rounded-lg border border-indigo-300 px-2 py-1" />
				</label>
				<button type="button" class="{btn} border border-indigo-300 text-indigo-900" onclick={clearLevel}>Clear level</button>
				<span class="text-xs text-slate-600">glyphs: <code>. o S F U D 1-9</code>; y grows downward; level 1 is the bottom; S must be on level 1 and F on the top level; stairs come in pairs (a U needs a D at the same x,y one level up) — painting one places its partner</span>
			</div>

			{#if issues.length}
				<ul class="text-xs text-amber-800 list-disc pl-5" data-testid="local-issues">
					{#each issues as issue}<li>{issue}</li>{/each}
				</ul>
			{/if}
		</section>

		<section aria-label="Map settings" class="space-y-3">
			<div class="grid grid-cols-2 gap-3">
				<label class="col-span-2 block text-sm">Map id <span class="text-xs text-slate-600">(a-z 0-9 _ -)</span>
					<input class={field} bind:value={id} oninput={invalidate} readonly={existing} placeholder="my_map" />
				</label>
				<label class="col-span-2 block text-sm">Name
					<input class={field} bind:value={name} oninput={invalidate} />
				</label>
				<label class="col-span-2 block text-sm">Description
					<textarea class={field} rows="2" bind:value={description} oninput={invalidate}></textarea>
				</label>
				<label class="block text-sm">Difficulty
					<select class={field} bind:value={difficulty} onchange={invalidate}>
						<option>easy</option><option>medium</option><option>hard</option>
					</select>
				</label>
				<label class="block text-sm">Max moves (1–10000)
					<input type="number" min="1" max="10000" class={field} bind:value={maxMoves} oninput={invalidate} />
				</label>
			</div>

			<div class="rounded-xl border border-indigo-200 bg-white p-3 space-y-3">
				<label class="block text-sm">Admin key
					<input
						type="password"
						class={field}
						bind:value={adminKey}
						oninput={(e) => setAdminKey(e.currentTarget.value)}
						autocomplete="off"
						placeholder="X-Admin-Key (kept for this tab only)"
					/>
				</label>
				<div class="flex flex-wrap gap-2">
					<button type="button" class="{btn} border border-indigo-300 text-indigo-900 hover:bg-indigo-50" disabled={busy} onclick={validate} data-testid="validate">Validate</button>
					<button type="button" class="{btn} bg-indigo-600 text-white hover:bg-indigo-700" disabled={busy || issues.length > 0 || !id.trim() || !name.trim()} onclick={save} data-testid="save">
						{existing ? 'Update map' : 'Create map'}
					</button>
					{#if existing}
						<button type="button" class="{btn} border border-rose-300 text-rose-800 hover:bg-rose-50" disabled={busy} onclick={remove}>Delete</button>
					{/if}
				</div>
			</div>

			{#if validation}
				<div
					class="rounded-xl border p-3 text-sm {validation.valid ? 'border-emerald-300 bg-emerald-50 text-emerald-900' : 'border-rose-300 bg-rose-50 text-rose-900'}"
					role="status"
					data-testid="validation"
				>
					<strong>{validation.valid ? 'Map is valid.' : 'Map has problems:'}</strong>
					{#if validation.issues.length}
						<ul class="mt-1 list-disc pl-5">
							{#each validation.issues as issue}<li>{issue.message}</li>{/each}
						</ul>
					{/if}
					{#if validation.valid || validation.parMoves !== null}
						<p class="mt-1" data-testid="validation-par">
							{validation.finishReachable && validation.parMoves !== null
								? `Par (shortest route): ${validation.parMoves} moves.`
								: 'The finish cannot be reached from the start.'}
						</p>
					{/if}
				</div>
			{/if}
		</section>
	</div>
</div>
