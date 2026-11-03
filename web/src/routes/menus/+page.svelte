<!-- The list's home, shown while no menu is chosen from the column: where
     the list is, how the scan went, and the scripts that aren't menus. -->
<script lang="ts">
	import { ApiError } from '#lib/api/client.ts';
	import { live, scanStarted } from '#lib/api/live.svelte.ts';
	import type { Skipped } from '#lib/api/types/catalog.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';
	import { rescan } from '#lib/menus/api.ts';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const t = m.menus;

	let source = $derived(data.status.source);
	let reply = $derived(data.menus);
	let menus = $derived(reply?.menus ?? []);
	let notMenus = $derived((reply?.skipped ?? []).filter((s) => s.reason === 'no-quest' || s.reason === 'empty'));
	let unreadable = $derived((reply?.skipped ?? []).filter((s) => s.reason === 'unreadable'));
	let inactive = $derived(menus.filter((s) => !s.active).length);
	let showNotMenus = $state(false);
	let error = $state('');

	async function startScan() {
		error = '';
		try {
			scanStarted(await rescan());
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
		}
	}

	let scanError = $derived(data.menusError || live.scan.error);

	useShortcuts(() => (source ? [{ key: 'r', shown: 'R', label: t.rescan, disabled: live.scan.scanning, run: startScan }] : []));
</script>

<div class="bar">
	<h1 class="caps">{t.title}</h1>
	{#if menus.length}
		<span class="muted n">{t.count({ count: menus.length })}{#if inactive}{' · '}{t.inactiveCount({ count: inactive })}{/if}</span>
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

{#if !reply && live.scan.scanning}
	<p class="empty">{t.firstScan}</p>
{:else if reply && !menus.length}
	<p class="empty">{t.none}</p>
{:else if reply}
	<p class="lead">{t.pickOne}</p>
{/if}

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

<style>
	.bar {
		display: flex;
		align-items: baseline;
		gap: var(--space-4);
	}
	h1 {
		font-size: var(--step-3);
		font-weight: 600;
		line-height: 1;
		position: relative;
		top: 0.21em;
	}
	.bar .n {
		white-space: nowrap;
	}
	.where {
		margin: var(--space-3) 0 0;
		max-width: 80ch;
		overflow-wrap: anywhere;
	}
	.path {
		font-family: var(--font-mono);
		font-size: var(--step--1);
	}
	.lead {
		margin: var(--space-4) 0 0;
		max-width: 60ch;
	}
	.empty {
		margin-top: var(--space-5);
		max-width: 60ch;
	}
	.notice {
		margin-top: var(--space-3);
		max-width: 70ch;
	}
	.skipped {
		margin-top: var(--space-5);
		max-width: 80ch;
	}
	.skipped > p {
		margin: 0 0 var(--space-2);
	}
	.files {
		list-style: none;
		margin: var(--space-2) 0 var(--space-4);
		padding: 0;
		font-size: var(--step--1);
	}
	.files li {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1) var(--space-3);
		padding: var(--space-1) 0;
		border-bottom: 1px solid var(--rule);
	}
	.files .title {
		min-width: 12rem;
	}
	.file {
		font-family: var(--font-mono);
	}
	.files .inactive .title {
		color: var(--muted);
	}
</style>
