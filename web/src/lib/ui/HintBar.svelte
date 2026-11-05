<!-- The game's button hints: each is a real shortcut, and clicking it does the same. -->
<script lang="ts">
	import { m } from '#lib/copy/messages.ts';
	import { active, onMac } from './shortcuts.svelte.ts';

	let hints = $derived(active().filter((s) => !s.hidden));
</script>

{#if hints.length}
	<nav class="hints" aria-label={m.ui.shortcuts}>
		{#each hints as h (`${h.command ? 'cmd+' : ''}${h.shift ? 'shift+' : ''}${h.key}`)}
			<button type="button" class="hint caps" class:writes={h.writes} onclick={h.run} disabled={h.disabled}>
				<kbd>{h.shift ? (onMac ? '⇧' : 'Shift ') : ''}{h.command ? (onMac ? '⌘ ' : 'Ctrl ') : ''}{h.shown ?? h.key}</kbd>{h.label}
			</button>
		{/each}
	</nav>
{/if}

<style>
	/* Along the bottom of the screen, like the game's own button hints. */
	.hints {
		position: fixed;
		inset: auto 0 0 0;
		display: flex;
		flex-wrap: wrap;
		justify-content: safe flex-end;
		gap: var(--space-2) var(--space-5);
		padding: var(--space-2) var(--space-6);
		background: var(--ground);
		border-top: 1px solid var(--rule);
		z-index: 1;
	}
	.hint {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		background: none;
		border: 0;
		padding: 0;
		color: var(--muted);
		font-size: var(--step-0);
		cursor: pointer;
	}
	.hint:hover:not(:disabled) {
		color: var(--ink);
	}
	.hint:disabled {
		opacity: 0.45;
		cursor: not-allowed;
	}
	kbd {
		font: inherit;
		color: var(--ink);
		border: 1px solid var(--rule-strong);
		border-radius: var(--radius);
		padding: 0 0.4em;
		min-width: 1.6em;
		text-align: center;
	}
	/* A hint that writes to disk looks like Save's button (theme.css,
	   .button.save): the key in amber with its glow, breathing, and the
	   label in amber, so no way to write ever looks like any other key. */
	.hint.writes {
		color: var(--warn);
	}
	.hint.writes kbd {
		background: var(--warn);
		border-color: var(--warn);
		color: var(--panel);
		font-weight: 600;
		box-shadow: 0 2px 14px -2px color-mix(in oklab, var(--warn) 55%, transparent);
		animation: save-breathe 2.8s ease-in-out infinite;
	}
	.hint.writes:hover:not(:disabled) kbd {
		border-color: var(--ink);
	}
	.hint.writes:disabled kbd {
		box-shadow: none;
		animation: none;
	}
	@media (prefers-reduced-motion: reduce) {
		.hint.writes kbd {
			animation: none;
		}
	}
</style>
