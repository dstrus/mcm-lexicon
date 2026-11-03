<!-- The game's selection marker. Hollow, in ink, it marks something chosen
     for a job rather than selected: the recording A adds steps to, and an
     option recorded in it. Filled in amber, it marks a step not saved yet. -->
<script lang="ts">
	let { size = 9, hollow = false, fill = 'var(--accent)' }: { size?: number; hollow?: boolean; fill?: string } = $props();
</script>

<svg width={size} height={size} viewBox="0 0 10 10" aria-hidden="true">
	{#if hollow}
		<path d="M5 0.9 9.1 5 5 9.1 0.9 5Z" fill="none" stroke="currentColor" stroke-width="1.3" />
	{:else}
		<path d="M5 0 10 5 5 10 0 5Z" {fill} />
	{/if}
</svg>
