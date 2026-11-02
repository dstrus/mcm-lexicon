<!-- One menu: its pages as tabs, the current page's options as an item
     table, and the selected option in the pane. While editing, with a
     recording chosen, a Recorded column shows each option's value in it,
     edited in the table: a change records the option, and Record all
     settings on this page records every one whose value is known. -->
<script lang="ts">
	import { goto, invalidate } from '$app/navigation';
	import { ApiError } from '#lib/api/client.ts';
	import type { Option } from '#lib/api/types/catalog.ts';
	import { m } from '#lib/copy/messages.ts';
	import { menuHref } from '#lib/menus/api.ts';
	import OptionPane from '#lib/menus/OptionPane.svelte';
	import Adder from '#lib/recordings/Adder.svelte';
	import ChooseRecording from '#lib/recordings/ChooseRecording.svelte';
	import InlineValue from '#lib/recordings/InlineValue.svelte';
	import Switch from '#lib/ui/Switch.svelte';
	import { addAll, addStep, recordingsDependency } from '#lib/recordings/api.ts';
	import { OpenRecording } from '#lib/recordings/open.svelte.ts';
	import { known, recordable, stepFor, changed as stepChanged, value as stepValue } from '#lib/recordings/steps.ts';
	import { leave, param, setParams } from '#lib/ui/address.svelte.ts';
	import Selector from '#lib/menus/Selector.svelte';
	import { allowed, changed, hex, show } from '#lib/menus/values.ts';
	import Diamond from '#lib/ui/Diamond.svelte';
	import Tabs from '#lib/ui/Tabs.svelte';
	import Up from '#lib/ui/Up.svelte';
	import { fitToViewport } from '#lib/ui/fit.ts';
	import { Selection } from '#lib/ui/selection.svelte.ts';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const t = m.menu;

	let menu = $derived(data.menu);
	let page = $derived(menu.pages[data.at]);
	let options = $derived(page?.options ?? []);
	let pageText = (p: { name: string; text: string }) => (p.name === '' ? t.mainPage : p.text);
	// A menu without pages has one, unnamed, and no tabs.
	let tabbed = $derived(menu.pages.length > 1 || (menu.pages[0]?.name ?? '') !== '');
	let withSaved = $derived(menu.pages.some((p) => p.options.some((o) => o.saved)));

	// Rows are kept by position, which the address holds as ?item=.
	let rows = $derived(options.map((option, i) => ({ option, i })));
	const selection = new Selection(() => rows, (r) => String(r.i), { param: 'item' });
	let current = $derived(selection.current);

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
	const base = (o: Option) => stepFor(menu.name, page?.name ?? '', o);
	const recorded = (o: Option) => open.find(base(o));
	// What a cell shows and starts from: the recorded value, else the
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
	let inline = $state<InlineValue>();
	let editingCell = $derived(recording && param('mode') === 'edit' && selection.current ? selection.current.i : -1);

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
	function editCell(r: (typeof rows)[number]) {
		selection.select(r);
		if (flippable(r.option)) return flip(r.option);
		if (r.option.type === 'text') return record(r.option, '');
		setParams({ item: String(r.i), mode: 'edit' });
	}

	// Record all settings on this page: every option whose value is known
	// and whose label can be matched exactly; ones recorded already keep
	// their values.
	let recordableNow = $derived(options.filter((o) => recordable(o) && o.type !== 'text' && !!known(o)));
	async function recordAll() {
		if (working) return;
		if (!open.name && !(await choose())) return;
		working = true;
		error = '';
		note = '';
		try {
			const r = await addAll(open.name, recordableNow.map(base));
			const skipped = options.length - recordableNow.length;
			note = [
				t.recordedAdded({ count: r.added, recording: open.name }),
				r.kept ? t.recordedKept({ count: r.kept }) : '',
				skipped ? t.recordedSkipped({ count: skipped }) : ''
			]
				.filter(Boolean)
				.join(' ');
			await invalidate(recordingsDependency);
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
		} finally {
			working = false;
		}
	}

	let sel = $derived(selection.current);
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
					...(recording && sel && canEdit(sel.option)
						? flippable(sel.option)
							? [{ key: ' ', shown: 'Space', label: on(sel.option) ? m.edit.turnOff : m.edit.turnOn, disabled: working, run: () => flip(sel.option) }]
							: sel.option.type === 'text'
								? []
								: [{ key: 'e', shown: 'E', label: m.edit.edit, disabled: working, run: () => setParams({ mode: 'edit' }) }]
						: []),
					{ key: 'Escape', shown: 'Esc', label: t.up, run: () => goto(`/menus?item=${encodeURIComponent(menu.id)}`) }
				]
	);
