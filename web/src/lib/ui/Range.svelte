<!-- A slider's value on its range: a hairline track with the diamond as
     its knob, beside the number field that takes an exact value. Dragging
     or the arrow keys move the value; oncommit fires when the pointer lets
     go, for an editor that applies on release. -->
<script lang="ts">
	let {
		value = $bindable(),
		min,
		max,
		step,
		label,
		oncommit,
		onkeydown
	}: {
		value: string; // the step's value as text
		min: number;
		max: number;
		step: number; // 0 for any
		label: string; // for assistive technology
		oncommit?: () => void;
		onkeydown?: (e: KeyboardEvent) => void;
	} = $props();

	// What the knob shows: the value on the range, or its start while the
	// field holds nothing a slider can take.
	let at = $derived.by(() => {
		const n = Number(value.trim());
		if (value.trim() === '' || Number.isNaN(n)) return min;
		return Math.min(max, Math.max(min, n));
	});
</script>

<input type="range" {min} {max} step={step || 'any'} value={at} aria-label={label} oninput={(e) => (value = e.currentTarget.value)} onpointerup={oncommit} {onkeydown} />

<style>
	input {
		flex: 1 1 6rem;
		min-width: 5rem;
	}
</style>
