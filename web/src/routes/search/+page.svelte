<!-- One box, and every option in every menu that matches it, grouped under
     its menu. The query lives in the address, so a search can be linked to
     and Back returns to it. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.ts';
	import type { Result, SearchReply } from '#lib/api/types/catalog.ts';
	import { m } from '#lib/copy/messages.ts';
	import { count } from '#lib/format.ts';
	import { menuHref, search } from '#lib/menus/api.ts';
	import OptionPane from '#lib/menus/OptionPane.svelte';
	import Adder from '#lib/recordings/Adder.svelte';
	import ChooseRecording from '#lib/recordings/ChooseRecording.svelte';
	import { OpenRecording } from '#lib/recordings/open.svelte.ts';
	import Selector from '#lib/menus/Selector.svelte';
	import { allowed } from '#lib/menus/values.ts';
	import Diamond from '#lib/ui/Diamond.svelte';
	import { setParams } from '#lib/ui/address.svelte.ts';
	import { fitToViewport } from '#lib/ui/fit.ts';
	import { Selection } from '#lib/ui/selection.svelte.ts';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import { onMount } from 'svelte';

	const t = m.search;

	let query = $state(page.url.searchParams.get('q') ?? '');
	let reply = $state<SearchReply | null>(null);
	let error = $state('');
	let input = $state<HTMLInputElement>();

	// Each keystroke searches after a short pause, and the latest one wins.
	let timer: ReturnType<typeof setTimeout> | undefined;
	let asked = 0;
	function run(q: string) {
		clearTimeout(timer);
		timer = setTimeout(async () => {
			const mine = ++asked;
			error = '';
			try {
				const r = q.trim() ? await search(q) : null;
				if (mine === asked) reply = r;
			} catch (e) {
				if (mine === asked) error = e instanceof ApiError ? e.message : String(e);
			}
		}, 150);
	}
	function typed() {
		setParams({ q: query.trim() ? query : null, item: null }, 'replace');
		run(query);
	}
	onMount(() => {
		input?.focus();
		if (query) run(query);
	});

	let results = $derived(reply?.results ?? []);
	const key = (r: Result) => `${r.menuId}/${r.page}/${r.position}`;
	const selection = new Selection(() => results, key, { param: 'item' });
	let current = $derived(selection.current);
	const openMenu = (r: Result) => goto(menuHref(r.menuId, r.page, r.position));
	const pageText = (r: Result) => (r.pageName === '' ? '' : r.pageText);

	// The arrows move through results while the box has focus, and Enter
	// opens the selected one's menu.
	function keys(e: KeyboardEvent) {
		if (e.key === 'ArrowDown') selection.move(1);
		else if (e.key === 'ArrowUp') selection.move(-1);
		else if (e.key === 'Enter' && current) openMenu(current);
		else return;
		e.preventDefault();
	}

	// The open recording, for the pane's In <recording> section.
	const open = new OpenRecording();
	let chooser = $state<ChooseRecording>();
	const choose = () => chooser?.choose() ?? Promise.resolve(null);

	useShortcuts(() => [
		{ key: 'ArrowDown', label: m.common.next, hidden: true, run: () => selection.move(1) },
		{ key: 'ArrowUp', label: m.common.previous, hidden: true, run: () => selection.move(-1) },
		{ key: 'Enter', label: t.openMenu, disabled: !current, run: () => current && openMenu(current) }
	]);
</script>

<h1 class="caps">{t.title}</h1>
<p class="muted lead">{t.lead}</p>
<div class="box">
	<input bind:this={input} type="search" bind:value={query} oninput={typed} onkeydown={keys} placeholder={t.input} aria-label={t.inputLabel} spellcheck="false" autocomplete="off" />
	{#if reply}<span class="muted n">{t.results({ count: reply.total })}</span>{/if}
</div>
{#if error}<p class="notice error" role="alert">{error}</p>{/if}
{#if reply && reply.total > results.length}<p class="muted cut">{t.cut({ shown: count(results.length), count: reply.total })}</p>{/if}

{#if reply && !results.length}
	<p class="muted">{m.common.noMatch({ query: reply.query })}</p>
{:else if results.length}
	<div class="split">
		<table>
			<thead>
				<tr>
					<th class="mark" aria-label={m.common.selected}></th>
					<th class="caps">{m.menu.option}</th>
					<th class="caps">{t.page}</th>
					<th class="caps">{m.menu.type}</th>
					<th class="caps allowed">{m.menu.allowed}</th>
				</tr>
			</thead>
			<tbody>
				{#each results as r, i (key(r))}
					{#if r.menuId !== results[i - 1]?.menuId}
						<tr class="group"><th colspan="5" scope="rowgroup" class="caps"><a href={menuHref(r.menuId)}>{r.menuText}</a></th></tr>
					{/if}
					<tr aria-selected={selection.is(r)} onclick={() => selection.select(r)} ondblclick={() => openMenu(r)}>
						<td class="mark">{#if selection.is(r)}<Diamond />{/if}</td>
						<td class="option"><Selector text={r.option.text} raw={r.option.label} certainty={r.option.certainty} /></td>
						<td class="page">{pageText(r)}</td>
						<td class="type caps">{r.option.control === 'stepper' ? m.option.types.stepper : m.option.types[r.option.type]}</td>
						<td class="allowed">{allowed(r.option)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
		<aside class="pane" aria-label={t.pane} use:fitToViewport>
			{#if current}
				{@const r = current}
				{#key key(r)}
					<div class="enter">
						<OptionPane option={r.option} page={pageText(r)} menu={{ text: r.menuText, href: menuHref(r.menuId, r.page, r.position) }}>
							{#snippet action()}<Adder option={r.option} menu={r.menuName} pageName={r.pageName} {open} {choose} />{/snippet}
						</OptionPane>
						<p class="open"><a class="textbutton" href={menuHref(r.menuId, r.page, r.position)}>{t.openMenu}</a></p>
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
	.lead {
		margin: var(--space-2) 0 var(--space-4);
		max-width: 65ch;
	}
	.box {
		display: flex;
		align-items: baseline;
		gap: var(--space-4);
		max-width: 44rem;
	}
	.box input {
		flex: 1;
	}
	.n {
		white-space: nowrap;
	}
	.cut {
		margin-top: var(--space-2);
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
	/* A menu's results sit under its name, in the accent, as a list's groups do. */
	.group th {
		text-align: left;
		padding: var(--space-5) var(--space-3) var(--space-1);
		font-size: var(--step--1);
		font-weight: 500;
		border-bottom: 1px solid var(--rule-strong);
	}
	.group a {
		color: var(--accent);
		text-decoration: none;
	}
	.group a:hover {
		text-decoration: underline;
	}
	tbody tr.group:first-child th {
		padding-top: var(--space-3);
	}
	.mark {
		width: 1.5rem;
		padding-right: 0;
		vertical-align: middle;
	}
	.option {
		max-width: 26rem;
	}
	.page,
	.allowed {
		color: var(--muted);
		font-size: var(--step--1);
	}
	.type {
		font-size: var(--step--1);
		color: var(--muted);
		white-space: nowrap;
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
	.open {
		margin-top: var(--space-2);
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
