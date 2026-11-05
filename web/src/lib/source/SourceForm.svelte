<!-- Where the mod list is: asked when MCM Lexicon finds none, or opened in
     Settings to change the one in use. For a machine without the list, a
     catalog file can be opened instead. -->
<script lang="ts">
	import { tick, untrack } from 'svelte';
	import { invalidateAll } from '$app/navigation';
	import { ApiError } from '#lib/api/client.ts';
	import { checkSource, setSource } from '#lib/api/status.ts';
	import type { Source } from '#lib/api/types/modlist.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';
	import PathPicker from './PathPicker.svelte';

	let {
		current,
		changes = 0,
		close
	}: {
		current?: { path: string; profile: string; catalog: string }; // what's in use, when changing it, in a page that has its own heading
		changes?: number; // recordings the draft changes, which another list discards
		close?: (saved: boolean) => Promise<void>; // ends a change
	} = $props();

	const t = m.source;
	const id = $props.id();

	let path = $state(untrack(() => current?.path ?? ''));
	let profile = $state(untrack(() => current?.profile ?? ''));
	let catalog = $state(untrack(() => current?.catalog ?? ''));
	let listError = $state('');
	let catalogError = $state('');
	let saving = $state(false);
	let browsing = $state(false);
	let input = $state<HTMLInputElement>();

	// A mod list or a catalog file, one at a time, so which one a button
	// opens is never in doubt: the list unless a catalog is what's in use,
	// and the other a link away, since few will need a catalog. What's typed
	// in the half not shown is kept, but not used.
	let mode = $state<'list' | 'catalog'>(untrack(() => (current?.catalog ? 'catalog' : 'list')));
	let catalogInput = $state<HTMLInputElement>();
	let browsingCatalog = $state(false);
	// The picker opens on the folder of the file named, if any.
	let catalogFolder = $derived(catalog.trim().replace(/[\\/][^\\/]*$/, ''));
	async function endBrowseCatalog(picked?: string) {
		if (picked !== undefined) catalog = picked;
		browsingCatalog = false;
		await tick();
		catalogInput?.focus();
	}
	async function switchMode() {
		mode = mode === 'list' ? 'catalog' : 'list';
		await tick();
		(mode === 'list' ? input : catalogInput)?.focus();
	}

	// What's at the folder in the field, read as it changes (a moment after
	// typing stops): for an MO2 instance, its profiles, offered as the
	// profile's choices, with the one in use chosen, or for another instance
	// the one MO2 has selected.
	let found = $state<Source | null>(null);
	let checking = $state(false);
	let checkError = $state('');
	let checks = 0;
	$effect(() => {
		const want = path.trim();
		const n = ++checks;
		found = null;
		checkError = '';
		listError = ''; // Use's refusal was about the folder before
		if (!want) return;
		checking = true;
		const timer = setTimeout(async () => {
			try {
				const src = await checkSource(want);
				if (n !== checks) return;
				found = src;
				profile = src.kind === 'mo2' && src.path === current?.path && src.profiles.includes(current.profile) ? current.profile : src.profile;
			} catch (e) {
				if (n === checks) checkError = e instanceof ApiError ? e.message : String(e);
			} finally {
				if (n === checks) checking = false;
			}
		}, 300);
		return () => clearTimeout(timer);
	});

	// endBrowse closes the picker, filling the field with the folder picked,
	// if one was, and goes back to the field once the dialog is gone (which
	// gives focus back to Browse as it closes).
	async function endBrowse(picked?: string) {
		if (picked !== undefined) path = picked;
		browsing = false;
		await tick();
		input?.focus();
	}

	async function choose(req: Parameters<typeof setSource>[0], report: (message: string) => void) {
		saving = true;
		listError = '';
		catalogError = '';
		try {
			await setSource(req);
			await close?.(true); // settles before the reload, which it would cancel
			await invalidateAll(); // reload the status, which now has the list
		} catch (e) {
			report(e instanceof ApiError ? e.message : String(e));
		} finally {
			saving = false;
		}
	}

	function submitList(event: SubmitEvent) {
		event.preventDefault();
		choose({ path, profile: found?.kind === 'mo2' ? profile : undefined }, (message) => (listError = message));
	}

	function submitCatalog(event: SubmitEvent) {
		event.preventDefault();
		choose({ catalog }, (message) => (catalogError = message));
	}
</script>

