<!-- Every menu in the list, as an item table with the selected one in the
     pane; the list's home. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { ApiError } from '#lib/api/client.ts';
	import { live, scanStarted } from '#lib/api/live.svelte.ts';
	import type { Skipped, Summary } from '#lib/api/types/catalog.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';
	import { count } from '#lib/format.ts';
	import { menuHref, rescan } from '#lib/menus/api.ts';
	import Selector from '#lib/menus/Selector.svelte';
	import Diamond from '#lib/ui/Diamond.svelte';
	import Segmented from '#lib/ui/Segmented.svelte';
	import SortHeader from '#lib/ui/SortHeader.svelte';
	import { clipped } from '#lib/ui/clipped.ts';
	import { fitToViewport } from '#lib/ui/fit.ts';
	import { Selection } from '#lib/ui/selection.svelte.ts';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import { sorted, type Sort } from '#lib/ui/sort.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const t = m.menus;

	let source = $derived(data.status.source);
	let reply = $derived(data.menus);
	let menus = $derived(reply?.menus ?? []);
	let notMenus = $derived((reply?.skipped ?? []).filter((s) => s.reason === 'no-quest' || s.reason === 'empty'));
	let unreadable = $derived((reply?.skipped ?? []).filter((s) => s.reason === 'unreadable'));
	let inactive = $derived(menus.filter((s) => !s.active).length);

	let which = $state('all');
	let query = $state('');
	let sort = $state<Sort>({ key: 'menu', descending: false });
	let showNotMenus = $state(false);
	let error = $state('');

	let shown = $derived.by(() => {
		const q = query.trim().toLowerCase();
		const list = menus.filter((s) => (which === 'all' || s.active) && (!q || s.nameText.toLowerCase().includes(q) || s.name.toLowerCase().includes(q) || s.mod.toLowerCase().includes(q)));
		return sorted(list, sort, {
			menu: (s) => s.nameText,
			mod: (s) => s.mod,
			kind: (s) => t.kinds[s.kind],
			pages: (s) => s.pages,
			options: (s) => s.options
		});
	});

	const selection = new Selection(() => shown, (s) => s.id, { param: 'item' });
	let current = $derived(selection.current);
	const open = (s: Summary) => goto(menuHref(s.id));

	async function startScan() {
		error = '';
		try {
			scanStarted(await rescan());
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
		}
	}

	let scanError = $derived(data.menusError || live.scan.error);

	useShortcuts(() => [
		{ key: 'ArrowDown', label: m.common.next, hidden: true, run: () => selection.move(1) },
		{ key: 'ArrowUp', label: m.common.previous, hidden: true, run: () => selection.move(-1) },
		{ key: 'Enter', label: m.common.open, disabled: !current, run: () => current && open(current) },
		...(source ? [{ key: 'r', shown: 'R', label: t.rescan, disabled: live.scan.scanning, run: startScan }] : [])
	]);
</script>

