<!-- One recording in the editor: its name, where Save writes it, whether
     the draft differs from what's there (the Draft Rule), and its Steps and
     Files. Nothing is written until Save. -->
<script lang="ts">
	import { goto, invalidate } from '$app/navigation';
	import { ApiError } from '#lib/api/client.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';
	import { discardDraft, recordingHref, recordingsDependency, save } from '#lib/recordings/api.ts';
	import { change } from '#lib/recordings/draft.ts';
	import Tabs from '#lib/ui/Tabs.svelte';
	import Up from '#lib/ui/Up.svelte';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
	const t = m.recording;
	const d = m.draft;

	let ed = $derived(data.editor);
	let draft = $derived(ed.draft);
	let target = $derived(ed.target);
	let stepFiles = $derived(ed.files.filter((f) => f.change !== 'removed').length - 1); // without the info file
	let writes = $derived(ed.files.filter((f) => f.change !== 'same').length);
	let working = $state(false);
	let error = $state('');

	// Where Save writes, as a reader names it.
	let folder = $derived(target.kind === 'overwrite' ? d.overwrite : target.kind === 'data' ? d.data : target.name);
	const time = (iso: string) => new Date(iso).toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' });

	async function run(job: () => Promise<unknown>) {
		working = true;
		error = '';
		try {
			await job();
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
		} finally {
			working = false;
		}
	}
	const saveNow = () =>
		run(async () => {
			await save(draft.name);
			await invalidate(recordingsDependency);
		});
	const discard = () =>
		run(async () => {
			await discardDraft(draft.name);
			if (!ed.recording) return goto('/recordings', { invalidateAll: true });
			await invalidate(recordingsDependency);
		});

	// Choosing where Save writes: a mod folder from the list, or another by
	// name; with a catalog file open, a folder by its path.
	let other = $state(false);
	let otherName = $state('');
	const otherValue = '\u0000other';
	function chooseTarget(value: string) {
		if (value === otherValue) {
			other = true;
			otherName = '';
			return;
		}
		other = false;
		run(() => change(ed, (x) => (x.target = value)));
	}
	function useOther(e: SubmitEvent) {
		e.preventDefault();
		if (!otherName.trim()) return;
		run(async () => {
			await change(ed, (x) => (x.target = otherName.trim()));
			other = false;
		});
	}

	let canSave = $derived(!ed.unsaved && !working);
	useShortcuts(() => [{ key: 'Enter', label: m.common.save, command: true, disabled: !canSave, hidden: !ed.changed, run: saveNow }]);
</script>

<Up href="/recordings?item={encodeURIComponent(draft.name)}" label={t.up} />
<h1 class="caps">{draft.name}</h1>
<p class="muted sub">
	{#if draft.title && draft.title !== draft.name}{draft.title}{' · '}{/if}{#if target.kind !== 'none'}{folder}{' · '}{/if}{t.stepsInFiles({ steps: t.stepCount({ count: draft.steps.length }), files: t.fileCount({ count: stepFiles }) })}
</p>

<div class="draft">
	{#if ed.changed && target.kind !== 'none'}
		<p class="line">
			{ed.files.some((f) => f.old) ? d.notSavedBackup({ count: writes, folder }) : d.notSaved({ count: writes, folder })}
		</p>
	{:else if ed.saved}
		<p class="line">{ed.saved.backup ? d.savedBackup({ folder, time: time(ed.saved.at), backup: ed.saved.backup }) : d.saved({ folder, time: time(ed.saved.at) })}</p>
	{/if}
	{#if ed.changed}
		<div class="actions">
			<button type="button" class="button primary" disabled={!canSave} onclick={saveNow} title={ed.unsaved === 'running' ? m.common.gameRunning : undefined}>{m.common.save}</button>
			{#if ed.drafted}<button type="button" class="textbutton" disabled={working} onclick={discard}>{ed.recording ? d.discard : d.discardNew}</button>{/if}
			{#if ed.drafted}<span class="muted small">{d.keptInMemory}</span>{/if}
		</div>
	{/if}

	{#if target.kind === 'data'}
		<!-- A Data folder: there's nowhere else to write. -->
	{:else if target.kind === 'folder' || target.kind === 'none'}
		<form class="target" onsubmit={useOther}>
			<label>
				<span class="caps">{d.saveTo}</span>
				<input bind:value={otherName} placeholder={target.name || d.folderPath} autocomplete="off" spellcheck="false" />
			</label>
			<button class="button" disabled={working || !otherName.trim()}>{d.use}</button>
		</form>
	{:else}
		<div class="target">
			<label>
				<span class="caps">{d.saveTo}</span>
				<select value={other ? otherValue : target.name} disabled={working} onchange={(e) => chooseTarget(e.currentTarget.value)}>
					{#each ed.targets as name (name)}<option value={name}>{name}</option>{/each}
					<option value="overwrite">{d.overwrite}</option>
					<option value={otherValue}>{d.otherMod}</option>
				</select>
			</label>
			{#if other}
				<form class="other" onsubmit={useOther}>
					<input bind:value={otherName} placeholder={d.modName} aria-label={d.modName} autocomplete="off" spellcheck="false" />
					<button class="button" disabled={working || !otherName.trim()}>{d.use}</button>
				</form>
			{/if}
		</div>
	{/if}

	{#if ed.unsaved === 'running'}<p class="notice warn">{d.running}</p>{/if}
	{#if target.kind === 'none'}<p class="notice">{d.chooseFolder}</p>{/if}
	{#if target.kind === 'new-mod' && target.exists}<p class="notice"><Fill text={d.notListed.text}>{#snippet mod()}<span class="raw">{target.name}</span>{/snippet}</Fill></p>
	{:else if target.kind === 'new-mod'}<p class="notice"><Fill text={d.newMod.text}>{#snippet mod()}<span class="raw">{target.name}</span>{/snippet}</Fill></p>{/if}
	{#if target.kind === 'mod' && !target.enabled}<p class="notice warn">{d.unticked({ mod: target.name })}</p>{/if}
	{#if target.losesTo}<p class="notice warn">{d.losesTo({ mod: target.losesTo })}</p>{/if}
	{#each ed.problems as p, i (i)}<p class="notice error">{p}</p>{/each}
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}
</div>

<Tabs
	label={t.tabs}
	tabs={[
		{ href: recordingHref(draft.name), label: t.steps, count: draft.steps.length },
		{ href: recordingHref(draft.name, 'files'), label: t.files, count: ed.files.length }
	]}
/>
{@render children()}

<style>
	h1 {
		font-size: var(--step-3);
		font-weight: 600;
		line-height: 1;
		overflow-wrap: anywhere;
	}
	.sub {
		margin-top: var(--space-2);
	}
	.draft {
		display: grid;
		gap: var(--space-2);
		margin: var(--space-3) 0 var(--space-4);
		max-width: 70ch;
	}
	.line {
		margin: 0;
		color: var(--muted);
	}
	.actions,
	.target,
	.other {
		display: flex;
		align-items: end;
		flex-wrap: wrap;
		gap: var(--space-3);
	}
	.actions {
		align-items: baseline;
	}
	.target label {
		display: grid;
		gap: var(--space-1);
		min-width: 16rem;
	}
	.target label span {
		color: var(--muted);
		font-size: var(--step--1);
	}
	.small {
		font-size: var(--step--1);
	}
	.raw {
		font-family: var(--font-mono);
		font-size: var(--step--1);
	}
	.notice {
		margin: 0;
	}
</style>
