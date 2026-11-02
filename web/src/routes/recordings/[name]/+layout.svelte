<!-- One recording: its name, where it lives, and its Steps and Files. -->
<script lang="ts">
	import { m } from '#lib/copy/messages.ts';
	import { recordingHref } from '#lib/recordings/api.ts';
	import Tabs from '#lib/ui/Tabs.svelte';
	import Up from '#lib/ui/Up.svelte';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
	const t = m.recording;
	let rec = $derived(data.detail.recording!);
</script>

<Up href="/recordings?item={encodeURIComponent(rec.name)}" label={t.up} />
<h1 class="caps">{rec.name}</h1>
<p class="muted sub">
	{#if rec.title && rec.title !== rec.name}{rec.title}{' · '}{/if}{rec.mod}{' · '}{t.stepsInFiles({ steps: t.stepCount({ count: rec.steps.length }), files: t.fileCount({ count: rec.files.length }) })}
</p>
<Tabs
	label={t.tabs}
	tabs={[
		{ href: recordingHref(rec.name), label: t.steps, count: rec.steps.length },
		{ href: recordingHref(rec.name, 'files'), label: t.files, count: rec.files.length }
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
</style>