<div class="bar">
	<h1 class="caps">{t.title}</h1>
	{#if menus.length}
		<Segmented
			bind:value={which}
			options={[
				{ value: 'all', label: t.which.all },
				{ value: 'active', label: t.which.active }
			]}
			label={t.which.label}
		/>
		<span class="muted n" use:clipped>{t.count({ count: menus.length })}{#if inactive}{' · '}{t.inactiveCount({ count: inactive })}{/if}</span>
		<input type="search" bind:value={query} placeholder={t.filter} aria-label={t.filterLabel} />
	{/if}
</div>
<p class="muted where">
	{#if source?.kind === 'mo2'}
		<Fill text={t.inInstance.text}>{#snippet path()}<span class="path">{source.path}</span>{/snippet}{#snippet profile()}<span class="path">{source.profile}</span>{/snippet}</Fill>
	{:else if source}
		<Fill text={t.inData.text}>{#snippet path()}<span class="path">{source.path}</span>{/snippet}</Fill>
	{:else}
		<Fill text={t.inCatalog.text}>{#snippet path()}<span class="path">{data.status.catalogFile}</span>{/snippet}{#snippet list()}<span class="path">{reply?.list.path ?? ''}</span>{/snippet}</Fill>
	{/if}
</p>

{#if error}<p class="notice error" role="alert">{error}</p>{/if}
{#if scanError}<p class="notice error" role="alert">{t.scanFailed({ error: scanError })}</p>{/if}

{#snippet scripts(list: Skipped[])}
	<ul class="files">
		{#each list as s (s.path)}
			<li class:inactive={s.plugin && !s.active}>
				{#if s.nameText}<span class="title">{s.nameText}</span>{/if}
				<span class="file">{s.path.replace(/^Scripts\//i, '')}</span>
				<span class="muted">{s.plugin || s.mod}{#if s.archive}{' · '}{m.common.inArchive({ archive: s.archive })}{/if}{#if s.plugin && !s.active}{' · '}{t.notActive}{/if}</span>
			</li>
		{/each}
	</ul>
{/snippet}

{#if notMenus.length || unreadable.length}
	<div class="skipped">
		{#if notMenus.length}
			<p class="muted">{t.notMenus({ count: notMenus.length })} <button type="button" class="textbutton" onclick={() => (showNotMenus = !showNotMenus)} aria-expanded={showNotMenus}>{showNotMenus ? t.hideThem : t.showThem}</button></p>
			{#if showNotMenus}{@render scripts(notMenus)}{/if}
		{/if}
		{#if unreadable.length}
			<p class="notice warn">{t.unreadable({ count: unreadable.length })}</p>
			<ul class="files">
				{#each unreadable as s (s.path)}<li><span class="file">{s.path}</span> <span class="muted">{s.mod}{#if s.archive}{' · '}{m.common.inArchive({ archive: s.archive })}{/if}: {s.detail}</span></li>{/each}
			</ul>
		{/if}
	</div>
{/if}

{#if !reply && live.scan.scanning}
	<p class="empty">{t.firstScan}</p>
{:else if reply && !menus.length}
	<p class="empty">{t.none}</p>
{:else if reply && !shown.length}
	<p class="muted">{m.common.noMatch({ query })}</p>
{:else if reply}
	<div class="split">
		<table>
			<thead>
				<tr>
					<th class="mark" aria-label={m.common.selected}></th>
					<SortHeader label={t.menu} key="menu" bind:sort />
					<SortHeader label={t.mod} key="mod" bind:sort class="mod" />
					<SortHeader label={t.kind} key="kind" bind:sort />
					<SortHeader label={t.pages} key="pages" bind:sort descendingFirst align="end" />
					<SortHeader label={t.options} key="options" bind:sort descendingFirst align="end" />
				</tr>
			</thead>
			<tbody>
				{#each shown as s (s.id)}
					<tr aria-selected={selection.is(s)} class:inactive={!s.active} onclick={() => selection.select(s)} ondblclick={() => open(s)}>
						<td class="mark">{#if selection.is(s)}<Diamond />{/if}</td>
						<td class="name"><a href={menuHref(s.id)}><Selector text={s.nameText} raw={s.name} /></a></td>
						<td class="mod muted"><span use:clipped>{s.mod}</span></td>
						<td class="kind caps">{t.kinds[s.kind]}</td>
						<td class="n">{s.pages ? count(s.pages) : ''}</td>
						<td class="n">{count(s.options)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
		<aside class="pane" aria-label={t.pane} use:fitToViewport>
			{#if current}
				<h2 class="caps">{current.nameText}</h2>
				<p class="muted">{t.kinds[current.kind]} · {current.mod}</p>
				<dl>
					{#if current.name !== current.nameText}<dt class="caps">{m.option.selector}</dt><dd class="raw">{current.name}</dd>{/if}
					{#if current.plugin}<dt class="caps">{t.plugin}</dt><dd>{current.plugin}</dd>{/if}
					<dt class="caps">{current.kind === 'mcmhelper' ? t.config : t.script}</dt><dd>{current.script}</dd>
					<dt class="caps">{t.definedIn}</dt><dd><span class="raw">{current.source.path}</span>{#if current.source.archive}<span class="muted small">{m.common.inArchive({ archive: current.source.archive })}</span>{/if}</dd>
					{#if current.pages}<dt class="caps">{t.pages}</dt><dd>{count(current.pages)}</dd>{/if}
					<dt class="caps">{t.options}</dt><dd>{count(current.options)}</dd>
					<dt class="caps">{t.certainty}</dt>
					<dd>{#if !current.options}{t.noOptions}{:else if current.uncertain}{t.someUncertain({ uncertain: current.uncertain, count: current.options })}{:else}{t.allExact({ count: current.options })}{/if}</dd>
				</dl>
				<div class="actions"><a class="button primary" href={menuHref(current.id)}>{m.common.open}</a></div>
				{#if !current.active}<p class="notice warn">{t.inactive({ plugin: current.plugin })}</p>{/if}
				{#each current.notes as note, i (i)}
					{#if note.kind === 'no-plugin'}<p class="notice">{t.noPlugin}</p>{:else if note.kind === 'plugin-guessed'}<p class="notice">{t.pluginGuessed}</p>{:else if note.kind === 'overridden'}<p class="notice">{t.overridden({ mod: note.detail })}</p>{:else if note.kind === 'name-from-script'}<p class="notice">{t.nameFromScript}</p>{:else if note.kind === 'pages-guessed'}<p class="notice">{t.pagesGuessed}</p>{:else if note.kind === 'framework'}<p class="notice">{t.framework({ script: note.detail })}</p>{:else if note.kind === 'source-stale'}<p class="notice">{t.sourceStale}</p>{/if}
				{/each}
			{/if}
		</aside>
	</div>
{/if}

<style>
	/* Keeps the filter over the table, not the pane. */
	.bar {
		display: flex;
		align-items: baseline;
		gap: var(--space-4);
		margin-right: calc(clamp(18rem, 26vw, 24rem) + var(--space-5));
	}
	h1 {
		font-size: var(--step-3);
		font-weight: 600;
		line-height: 1;
		position: relative;
		top: 0.21em;
	}
	@media (max-width: 90rem) {
		.bar {
			margin-right: 0;
		}
	}
	.bar .n {
		white-space: nowrap;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.bar input {
		margin-left: auto;
		width: 16rem;
		flex: 0 1 16rem;
		min-width: 9rem;
	}
	.where {
		margin: var(--space-3) 0 var(--space-4);
		max-width: 80ch;
	}
	.path {
		color: var(--ink);
		word-break: break-all;
	}
	.skipped {
		margin-bottom: var(--space-4);
	}
	.skipped > p {
		max-width: 80ch;
	}
	.files {
		list-style: none;
		margin: var(--space-2) 0 var(--space-5);
		padding: 0;
		columns: 3 20rem;
		column-gap: var(--space-6);
		font-size: var(--step--1);
	}
	.files li {
		break-inside: avoid;
		padding: 1px 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.files .title {
		color: var(--ink);
		margin-right: var(--space-1);
	}
	.file {
		font-family: var(--font-mono);
		color: var(--muted);
		margin-right: var(--space-1);
	}
	.files .inactive .title {
		color: var(--muted);
	}
	.empty {
		max-width: 60ch;
	}
	.split {
		display: grid;
		grid-template-columns: minmax(0, 1fr) clamp(18rem, 26vw, 24rem);
		gap: var(--space-5);
		align-items: start;
	}
	table {
		width: 100%;
		border-collapse: collapse;
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
	tr.inactive td {
		color: var(--muted);
	}
	.mark {
		width: 1.5rem;
		padding-right: 0;
		vertical-align: middle;
	}
	th.mark {
		border-bottom: 1px solid var(--rule-strong);
	}
	.name a {
		display: block;
		font-size: var(--step-1);
		text-decoration: none;
		color: inherit;
	}
	.name a:hover :global(.text) {
		text-decoration: underline;
		text-decoration-color: var(--accent);
		text-underline-offset: 0.2em;
	}
	td.mod {
		max-width: 18rem;
		font-size: var(--step--1);
	}
	td.mod span {
		display: block;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.kind {
		font-size: var(--step--1);
		color: var(--muted);
		white-space: nowrap;
	}
	td.n {
		text-align: right;
		width: 5.5rem;
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
	.pane dd .small {
		display: block;
		font-size: var(--step--1);
	}
	.raw {
		font-family: var(--font-mono);
		font-size: var(--step--1);
	}
	.actions a {
		text-decoration: none;
	}
	.pane .notice {
		margin-top: var(--space-3);
	}
	@media (max-width: 60rem) {
		.bar {
			margin-right: 0;
			flex-wrap: wrap;
		}
		.split {
			grid-template-columns: 1fr;
		}
		.pane {
			position: static;
			min-height: 0;
			border-left: 0;
			border-top: 1px solid var(--rule-strong);
		}
		.split :global(.mod) {
			display: none;
		}
	}
</style>
