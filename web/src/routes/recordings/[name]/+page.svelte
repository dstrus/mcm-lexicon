<!-- A recording's steps in replay order, under a heading per step file,
     each with its check, and the selected one in the pane. Read-only:
     editing comes with writing recordings. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import type { Check, Step } from '#lib/api/types/recordings.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';
	import { menuHref } from '#lib/menus/api.ts';
	import Selector from '#lib/menus/Selector.svelte';
	import { hex } from '#lib/menus/values.ts';
	import { changed, failing, fileHeading, kindType, sentence, value, word } from '#lib/recordings/steps.ts';
	import Diamond from '#lib/ui/Diamond.svelte';
	import { fitToViewport } from '#lib/ui/fit.ts';
	import { Selection } from '#lib/ui/selection.svelte.ts';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const t = m.recording;

	let rec = $derived(data.detail.recording!);
	let rows = $derived(rec.steps.map((step, i) => ({ step, check: data.detail.checks[i]!, n: i + 1 })));
	const selection = new Selection(() => rows, (r) => String(r.n), { param: 'item' });
	let current = $derived(selection.current);

	// What a step names, for reading: its option's text where the catalog
	// found it, otherwise its selector as written.
	const optionText = (s: Step, c: Check) => c.option?.text ?? s.option;
	const pageText = (s: Step, c: Check) => (c.option ? c.pageText : s.page);
	const menuLink = (c: Check) => (c.option ? menuHref(c.menuId, c.page, c.position) : c.menuId ? menuHref(c.menuId) : '');

	useShortcuts(() => [
		{ key: 'ArrowDown', label: m.common.next, hidden: true, run: () => selection.move(1) },
		{ key: 'ArrowUp', label: m.common.previous, hidden: true, run: () => selection.move(-1) },
		{ key: 'Escape', shown: 'Esc', label: t.up, run: () => goto(`/recordings?item=${encodeURIComponent(rec.name)}`) }
	]);
</script>

{#if !rows.length}
	<p class="empty">{t.noSteps}</p>
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
						<tr class="group"><th colspan="7" scope="rowgroup" class="caps">{fileHeading(rec.files[r.step.file]?.name ?? '')}</th></tr>
					{/if}
					<tr aria-selected={selection.is(r)} onclick={() => selection.select(r)}>
						<td class="mark">{#if selection.is(r)}<Diamond />{/if}</td>
						<td class="n muted">{r.n}</td>
						<td class="option"><Selector text={optionText(r.step, r.check)} raw={r.step.option} /></td>
						<td class="page muted">{pageText(r.step, r.check)}</td>
						<td class="type caps">{m.option.types[kindType[r.step.kind]]}</td>
						<td class="value" class:changed={changed(r.step, r.check.option)}>
							{#if r.step.kind === 'color'}<span class="swatch" style:background={hex(Number(r.step.value))}></span>{/if}{value(r.step, r.check.option)}
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
						{#if c.suggestion}
							<p class="suggestion">
								{#if c.suggestion.page !== s.page}
									<Fill text={t.suggestionOnPage.text}>{#snippet label()}<span class="raw">“{c.suggestion!.label}”</span>{/snippet}{#snippet page()}<span class="raw">“{c.suggestion!.page}”</span>{/snippet}</Fill>
								{:else}
									<Fill text={t.suggestion.text}>{#snippet label()}<span class="raw">“{c.suggestion!.label}”</span>{/snippet}</Fill>
								{/if}
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
							<dt class="caps">{t.file}</dt><dd class="raw">{rec.files[s.file]?.name}</dd>
						</dl>
						{#if menuLink(c)}<a class="textbutton" href={menuLink(c)}>{t.openInMenus}</a>{/if}
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
