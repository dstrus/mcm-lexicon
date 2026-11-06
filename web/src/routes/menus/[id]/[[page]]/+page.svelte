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
	import type { Fixture, Note, Option } from '#lib/api/types/catalog.ts';
	import { m } from '#lib/copy/messages.ts';
	import { menuHref } from '#lib/menus/api.ts';
	import { column } from '#lib/menus/column.svelte.ts';
	import { allowed, changed, edgeSpace, hex, show } from '#lib/menus/values.ts';
	import ChooseRecording from '#lib/recordings/ChooseRecording.svelte';
	import InlineValue from '#lib/recordings/InlineValue.svelte';
	import { addAll, addStep, getRecording, putDraft, recordingHref, recordingsDependency, setClicks } from '#lib/recordings/api.ts';
	import { OpenRecording } from '#lib/recordings/open.svelte.ts';
	import type { Step } from '#lib/api/types/recordings.ts';
	import { known, recordable, stepFor, changed as stepChanged, value as stepValue } from '#lib/recordings/steps.ts';
	import Diamond from '#lib/ui/Diamond.svelte';
	import Path from '#lib/ui/Path.svelte';
	import { clipped } from '#lib/ui/clipped.ts';
	import { aboveHints } from '#lib/ui/fit.ts';
	import Switch from '#lib/ui/Switch.svelte';
	import { leave, param, setParams } from '#lib/ui/address.svelte.ts';
	import { Selection } from '#lib/ui/selection.svelte.ts';
	import { holdSave, useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const t = m.menu;

	// A menu's note as a sentence, or "" for one that's said elsewhere.
	function menuNote(n: Note): string {
		const w = m.menus;
		switch (n.kind) {
			case 'no-plugin':
				return w.noPlugin;
			case 'plugin-guessed':
				return w.pluginGuessed;
			case 'name-from-script':
				return w.nameFromScript;
			case 'pages-guessed':
				return w.pagesGuessed;
			case 'framework':
				return w.framework({ script: n.detail });
			case 'source-stale':
				return w.sourceStale;
			case 'overridden':
				return w.overridden({ mod: n.detail });
		}
		return '';
	}
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
			.map((x) => ({ option: x.option!, i: x.i!, row: x.row, col: x.col }))
	);
	const selection = new Selection(() => rows, (r) => String(r.i), { param: 'item' });
	let current = $derived(selection.current);
	const selected = (i: number) => current?.i === i;

	// The arrow keys move as on the game's grid: ↑ ↓ within a column, ← →
	// to the nearest option in the column beside, and ← from the left
	// column to the page list, then to the column's list of menus. In
	// either list ↑ ↓ open the page or menu above or below, and → comes
	// back toward the options.
	let zone = $state<'options' | 'pages' | 'menus'>(column.active ? 'menus' : 'options');
	let inPages = $derived(zone === 'pages');
	$effect(() => {
		column.active = zone === 'menus';
	});
	// A click in the column, or leaving it, takes the keys back to the options.
	$effect(() => {
		if (!column.active && zone === 'menus') zone = 'options';
	});
	let listed = $derived(menu.pages.map((p, i) => ({ p, i })).filter((x) => !x.p.sameAs).map((x) => x.i));
	// pick selects the option at i, as a click on it does.
	function pick(i: number) {
		zone = 'options';
		const r = rows.find((r) => r.i === i);
		if (r) selection.select(r);
	}
	// Where the keys are headed, while that menu or page loads, so presses
	// in quick succession each move one further rather than from the same
	// place.
	let headed: { menu: string; page: number } | null = null;
	$effect(() => {
		if (headed && headed.menu === menu.id && (headed.page < 0 || headed.page === data.at)) headed = null;
	});
	function vertical(by: 1 | -1) {
		if (zone === 'menus') {
			const from = headed?.menu ?? menu.id;
			const at = column.ids.indexOf(from);
			const next = column.ids[Math.min(column.ids.length - 1, Math.max(0, at + by))];
			if (next && next !== from) {
				headed = { menu: next, page: -1 };
				goto(menuHref(next), { replace: true });
			}
			return;
		}
		if (inPages) {
			const from = headed?.menu === menu.id && headed.page >= 0 ? headed.page : data.at;
			const at = listed.indexOf(from);
			const next = listed[Math.min(listed.length - 1, Math.max(0, at + by))];
			if (next !== undefined && next !== from) {
				headed = { menu: menu.id, page: next };
				goto(menuHref(menu.id, next), { replace: true });
			}
			return;
		}
		const c = current;
		if (!c) return;
		const here = rows.filter((r) => r.col === c.col);
		const next = by > 0 ? here.find((r) => r.row > c.row) : here.findLast((r) => r.row < c.row);
		if (next) selection.moveTo(next);
	}
	let hasPages = $derived(paged && listed.length > 0);
	function across(by: 1 | -1) {
		if (zone === 'menus') {
			if (by > 0) zone = hasPages ? 'pages' : 'options';
			return;
		}
		if (zone === 'pages') {
			zone = by > 0 ? 'options' : column.ids.length ? 'menus' : 'pages';
			return;
		}
		const c = current;
		const beside = c ? rows.filter((r) => r.col === c.col + by) : [];
		if (!c || !beside.length) {
			if (by < 0) zone = hasPages ? 'pages' : column.ids.length ? 'menus' : 'options';
			return;
		}
		// The nearest row; between two as near, the one above.
		const next = beside.reduce((a, b) => (Math.abs(b.row - c.row) < Math.abs(a.row - c.row) ? b : a));
		selection.moveTo(next);
	}

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

	// A stepper: a text option that moves to its next value on each click.
	// The recording clicks it as many times as it takes to reach a value,
	// counted from the mod's default (MCM Helper's, or the first value its
	// script sets), and each click here moves one further, wrapping back
	// to the default, which is no clicks at all.
	const stepper = (o: Option) => o.type === 'text' && o.control === 'stepper' && (o.choices?.length ?? 0) > 1;
	const startOf = (o: Option) => (o.default?.kind === 'number' && o.default.number >= 0 && o.default.number < (o.choices?.length ?? 0) ? o.default.number : 0);
	const clicksOf = (o: Option) => open.clicks(base(o), page?.landing);
	const cycled = (o: Option, n: number) => o.choices![(startOf(o) + n) % o.choices!.length]!.text;
	async function cycle(o: Option) {
		if (working) return;
		working = true;
		error = '';
		try {
			await setClicks(open.name, base(o), (clicksOf(o) + 1) % o.choices!.length);
			await invalidate(recordingsDependency);
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
		} finally {
			working = false;
		}
	}
	function edit(r: { option: Option; i: number }) {
		pick(r.i);
		if (flippable(r.option)) return flip(r.option);
		if (stepper(r.option)) return cycle(r.option);
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
		if (stepper(r.option)) return cycle(r.option);
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
	// Counted by option, so a stepper's run of clicks is one change.
	let unsavedHere = $derived(open.editing ? new Set(open.steps.filter((s) => ofMenu(s) && open.dirty(s)).map((s) => `${s.page}\0${s.option}\0${s.index}\0${s.kind}`.toLowerCase())).size : 0);

	// discardMenu takes this menu's unsaved steps out of the draft: a step
	// the saved recording has goes back to its saved value, and one it
	// hasn't is removed. The rest of the draft is left as it is.
	async function discardMenu() {
		if (working || !open.name) return;
		working = true;
		error = '';
		note = '';
		try {
			// The menu's steps go back to the saved recording's, in its order,
			// where the draft's first step for the menu was; the rest of the
			// draft is left as it is.
			const n = unsavedHere;
			const ed = await getRecording(open.name);
			const saved = (ed.recording?.steps ?? []).filter(ofMenu);
			const others = ed.draft.steps.filter((s) => !ofMenu(s));
			const first = ed.draft.steps.findIndex(ofMenu);
			const at = first < 0 ? others.length : ed.draft.steps.slice(0, first).filter((s) => !ofMenu(s)).length;
			const steps: Step[] = [...others.slice(0, at), ...saved, ...others.slice(at)];
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
					holdSave,
					{ key: 'Escape', shown: 'Esc', label: m.common.cancel, run: () => leave('mode') }
				]
			: [
					{ key: 'ArrowDown', label: m.common.next, hidden: true, run: () => vertical(1) },
					{ key: 'ArrowUp', label: m.common.previous, hidden: true, run: () => vertical(-1) },
					{ key: 'ArrowRight', label: m.common.next, hidden: true, run: () => across(1) },
					{ key: 'ArrowLeft', label: m.common.previous, hidden: true, run: () => across(-1) },
					...(zone === 'options' && recording && current && canEdit(current.option)
						? flippable(current.option)
							? [{ key: ' ', shown: 'Space', label: on(current.option) ? m.edit.turnOff : m.edit.turnOn, disabled: working, run: () => flip(current!.option) }]
							: current.option.type === 'text'
								? []
								: [{ key: 'e', shown: 'E', label: m.edit.edit, disabled: working, run: () => setParams({ mode: 'edit' }) }]
						: []),
					...(zone === 'options' && open.editing && current && canEdit(current.option) && !recorded(current.option)
						? [{ key: 'a', shown: 'A', label: open.name ? m.adding.addTo({ recording: open.name }) : m.adding.addToAny, disabled: working, run: add }]
						: []),
					{ key: 'Escape', shown: 'Esc', label: t.up, run: () => goto('/menus') }
				]
	);
</script>

<h1 class="caps">{menu.nameText}</h1>
<p class="muted sub">{menu.mod}{#if menu.plugin}{' · '}{menu.plugin}{/if}{' · '}{m.menus.kinds[menu.kind]}</p>
{#if !menu.active}<p class="notice warn">{m.menus.inactive({ plugin: menu.plugin })}</p>{/if}
<!-- How the menu was read, where that's worth knowing: a guess, another
     copy, or a framework read along with it. -->
{#each menu.notes as n, i (i)}
	{@const text = menuNote(n)}
	{#if text}<p class="notice">{text}</p>{/if}
{/each}
{#if error}<p class="notice error" role="alert">{error}</p>{/if}

<!-- A value as the menu shows it, for reading: a toggle's switch, at rest,
     a text option's value, or the known value, a dash where it isn't known. -->
{#snippet shown(o: Option)}
	{@const k = known(o)}
	{#if o.type === 'toggle' && k}
		<Switch checked={k.kind === 'bool' && k.bool} disabled label={m.edit.toggleLabel({ option: o.text })} onchange={() => {}} />
	{:else if o.type === 'text'}
		{#if o.valueText || o.value}{o.valueText || o.value}{/if}
	{:else if k}
		{#if o.type === 'color'}<span class="swatch" style:background={hex(k.number)}></span>{/if}{show(o, k)}
	{:else}
		<span class="muted">—</span>
	{/if}
{/snippet}

{#snippet value(o: Option, i: number)}
	{#if recording && (o.disabled === 'always' || o.inert)}
		<!-- Grayed out in the game, or a line a click does nothing to: for
		     reading only. -->
		<span class="val readonly" title={o.inert ? m.adding.inert : m.adding.readOnly}>{@render shown(o)}</span>
	{:else if recording}
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
			{:else if stepper(o)}
				<!-- A stepper: the value it reaches after the recording's
				     clicks, and how many; each click here moves one on. -->
				{@const n = clicksOf(o)}
				<button type="button" class="click" title={m.adding.stepperTitle({ next: cycled(o, n + 1) })} aria-label={m.adding.stepperLabel({ option: o.text, value: cycled(o, n), count: n })} disabled={working} onclick={(e) => (e.stopPropagation(), edit({ option: o, i }))}>
					{#if n}<span class="caps word click-word">{m.adding.clicksShort({ count: n })}</span>{/if}{cycled(o, n)}
				</button>
			{:else if o.type === 'text'}
				<!-- A text option can only be clicked: its value is the menu's,
				     and a click on it is the step. -->
				<button type="button" class="click" title={rec ? m.adding.clickRecorded({ recording: open.name }) : m.adding.clickTitle({ recording: open.name })} aria-label={rec ? m.adding.clickRecorded({ recording: open.name }) : m.adding.recordThis({ option: o.text })} onclick={(e) => (e.stopPropagation(), edit({ option: o, i }))}>
					{#if rec}<span class="caps word click-word">{m.steps.click}</span>{/if}{#if o.valueText || o.value}{o.valueText || o.value}{:else}<span class="muted">—</span>{/if}
				</button>
			{:else if editingCell === i}
				<InlineValue bind:this={inline} kind={base(o).kind} option={o} value={valueNow(o).value} fresh={!recorded(o)} apply={(v) => record(o, v)} cancel={() => leave('mode')} />
			{:else if rec}
				<button type="button" class="cell" title={m.edit.editValue} onclick={(e) => (e.stopPropagation(), edit({ option: o, i }))}>
					{#if o.type === 'color'}<span class="swatch" style:background={hex(Number(rec.step.value))}></span>{/if}{stepValue(rec.step, o)}
				</button>
			{:else}
				<!-- Not recorded yet: the value the option has, muted, as the
				     editor starts from it; a dash when none is known. -->
				{@const k = known(o)}
				<button type="button" class="cell unrecorded" title={m.adding.notYetTitle({ recording: open.name })} aria-label={m.adding.recordThis({ option: o.text })} onclick={(e) => (e.stopPropagation(), edit({ option: o, i }))}>
					{#if k}{#if o.type === 'color'}<span class="swatch" style:background={hex(k.number)}></span>{/if}{show(o, k)}{:else}—{/if}
				</button>
			{/if}
		</span>
	{:else}
		<span class="val" class:changed={changed(o)} class:readonly={o.disabled === 'always'}>{@render shown(o)}</span>
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
		<li class="opt" class:disabled={o.disabled === 'always'} role="option" aria-selected={selected(x.i!)} style:grid-row={x.row + 1} style:grid-column={x.col + 1} onclick={() => !(editingCell >= 0) && pick(x.i!)}>
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

<div class="mcm" class:paged class:inpages={zone !== 'options'} style:--hints="{hintsHeight}px">
	{#if paged}
		<nav class="pages" class:active={inPages} aria-label={t.pages}>
			<ul>
				{#each menu.pages as p, i (i)}
					{#if !p.sameAs}
					<li>
						<a class="caps" href={menuHref(menu.id, i)} aria-current={i === data.at ? 'page' : undefined} onclick={() => (zone = 'options')}>
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
						<span class="sep">·</span> <span class="caps">{m.option.definedIn}</span> <span class="raw"><Path text={o.source.path} />{#if o.source.line}:{o.source.line}{/if}</span>{#if o.source.archive} {m.common.inArchive({ archive: o.source.archive })}{/if}
						{#if rec}<span class="sep">·</span> <a href={recordingHref(open.name) + `?item=${rec.n}`}>{s.stepOf({ n: rec.n, recording: open.name })}</a>{/if}
					</p>
					{#if o.certainty !== 'exact'}<p class="notice warn">{o.certainty === 'pattern' ? m.option.pattern : m.option.unknown}</p>{/if}
					{#if o.type === 'text' && o.valueChanges}<p class="notice">{m.option.valueChanges}</p>{/if}
					<!-- A click on an inert option does nothing, grayed out or not. -->
					{#if o.inert}<p class="notice">{m.option.inert}</p>
					{:else if o.disabled}<p class="notice">{o.disabled === 'always' ? m.option.disabledAlways : m.option.disabledSometimes}</p>{/if}
					{#if recording && stepper(o) && canEdit(o)}<p class="notice">{m.adding.stepperNote({ start: cycled(o, 0) })}</p>
					{:else if recording && o.type === 'text' && canEdit(o)}<p class="notice">{rec ? m.adding.clickNoteRecorded({ recording: open.name }) : m.adding.clickNote}</p>
					{:else if recording && o.type === 'keymap' && canEdit(o)}<p class="notice">{m.editor.controllerNote}</p>{/if}
					{#if recording && !canEdit(o) && o.disabled !== 'always' && !o.inert}<p class="notice">{m.adding.cantRecord}</p>{/if}
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
	/* The arrow keys in the page list: the current page takes the selected
	   haze, and the option the keys left keeps its diamond without it. */
	/* Room inside the keys' hairline, without moving the row. */
	.pages a {
		margin-inline: calc(-1 * var(--space-2));
		padding-inline: var(--space-2);
	}
	.pages.active a[aria-current='page'] {
		background: var(--select);
		outline: 1px solid var(--keys);
		outline-offset: -1px;
	}
	.mcm.inpages .grid .opt[aria-selected='true'] {
		background: none;
		outline: none;
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
	/* Where the arrow keys are: the haze, and a hairline of the accent
	   around it, so the place reads at a glance. */
	.grid .opt[aria-selected='true'] {
		background: var(--select);
		outline: 1px solid var(--keys);
		outline-offset: -1px;
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
	/* A text option's click: its value, plain, as the game shows it; a
	   hairline under it on hover says a click records it. */
	.click {
		padding: 0;
		background: none;
		border: 0;
		border-bottom: 1px solid transparent;
		font: inherit;
		color: inherit;
		text-align: right;
		cursor: pointer;
	}
	.click:hover,
	.click:focus-visible {
		border-bottom-color: var(--accent);
		outline: none;
	}
	/* A recorded click says so, since the value doesn't change. */
	.click-word {
		margin: 0 var(--space-2) 0 0;
		color: var(--ink);
	}
	/* Grayed out in the game, so here: the label and its value muted, and
	   nothing to edit. */
	.grid .opt.disabled .label,
	.val.readonly {
		color: var(--muted);
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
