<!-- MCM Lexicon's own settings: the mod list it reads, and the mod folder
     recordings are saved to in an MO2 instance. A save folder change is
     kept as it's made, editing or not, since it changes no recording;
     another list is chosen in its form, since it rescans and can discard
     the draft. Then the capture from the game, and copies of what was
     read, saved to files: the list's catalog, and for an MO2 instance a
     snapshot. -->
<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.ts';
	import CaptureSettings from '#lib/capture/CaptureSettings.svelte';
	import { exportCatalog, exportSnapshot, setSaveFolder } from '#lib/api/status.ts';
	import type { Status } from '#lib/api/types/api.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';
	import SaveCopy from '#lib/source/SaveCopy.svelte';
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

	// The names suggested for a copy: the list's folder, its profile, and
	// when, without the characters a file name can't hold.
	const safe = (s: string) => s.replace(/[\\/:*?"<>|]+/g, '-').replace(/\s+/g, ' ').trim();
	const two = (n: number) => String(n).padStart(2, '0');
	const now = new Date();
	const day = `${now.getFullYear()}-${two(now.getMonth() + 1)}-${two(now.getDate())}`;
	let listName = $derived(source?.path.split(/[\\/]/).filter(Boolean).pop() ?? '');
	let catalogName = $derived(safe([listName, source?.kind === 'mo2' ? source.profile : '', day].filter(Boolean).join(' ')));
	let snapshotName = $derived(safe(`snapshot-${listName}-${day.replaceAll('-', '')}-${two(now.getHours())}${two(now.getMinutes())}`));
	const megabytes = (bytes: number) => (bytes / (1024 * 1024)).toFixed(1);
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

{#if source || status.catalogFile}
	<section class="settings" id="capture" aria-labelledby="capture-title">
		<h2 id="capture-title" class="caps">{m.settings.capture.title}</h2>
		<CaptureSettings game={source?.game ?? ''} scanned={status.scan.scannedAt ?? ''} />
	</section>
{/if}
{#if source}
	<section class="settings" id="catalog-file" aria-labelledby="catalog-file-title">
		<h2 id="catalog-file-title" class="caps">{t.catalogTitle}</h2>
		<p class="lead">{t.catalogLead}</p>
		<SaveCopy name={catalogName} ext=".json" action={t.saveCatalog} working={t.savingCatalog} save={exportCatalog} done={(r) => t.catalogSaved({ count: r.menus })} />
	</section>
{/if}
{#if source?.kind === 'mo2'}
	<section class="settings" id="snapshot" aria-labelledby="snapshot-title">
		<h2 id="snapshot-title" class="caps">{t.snapshotTitle}</h2>
		<p class="lead">{t.snapshotLead}</p>
		<SaveCopy name={snapshotName} ext=".zip" action={t.makeSnapshot} working={t.makingSnapshot} save={exportSnapshot} done={(r) => t.snapshotSaved({ mods: r.mods, files: r.files, steps: r.steps, mb: megabytes(r.bytes) })} />
	</section>
{/if}

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
	.lead {
		max-width: 60ch;
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
