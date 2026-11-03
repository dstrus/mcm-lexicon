<!-- One menu, as SkyUI draws it: the name across the top, the pages down
     the left, and the options in two columns placed where the game puts
     them, each a label with its value at the right. The value is the open
     recording's where it has the option, else the menu's own; while
     editing, it changes in place, and any change records the option. The
     selected option's help and details sit at the foot, where SkyUI shows
     its info text. -->
<script lang="ts">
	import { goto, invalidate } from '$app/navigation';
	import { ApiError } from '#lib/api/client.ts';
	import type { Fixture, Option } from '#lib/api/types/catalog.ts';
	import { m } from '#lib/copy/messages.ts';
	import { menuHref } from '#lib/menus/api.ts';
	import { allowed, changed, edgeSpace, hex, show } from '#lib/menus/values.ts';
	import ChooseRecording from '#lib/recordings/ChooseRecording.svelte';
	import InlineValue from '#lib/recordings/InlineValue.svelte';
	import { addAll, addStep, getRecording, putDraft, recordingHref, recordingsDependency } from '#lib/recordings/api.ts';
	import { OpenRecording } from '#lib/recordings/open.svelte.ts';
	import type { Step } from '#lib/api/types/recordings.ts';
	import { known, recordable, sameOption, stepFor, changed as stepChanged, value as stepValue } from '#lib/recordings/steps.ts';
	import Diamond from '#lib/ui/Diamond.svelte';
	import { clipped } from '#lib/ui/clipped.ts';
	import { aboveHints } from '#lib/ui/fit.ts';
	import Switch from '#lib/ui/Switch.svelte';
	import { leave, param, setParams } from '#lib/ui/address.svelte.ts';
	import { Selection } from '#lib/ui/selection.svelte.ts';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const t = m.menu;
	const s = m.mcm;

	let menu = $derived(data.menu);
	let page = $derived(menu.pages[data.at]);
	let options = $derived(page?.options ?? []);
	let pageText = (p: { name: string; text: string }) => (p.name === '' ? t.mainPage : p.text);
	// A menu without pages has one, unnamed, and no page list.
	let paged = $derived(menu.pages.length > 1 || (menu.pages[0]?.name ?? '') !== '');

	// The page's slots, as the game places them: options and fixtures by
	// position, each in a row and a column of the game's grid (even
	// positions down the left, odd down the right), a slot the script
	// places at runtime in an unused row. Two slots in one cell, from
	// branches the game decides, take the next free rows. A page the script
	// places at runtime is one column, in the order its options are added,
	// without its headers.
	interface Slot {
		position: number;
		row: number;
		col: number;
		option?: Option;
		i?: number; // the option's position in the page
		fixture?: Fixture;
	}
	let placed = $derived(!!page?.placed);
	let slots = $derived.by((): Slot[] => {
		const out: Slot[] = options.map((option, i) => ({ position: placed ? option.position : i * 2, row: 0, col: 0, option, i }));
		if (placed) for (const fixture of page?.fixtures ?? []) out.push({ position: fixture.position, row: 0, col: 0, fixture });
		out.sort((a, b) => a.position - b.position);
		const taken = new Set<string>();
		for (const x of out) {
			x.col = x.position % 2;
			x.row = Math.floor(x.position / 2);
			while (taken.has(`${x.col}:${x.row}`)) x.row++;
			taken.add(`${x.col}:${x.row}`);
		}
		return out;
	});
	let twoColumns = $derived(placed && slots.some((x) => x.col === 1));
	// Rows are the options in reading order (down the left, then the right),
	// kept by their position in the page, which the address holds as ?item=.
	let rows = $derived(
		[...slots]
			.sort((a, b) => a.col - b.col || a.row - b.row)
			.filter((x) => x.option)
			.map((x) => ({ option: x.option!, i: x.i! }))
	);
	const selection = new Selection(() => rows, (r) => String(r.i), { param: 'item' });
	let current = $derived(selection.current);
	const selected = (i: number) => current?.i === i;

	// How many options share each type and label on the page, so a
	// duplicate says which of them it is.
	let sharing = $derived.by(() => {
		const n = new Map<string, number>();
		for (const o of options) n.set(`${o.type}\0${o.label}`, (n.get(`${o.type}\0${o.label}`) ?? 0) + 1);
		return n;
	});
	const typeLabel = (o: Option) => (o.control === 'stepper' ? m.option.types.stepper : m.option.types[o.type]);

	// The open recording, while editing: what it has of this page.
	const open = new OpenRecording();
	let chooser = $state<ChooseRecording>();
	const choose = () => chooser?.choose() ?? Promise.resolve(null);
	let recording = $derived(open.editing && !!open.name);
	// An option's step as this page makes it, and as the recording has it:
	// on a landing page, a step on the default page is the same option, and
	// a change to it stays on the page it's on.
	const named = (o: Option) => stepFor(menu.name, page?.name ?? '', o);
	const recorded = (o: Option) => open.find(named(o), page?.landing);
	const base = (o: Option) => {
		const r = recorded(o);
		return r ? { ...named(o), page: r.step.page } : named(o);
	};
	// What a value shows and starts from: the recorded value, else the
	// option's own.
	const valueNow = (o: Option) => recorded(o)?.step ?? base(o);
	const on = (o: Option) => valueNow(o).value.toLowerCase() === 'on';
	// A toggle flips in place when its value is known (recorded, saved, or
	// the mod's default); one that isn't is chosen outright, On or Off.
	const flippable = (o: Option) => o.type === 'toggle' && (!!recorded(o) || !!known(o));
	const canEdit = (o: Option) => recordable(o);

	let working = $state(false);
	let error = $state('');
	let note = $state('');
	// What covers the window's bottom edge, the info area and the hint strip
	// under it, so a row scrolled into view stops above both.
	let infoHeight = $state(0);
	let hintsHeight = $derived(infoHeight && (document.querySelector<HTMLElement>('.hints')?.offsetHeight ?? 0));
	let covered = $derived(infoHeight + hintsHeight);
	let inline = $state<InlineValue>();
	let editingCell = $derived(recording && param('mode') === 'edit' && current ? current.i : -1);

	// record puts the option in the open recording with this value: a new
	// step, or its step changed.
	async function record(o: Option, value: string) {
		if (working) return;
		working = true;
		error = '';
		try {
			await addStep(open.name, { ...base(o), value });
			await invalidate(recordingsDependency);
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
		} finally {
			working = false;
			leave('mode');
		}
	}
	const flip = (o: Option) => record(o, on(o) ? 'off' : 'on');
	function edit(r: { option: Option; i: number }) {
		selection.select(r);
		if (flippable(r.option)) return flip(r.option);
		if (r.option.type === 'text') return record(r.option, '');
		setParams({ item: String(r.i), mode: 'edit' });
	}
	// A records the selected option as it shows, asking which recording
	// first when none is chosen; an option whose value isn't known opens
	// its editor instead.
	async function add() {
		const r = current;
		if (!r || !canEdit(r.option) || working) return;
		if (!open.name && !(await choose())) return;
		if (recorded(r.option)) return;
		if (flippable(r.option) || known(r.option) || r.option.type === 'text') return record(r.option, valueNow(r.option).value);
		setParams({ mode: 'edit' });
	}

	// Record all settings on this page: every option whose value is known
	// and whose label can be matched exactly; ones recorded already keep
	// their values.
	let recordableNow = $derived(options.filter((o) => recordable(o) && o.type !== 'text' && !!known(o)));
	// The draft's unsaved steps for this menu, on every page: what Discard
	// takes back.
	const ofMenu = (s: Step) => s.mod.toLowerCase() === menu.name.toLowerCase();
	let unsavedHere = $derived(open.editing ? open.steps.filter((s) => ofMenu(s) && open.dirty(s)).length : 0);

	// discardMenu takes this menu's unsaved steps out of the draft: a step
	// the saved recording has goes back to its saved value, and one it
	// hasn't is removed. The rest of the draft is left as it is.
	async function discardMenu() {
		if (working || !open.name) return;
		working = true;
		error = '';
		note = '';
		try {
			const ed = await getRecording(open.name);
			const saved = ed.recording?.steps ?? [];
			const steps: Step[] = [];
			let n = 0;
			for (const s of ed.draft.steps) {
				const was = ofMenu(s) ? saved.find((w) => sameOption(w, s)) : undefined;
				if (!ofMenu(s) || (was && was.value === s.value && was.side === s.side)) steps.push(s);
				else if (was) (steps.push(was), n++);
				else n++;
			}
			await putDraft(open.name, { ...ed.draft, steps });
			note = t.discardedHere({ count: n });
			await invalidate(recordingsDependency);
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
		} finally {
			working = false;
		}
	}

	async function recordAll() {
		if (working) return;
		if (!open.name && !(await choose())) return;
		working = true;
		error = '';
		note = '';
		try {
			const r = await addAll(open.name, recordableNow.map(base));
			const skipped = options.length - recordableNow.length;
			note = [t.recordedAdded({ count: r.added, recording: open.name }), r.kept ? t.recordedKept({ count: r.kept }) : '', skipped ? t.recordedSkipped({ count: skipped }) : '']
				.filter(Boolean)
				.join(' ');
			await invalidate(recordingsDependency);
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
		} finally {
			working = false;
		}
	}

	useShortcuts(() =>
		editingCell >= 0
			? [
					// The cell's field takes these first; they're here so the strip shows them.
					{ key: 'Enter', label: m.edit.apply, disabled: working, run: () => inline?.done() },
					{ key: 'Escape', shown: 'Esc', label: m.common.cancel, run: () => leave('mode') }
				]
			: [
					{ key: 'ArrowDown', label: m.common.next, hidden: true, run: () => selection.move(1) },
					{ key: 'ArrowUp', label: m.common.previous, hidden: true, run: () => selection.move(-1) },
					...(recording && current && canEdit(current.option)
						? flippable(current.option)
							? [{ key: ' ', shown: 'Space', label: on(current.option) ? m.edit.turnOff : m.edit.turnOn, disabled: working, run: () => flip(current!.option) }]
							: current.option.type === 'text'
								? []
								: [{ key: 'e', shown: 'E', label: m.edit.edit, disabled: working, run: () => setParams({ mode: 'edit' }) }]
						: []),
					...(open.editing && current && canEdit(current.option) && !recorded(current.option)
						? [{ key: 'a', shown: 'A', label: open.name ? m.adding.addTo({ recording: open.name }) : m.adding.addToAny, disabled: working, run: add }]
						: []),
					{ key: 'Escape', shown: 'Esc', label: t.up, run: () => goto('/menus') }
				]
	);
