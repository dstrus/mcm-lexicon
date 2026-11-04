<!-- A text area for a JSON file, with its parts set apart as it's typed:
     the highlighted text sits beneath a text area whose own text is
     invisible, the two sharing one font, one wrap, and one scroll. The
     mirror's own color is the ink, so a word that isn't JSON (which the
     server will refuse) reads as plain text, not as a hole. -->
<script lang="ts">
	import Json from './Json.svelte';

	let {
		value = $bindable(),
		label,
		rows,
		box = $bindable(),
		disabled = false,
		onkeydown
	}: { value: string; label: string; rows?: number; box?: HTMLTextAreaElement; disabled?: boolean; onkeydown?: (e: KeyboardEvent) => void } = $props();

	let mirror = $state<HTMLElement>();
	function sync(e: Event) {
		if (!mirror) return;
		const t = e.currentTarget as HTMLTextAreaElement;
		mirror.scrollTop = t.scrollTop;
		mirror.scrollLeft = t.scrollLeft;
	}
</script>

<div class="code">
	<pre bind:this={mirror} aria-hidden="true"><Json text={value} />{'\n'}</pre>
	<textarea bind:this={box} bind:value spellcheck="false" autocapitalize="off" autocomplete="off" aria-label={label} {rows} {disabled} {onkeydown} onscroll={sync}></textarea>
</div>

<style>
	/* The field's material goes on the box, beneath the highlighted text;
	   the text area over it is clear, so it never dims the marks. */
	.code {
		position: relative;
		display: flex;
		flex-direction: column;
		min-height: 0;
		background: var(--glass-strong);
		border-radius: var(--radius);
	}
	/* The same box, font, and wrapping for both, so the letters line up. */
	pre,
	textarea {
		margin: 0;
		padding: var(--space-2) var(--space-3);
		font: 400 var(--step--1) / 1.55 var(--font-mono);
		white-space: pre-wrap;
		overflow-wrap: anywhere;
		tab-size: 4;
	}
	pre {
		position: absolute;
		inset: 0;
		overflow: hidden;
		border: 1px solid transparent;
		color: var(--ink);
		pointer-events: none;
	}
	textarea {
		position: relative;
		flex: 1;
		width: 100%;
		min-height: 8rem;
		resize: vertical;
		color: transparent;
		caret-color: var(--ink);
		background: transparent;
	}
	textarea::selection {
		color: transparent;
	}
</style>
