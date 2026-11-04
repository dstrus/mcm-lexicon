<!-- A JSON file's text with its parts set apart: keys in the accent (the
     words MCM Recorder reads, which decide what a step does), values in
     ink (what it fills in), punctuation muted, anything else as it is.
     Put it inside a pre or a block that keeps white space. -->
<script lang="ts">
	import { tokens } from './json.ts';

	let { text }: { text: string } = $props();
	let parts = $derived(tokens(text));
</script>

{#each parts as t, i (i)}{#if t.kind === 'text'}{t.text}{:else}<span class={t.kind}>{t.text}</span>{/if}{/each}

<style>
	.key {
		color: var(--accent);
	}
	.string,
	.number,
	.literal {
		color: var(--ink);
	}
	.punct {
		color: var(--muted);
	}
</style>
