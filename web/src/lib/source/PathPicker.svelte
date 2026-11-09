<!-- A dialog that browses one folder at a time, starting where MCM Lexicon
     runs from, for a mod list, a catalog file, or a folder to save a file
     in. For a list, folders that are one (an MO2 instance, or a Data
     folder), or lead to one (a game folder), are marked, one that leads to
     one opens straight onto it, and the folder listed is chosen. For a
     catalog, the files that are one are marked, and one is chosen by
     clicking it. For a folder, nothing is marked, and the one listed is
     chosen. -->
<script lang="ts">
	import { tick } from 'svelte';
	import { ApiError } from '#lib/api/client.ts';
	import { browseFolders } from '#lib/api/status.ts';
	import type { Folder, Kind, Listing, Looking } from '#lib/api/types/modlist.ts';
	import { m } from '#lib/copy/messages.ts';
	import { closeOnBackdrop } from '#lib/ui/backdrop.ts';
	import EntryIcon from './EntryIcon.svelte';

	let {
		looking = 'list',
		start = '',
		onpick,
		oncancel
	}: {
		looking?: Looking;
		start?: string; // the folder to open on; "" for where MCM Lexicon runs from
		onpick: (path: string) => void;
		oncancel: () => void;
	} = $props();

	const t = m.source.picker;

	let listing = $state<Listing>();
	let typed = $state(''); // the path field, which follows the listing until typed in
	let error = $state('');
	let loading = $state(false);
	let list = $state<HTMLElement>();
	let dialog: HTMLDialogElement;
	const id = $props.id();

	$effect(() => {
		dialog.showModal();
	});

	// go lists a folder. Focus moves to the list, unless the path was typed:
	// then it stays in the field, which keeps what was typed if it fails.
	async function go(path = '', typedIn = false) {
		error = '';
		loading = true;
		try {
			listing = await browseFolders(path, looking);
			typed = listing.path;
			if (!typedIn) {
				await tick();
				list?.querySelector('button')?.focus();
			}
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
			// A remembered folder that's gone: start where MCM Lexicon runs from.
			if (!listing && path) return go('');
		} finally {
			loading = false;
		}
	}
	// svelte-ignore state_referenced_locally
	go(start);

	// Enter lists the typed path, and doesn't submit the form the dialog sits in.
	function pathKey(e: KeyboardEvent) {
		if (e.key !== 'Enter') return;
		e.preventDefault();
		go(typed, true);
	}

	const kindLabel = (k: Kind | undefined) => (k === 'mo2' ? t.mo2 : t.data);
	const markLabel = (f: Folder) => (f.mark === 'here' ? kindLabel(f.kind) : f.mark === 'inside' ? t.dataInside : '');

	// Esc, or a click outside it, closes the dialog, and the form behind it
	// takes it away.
	function cancel(e: Event) {
		e.preventDefault();
		oncancel();
	}
</script>

