<!-- The first thing shown without a mod list: where it is, or, for a
     machine without the list, a catalog file to open instead. -->
<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { ApiError } from '#lib/api/client.ts';
	import { setSource } from '#lib/api/status.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';

	const t = m.source;

	let path = $state('');
	let profile = $state('');
	let catalog = $state('');
	let listError = $state('');
	let catalogError = $state('');
	let saving = $state(false);

	async function choose(req: Parameters<typeof setSource>[0], report: (message: string) => void) {
		saving = true;
		listError = '';
		catalogError = '';
		try {
			await setSource(req);
			await invalidateAll(); // reload the status, which now has the list
		} catch (e) {
			report(e instanceof ApiError ? e.message : String(e));
		} finally {
			saving = false;
		}
	}

	function submitList(event: SubmitEvent) {
		event.preventDefault();
		choose({ path, profile: profile.trim() || undefined }, (message) => (listError = message));
	}

	function submitCatalog(event: SubmitEvent) {
		event.preventDefault();
		choose({ catalog }, (message) => (catalogError = message));
	}
</script>

<form onsubmit={submitList}>
	<h1 class="caps">{t.title}</h1>
	<p>
		<Fill text={t.notFound.text}>{#snippet ini()}<code>ModOrganizer.ini</code>{/snippet}{#snippet mods()}<code>mods</code>{/snippet}{#snippet profiles()}<code>profiles</code>{/snippet}{#snippet data()}<code>Data</code>{/snippet}</Fill>
	</p>
	<label>
		<span class="caps">{t.folder}</span>
		<input bind:value={path} required autocomplete="off" spellcheck="false" />
	</label>
	<label>
		<span class="caps">{t.profile}</span>
		<input bind:value={profile} autocomplete="off" spellcheck="false" />
		<span class="muted small">{t.profileHint}</span>
	</label>
	<div><button class="button primary" disabled={saving}>{t.use}</button></div>
	{#if listError}
		<p class="notice error" role="alert">{listError}</p>
	{/if}
</form>

<form class="catalog" onsubmit={submitCatalog}>
	<h2 class="caps">{t.orCatalog}</h2>
	<p>{t.catalogLead}</p>
	<label>
		<span class="caps">{t.catalogFile}</span>
		<input bind:value={catalog} required autocomplete="off" spellcheck="false" />
	</label>
	<div><button class="button" disabled={saving}>{t.open}</button></div>
	{#if catalogError}
		<p class="notice error" role="alert">{catalogError}</p>
	{/if}
</form>

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
	h2 {
		font-size: var(--step-2);
		font-weight: 600;
	}
	p {
		max-width: 65ch;
	}
	/* File and folder names, in the File font rather than the browser's. */
	code {
		font-family: var(--font-mono);
		font-size: 0.9em;
	}
	label {
		display: grid;
		gap: var(--space-1);
	}
	label span {
		font-size: var(--step--1);
		color: var(--muted);
	}
	/* The second way in, under a hairline. */
	.catalog {
		margin-top: var(--space-7);
		padding-top: var(--space-5);
		border-top: 1px solid var(--rule);
	}
	.small {
		font-size: var(--step--1);
	}
</style>
