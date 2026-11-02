<!-- A sortable column head. Clicking sorts by it, and again reverses. -->
<script lang="ts">
	import type { Sort } from './sort.ts';

	let {
		label,
		key,
		sort = $bindable(),
		descendingFirst = false,
		align = 'start',
		class: className = ''
	}: {
		label: string;
		key: string;
		sort: Sort;
		descendingFirst?: boolean; // counts and dates read best largest or newest first
		align?: 'start' | 'end';
		class?: string; // for the table to hide the column at some widths
	} = $props();

	let on = $derived(sort.key === key);
</script>

<th aria-sort={on ? (sort.descending ? 'descending' : 'ascending') : 'none'} class={[className, { end: align === 'end' }]}>
	<button
		type="button"
		class="caps"
		class:on
		onclick={() => (sort = { key, descending: on ? !sort.descending : descendingFirst })}
	>
		{label}
		<svg class:flip={on && !sort.descending} class:hide={!on} width="8" height="8" viewBox="0 0 8 8" aria-hidden="true">
			<path d="M0 2h8L4 7z" fill="currentColor" />
		</svg>
	</button>
</th>

<style>
	th {
		text-align: left;
		padding: 0;
		border-bottom: 1px solid var(--rule-strong);
		font-weight: inherit;
		white-space: nowrap;
	}
	th.end {
		text-align: right;
	}
	button {
		background: none;
		border: 0;
		padding: var(--space-2) var(--space-3);
		font-size: var(--step--1);
		color: var(--muted);
		cursor: pointer;
		width: 100%;
		text-align: inherit;
	}
	button:hover,
	button.on {
		color: var(--accent);
	}
	svg {
		margin-left: var(--space-1);
		vertical-align: 1px;
	}
	.flip {
		transform: rotate(180deg);
	}
	.hide {
		visibility: hidden;
	}
</style>