<dialog bind:this={dialog} aria-labelledby="{id}-title" oncancel={cancel} {...closeOnBackdrop(cancel)}>
	<h2 id="{id}-title" class="caps">{looking === 'catalog' ? t.titleCatalog : looking === 'capture' ? t.titleCapture : looking === 'folder' ? t.titleFolder : t.title}</h2>
	{#if listing}
		<div class="where">
			{#if listing.roots.length > 1}
				<select aria-label={t.drive} value={listing.roots.find((r) => listing!.path.startsWith(r)) ?? ''} onchange={(e) => go(e.currentTarget.value)}>
					{#each listing.roots as r (r)}<option value={r}>{r}</option>{/each}
				</select>
			{/if}
			<input class="path" bind:value={typed} onkeydown={pathKey} aria-label={t.path} autocomplete="off" spellcheck="false" />
		</div>
		{#if listing.mark === 'here'}<p class="found">{listing.kind === 'mo2' ? t.isMo2 : t.isData}</p>{/if}
		<ul bind:this={list} class:loading>
			{#if listing.parent}
				<li>
					<button type="button" class="row up" onclick={() => go(listing!.parent)}>
						<span class="label"><EntryIcon kind="up" /><span class="caps">{t.up}</span></span>
					</button>
				</li>
			{/if}
			{#each listing.folders as f (f.path)}
				<li>
					<button type="button" class="row" onclick={() => go(f.inside ?? f.path)}>
						<span class="label"><EntryIcon kind="folder" /><span class="name">{f.name}</span></span>
						{#if f.mark}<span class="mark caps" class:here={f.mark === 'here'}>{markLabel(f)}</span>{/if}
					</button>
				</li>
			{/each}
			<!-- A catalog or capture file is chosen by clicking it; any other
			     file is there to tell where you are. -->
			{#each listing.files as f (f.path)}
				{#if f.mark === 'here'}
					<li>
						<button type="button" class="row" onclick={() => onpick(f.path)}>
							<span class="label"><EntryIcon kind="file" /><span class="name">{f.name}</span></span>
							<span class="mark caps here">{looking === 'capture' ? t.capture : t.catalog}</span>
						</button>
					</li>
				{:else}
					<li class="row file"><span class="label"><EntryIcon kind="file" /><span class="name">{f.name}</span></span></li>
				{/if}
			{/each}
			{#if !listing.folders.length && !listing.files.length}
				<li class="muted empty">{t.empty}</li>
			{/if}
		</ul>
	{/if}
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}
	<div class="buttons">
		{#if looking === 'list' || looking === 'folder'}
			<button type="button" class={['button', (looking === 'folder' || listing?.mark === 'here') && 'primary']} disabled={!listing} onclick={() => listing && onpick(listing.path)}>{t.choose}</button>
		{:else if listing && !listing.files.some((f) => f.mark === 'here')}
			<p class="muted small">{looking === 'capture' ? t.noCaptures : t.noCatalogs}</p>
		{/if}
		<button type="button" class="button" onclick={oncancel}>{m.common.cancel}</button>
	</div>
</dialog>

<style>
	/* As Quick Find's, wider for paths. */
	dialog[open] {
		display: grid;
		gap: var(--space-3);
		width: min(40rem, 90vw);
		margin: 12vh auto auto;
		padding: var(--space-5);
		background: var(--smoke);
		color: var(--ink);
		border: 1px solid var(--rule-strong);
		border-radius: var(--radius);
		box-shadow: 0 1.5rem 3rem rgb(0 0 0 / 0.45);
	}
	dialog::backdrop {
		background: rgb(0 0 0 / 0.45);
	}
	h2 {
		font-size: var(--step-1);
		font-weight: 600;
	}
	.where {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}
	.path {
		flex: 1;
		min-width: 0;
	}
	.found {
		font-size: var(--step--1);
	}
	ul {
		list-style: none;
		margin: 0;
		padding: 0;
		max-height: min(24rem, 45vh);
		overflow-y: auto;
		border-top: 1px solid var(--rule-strong);
	}
	ul.loading {
		opacity: 0.6;
	}
	.row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-4);
		width: 100%;
		padding: var(--space-2) var(--space-3);
		border: 0;
		border-bottom: 1px solid var(--rule);
		background: none;
		color: var(--ink);
		font: inherit;
		text-align: left;
		cursor: pointer;
	}
	button.row:hover {
		background: var(--glass);
	}
	.row:focus-visible {
		background: var(--select);
	}
	.up,
	.file {
		color: var(--muted);
	}
	.up .caps {
		font-size: var(--step--1);
	}
	.file {
		cursor: default;
	}
	.label {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		min-width: 0;
	}
	.name {
		min-width: 0;
		overflow-wrap: anywhere;
	}
	.mark {
		flex: none;
		font-size: var(--step--1);
		color: var(--muted);
	}
	.mark.here {
		color: var(--ink);
	}
	.empty {
		padding: var(--space-2) var(--space-3);
	}
	.buttons {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}
	.buttons .small {
		margin: 0 auto 0 0;
		font-size: var(--step--1);
	}
</style>
