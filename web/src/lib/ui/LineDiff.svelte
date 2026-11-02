<!-- One file against the one it replaces, line by line, as Hermaeus shows
     an override against its base: unchanged runs folded, added lines in
     the accent's wash, removed ones in danger's, and within a paired line
     the words that differ marked. -->
<script lang="ts">
	import { m } from '#lib/copy/messages.ts';
	import { diffLines, hunks, markWords } from './linediff.ts';

	let { before, after, label }: { before: string; after: string; label: string } = $props();

	const t = m.ui;
	let lines = $derived(diffLines(before.replaceAll('\r\n', '\n'), after.replaceAll('\r\n', '\n')));
	let parts = $derived(hunks(lines));
	let words = $derived(markWords(lines));
</script>

<div class="diff" role="table" aria-label={label}>
	{#each parts as h, i (i)}
		{#if 'skipped' in h}
			<div class="skipped muted">{t.linesUnchanged({ count: h.skipped })}</div>
		{:else}
			{#each h.lines as l, j (j)}
				<div class="line {l.kind}" role="row">
					<span class="sign" role="cell" aria-label={l.kind === 'same' ? t.lineSame : l.kind === 'added' ? t.lineAdded : t.lineRemoved}>{l.kind === 'added' ? '+' : l.kind === 'removed' ? '−' : ''}</span>
					<span role="cell">{#if words.get(l)}{#each words.get(l) ?? [] as p, k (k)}{#if p.changed}<mark>{p.text}</mark>{:else}{p.text}{/if}{/each}{:else}{l.text || ' '}{/if}</span>
				</div>
			{/each}
		{/if}
	{/each}
</div>

<style>
	.diff {
		margin: 0;
		font: 400 var(--step--1) / 1.55 var(--font-mono);
	}
	.line {
		display: grid;
		grid-template-columns: 1.25rem minmax(0, 1fr);
		white-space: pre-wrap;
		overflow-wrap: anywhere;
		tab-size: 4;
	}
	.line.same {
		color: var(--muted);
	}
	.line.added {
		background: color-mix(in srgb, var(--accent) 16%, transparent);
	}
	.line.removed {
		background: color-mix(in srgb, var(--danger) 10%, transparent);
		color: var(--muted);
	}
	/* Within a paired line, the words that differ carry a stronger wash. */
	.line mark {
		color: inherit;
		border-radius: 1px;
	}
	.line.added mark {
		background: color-mix(in srgb, var(--accent) 34%, transparent);
	}
	.line.removed mark {
		background: color-mix(in srgb, var(--danger) 24%, transparent);
		color: var(--ink);
	}
	.sign {
		text-align: center;
		user-select: none;
	}
	.line.added .sign {
		color: var(--accent);
	}
	.line.removed .sign {
		color: var(--danger);
	}
	.skipped {
		padding: var(--space-1) 0 var(--space-1) 1.25rem;
		font-family: var(--font-text);
		font-style: italic;
		border-block: 1px dashed var(--rule);
		margin: var(--space-1) 0;
	}
</style>
