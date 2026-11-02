<!-- One recording: its name, and its Steps and Files. While editing, also
     where Save writes it and whether the draft differs from what's there
     (the Draft Rule), with the way to Changes, where Save is. Read-only, it
     shows what's on disk. -->
<script lang="ts">
	import { goto, invalidate } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.ts';
	import type { Status } from '#lib/api/types/api.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';
	import { discardDraft, recordingHref, recordingsDependency, restore } from '#lib/recordings/api.ts';
	import { change } from '#lib/recordings/draft.ts';
	import Tabs from '#lib/ui/Tabs.svelte';
	import Up from '#lib/ui/Up.svelte';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
	const t = m.recording;
	const d = m.draft;

	let ed = $derived(data.editor);
	let draft = $derived(ed.draft);
	let target = $derived(ed.target);
	let editing = $derived((page.data.status as Status).editing);
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
	const discard = () =>
		run(async () => {
			await discardDraft(draft.name);
			if (!ed.recording) return goto('/recordings', { invalidateAll: true });
			await invalidate(recordingsDependency);
		});
	const undelete = () =>
		run(async () => {
			await restore(draft.name);
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
</script>

<Up href="/recordings?item={encodeURIComponent(draft.name)}" label={t.up} />
<h1 class="caps">{draft.name}</h1>
<p class="muted sub">
	{#if draft.title && draft.title !== draft.name}{draft.title}{' · '}{/if}{#if target.kind !== 'none'}{folder}{' · '}{/if}{t.stepsInFiles({ steps: t.stepCount({ count: draft.steps.length }), files: t.fileCount({ count: Math.max(0, stepFiles) }) })}
</p>

{#if editing}
	<div class="draft">
		<!-- Two lines' room whatever it says, so the table below doesn't move
		     when the first change is made in it. -->
		<p class="line">
			{#if ed.deleted}
				{d.deletes({ count: writes, folder })}
				<button type="button" class="textbutton" disabled={working} onclick={undelete}>{d.restore}</button>
			{:else if ed.changed && target.kind !== 'none'}
				{ed.files.some((f) => f.old) ? d.notSavedBackup({ count: writes, folder }) : d.notSaved({ count: writes, folder })}
				<a class="textbutton" href="/changes">{d.review}</a>
				{#if ed.drafted}<button type="button" class="textbutton" disabled={working} onclick={discard}>{ed.recording ? d.discard : d.discardNew}</button>{/if}
			{:else if ed.saved}
				{ed.saved.backup ? d.savedBackup({ folder, time: time(ed.saved.at), backup: ed.saved.backup }) : d.saved({ folder, time: time(ed.saved.at) })}
			{:else if target.kind !== 'none'}
				{d.upToDate({ folder })}
			{/if}
		</p>

		{#if ed.deleted || target.kind === 'data'}
			<!-- Nothing to choose: a deletion removes the files where they are, and a Data folder is the only place. -->
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

		{#if !ed.deleted}
			{#if target.kind === 'none'}<p class="notice">{d.chooseFolder}</p>{/if}
			{#if target.kind === 'new-mod' && target.exists}<p class="notice"><Fill text={d.notListed.text}>{#snippet mod()}<span class="raw">{target.name}</span>{/snippet}</Fill></p>
			{:else if target.kind === 'new-mod'}<p class="notice"><Fill text={d.newMod.text}>{#snippet mod()}<span class="raw">{target.name}</span>{/snippet}</Fill></p>{/if}
			{#if target.kind === 'mod' && !target.enabled}<p class="notice warn">{d.disabled({ mod: target.name })}</p>{/if}
			{#if target.losesTo}<p class="notice warn">{d.losesTo({ mod: target.losesTo })}</p>{/if}
			{#each ed.problems as p, i (i)}<p class="notice error">{p}</p>{/each}
		{/if}
		{#if error}<p class="notice error" role="alert">{error}</p>{/if}
	</div>
{/if}

<Tabs
	label={t.tabs}
	tabs={[
		{ href: recordingHref(draft.name), label: t.steps, count: draft.steps.length },
		{ href: recordingHref(draft.name, 'files'), label: t.files, count: ed.files.length },
		{ href: recordingHref(draft.name, 'info'), label: t.infoTab }
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
	/* Two lines' room whatever it says, so the table doesn't move. */
	.line {
		margin: 0;
		min-height: 3.1em;
		color: var(--muted);
	}
	.line .textbutton {
		margin-left: var(--space-2);
	}
	.target,
	.other {
		display: flex;
		align-items: end;
		flex-wrap: wrap;
		gap: var(--space-3);
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
	.raw {
		font-family: var(--font-mono);
		font-size: var(--step--1);
	}
	.notice {
		margin: 0;
	}
</style>