<div class="intro">
	{#if !current}<h1 class="caps">{t.title}</h1>{/if}
	<p>
		{#if current}
			<Fill text={t.change.text}>{#snippet flag()}<code>-list</code>{/snippet}</Fill>
		{:else}
			<Fill text={t.notFound.text}>{#snippet ini()}<code>ModOrganizer.ini</code>{/snippet}{#snippet mods()}<code>mods</code>{/snippet}{#snippet profiles()}<code>profiles</code>{/snippet}{#snippet data()}<code>Data</code>{/snippet}</Fill>
		{/if}
	</p>
	{#if changes}<p class="notice warn">{t.discards({ count: changes })}</p>{/if}
</div>

{#if mode === 'list'}
<form onsubmit={submitList}>
	<!-- What's in use, when it's the other kind: this half would replace it,
	     which is worth amber. -->
	{#if current?.catalog}<p class="notice warn">{t.catalogInUse}</p>{/if}
	<div class="field">
		<label class="caps" for="{id}-folder">{t.folder}</label>
		<div class="entry">
			<input id="{id}-folder" bind:this={input} bind:value={path} required autocomplete="off" spellcheck="false" />
			<button type="button" class="button" aria-expanded={browsing} onclick={() => (browsing = true)}>{t.picker.browse}</button>
		</div>
		{#if browsing}<PathPicker start={path} onpick={endBrowse} oncancel={() => endBrowse()} />{/if}
	</div>
	{#if checking}
		<p class="muted small">{t.checking}</p>
	{:else if checkError && !listError}
		<!-- Said once: Use's refusal below says the same, in red. -->
		<p class="muted small">{checkError}</p>
	{:else if found?.kind === 'mo2'}
		<div class="field">
			<label class="caps" for="{id}-profile">{t.profile}</label>
			<select id="{id}-profile" bind:value={profile}>
				{#each found.profiles as p (p)}<option value={p}>{p}</option>{/each}
			</select>
			<span class="muted small">{t.profileHint}</span>
		</div>
	{:else if found}
		<p class="muted small">{t.noProfiles}</p>
	{/if}
	<div class="buttons">
		<button class="button primary" disabled={saving}>{t.use}</button>
		{#if close}<button type="button" class="button" onclick={() => close(false)} disabled={saving}>{m.common.cancel}</button>{/if}
	</div>
	{#if listError}
		<p class="notice error" role="alert">{listError}</p>
	{/if}
</form>
{:else}
	<form class="catalog" onsubmit={submitCatalog}>
		{#if current?.path}<p class="notice warn">{t.listInUse}</p>{/if}
		<p>{t.catalogLead}</p>
		<div class="field">
			<label class="caps" for="{id}-catalog">{t.catalogFile}</label>
			<div class="entry">
				<input id="{id}-catalog" bind:this={catalogInput} bind:value={catalog} required autocomplete="off" spellcheck="false" />
				<button type="button" class="button" aria-expanded={browsingCatalog} onclick={() => (browsingCatalog = true)}>{t.picker.browse}</button>
			</div>
			{#if browsingCatalog}<PathPicker looking="catalog" start={catalogFolder} onpick={endBrowseCatalog} oncancel={() => endBrowseCatalog()} />{/if}
		</div>
		<div class="buttons">
			<button class="button primary" disabled={saving}>{t.open}</button>
			{#if close}<button type="button" class="button" onclick={() => close(false)} disabled={saving}>{m.common.cancel}</button>{/if}
		</div>
		{#if catalogError}
			<p class="notice error" role="alert">{catalogError}</p>
		{/if}
	</form>
{/if}

<p class="instead"><button type="button" class="textbutton" onclick={switchMode}>{mode === 'list' ? t.orCatalog : t.orList}</button></p>

<style>
	form {
		display: grid;
		gap: var(--space-4);
		max-width: 44rem;
	}
	h1 {
		font-size: var(--step-3);
		font-weight: 600;
		line-height: 1;
	}
	p {
		max-width: 65ch;
	}
	/* File and folder names, in the File font rather than the browser's. */
	code {
		font-family: var(--font-mono);
		font-size: 0.9em;
	}
	.field {
		display: grid;
		gap: var(--space-1);
	}
	label {
		font-size: var(--step--1);
		color: var(--muted);
	}
	.entry {
		display: flex;
		gap: var(--space-3);
	}
	.entry input {
		flex: 1;
		min-width: 0;
	}
	.field select {
		justify-self: start;
		min-width: 16rem;
		max-width: 100%;
	}
	.buttons {
		display: flex;
		gap: var(--space-3);
	}
	.intro {
		display: grid;
		gap: var(--space-4);
		max-width: 44rem;
		margin-bottom: var(--space-4);
	}
	.intro p {
		margin: 0;
	}
	/* The other way in, a link below the one shown. */
	.instead {
		margin: var(--space-5) 0 0;
	}
	.small {
		font-size: var(--step--1);
	}
</style>