</script>

<h1 class="caps">{menu.nameText}</h1>
<p class="muted sub">{menu.mod}{#if menu.plugin}{' · '}{menu.plugin}{/if}{' · '}{m.menus.kinds[menu.kind]}</p>
{#if !menu.active}<p class="notice warn">{m.menus.inactive({ plugin: menu.plugin })}</p>{/if}
{#if error}<p class="notice error" role="alert">{error}</p>{/if}

{#snippet value(o: Option, i: number)}
	{#if recording}
		{@const rec = recorded(o)}
		<span class="val" class:changed={!!rec && stepChanged(rec.step, o)}>
			<!-- What the recording holds of it: a hollow diamond for a step
			     that's saved, an amber one for a step the draft hasn't saved. -->
			{#if rec && canEdit(o) && editingCell !== i}
				{@const dirty = open.dirty(rec.step)}
				<span class="mark" title={dirty ? m.adding.unsavedStep({ recording: open.name }) : m.adding.inRecording({ recording: open.name })}>
					{#if dirty}<Diamond size={7} fill="var(--warn)" />{:else}<Diamond size={7} hollow />{/if}
					<span class="visually-hidden">{dirty ? m.adding.unsavedStep({ recording: open.name }) : m.adding.inRecording({ recording: open.name })}</span>
				</span>
			{/if}
			{#if !canEdit(o)}
				<span class="muted" title={m.adding.cantRecord}>—</span>
			{:else if flippable(o)}
				<span class="flip" title={rec ? undefined : m.adding.notYetTitle({ recording: open.name })}>
					{#if rec}{stepValue(rec.step, o)}{/if}
					<Switch checked={on(o)} dim={!rec} disabled={working} label={m.edit.toggleLabel({ option: o.text })} onchange={() => flip(o)} />
				</span>
			{:else if editingCell === i}
				<InlineValue bind:this={inline} kind={base(o).kind} option={o} value={valueNow(o).value} apply={(v) => record(o, v)} cancel={() => leave('mode')} />
			{:else if rec}
				<button type="button" class="cell" title={m.edit.editValue} disabled={o.type === 'text'} onclick={(e) => (e.stopPropagation(), edit({ option: o, i }))}>
					{#if o.type === 'color'}<span class="swatch" style:background={hex(Number(rec.step.value))}></span>{/if}{stepValue(rec.step, o)}
				</button>
			{:else}
				<button type="button" class="cell unrecorded" title={m.adding.notYetTitle({ recording: open.name })} aria-label={m.adding.recordThis({ option: o.text })} onclick={(e) => (e.stopPropagation(), edit({ option: o, i }))}>—</button>
			{/if}
		</span>
	{:else}
		{@const k = known(o)}
		<span class="val" class:changed={changed(o)}>
			{#if o.type === 'toggle' && k}
				<Switch checked={k.kind === 'bool' && k.bool} disabled label={m.edit.toggleLabel({ option: o.text })} onchange={() => {}} />
			{:else if o.type === 'text'}
				{#if o.valueText || o.value}{o.valueText || o.value}{/if}
			{:else if k}
				{#if o.type === 'color'}<span class="swatch" style:background={hex(k.number)}></span>{/if}{show(o, k)}
			{:else}
				<span class="muted">—</span>
			{/if}
		</span>
	{/if}
{/snippet}

{#snippet slot(x: Slot)}
	{#if x.fixture?.kind === 'header'}
		<li class="header caps" style:grid-row={x.row + 1} style:grid-column={x.col + 1}>{x.fixture.text || x.fixture.label}{#if x.fixture.certainty !== 'exact'}<span class="word">{x.fixture.certainty}</span>{/if}</li>
	{:else if x.fixture}
		<li class="empty" aria-hidden="true" style:grid-row={x.row + 1} style:grid-column={x.col + 1}></li>
	{:else if x.option}
		{@const o = x.option}
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<li class="opt" role="option" aria-selected={selected(x.i!)} style:grid-row={x.row + 1} style:grid-column={x.col + 1} onclick={() => !(editingCell >= 0) && selection.select({ option: o, i: x.i! })}>
			<span class="marker">{#if selected(x.i!)}<Diamond />{/if}</span>
			<span class="label">
				{o.text || o.label || m.values.noLabel}
				{#if o.index > 0}<span class="word">{t.indexOf({ n: o.index, count: sharing.get(`${o.type}\0${o.label}`) ?? o.index })}</span>{/if}
				{#if o.certainty !== 'exact'}<span class="word">{o.certainty}</span>{/if}
			</span>
			{@render value(o, x.i!)}
		</li>
	{/if}
{/snippet}

<div class="mcm" class:paged style:--hints="{hintsHeight}px">
	{#if paged}
		<nav class="pages" aria-label={t.pages}>
			<ul>
				{#each menu.pages as p, i (i)}
					{#if !p.sameAs}
					<li>
						<a class="caps" href={menuHref(menu.id, i)} aria-current={i === data.at ? 'page' : undefined}>
							<span class="marker">{#if i === data.at}<Diamond />{/if}</span>
							<span class="clip" use:clipped>{pageText(p)}</span>
							<span class="count">{p.options.length}</span>
						</a>
					</li>
					{/if}
				{/each}
			</ul>
		</nav>
	{/if}
	<div class="panel">
		{#if !options.length}
			<p class="none muted">{paged ? t.noOptions : t.noPages}</p>
		{:else}
			<ul class="grid" class:two={twoColumns} role="listbox" aria-label={t.option} style:--covered="{covered}px">
				{#each slots as x, i (i)}{@render slot(x)}{/each}
			</ul>
		{/if}
		{#if open.editing && options.length}
			<div class="tools">
				<button type="button" class="button" disabled={working || !recordableNow.length} onclick={recordAll}>{t.recordAll}</button>
				{#if unsavedHere}<button type="button" class="textbutton" disabled={working} onclick={discardMenu}>{t.discardHere({ count: unsavedHere })}</button>{/if}
				{#if note}<p class="muted" role="status">{note}</p>{/if}
			</div>
		{/if}
	</div>
	<div class="info" aria-live="polite" use:aboveHints bind:offsetHeight={infoHeight}>
		{#if current}
			{@const o = current.option}
			{@const rec = recorded(o)}
			{#key current.i}
				<div class="enter">
					{#if o.helpText}<p class="help">{o.helpText}</p>{:else}<p class="help muted">{s.noHelp}</p>{/if}
					<p class="details muted">
						<span class="caps">{m.option.selector}</span> <span class="raw">{o.label}</span>{#if edgeSpace(o.label)} <span class="warn">{m.values.edgeSpace}</span>{/if}
						{#if o.index > 0}<span class="sep">·</span> <span class="caps">{m.option.index}</span> {o.index}{/if}
						<span class="sep">·</span> <span class="caps">{m.option.type}</span> {typeLabel(o)}
						<span class="sep">·</span> {allowed(o)}
						{#if o.default}<span class="sep">·</span> <span class="caps">{m.option.default}</span> {show(o, o.default)}{/if}
						{#if o.saved}<span class="sep">·</span> <span class="caps">{m.option.saved}</span> <span class:changed={changed(o)}>{show(o, o.saved)}</span>{/if}
						<span class="sep">·</span> <span class="caps">{m.option.definedIn}</span> <span class="raw">{o.source.path}</span>{#if o.source.line}:{o.source.line}{/if}{#if o.source.archive} {m.common.inArchive({ archive: o.source.archive })}{/if}
						{#if rec}<span class="sep">·</span> <a href={recordingHref(open.name) + `?item=${rec.n}`}>{s.stepOf({ n: rec.n, recording: open.name })}</a>{/if}
					</p>
					{#if o.certainty !== 'exact'}<p class="notice warn">{o.certainty === 'pattern' ? m.option.pattern : m.option.unknown}</p>{/if}
					{#if o.type === 'text' && o.valueChanges}<p class="notice">{m.option.valueChanges}</p>{/if}
					{#if recording && !canEdit(o)}<p class="notice">{m.adding.cantRecord}</p>{/if}
				</div>
			{/key}
		{:else if options.length}
			<p class="help muted">{s.pick}</p>
		{/if}
	</div>
</div>

<ChooseRecording bind:this={chooser} />

<style>
	h1 {
		font-size: var(--step-3);
		font-weight: 600;
		line-height: 1;
	}
	.sub {
		margin-top: var(--space-2);
	}
	.notice {
		margin-top: var(--space-2);
		max-width: 70ch;
	}
	/* The config panel: pages down the left, options to their right, and the
	   info text under both, as SkyUI's. */
	.mcm {
		display: grid;
		grid-template-columns: minmax(0, 1fr);
		grid-template-rows: auto auto;
		gap: var(--space-4) var(--space-6);
		margin-top: var(--space-5);
		/* The info area ends the page at the hint strip's top edge: this
		   takes back the part of main's bottom padding (5rem) the strip
		   doesn't cover, so scrolled to the end there's no gap. */
		margin-bottom: calc(var(--hints, 5rem) - 5rem);
	}
	.mcm.paged {
		grid-template-columns: 13rem minmax(0, 1fr);
	}
	.pages ul,
	.grid {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.pages a {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-1) 0;
		font-size: var(--step-1);
		color: var(--muted);
		text-decoration: none;
	}
	.pages a:hover {
		color: var(--ink);
	}
	.pages a[aria-current='page'] {
		color: var(--accent);
		font-weight: 600;
	}
	.pages .count {
		margin-left: auto;
		padding-left: var(--space-2);
		font-size: var(--step--1);
		font-weight: 400;
		font-variant-numeric: tabular-nums;
		color: var(--muted);
	}
	.marker {
		flex: none;
		display: inline-flex;
		width: 9px;
	}
	.clip {
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	/* The options, on the game's grid: a row and a column each, so the two
	   columns line up row for row, and a slot nothing fills stays blank. */
	.grid {
		display: grid;
		grid-template-columns: minmax(0, 1fr);
		grid-auto-rows: minmax(2.4rem, auto);
		gap: 0 var(--space-6);
		align-items: stretch;
	}
	.grid.two {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}
	.grid li {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		min-height: 2.4rem;
		padding: var(--space-1) var(--space-2);
		border-bottom: 1px solid var(--rule);
	}
	.grid .opt {
		cursor: default;
	}
	.grid .opt:hover:not([aria-selected='true']) {
		background: var(--glass);
	}
	.grid .opt[aria-selected='true'] {
		background: var(--select);
	}
	.label {
		flex: 1 1 auto;
		min-width: 0;
		overflow-wrap: anywhere;
	}
	/* The value at the right, as the game shows it. */
	.val {
		flex: none;
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		max-width: 50%;
		margin-left: auto;
		text-align: right;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.changed {
		color: var(--changed);
	}
	/* A header is a group heading in the list; a gap is a row with nothing. */
	.header {
		color: var(--accent);
		font-size: var(--step--1);
		font-weight: 500;
		border-bottom-color: var(--rule-strong);
	}
	.empty {
		border-bottom-color: transparent;
	}
	/* Certainty, and a duplicate's number: a word after the label, never a color. */
	.word {
		display: inline-block;
		margin-left: var(--space-2);
		font-size: var(--step--1);
		font-weight: 400;
		color: var(--muted);
	}
	.swatch {
		display: inline-block;
		width: 9px;
		height: 9px;
		margin-right: var(--space-1);
		border: 1px solid var(--rule-strong);
	}
	.flip {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
	}
	/* Any other value's cell is a button that turns into its editor, drawn
	   as a field at rest so it reads as editable before it's clicked: the
	   same glass and hairline as an input, a dash in it where the option
	   isn't recorded yet. A text option's cell, which can't be changed, is
	   plain. */
	.cell {
		min-width: 3.5rem;
		padding: 0.1em var(--space-2);
		background: var(--glass-strong);
		border: 1px solid var(--rule-strong);
		border-radius: var(--radius);
		font: inherit;
		color: inherit;
		text-align: right;
		cursor: text;
	}
	.cell:hover:not(:disabled),
	.cell:focus-visible {
		border-color: var(--accent);
		outline: none;
	}
	.cell:disabled {
		padding: 0;
		min-width: 0;
		background: none;
		border-color: transparent;
		cursor: default;
	}
	.cell.unrecorded {
		color: var(--muted);
	}
	.mark {
		flex: none;
		display: inline-flex;
		color: var(--muted);
	}
	.visually-hidden {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}
	.none {
		margin: 0;
	}
	.tools {
		display: flex;
		align-items: baseline;
		flex-wrap: wrap;
		gap: var(--space-3);
		margin-top: var(--space-4);
	}
	.tools p {
		margin: 0;
		font-size: var(--step--1);
	}
	/* The info text, as SkyUI's: the selected option's help, then its
	   details in one line of label caps and raw text. On a page taller
	   than the window it sticks above the hint strip, so the selected
	   option and what it is are always in view together. */
	.info {
		grid-column: 1 / -1;
		position: sticky;
		min-height: 5.5rem;
		/* Out to main's edges, so nothing shows past it as the page scrolls. */
		margin-inline: calc(-1 * var(--space-6));
		padding: var(--space-3) var(--space-6);
		background: var(--panel);
		border-top: 1px solid var(--rule-strong);
	}
	.grid .opt {
		scroll-margin-bottom: var(--covered, 0px);
	}
	/* Help keeps the lines its author broke it into. */
	.help {
		margin: 0;
		max-width: 70ch;
		white-space: pre-line;
	}
	.details {
		margin: var(--space-2) 0 0;
		font-size: var(--step--1);
		line-height: 1.7;
		overflow-wrap: anywhere;
	}
	.details .caps {
		font-size: inherit;
	}
	.details .sep {
		margin: 0 var(--space-1);
	}
	.raw {
		font-family: var(--font-mono);
	}
	.enter {
		animation: rise 180ms cubic-bezier(0.16, 1, 0.3, 1);
	}
	@keyframes rise {
		from {
			opacity: 0;
			transform: translateY(6px);
		}
	}
	.warn {
		color: var(--warn);
	}
	@media (max-width: 60rem) {
		.mcm.paged {
			grid-template-columns: minmax(0, 1fr);
		}
		.pages ul {
			display: flex;
			flex-wrap: wrap;
			gap: 0 var(--space-4);
		}
		.pages .count {
			margin-left: var(--space-1);
		}
	}
</style>
