<!-- The catalog now against one saved before: what changed, menu by menu,
     and which recorded steps the changes break. The pane's counts are the
     doorways that filter the table. -->
<script lang="ts">
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.ts';
	import type { ListInfo, OptionDiff } from '#lib/api/types/catalog.ts';
	import type { CompareReply } from '#lib/api/types/recordings.ts';
	import { m } from '#lib/copy/messages.ts';
	import { count } from '#lib/format.ts';
	import { compare, recordingHref } from '#lib/recordings/api.ts';
	import { sentence, word } from '#lib/recordings/steps.ts';
	import Diamond from '#lib/ui/Diamond.svelte';
	import { param, setParams } from '#lib/ui/address.svelte.ts';
	import { fitToViewport } from '#lib/ui/fit.ts';
	import { Selection } from '#lib/ui/selection.svelte.ts';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import { onMount } from 'svelte';

	const t = m.compare;

	let path = $state(page.url.searchParams.get('path') ?? '');
	let reply = $state<CompareReply | null>(null);
	let error = $state('');
	let working = $state(false);

	async function run(event?: SubmitEvent) {
		event?.preventDefault();
		if (!path.trim()) return;
		working = true;
		error = '';
		try {
			reply = await compare(path);
			setParams({ path: path.trim() }, 'replace');
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
			reply = null;
		} finally {
			working = false;
		}
	}
	onMount(() => {
		if (path) run();
	});

	// A row of the table: what changed, under its menu's or recording's name.
	interface Row {
		group: string;
		groupHref?: string;
		what: string;
		change: string;
		before: string;
		after: string;
		door: string;
		warn?: boolean;
	}
	const door = (o: OptionDiff) =>
		o.change === 'added' ? 'optionsAdded' : o.change === 'removed' ? 'optionsRemoved' : o.change === 'renamed' ? 'optionsRenamed' : o.change === 'page' ? 'pages' : 'valuesChanged';
	let rows = $derived.by((): Row[] => {
		if (!reply) return [];
		const out: Row[] = [];
		for (const md of reply.diff.menus) {
			if (md.change !== 'changed') {
				out.push({ group: md.nameText, what: t.wholeMenu, change: t.changes[md.change], before: '', after: '', door: md.change === 'added' ? 'menusAdded' : 'menusRemoved' });
				continue;
			}
			for (const o of md.options) {
				const what = o.change === 'page' ? o.pageText : [o.pageText, o.text || o.label].filter(Boolean).join(' · ');
				const change = o.change === 'page' ? (o.after ? t.pageAdded : t.pageRemoved) : t.changes[o.change];
				out.push({ group: md.nameText, what, change, before: o.change === 'page' ? '' : o.before, after: o.change === 'page' ? '' : o.after, door: door(o) });
			}
		}
		for (const b of reply.breaks) {
			const step = { mod: b.mod, page: b.page, option: b.option, kind: b.kind, value: '', side: '', state: '', index: 0, file: 0, wildcard: false };
			out.push({ group: b.recording, groupHref: recordingHref(b.recording) + `?item=${b.step}`, what: `${t.step({ n: b.step })} · ${[b.mod, b.page, b.option].filter(Boolean).join(' · ')}`, change: word(b.check), before: '', after: sentence(b.check, step), door: 'breaks', warn: true });
		}
		return out;
	});

	// The doorways: each count, and what it shows.
	let doors = $derived.by(() => {
		const c = reply?.diff.counts;
		if (!c || !reply) return [];
		return [
			{ id: 'all', label: t.all, n: rows.length },
			{ id: 'menusAdded', label: t.menusAdded, n: c.menusAdded },
			{ id: 'menusRemoved', label: t.menusRemoved, n: c.menusRemoved },
			{ id: 'optionsAdded', label: t.optionsAdded, n: c.optionsAdded },
			{ id: 'optionsRemoved', label: t.optionsRemoved, n: c.optionsRemoved },
			{ id: 'optionsRenamed', label: t.optionsRenamed, n: c.optionsRenamed },
			{ id: 'valuesChanged', label: t.valuesChanged, n: c.valuesChanged },
			{ id: 'breaks', label: t.breaks, n: reply.breaks.length }
		];
	});
	let most = $derived(Math.max(1, ...doors.filter((d) => d.id !== 'all').map((d) => d.n)));
	const selection = new Selection(() => doors, (d) => d.id);
	let marked = $derived(selection.current);
	let show = $derived(param('show') ?? 'all');
	let shown = $derived(show === 'all' ? rows : rows.filter((r) => r.door === show));
	const open = (id: string) => setParams({ show: id === 'all' ? null : id });

	const scanned = (l: ListInfo) => t.scannedFrom({ list: l.name, profile: l.profile || '—', when: new Date(l.scannedAt).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' }) });

	useShortcuts(() =>
		reply
			? [
					{ key: 'ArrowDown', label: m.common.next, hidden: true, run: () => selection.move(1) },
					{ key: 'ArrowUp', label: m.common.previous, hidden: true, run: () => selection.move(-1) },
					{ key: 'Enter', label: marked ? marked.label : m.common.open, disabled: !marked, run: () => marked && open(marked.id) }
				]
			: []
	);
</script>

<h1 class="caps">{t.title}</h1>
<p class="muted lead">{t.lead}</p>
<form class="choose" onsubmit={run}>
	<label>
		<span class="caps">{t.file}</span>
		<input bind:value={path} required autocomplete="off" spellcheck="false" />
	</label>
	<button class="button primary" disabled={working}>{t.run}</button>
</form>
{#if error}<p class="notice error" role="alert">{error}</p>{/if}

{#if reply}
	<div class="split">
		<div>
			{#if !rows.length}
				<p class="muted">{t.nothing}</p>
			{:else if !shown.length}
				<p class="muted">{show === 'breaks' ? t.noBreaks : t.nothing}</p>
			{:else}
				<table>
					<thead>
						<tr>
							<th class="caps">{t.what}</th>
							<th class="caps">{t.change}</th>
							<th class="caps">{t.before}</th>
							<th class="caps">{t.after}</th>
						</tr>
					</thead>
					<tbody>
						{#each shown as r, i (i)}
							{#if r.group !== shown[i - 1]?.group}
								<tr class="group"><th colspan="4" scope="rowgroup" class="caps">{#if r.groupHref}<a href={r.groupHref}>{r.group}</a>{:else}{r.group}{/if}</th></tr>
							{/if}
							<tr>
								<td class="what">{r.what}</td>
								<td class="change caps" class:warn={r.warn}>{r.change}</td>
								<td class="before"><del>{r.before}</del></td>
								<td class="after">{#if r.warn}{r.after}{:else}<ins>{r.after}</ins>{/if}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}
		</div>
		<aside class="pane" aria-label={t.changed} use:fitToViewport>
			<dl class="sides">
				<dt class="caps">{t.older}</dt><dd>{scanned(reply.diff.old)}</dd>
				<dt class="caps">{t.current}</dt><dd>{scanned(reply.diff.new)}</dd>
			</dl>
			<h2 class="caps">{t.changed}</h2>
			<ul class="doors">
				{#each doors as d (d.id)}
					<li class:marked={selection.is(d)} class:current={show === d.id}>
						<span class="mark">{#if selection.is(d)}<Diamond />{/if}</span>
						<button type="button" class="caps" onclick={() => { selection.select(d); open(d.id); }} onfocus={() => selection.select(d)} aria-pressed={show === d.id}>{d.label}</button>
						<span class="n">{count(d.n)}</span>
						{#if d.id !== 'all'}<span class="bar" style:width="{(d.n / most) * 100}%"></span>{/if}
					</li>
				{/each}
			</ul>
		</aside>
	</div>
{/if}

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
	.choose {
		display: flex;
		align-items: end;
		gap: var(--space-3);
		max-width: 44rem;
		margin-bottom: var(--space-4);
	}
	.choose label {
		display: grid;
		gap: var(--space-1);
		flex: 1;
	}
	.choose label span {
		font-size: var(--step--1);
		color: var(--muted);
	}
	.choose .button {
		padding-block: var(--space-2);
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
	thead th {
		text-align: left;
		font-size: var(--step--1);
		color: var(--muted);
		font-weight: inherit;
		padding: var(--space-2) var(--space-3);
		border-bottom: 1px solid var(--rule-strong);
	}
	td {
		padding: var(--space-2) var(--space-3);
		border-bottom: 1px solid var(--rule);
		vertical-align: baseline;
	}
	.group th {
		text-align: left;
		padding: var(--space-5) var(--space-3) var(--space-1);
		color: var(--accent);
		font-size: var(--step--1);
		font-weight: 500;
		border-bottom: 1px solid var(--rule-strong);
	}
	.group a {
		color: inherit;
		text-decoration: none;
	}
	.group a:hover {
		text-decoration: underline;
	}
	tbody tr.group:first-child th {
		padding-top: var(--space-3);
	}
	.change {
		font-size: var(--step--1);
		color: var(--muted);
		white-space: nowrap;
	}
	.change.warn {
		color: var(--warn);
	}
	.before,
	.after {
		font-size: var(--step--1);
		overflow-wrap: anywhere;
	}
	del {
		color: var(--muted);
	}
	ins {
		text-decoration: none;
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
	.sides {
		display: grid;
		grid-template-columns: 5rem 1fr;
		gap: var(--space-1) var(--space-3);
		margin: 0 0 var(--space-5);
		font-size: var(--step--1);
	}
	.sides dt {
		color: var(--muted);
		padding-top: 0.15em;
	}
	.sides dd {
		margin: 0;
	}
	.pane h2 {
		font-size: var(--step--1);
		font-weight: 500;
		color: var(--accent);
		padding-bottom: var(--space-1);
		border-bottom: 1px solid var(--rule-strong);
	}
	/* The Numbers Are Doors: each count is the way into its rows. */
	.doors {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.doors li {
		position: relative;
		display: grid;
		grid-template-columns: 1.25rem 1fr auto;
		align-items: baseline;
		padding: var(--space-2) var(--space-1) var(--space-2) 0;
		border-bottom: 1px solid var(--rule);
	}
	.doors li.marked {
		background: var(--select);
	}
	.mark {
		display: inline-flex;
		align-self: center;
		padding-left: var(--space-1);
	}
	.doors button {
		background: none;
		border: 0;
		padding: 0;
		text-align: left;
		font-size: var(--step-1);
		color: var(--muted);
		cursor: pointer;
	}
	.doors button:hover,
	.doors li.current button {
		color: var(--ink);
	}
	.doors .n {
		color: var(--muted);
	}
	.bar {
		position: absolute;
		left: 1.25rem;
		bottom: -1px;
		max-width: calc(100% - 1.25rem);
		height: 2px;
		background: var(--accent);
		opacity: 0.55;
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
