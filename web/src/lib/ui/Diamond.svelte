<!-- The game's selection marker. -->
<script lang="ts">
	let { size = 9 }: { size?: number } = $props();
</script>

<svg width={size} height={size} viewBox="0 0 10 10" aria-hidden="true"><path d="M5 0 10 5 5 10 0 5Z" fill="var(--accent)" /></svg>
