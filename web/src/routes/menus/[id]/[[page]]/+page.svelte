<!-- One menu: its pages as tabs, the current page's options as an item
     table, and the selected option in the pane. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import type { Option } from '#lib/api/types/catalog.ts';
	import { m } from '#lib/copy/messages.ts';
	import { menuHref } from '#lib/menus/api.ts';
	import OptionPane from '#lib/menus/OptionPane.svelte';
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

	useShortcuts(() => [
		{ key: 'ArrowDown', label: m.common.next, hidden: true, run: () => selection.move(1) },
		{ key: 'ArrowUp', label: m.common.previous, hidden: true, run: () => selection.move(-1) },
		{ key: 'Escape', shown: 'Esc', label: t.up, run: () => goto(`/menus?item=${encodeURIComponent(menu.id)}`) }
	]);
</script>

<Up href="/menus?item={encodeURIComponent(menu.id)}" label={t.up} />
<h1 class="caps">{menu.nameText}</h1>
<p class="muted sub">{menu.mod}{#if menu.plugin}{' · '}{menu.plugin}{/if}{' · '}{m.menus.kinds[menu.kind]}</p>
{#if !menu.active}<p class="notice warn">{m.menus.inactive({ plugin: menu.plugin })}</p>{/if}

{#if tabbed}
	<Tabs label={t.pages} tabs={menu.pages.map((p, i) => ({ href: menuHref(menu.id, i), label: pageText(p), count: p.options.length }))} />
{/if}

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
				</tr>
			</thead>
			<tbody>
				{#each rows as r (r.i)}
					{@const o = r.option}
					<tr aria-selected={selection.is(r)} onclick={() => selection.select(r)}>
						<td class="mark">{#if selection.is(r)}<Diamond />{/if}</td>
						<td class="option">
							<Selector text={o.text} raw={o.label} />
							{#if o.index > 0}<span class="caps word">{t.indexOf({ n: o.index, count: sharing.get(`${o.type}\0${o.label}`) ?? o.index })}</span>{/if}
							{#if o.certainty !== 'exact'}<span class="caps word">{o.certainty}</span>{/if}
						</td>
						<td class="type caps">{typeLabel(o)}</td>
						<td class="allowed">{allowed(o)}</td>
						<td class="value">{#if o.default}{#if o.type === 'color'}<span class="swatch" style:background={hex(o.default.number)}></span>{/if}{show(o, o.default)}{/if}</td>
						{#if withSaved}
							<td class="value" class:changed={changed(o)}>{#if o.saved}{#if o.type === 'color'}<span class="swatch" style:background={hex(o.saved.number)}></span>{/if}{show(o, o.saved)}{/if}</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
		<aside class="pane" aria-label={t.pane} use:fitToViewport>
			{#if current}
				{#key current.i}
					<div class="enter"><OptionPane option={current.option} page={page && page.name !== '' ? pageText(page) : ''} /></div>
				{/key}
			{/if}
		</aside>
	</div>
{/if}

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
