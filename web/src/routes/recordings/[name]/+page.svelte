<!-- A recording's steps in replay order, under a heading per step file,
     each with its check, and the selected one in the pane, where it's
     moved and removed. Values change in the table: a toggle's checkbox
     flips it, and any other value's cell becomes its editor (?mode=edit).
     Every change goes to the draft; nothing is written until Save. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import type { Status } from '#lib/api/types/api.ts';
	import { ApiError } from '#lib/api/client.ts';
	import type { Check, Step } from '#lib/api/types/recordings.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';
	import { menuHref } from '#lib/menus/api.ts';
	import Selector from '#lib/menus/Selector.svelte';
	import { hex } from '#lib/menus/values.ts';
	import { change, move } from '#lib/recordings/draft.ts';
	import { changed, failing, fileHeading, kindType, sentence, value, word } from '#lib/recordings/steps.ts';
	import InlineValue from '#lib/recordings/InlineValue.svelte';
	import Switch from '#lib/ui/Switch.svelte';
	import Diamond from '#lib/ui/Diamond.svelte';
	import { leave, param, setParams } from '#lib/ui/address.svelte.ts';
	import { fitToViewport } from '#lib/ui/fit.ts';
	import { Selection } from '#lib/ui/selection.svelte.ts';
	import { deleteShortcuts, useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const t = m.recording;
	const e = m.edit;

	let ed = $derived(data.editor);
	let steps = $derived(ed.draft.steps);
	let rows = $derived(steps.map((step, i) => ({ step, check: ed.checks[i]!, n: i + 1 })));
	const selection = new Selection(() => rows, (r) => String(r.n), { param: 'item' });
	let current = $derived(selection.current);
	// The step files' names, by the position steps give them.
	let fileNames = $derived(ed.files.filter((f) => f.change !== 'removed').slice(1).map((f) => f.name));

	// What a step names, for reading: its option's text where the catalog
	// found it, otherwise its selector as written.
	const optionText = (s: Step, c: Check) => c.option?.text ?? s.option;
	const pageText = (s: Step, c: Check) => (c.option ? c.pageText : s.page);
	// Whether JContainers matches characters beyond plain ASCII against the
	// game's strings isn't confirmed yet (PLAN.md §4), so such steps say so.
	const beyondASCII = (s: Step) => /[^\x00-\x7f]/.test(s.mod + s.page + s.option + (s.kind === 'choose' || s.kind === 'text' ? s.value : ''));
	const menuLink = (c: Check) => (c.option ? menuHref(c.menuId, c.page, c.position) : c.menuId ? menuHref(c.menuId) : '');

	// Steps change only while editing, and not in a recording the draft
	// deletes. Then a toggle flips in place, a click has nothing to set, and
	// anything else is edited in its cell.
	let canEdit = $derived((page.data.status as Status).editing && !ed.deleted);
	const editable = (s: Step) => canEdit && s.kind !== 'toggle' && s.kind !== 'click';
	let editing = $derived(param('mode') === 'edit' && !!current && editable(current.step));
	let working = $state(false);
	let error = $state('');
	let inline = $state<InlineValue>();

	async function apply(edit: (s: Step[]) => void, select?: number) {
		working = true;
		error = '';
		try {
			await change(ed, (d) => edit(d.steps));
			if (select !== undefined) await setParams({ item: select > 0 ? String(select) : null, mode: null }, 'replace');
		} catch (err) {
			error = err instanceof ApiError ? err.message : String(err);
		} finally {
			working = false;
		}
	}
	const at = () => (current ? current.n - 1 : -1);
	const isOn = (s: Step) => s.value.toLowerCase() === 'on';
	// flip turns the toggle step at i on or off.
	const flip = (i: number) => {
		const s = steps[i];
		if (!s || s.kind !== 'toggle' || working || !canEdit) return;
		apply((x) => (x[i]!.value = isOn(s) ? 'off' : 'on'));
	};
	// E: a toggle flips; anything else's cell becomes its editor.
	const startEdit = () => {
		if (!current) return;
		if (current.step.kind === 'toggle') flip(at());
		else if (editable(current.step)) setParams({ mode: 'edit' });
	};
	async function setValue(value: string) {
		const i = at();
		if (i < 0) return;
		await apply((x) => (x[i]!.value = value));
		leave('mode');
	}
	// A value's cell: clicking it selects the row and starts editing it.
	function editRow(r: (typeof rows)[number]) {
		selection.select(r);
		setParams({ item: String(r.n), mode: 'edit' });
	}
	const moveBy = (by: -1 | 1) => {
		const i = at();
		if (i < 0 || i + by < 0 || i + by >= steps.length) return;
		apply((s) => move(s, i, by), i + by + 1);
	};
	const remove = () => {
		const i = at();
		if (i < 0) return;
		apply((s) => s.splice(i, 1), Math.min(i + 1, steps.length - 1));
	};
	// The suggestion replaces the step's selector, and its page when the
	// option is on another one.
	const useSuggestion = () => {
		const i = at();
		const sug = current?.check.suggestion;
		if (i < 0 || !sug) return;
		apply((s) => {
			s[i]!.option = sug.label;
			s[i]!.page = sug.page;
			if (s[i]!.kind === 'click') s[i]!.value = sug.label;
		});
	};

	let up = $derived(`/recordings?item=${encodeURIComponent(ed.draft.name)}`);
	useShortcuts(() =>
		!canEdit
			? [
					{ key: 'ArrowDown', label: m.common.next, hidden: true, run: () => selection.move(1) },
					{ key: 'ArrowUp', label: m.common.previous, hidden: true, run: () => selection.move(-1) },
					{ key: 'Escape', shown: 'Esc', label: t.up, run: () => goto(up) }
				]
			: editing
			? [
					// The cell's field takes these keys first; they're here so the
					// strip shows them.
					{ key: 'Enter', label: e.apply, disabled: working, run: () => inline?.done() },
					{ key: 'Escape', shown: 'Esc', label: m.common.cancel, run: () => leave('mode') }
				]
			: [
					{ key: 'ArrowDown', label: m.common.next, hidden: true, run: () => selection.move(1) },
					{ key: 'ArrowUp', label: m.common.previous, hidden: true, run: () => selection.move(-1) },
					{ key: ' ', shown: 'Space', label: current && isOn(current.step) ? e.turnOff : e.turnOn, hidden: current?.step.kind !== 'toggle', disabled: working, run: () => flip(at()) },
					{ key: 'e', shown: 'E', label: e.edit, hidden: current?.step.kind === 'toggle', disabled: !current || !editable(current.step) || working, run: startEdit },
					{ key: 'u', shown: 'U', label: e.moveUp, disabled: !current || at() === 0 || working, run: () => moveBy(-1) },
					{ key: 'd', shown: 'D', label: e.moveDown, disabled: !current || at() === steps.length - 1 || working, run: () => moveBy(1) },
					...deleteShortcuts({ label: e.remove, disabled: !current || working, run: remove }),
					{ key: 'Escape', shown: 'Esc', label: t.up, run: () => goto(up) }
				]
	);
</script>

{#if !rows.length}
	<p class="empty">{e.noSteps}</p>
{:else}
	<div class="split">
		<table>
			<thead>
				<tr>
					<th class="mark" aria-label={m.common.selected}></th>
					<th class="caps n">{t.number}</th>
					<th class="caps">{t.option}</th>
					<th class="caps page">{t.page}</th>
					<th class="caps">{t.type}</th>
					<th class="caps">{t.value}</th>
					<th class="caps">{t.check}</th>
				</tr>
			</thead>
			<tbody>
				{#each rows as r, i (r.n)}
					{#if r.step.file !== rows[i - 1]?.step.file}
						<tr class="group"><th colspan="7" scope="rowgroup" class="caps">{fileHeading(fileNames[r.step.file] ?? '')}</th></tr>
					{/if}
					<tr aria-selected={selection.is(r)} onclick={() => !editing && selection.select(r)}>
						<td class="mark">{#if selection.is(r)}<Diamond />{/if}</td>
						<td class="n muted">{r.n}</td>
						<td class="option"><Selector text={optionText(r.step, r.check)} raw={r.step.option} /></td>
						<td class="page muted">{pageText(r.step, r.check)}</td>
						<td class="type caps">{m.option.types[kindType[r.step.kind]]}</td>
						<td class="value" class:changed={changed(r.step, r.check.option)}>
							{#if r.step.kind === 'toggle' && canEdit}
								<span class="flip">
									<Switch checked={isOn(r.step)} disabled={working} label={e.toggleLabel({ option: optionText(r.step, r.check) })} onchange={() => flip(r.n - 1)} />
									{value(r.step, r.check.option)}
								</span>
							{:else if editing && selection.is(r)}
								<InlineValue bind:this={inline} kind={r.step.kind} option={r.check.option} value={r.step.value} apply={setValue} cancel={() => leave('mode')} />
							{:else if editable(r.step)}
								<button type="button" class="cell" title={e.editValue} onclick={(ev) => (ev.stopPropagation(), editRow(r))}>
									{#if r.step.kind === 'color'}<span class="swatch" style:background={hex(Number(r.step.value))}></span>{/if}{value(r.step, r.check.option)}
								</button>
							{:else}
								{value(r.step, r.check.option)}
							{/if}
						</td>
						<td class="word caps" class:warn={failing(r.check)}>{word(r.check)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
		<aside class="pane" aria-label={t.pane} use:fitToViewport>
			{#if current}
				{@const s = current.step}
				{@const c = current.check}
				{#key current.n}
					<div class="enter">
						<h2 class="caps">{optionText(s, c) || m.values.noLabel}</h2>
						<p class="muted">{c.menuText || s.mod}{#if pageText(s, c)}{' · '}{pageText(s, c)}{/if}</p>
							<p class="check" class:warn={failing(c)}>{sentence(c, s)}</p>
							{#if beyondASCII(s)}<p class="notice">{e.beyondASCII}</p>{/if}
							{#if c.suggestion}
								<p class="suggestion">
									{#if c.suggestion.page !== s.page}
										<Fill text={t.suggestionOnPage.text}>{#snippet label()}<span class="raw">“{c.suggestion!.label}”</span>{/snippet}{#snippet page()}<span class="raw">“{c.suggestion!.page}”</span>{/snippet}</Fill>
									{:else}
										<Fill text={t.suggestion.text}>{#snippet label()}<span class="raw">“{c.suggestion!.label}”</span>{/snippet}</Fill>
									{/if}
									{#if canEdit}<button type="button" class="textbutton" disabled={working} onclick={useSuggestion}>{e.useIt}</button>{/if}
								</p>
							{/if}
							<dl>
								<dt class="caps">{t.menu}</dt><dd class="raw">{s.mod}</dd>
								{#if s.page}<dt class="caps">{t.page}</dt><dd class="raw">{s.page}</dd>{/if}
								<dt class="caps">{t.selector}</dt><dd class="raw">{s.option}</dd>
								<dt class="caps">{t.type}</dt><dd>{m.option.types[kindType[s.kind]]}</dd>
								<dt class="caps">{t.value}</dt>
								<dd class:changed={changed(s, c.option)}>{value(s, c.option)}<span class="raw muted small">{t.asWritten}: {s.kind} {s.value}</span></dd>
								{#if s.index}<dt class="caps">{t.index}</dt><dd>{s.index}</dd>{/if}
								{#if s.state}<dt class="caps">{t.state}</dt><dd class="raw">{s.state}</dd>{/if}
								{#if s.side}<dt class="caps">{t.side}</dt><dd>{s.side}</dd>{/if}
								<dt class="caps">{t.file}</dt><dd class="raw">{fileNames[s.file]}</dd>
							</dl>
							{#if canEdit}<div class="actions">
								{#if s.kind === 'toggle'}
									<button type="button" class="button primary" disabled={working} onclick={startEdit}>{isOn(s) ? e.turnOff : e.turnOn}</button>
								{:else if editable(s)}
									<button type="button" class="button primary" disabled={working || editing} onclick={startEdit}>{e.edit}</button>
								{/if}
								<button type="button" class="button" disabled={working || current.n === 1} onclick={() => moveBy(-1)}>{e.moveUp}</button>
								<button type="button" class="button" disabled={working || current.n === steps.length} onclick={() => moveBy(1)}>{e.moveDown}</button>
								<button type="button" class="button" disabled={working} onclick={remove}>{e.remove}</button>
							</div>{/if}
							{#if menuLink(c)}<a class="textbutton open" href={menuLink(c)}>{t.openInMenus}</a>{/if}
						{#if error}<p class="notice error" role="alert">{error}</p>{/if}
					</div>
				{/key}
			{/if}
		</aside>
	</div>
{/if}

<style>
	.empty {
		margin-top: var(--space-5);
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
	thead th {
		text-align: left;
		font-size: var(--step--1);
		color: var(--muted);
		font-weight: inherit;
		padding: var(--space-2) var(--space-3);
		border-bottom: 1px solid var(--rule-strong);
		white-space: nowrap;
	}
	thead th.n {
		text-align: right;
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
	/* A step file's steps sit under its name, as a list's groups do. */
	.group th {
		text-align: left;
		padding: var(--space-5) var(--space-3) var(--space-1);
		color: var(--accent);
		font-size: var(--step--1);
		font-weight: 500;
		border-bottom: 1px solid var(--rule-strong);
	}
	tbody tr.group:first-child th {
		padding-top: var(--space-3);
	}
	.mark {
		width: 1.5rem;
		padding-right: 0;
		vertical-align: middle;
	}
	td.n {
		text-align: right;
		width: 2.5rem;
	}
	.option {
		max-width: 24rem;
	}
	.page,
	.type {
		font-size: var(--step--1);
	}
	.type,
	.word {
		color: var(--muted);
		white-space: nowrap;
	}
	.word {
		font-size: var(--step--1);
	}
	.word.warn {
		color: var(--warn);
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
	.pane h2 {
		font-size: var(--step-2);
		font-weight: 600;
		margin-bottom: var(--space-1);
		overflow-wrap: anywhere;
	}
	.check {
		margin-top: var(--space-3);
		max-width: 60ch;
	}
	.check.warn {
		color: var(--warn);
	}
	.suggestion {
		margin-top: var(--space-2);
	}
	.pane dl {
		display: grid;
		grid-template-columns: 7.5rem 1fr;
		gap: var(--space-1) var(--space-3);
		margin: var(--space-4) 0;
	}
	.pane dt {
		color: var(--muted);
		font-size: var(--step--1);
		padding-top: 0.15em;
	}
	.pane dd {
		margin: 0;
		overflow-wrap: anywhere;
	}
	.raw {
		font-family: var(--font-mono);
		font-size: var(--step--1);
	}
	dd .small {
		display: block;
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}
	.open {
		display: inline-block;
		margin-top: var(--space-3);
	}
	.suggestion .textbutton {
		margin-left: var(--space-2);
	}
	/* A toggle's cell is its checkbox, with the value beside it. */
	.flip {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		cursor: pointer;
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
	.cell:hover,
	.cell:focus-visible {
		border-bottom-color: var(--rule-strong);
		outline: none;
	}
	.notice {
		margin-top: var(--space-3);
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
	@media (max-width: 70rem) {
		.page {
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
