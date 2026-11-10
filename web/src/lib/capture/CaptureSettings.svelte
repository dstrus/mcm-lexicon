<!-- The capture from the game, in Settings: where the plugin's captures are
     read from (or the file chosen instead), the one added and how much of
     it matched, CAPTURE IN GAME to ask for a new one at the next load of a
     save, and whether trackers are included and the capture's values are
     shown. While a request waits, it looks again every few seconds, and
     when a new capture lands the menus reload with its values. -->
<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { ApiError } from '#lib/api/client.ts';
	import type { Status } from '#lib/api/types/capture.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';
	import { count, realTime } from '#lib/format.ts';
	import PathPicker from '#lib/source/PathPicker.svelte';
	import Path from '#lib/ui/Path.svelte';
	import Switch from '#lib/ui/Switch.svelte';
	import { cancelRequest, getCapture, requestCapture, setCaptureSettings } from './api.ts';

	let {
		game,
		scanned
	}: {
		game: string; // the list's game as MO2 names it; "" when not known
		scanned: string; // when the list was last scanned: a capture is matched against each new catalog
	} = $props();

	const t = m.settings.capture;
	const releases = 'https://github.com/dstrus/mcm-lexicon/releases/latest';

	let st = $state<Status>();
	let working = $state(false);
	let error = $state('');
	let browsing = $state(false);

	// A capture that lands changes the catalog, so the menus load again.
	async function update(next: Status) {
		const landed = next.capture?.captured !== st?.capture?.captured || next.capture?.path !== st?.capture?.path;
		const first = !st;
		st = next;
		if (landed && !first) await invalidateAll();
	}
	async function act(fn: () => Promise<Status>) {
		working = true;
		error = '';
		try {
			await update(await fn());
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
		} finally {
			working = false;
		}
	}
	const settings = (change: Partial<{ use: boolean; includeTrackers: boolean; file: string }>) =>
		act(() => setCaptureSettings({ use: st!.use, includeTrackers: st!.includeTrackers, file: st!.file, ...change }));

	$effect(() => {
		void game; // another list may be another game's folder
		void scanned; // a scan that ends matches the capture against its catalog
		getCapture()
			.then(update)
			.catch((e) => (error = e instanceof ApiError ? e.message : String(e)));
	});
	// While a request waits for the game, or a capture for the scan, look
	// again every few seconds.
	$effect(() => {
		if (!st?.requested && !st?.pending) return;
		const timer = setInterval(() => getCapture().then(update).catch(() => {}), 3000);
		return () => clearInterval(timer);
	});

	function picked(path?: string) {
		browsing = false;
		if (path) settings({ file: path });
	}
	let names = $derived(st?.trackers.map((tr) => tr.name).join(', ') ?? '');
	let report = $derived(st?.capture?.report);
	let matched = $derived(report ? report.byPosition + report.byLabel + report.byPattern : 0);
</script>

<p class="lead">{t.lead}</p>
<p><a href={releases} target="_blank" rel="noreferrer">{t.download}</a></p>

{#if st}
	{#if st.capture && report}
		<dl class="stats">
			<div><dt class="caps">{t.captured}</dt><dd>{realTime(Date.parse(st.capture.captured))}</dd></div>
			{#if !st.pending}
				<div><dt class="caps">{t.menus}</dt><dd>{count(report.menus)}</dd></div>
				<div><dt class="caps">{t.matched}</dt><dd>{t.matchedOf({ matched: count(matched), options: count(report.options) })}</dd></div>
				<div><dt class="caps">{t.values}</dt><dd>{count(report.values)}</dd></div>
				{#if report.optionsAdded}<div><dt class="caps">{t.added}</dt><dd>{count(report.optionsAdded)}</dd></div>{/if}
			{/if}
		</dl>
		{#if st.pending}<p class="note">{t.pending}</p>{/if}
		{#if st.capture.unfinished}<p class="notice warn">{t.unfinished({ step: st.capture.unfinished })}</p>
		{:else if st.capture.stopped}<p class="notice warn">{t.stopped({ why: st.capture.stopped })}</p>{/if}
	{:else if !st.error}
		<p class="muted">{t.none}</p>
	{/if}
	{#if st.error}<p class="notice error">{st.error}</p>{/if}

	{#if st.requested}
		<p class="notice">{t.requested({ when: realTime(Date.parse(st.requested)) })}</p>
		<div><button type="button" class="button" disabled={working} onclick={() => act(cancelRequest)}>{t.cancel}</button></div>
	{:else}
		<div><button type="button" class="button primary" disabled={working || !st.folder} onclick={() => act(requestCapture)}>{t.request}</button></div>
		{#if !st.folder}<p class="note">{t.noFolder({ game: game || 'Skyrim' })}</p>{/if}
	{/if}

	<div class="field">
		<span class="caps label">{t.includeTrackers}</span>
		<span class="row">
			<Switch checked={st.includeTrackers} label={t.includeTrackers} disabled={working} onchange={(on) => settings({ includeTrackers: on })} />
			<span>{st.includeTrackers ? m.recordings.on : m.recordings.off}</span>
		</span>
		<p class="note">{t.trackersNote({ names })}</p>
	</div>
	<div class="field">
		<span class="caps label">{t.use}</span>
		<span class="row">
			<Switch checked={st.use} label={t.use} disabled={working} onchange={(on) => settings({ use: on })} />
			<span>{st.use ? m.recordings.on : m.recordings.off}</span>
		</span>
		<p class="note">{t.useNote}</p>
	</div>

	<p class="result">
		{#if st.file}
			<Fill text={t.file.text}>{#snippet path()}<span class="raw"><Path text={st!.file} /></span>{/snippet}</Fill>
		{:else if st.folder}
			<Fill text={t.folder.text}>{#snippet path()}<span class="raw"><Path text={st!.folder} /></span>{/snippet}</Fill>
		{/if}
	</p>
	<div class="buttons">
		<button type="button" class="button" disabled={working} aria-expanded={browsing} onclick={() => (browsing = true)}>{t.chooseFile}</button>
		{#if st.file}<button type="button" class="button" disabled={working} onclick={() => settings({ file: '' })}>{t.useFolder}</button>{/if}
	</div>
	{#if browsing}<PathPicker looking="capture" start={st.file ? st.file.replace(/[\\/][^\\/]*$/, '') : st.folder} onpick={picked} oncancel={() => picked()} />{/if}
{/if}
{#if error}<p class="notice error" role="alert">{error}</p>{/if}

<style>
	p {
		margin: 0;
	}
	.lead {
		max-width: 60ch;
	}
	.stats {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2) var(--space-5);
		margin: 0;
	}
	.stats div {
		display: grid;
		gap: var(--space-1);
	}
	.stats dd {
		margin: 0;
		font-variant-numeric: tabular-nums;
	}
	dt,
	.label {
		font-size: var(--step--1);
		color: var(--muted);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		justify-items: start;
	}
	.row,
	.buttons {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
	}
	.buttons {
		gap: var(--space-3);
	}
	.note {
		font-size: var(--step--1);
		color: var(--muted);
		max-width: 60ch;
	}
	.raw {
		font-family: var(--font-mono);
		font-size: var(--step--1);
	}
</style>