</script>

<Up href="/menus?item={encodeURIComponent(menu.id)}" label={t.up} />
<h1 class="caps">{menu.nameText}</h1>
<p class="muted sub">{menu.mod}{#if menu.plugin}{' · '}{menu.plugin}{/if}{' · '}{m.menus.kinds[menu.kind]}</p>
{#if !menu.active}<p class="notice warn">{m.menus.inactive({ plugin: menu.plugin })}</p>{/if}

{#if tabbed}
	<Tabs label={t.pages} tabs={menu.pages.map((p, i) => ({ href: menuHref(menu.id, i), label: pageText(p), count: p.options.length }))} />
{/if}

{#if open.editing && options.length}
	<div class="tools">
		<button type="button" class="button" disabled={working || !recordableNow.length} onclick={recordAll}>{t.recordAll}</button>
		{#if note}<p class="muted" role="status">{note}</p>{/if}
	</div>
{/if}
{#if error}<p class="notice error" role="alert">{error}</p>{/if}

{#if !options.length}
	<p class="empty">{tabbed ? t.noOptions : t.noPages}</p>
{:else}
	<div class="split">
		<table>
			<thead>
				<tr>
					<th class="mark" aria-label={m.common.selected}></th>
					<th class="caps">{t.option}</th>
					<th class="caps">{t.type}</th>
					<th class="caps allowed">{t.allowed}</th>
					<th class="caps">{t.default}</th>
					{#if withSaved}<th class="caps">{t.saved}</th>{/if}
					{#if recording}<th class="caps" title={open.name}>{t.recorded}</th>{/if}
				</tr>
			</thead>
			<tbody>
				{#each rows as r (r.i)}
					{@const o = r.option}
					<tr aria-selected={selection.is(r)} onclick={() => selection.select(r)}>
						<td class="mark">{#if selection.is(r)}<Diamond />{/if}</td>
						<td class="option">
							<Selector text={o.text} raw={o.label} certainty={o.certainty} />
							{#if o.index > 0}<span class="caps word">{t.indexOf({ n: o.index, count: sharing.get(`${o.type}\0${o.label}`) ?? o.index })}</span>{/if}
							{#if o.certainty !== 'exact'}<span class="caps word">{o.certainty}</span>{/if}
						</td>
						<td class="type caps">{typeLabel(o)}</td>
						<td class="allowed">{allowed(o)}</td>
						<td class="value">{#if o.default}{#if o.type === 'color'}<span class="swatch" style:background={hex(o.default.number)}></span>{/if}{show(o, o.default)}{/if}</td>
						{#if withSaved}
							<td class="value" class:changed={changed(o)}>{#if o.saved}{#if o.type === 'color'}<span class="swatch" style:background={hex(o.saved.number)}></span>{/if}{show(o, o.saved)}{/if}</td>
						{/if}
						{#if recording}
							{@const rec = recorded(o)}
							<td class="value rec" class:changed={!!rec && stepChanged(rec.step, o)}>
								{#if !canEdit(o)}
									<span class="muted" title={m.adding.cantRecord}></span>
								{:else if flippable(o)}
									<span class="flip" title={rec ? undefined : m.adding.notYetTitle({ recording: open.name })}>
										<Switch checked={on(o)} dim={!rec} disabled={working} label={m.edit.toggleLabel({ option: o.text })} onchange={() => flip(o)} />
										{#if rec}{stepValue(rec.step, o)}{/if}
									</span>
								{:else if editingCell === r.i}
									<InlineValue bind:this={inline} kind={base(o).kind} option={o} value={valueNow(o).value} apply={(v) => record(o, v)} cancel={() => leave('mode')} />
								{:else if rec}
									<button type="button" class="cell" title={m.edit.editValue} disabled={o.type === 'text'} onclick={(e) => (e.stopPropagation(), editCell(r))}>
										{#if o.type === 'color'}<span class="swatch" style:background={hex(Number(rec.step.value))}></span>{/if}{stepValue(rec.step, o)}
									</button>
								{:else}
									<button type="button" class="cell unrecorded" title={m.adding.notYetTitle({ recording: open.name })} aria-label={m.adding.recordThis({ option: o.text })} onclick={(e) => (e.stopPropagation(), editCell(r))}>—</button>
								{/if}
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
		<aside class="pane" aria-label={t.pane} use:fitToViewport>
			{#if current}
				{#key current.i}
					<div class="enter">
						<OptionPane option={current.option} page={page && page.name !== '' ? pageText(page) : ''}>
							{#snippet action()}<Adder option={current.option} menu={menu.name} pageName={page?.name ?? ''} {open} {choose} />{/snippet}
						</OptionPane>
					</div>
				{/key}
			{/if}
		</aside>
	</div>
{/if}

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
	.empty {
		margin-top: var(--space-5);
		max-width: 60ch;
	}
	.split {
		display: grid;
		grid-template-columns: minmax(0, 1fr) clamp(18rem, 26vw, 24rem);
		gap: var(--space-5);
		align-items: start;
		margin-top: var(--space-4);
	}
	table {
		width: 100%;
		border-collapse: collapse;
	}
	th {
		text-align: left;
		font-size: var(--step--1);
		color: var(--muted);
		font-weight: inherit;
		padding: var(--space-2) var(--space-3);
		border-bottom: 1px solid var(--rule-strong);
		white-space: nowrap;
	}
	td {
		padding: var(--space-2) var(--space-3);
		border-bottom: 1px solid var(--rule);
		vertical-align: baseline;
	}
	tbody tr {
		cursor: default;
	}
	tbody tr[aria-selected]:not([aria-selected='true']):hover td {
		background: var(--glass);
	}
	tr[aria-selected='true'] td {
		background: var(--select);
	}
	.mark {
		width: 1.5rem;
		padding-right: 0;
		vertical-align: middle;
	}
	.option {
		max-width: 28rem;
	}
	/* Certainty, when it isn't exact: a word after the label, never a color. */
	.word {
		display: inline-block;
		margin-top: var(--space-1);
		font-size: var(--step--1);
		color: var(--muted);
	}
	.type {
		font-size: var(--step--1);
		color: var(--muted);
		white-space: nowrap;
	}
	.allowed {
		color: var(--muted);
		font-size: var(--step--1);
	}
	td.value {
		white-space: nowrap;
	}
	.changed {
		color: var(--changed);
	}
	.swatch {
		display: inline-block;
		width: 9px;
		height: 9px;
		margin-right: var(--space-2);
		border: 1px solid var(--rule-strong);
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
	/* A toggle's recorded cell is its switch; one not recorded yet is
	   dimmed, showing what the menu has, so a click records the other. */
	.flip {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
	}
	/* Any other value's cell is a button that turns into its editor. */
	.cell {
		background: none;
		border: 0;
		padding: 0;
		font: inherit;
		color: inherit;
		text-align: left;
		cursor: text;
		border-bottom: 1px dashed transparent;
	}
	.cell:hover:not(:disabled),
	.cell:focus-visible {
		border-bottom-color: var(--rule-strong);
		outline: none;
	}
	.cell:disabled {
		cursor: default;
	}
	.cell.unrecorded {
		color: var(--muted);
	}
	.pane {
		position: sticky;
		top: 0;
		display: flex;
		flex-direction: column;
		min-height: calc(100vh - 7rem);
		padding: var(--space-5);
		background: var(--panel);
		border-left: 1px solid var(--rule-strong);
		overflow-y: auto;
	}
	/* The pane's content enters with a short rise when the selection moves. */
	.enter {
		animation: rise 180ms cubic-bezier(0.16, 1, 0.3, 1);
	}
	@keyframes rise {
		from {
			opacity: 0;
			transform: translateY(6px);
		}
	}
	@media (max-width: 70rem) {
		.allowed {
			display: none;
		}
	}
	@media (max-width: 60rem) {
		.split {
			grid-template-columns: 1fr;
		}
		.pane {
			position: static;
			min-height: 0;
			border-left: 0;
			border-top: 1px solid var(--rule-strong);
		}
	}
</style>
