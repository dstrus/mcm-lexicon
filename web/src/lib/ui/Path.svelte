<!-- A file path, breakable only after its separators, so a long one wraps
     between folders rather than inside a name. -->
<script lang="ts">
	let { text }: { text: string } = $props();
	let parts = $derived(text.split(/(?<=[/\\])/));
</script>

{#each parts as p, i (i)}{p}<wbr />{/each}
