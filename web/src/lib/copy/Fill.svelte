<!-- A message whose placeholders hold markup, not just text: each {name}
     in it is replaced by the snippet passed as name, and anything else by
     its value. For “Its mod, <span class="id">X</span>, is switched off”. -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let { text, ...parts }: { text: string } & Record<string, Snippet | string | number> = $props();

	let pieces = $derived(text.split(/(\{[A-Za-z][A-Za-z0-9]*\})/).filter(Boolean));
</script>

{#each pieces as piece, i (i)}{@const name = /^\{(.+)\}$/.exec(piece)?.[1]}{#if name !== undefined && typeof parts[name] === 'function'}{@render (parts[name] as Snippet)()}{:else if name !== undefined && parts[name] !== undefined}{parts[name]}{:else}{piece}{/if}{/each}
