<!-- The selected option in the details pane: its text, then what a
     recording needs to know about it, then where it's defined. -->
<script lang="ts">
	import type { Option } from '#lib/api/types/catalog.ts';
	import { m } from '#lib/copy/messages.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import type { Snippet } from 'svelte';
	import { allowed, changed, edgeSpace, hex, number, show } from './values.ts';

	// action, when given, sits under the heading: the option in the open
	// recording, while editing.
	let { option: o, page, menu, action }: { option: Option; page: string; menu?: { text: string; href: string }; action?: Snippet } = $props();

	const t = m.option;
	let typeLabel = $derived(o.control === 'stepper' ? t.types.stepper : t.types[o.type]);
</script>

<h2 class="caps">{o.text || m.values.noLabel}</h2>
{#if menu}<p class="muted"><a href={menu.href}>{menu.text}</a>{#if page}{' · '}{page}{/if}</p>{:else if page}<p class="muted">{page}</p>{/if}
{@render action?.()}

<dl>
	<dt class="caps">{t.selector}</dt>
	<dd>
		<span class="raw">{o.label}</span>
		{#if edgeSpace(o.label)}<span class="muted small">{m.values.edgeSpace}</span>{/if}
	</dd>
	<dt class="caps">{t.type}</dt>
	<dd>{typeLabel}</dd>
	{#if o.index > 0}
		<dt class="caps">{t.index}</dt>
		<dd>{t.indexOf({ n: o.index })}</dd>
	{/if}
	{#if o.type === 'text' && (o.value || o.valueText)}
		<dt class="caps">{t.value}</dt>
		<dd>
			{o.valueText || o.value}
			{#if o.valueCertainty !== 'exact'}<span class="caps word">{o.valueCertainty}</span>{/if}
			{#if o.value && o.value !== o.valueText}<span class="raw small">{o.value}</span>{/if}
		</dd>
	{/if}
	{#if o.slider}
		<dt class="caps">{t.range}</dt>
		<dd>{allowed(o)}{#if o.slider.format.replace(/\{\d\}/, '').trim()}<span class="muted small">{t.shownAs({ example: number(o.slider.max, o.slider.format) })}</span>{/if}</dd>
	{/if}
	{#if o.choices?.length}
		<dt class="caps">{t.choices}</dt>
		<dd>
			<ol class="choices">
				{#each o.choices as c, i (i)}
					<li>{c.text}{#if c.raw !== c.text}<span class="raw">{c.raw}</span>{/if}</li>
				{/each}
			</ol>
		</dd>
	{/if}
	{#if o.default}
		<dt class="caps">{t.default}</dt>
		<dd class="value">{#if o.type === 'color'}<span class="swatch" style:background={hex(o.default.number)}></span>{/if}{show(o, o.default)}</dd>
	{/if}
	{#if o.saved}
		<dt class="caps">{t.saved}</dt>
		<dd class="value" class:changed={changed(o)}>{#if o.type === 'color'}<span class="swatch" style:background={hex(o.saved.number)}></span>{/if}{show(o, o.saved)}</dd>
	{/if}
	{#if o.setting}
		<dt class="caps">{t.setting}</dt>
		<dd class="raw">{o.setting}</dd>
	{/if}
	<dt class="caps">{t.definedIn}</dt>
	<dd><span class="raw">{o.source.path}</span>{#if o.source.archive}<span class="muted small">{m.common.inArchive({ archive: o.source.archive })}</span>{/if}<span class="muted small">{o.source.mod}</span></dd>
</dl>

{#if o.certainty !== 'exact'}
	<p class="notice warn">{o.certainty === 'pattern' ? t.pattern : t.unknown}</p>
{/if}
{#if o.inert}
	<p class="notice">{t.inert}</p>
{:else if o.disabled}
	<p class="notice">{o.disabled === 'always' ? t.disabledAlways : t.disabledSometimes}</p>
{/if}
{#if o.type === 'text' && o.valueChanges}
	<p class="notice">{t.valueChanges}</p>
{/if}
{#each o.notes as note, i (i)}
	{#if note.kind === 'stepper'}<p class="notice">{t.stepper}</p>{:else if note.kind === 'repeats'}<p class="notice">{t.repeats}</p>{/if}
{/each}
{#if o.saved && changed(o)}
	<p class="notice small"><Fill text={t.savedNote.text}>{#snippet value()}<span class="changed">{show(o, o.saved!)}</span>{/snippet}</Fill></p>
{/if}
{#if o.helpText}
	<section class="help">
		<h3 class="caps">{t.help}</h3>
		<p>{o.helpText}</p>
	</section>
{/if}

<style>
	h2 {
		font-size: var(--step-2);
		font-weight: 600;
		margin-bottom: var(--space-1);
		overflow-wrap: anywhere;
	}
	h2 + p a {
		text-decoration-color: color-mix(in srgb, var(--accent) 75%, var(--ground));
	}
	dl {
		display: grid;
		grid-template-columns: 7.5rem 1fr;
		gap: var(--space-1) var(--space-3);
		margin: var(--space-4) 0;
	}
	dt {
		color: var(--muted);
		font-size: var(--step--1);
		padding-top: 0.15em;
	}
	dd {
		margin: 0;
		overflow-wrap: anywhere;
		min-width: 0;
	}
	/* Certainty, when it isn't exact: a word after the value, never a color. */
	.word {
		margin-left: var(--space-2);
		font-size: var(--step--1);
		color: var(--muted);
	}
	.raw {
		font-family: var(--font-mono);
		font-size: var(--step--1);
	}
	.choices .raw,
	dd .small {
		display: block;
		color: var(--muted);
	}
	.small {
		font-size: var(--step--1);
	}
	.choices {
		margin: 0;
		padding-left: 1.4em;
	}
	.choices li::marker {
		color: var(--muted);
		font-size: var(--step--1);
	}
	.value {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}
	.changed {
		color: var(--changed);
	}
	.swatch {
		width: 9px;
		height: 9px;
		flex: none;
		border: 1px solid var(--rule-strong);
	}
	.notice {
		margin-top: var(--space-3);
		max-width: 60ch;
	}
	.help {
		margin-top: var(--space-4);
		padding-top: var(--space-3);
		border-top: 1px solid var(--rule);
	}
	.help h3 {
		font-size: var(--step--1);
		color: var(--muted);
		margin-bottom: var(--space-2);
	}
	.help p {
		font-size: var(--step--1);
		line-height: 1.6;
		white-space: pre-wrap;
		max-width: 65ch;
	}
</style>
