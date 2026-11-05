<!-- MCM Lexicon's own settings: the mod list it reads, and the mod folder
     recordings are saved to in an MO2 instance. A save folder change is
     kept as it's made, editing or not, since it changes no recording;
     another list is chosen in its form, since it rescans and can discard
     the draft. -->
<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.ts';
	import { setSaveFolder } from '#lib/api/status.ts';
	import type { Status } from '#lib/api/types/api.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';
	import SourceForm from '#lib/source/SourceForm.svelte';
	import Path from '#lib/ui/Path.svelte';
	import { leave, param, setParams } from '#lib/ui/address.svelte.ts';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import Switch from '#lib/ui/Switch.svelte';

	const t = m.settings;

	let status = $derived(page.data.status as Status);
	let source = $derived(status.source);

	// Choosing another list has its own address, so Back ends it.
	let changing = $derived(param('mode') === 'source');
	const closeSource = () => leave('mode'); // chosen or not
	useShortcuts(() => (changing ? [{ key: 'Escape', shown: 'Esc', label: m.common.cancel, run: closeSource }] : []));
	let folder = $derived(status.saveFolder);
	let working = $state(false);
	let error = $state('');

	// The name as typed, started again from the settings whenever they change.
	let name = $state('');
	$effect(() => {
		name = folder.name;
	});

	async function save(nextName: string, noDelete: boolean) {
		working = true;
		error = '';
		try {
			await setSaveFolder(nextName, noDelete);
			await invalidateAll();
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
			name = folder.name;
		} finally {
			working = false;
		}
	}
	// The name is kept when the field is left or Enter is pressed; Esc puts
	// it back.
	function commit() {
		const v = name.trim();
		if (v === folder.name) return;
		save(v, folder.noDelete);
	}
	function keys(e: KeyboardEvent) {
		if (e.key === 'Escape') name = folder.name;
	}
</script>

<h1 class="caps">{t.title}</h1>

<section class="settings" id="mod-list" aria-labelledby="mod-list-title">
	<h2 id="mod-list-title" class="caps">{t.modList}</h2>
	{#if changing}
		<SourceForm current={{ path: source?.path ?? '', profile: source?.profile ?? '', catalog: status.catalogFile }} changes={status.changes} close={closeSource} />
	{:else}
		<p class="result">
			{#if source?.kind === 'mo2'}
				<Fill text={m.menus.inInstance.text}>{#snippet path()}<span class="raw"><Path text={source.path} /></span>{/snippet}</Fill>
				<span class="line"><Fill text={m.menus.inProfile.text}>{#snippet profile()}<span class="raw">{source.profile}</span>{/snippet}</Fill></span>
			{:else if source}
				<Fill text={m.menus.inData.text}>{#snippet path()}<span class="raw"><Path text={source.path} /></span>{/snippet}</Fill>
			{:else if status.catalogFile}
				<Fill text={t.inCatalog.text}>{#snippet path()}<span class="raw"><Path text={status.catalogFile} /></span>{/snippet}</Fill>
			{/if}
		</p>
		<div><button type="button" class="button" onclick={() => setParams({ mode: 'source' })}>{t.changeList}</button></div>
		<p class="note">{t.modListNote}</p>
	{/if}
</section>

<section class="settings" aria-labelledby="save-folder">
	<h2 id="save-folder" class="caps">{t.saveFolder}</h2>
	<div class="field">
		<label for="name" class="caps">{t.name}</label>
		<input id="name" bind:value={name} disabled={working} onchange={commit} onkeydown={keys} autocomplete="off" spellcheck="false" />
	</div>
	<div class="field">
		<span class="caps label">{t.noDelete}</span>
		<span class="row">
			<Switch checked={folder.noDelete} label={t.noDelete} disabled={working} onchange={(on) => save(name.trim(), on)} />
			<span>{folder.noDelete ? m.recordings.on : m.recordings.off}</span>
		</span>
		<p class="note">{t.noDeleteNote}</p>
	</div>
	<p class="result"><Fill text={t.folder.text}>{#snippet folder()}<span class="raw">{status.saveFolder.folder}</span>{/snippet}</Fill></p>
	<p class="note">{t.otherMods}</p>
	<p class="note">{t.moved}</p>
	{#if status.source?.kind === 'data'}<p class="notice">{t.dataFolder}</p>
	{:else if status.catalogFile}<p class="notice">{t.catalogFile}</p>{/if}
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}
</section>

<style>
	h1 {
		font-size: var(--step-3);
		font-weight: 600;
		line-height: 1;
		position: relative;
		top: 0.21em;
	}
	h2 {
		font-size: var(--step-1);
		font-weight: 600;
		color: var(--accent);
	}
	.settings {
		display: grid;
		gap: var(--space-4);
		max-width: 36rem;
		margin-top: var(--space-6);
	}
	.settings + .settings {
		margin-top: var(--space-7);
		padding-top: var(--space-5);
		border-top: 1px solid var(--rule);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		justify-items: start;
	}
	label,
	.label {
		font-size: var(--step--1);
		color: var(--muted);
	}
	input {
		width: 100%;
	}
	p {
		margin: 0;
	}
	.row {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
	}
	.note {
		font-size: var(--step--1);
		color: var(--muted);
		max-width: 60ch;
	}
	/* The profile, on a line of its own under a path that may be long. */
	.line {
		display: block;
	}
	.raw {
		font-family: var(--font-mono);
		font-size: var(--step--1);
	}
</style>
