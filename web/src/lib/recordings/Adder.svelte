<!-- The selected option in the open recording, in the pane, while editing:
     its value there, edited in place. An option the recording has shows its
     step's value with Apply; one it hasn't shows the option's own
     value with Add to <recording>. With no recording chosen, it asks for
     one first. Read-only, it shows nothing. -->
<script lang="ts">
	import { invalidate } from '$app/navigation';
	import { ApiError } from '#lib/api/client.ts';
	import type { Option } from '#lib/api/types/catalog.ts';
	import { m } from '#lib/copy/messages.ts';
	import { addStep, recordingHref, recordingsDependency } from '#lib/recordings/api.ts';
	import type { OpenRecording } from '#lib/recordings/open.svelte.ts';
	import { recordable, stepFor } from '#lib/recordings/steps.ts';
	import ValueEditor from '#lib/recordings/ValueEditor.svelte';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';

	let {
		option,
		menu,
		pageName,
		open,
		choose,
		landing = false
	}: {
		option: Option;
		menu: string; // the menu's raw name, as a recording calls it
		pageName: string; // raw; "" for a menu's default page
		open: OpenRecording;
		choose: () => Promise<string | null>; // asks which recording, when none is chosen
		landing?: boolean; // the menu opens on a copy of its page, so a step on the default page is this option too
	} = $props();

	const t = m.adding;

	let named = $derived(stepFor(menu, pageName, option));
	let existing = $derived(open.find(named, landing));
	// A change to a recorded step stays on the page it's on.
	let base = $derived(existing ? { ...named, page: existing.step.page } : named);
	let can = $derived(recordable(option));

	// The value being edited: the step's when recorded, the option's own when
	// not. Started again when another option, or another value, arrives.
	let value = $state('');
	let side = $state('');
	let problem = $state('');
	let error = $state('');
	let done = $state('');
	let working = $state(false);
	let from = $derived(existing ? { value: existing.step.value, side: existing.step.side } : { value: base.value, side: base.side });
	$effect(() => {
		value = from.value;
		side = from.side;
		error = '';
	});
	$effect(() => {
		void option;
		done = '';
	});
	let edited = $derived(value !== from.value || side !== from.side);

	async function submit(e?: SubmitEvent) {
		e?.preventDefault();
		if (problem || working || !can || (existing && !edited)) return;
		working = true;
		error = '';
		try {
			const r = await addStep(open.name, { ...base, value, side });
			done = r.replaced ? t.changed({ recording: open.name, n: r.number }) : t.added({ recording: open.name, n: r.number });
			await invalidate(recordingsDependency);
		} catch (err) {
			error = err instanceof ApiError ? err.message : String(err);
		} finally {
			working = false;
		}
	}

	let label = $derived(existing ? t.apply : t.addTo({ recording: open.name }));
	useShortcuts(() =>
		!open.editing
			? []
			: !open.name
				? [{ key: 'a', shown: 'A', label: t.addToAny, run: choose }]
				: [{ key: 'a', shown: 'A', label, disabled: !can || !!problem || working || (!!existing && !edited), run: () => submit() }]
	);
</script>

{#if open.editing}
	<section class="recorded" aria-label={t.inRecording({ recording: open.name || '…' })}>
		{#if !open.name}
			<button type="button" class="button primary" onclick={choose}>{t.addToAny}</button>
		{:else}
			<h3 class="caps">{t.inRecording({ recording: open.name })}</h3>
			{#if !can}
				<p class="muted">{option.inert ? t.inert : option.disabled === 'always' ? t.readOnly : t.cantRecord}</p>
			{:else}
				<p class="muted">
					{#if existing}{t.stepN({ n: existing.n })} <a href={recordingHref(open.name) + `?item=${existing.n}`}>{t.openIt}</a>{:else}{t.notYet}{/if}
				</p>
				<form onsubmit={submit}>
					<ValueEditor kind={base.kind} {option} bind:value bind:side bind:problem />
					<button class="button primary" disabled={!!problem || working || (!!existing && !edited)}>{label}</button>
				</form>
			{/if}
			{#if done}<p class="notice" role="status">{done}</p>{/if}
			{#if error}<p class="notice error" role="alert">{error}</p>{/if}
		{/if}
	</section>
{/if}

<style>
	.recorded {
		display: grid;
		gap: var(--space-2);
		margin: var(--space-4) 0;
		padding: var(--space-3) 0;
		border-block: 1px solid var(--rule);
	}
	h3 {
		font-size: var(--step--1);
		font-weight: 500;
		color: var(--muted);
		overflow-wrap: anywhere;
	}
	p {
		margin: 0;
	}
	form {
		display: grid;
		gap: var(--space-3);
		justify-items: start;
	}
	.notice {
		max-width: 60ch;
	}
</style>
