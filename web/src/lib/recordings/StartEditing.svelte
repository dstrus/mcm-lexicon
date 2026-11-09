<!-- Asked when a value is changed while MCM Lexicon is read-only, so the
     change needn't wait for the switch at the column's top: Start editing
     turns editing on, as that switch does, and the change goes ahead.
     ask() opens it and says whether editing is on now. -->
<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { ApiError } from '#lib/api/client.ts';
	import { setEditing } from '#lib/api/status.ts';
	import { m } from '#lib/copy/messages.ts';
	import { tick } from 'svelte';

	const t = m.shell;

	let dialog = $state<HTMLDialogElement>();
	let start = $state<HTMLButtonElement>();
	let working = $state(false);
	let error = $state('');
	let settle: ((on: boolean) => void) | null = null;

	export async function ask(): Promise<boolean> {
		error = '';
		dialog?.showModal();
		const answer = new Promise<boolean>((resolve) => (settle = resolve));
		await tick();
		start?.focus();
		return answer;
	}

	async function begin() {
		working = true;
		error = '';
		try {
			await setEditing(true);
			await invalidateAll();
			settle?.(true);
			settle = null;
			dialog?.close();
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
		} finally {
			working = false;
		}
	}
	// Closed any other way (Cancel, Esc, a click outside): still read-only.
	function closed() {
		settle?.(false);
		settle = null;
	}
</script>

<dialog bind:this={dialog} aria-labelledby="start-editing-title" onclick={(e) => e.target === dialog && dialog?.close()} onclose={closed}>
	<h2 id="start-editing-title" class="caps">{t.startTitle}</h2>
	<p>{t.startSays}</p>
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}
	<div class="buttons">
		<button type="button" class="button primary" bind:this={start} disabled={working} onclick={begin}>{t.startEditing}</button>
		<button type="button" class="textbutton" disabled={working} onclick={() => dialog?.close()}>{m.common.cancel}</button>
	</div>
</dialog>

<style>
	/* As the layout's dialog for leaving editing. */
	dialog {
		width: min(30rem, 90vw);
		margin: 18vh auto auto;
		padding: var(--space-4);
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
	p {
		margin: var(--space-2) 0 var(--space-4);
	}
	.buttons {
		display: flex;
		align-items: baseline;
		flex-wrap: wrap;
		gap: var(--space-2);
	}
</style>
