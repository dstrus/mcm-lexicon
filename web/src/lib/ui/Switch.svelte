<!-- On or off, as a sliding switch: a squared track (the system's one 3px
     radius, never a pill) with a square knob that slides right and fills
     the track with the accent when on. dim draws it at rest, for a value
     shown but not yet set (an option not recorded yet). -->
<script lang="ts">
	let {
		checked,
		label,
		disabled = false,
		dim = false,
		onchange
	}: {
		checked: boolean;
		label: string; // for assistive technology
		disabled?: boolean;
		dim?: boolean;
		onchange: (checked: boolean) => void;
	} = $props();
</script>

<button
	type="button"
	role="switch"
	class="switch"
	class:dim
	aria-checked={checked}
	aria-label={label}
	{disabled}
	onclick={(e) => {
		e.stopPropagation();
		onchange(!checked);
	}}
>
	<span class="knob" aria-hidden="true"></span>
</button>

<style>
	.switch {
		position: relative;
		flex: none;
		width: 1.75rem;
		height: 0.95rem;
		padding: 0;
		background: var(--glass-strong);
		border: 1px solid var(--rule-strong);
		border-radius: var(--radius);
		cursor: pointer;
		transition:
			background-color 140ms cubic-bezier(0.16, 1, 0.3, 1),
			border-color 140ms cubic-bezier(0.16, 1, 0.3, 1);
	}
	.knob {
		position: absolute;
		top: 50%;
		left: 2px;
		width: 0.6rem;
		height: 0.6rem;
		background: var(--muted);
		border-radius: 2px;
		transform: translateY(-50%);
		transition:
			transform 140ms cubic-bezier(0.16, 1, 0.3, 1),
			background-color 140ms cubic-bezier(0.16, 1, 0.3, 1);
	}
	.switch[aria-checked='true'] {
		background: var(--accent);
		border-color: var(--accent);
	}
	.switch[aria-checked='true'] .knob {
		background: var(--on-accent);
		transform: translate(calc(1.75rem - 0.6rem - 6px), -50%);
	}
	.switch:hover:not(:disabled) {
		border-color: var(--ink);
	}
	.switch[aria-checked='true']:hover:not(:disabled) {
		border-color: var(--accent);
	}
	.switch:focus-visible {
		outline: 2px solid var(--accent);
		outline-offset: 2px;
	}
	.switch:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
	/* Shown but not set: the track at rest, whatever it holds. */
	.switch.dim {
		opacity: 0.45;
	}
	.switch.dim:hover:not(:disabled) {
		opacity: 1;
	}
	@media (prefers-reduced-motion: reduce) {
		.switch,
		.knob {
			transition: none;
		}
	}
</style>
