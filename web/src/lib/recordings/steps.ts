// How a recording's steps read: their values in their option's terms, and
// their checks as words.
import type { Option } from '#lib/api/types/catalog.ts';
import type { Check, Counts, Kind, Step } from '#lib/api/types/recordings.ts';
import { m } from '#lib/copy/messages.ts';
import { keyName } from '#lib/menus/keys.ts';
import { hex, number, same } from '#lib/menus/values.ts';

const t = m.steps;

// The option type each kind of step plays, for the Type column.
export const kindType: Record<Kind, keyof typeof m.option.types> = {
	toggle: 'toggle',
	slider: 'slider',
	choose: 'menu',
	shortcut: 'keymap',
	color: 'color',
	text: 'input',
	click: 'text'
};

// value is what a step sets, in its option's terms where the catalog knows
// the option.
export function value(s: Step, o: Option | null | undefined): string {
	switch (s.kind) {
		case 'toggle':
			return s.value.toLowerCase() === 'on' ? m.values.on : s.value.toLowerCase() === 'off' ? m.values.off : s.value;
		case 'slider': {
			const n = Number(s.value);
			return Number.isNaN(n) ? s.value : number(n, o?.slider?.format);
		}
		case 'choose':
			return o?.choices?.find((c) => c.raw.toLowerCase() === s.value.toLowerCase())?.text ?? s.value;
		case 'shortcut':
			return keyName(Number(s.value));
		case 'color':
			return hex(Number(s.value));
		case 'text':
			return `“${s.value}”`;
		case 'click':
			return s.side.toLowerCase() === 'right' ? t.clickRight : t.click;
	}
	return s.value;
}

// changed reports whether a step sets its option to something other than
// the option's known default: the Changed Rule's copper.
export function changed(s: Step, o: Option | null | undefined): boolean {
	const d = o?.default;
	if (!d) return false;
	switch (s.kind) {
		case 'toggle':
			return d.kind === 'bool' && d.bool !== (s.value.toLowerCase() === 'on');
		case 'slider':
		case 'shortcut':
		case 'color':
			return !same(d, { kind: 'number', number: Number(s.value), bool: false, string: '' });
		case 'choose': {
			const i = o?.choices?.findIndex((c) => c.raw.toLowerCase() === s.value.toLowerCase()) ?? -1;
			return i >= 0 && d.kind === 'number' && d.number !== i;
		}
	}
	return false;
}

// word is a check's status as the step table shows it, or "" for a match.
export function word(c: Check): string {
	switch (c.status) {
		case 'not-found':
			return t.words.notFound;
		case 'not-allowed':
			return t.words.notAllowed;
		case 'mod-missing':
			return t.words.modMissing;
		case 'unchecked':
			return t.words.unchecked;
		case 'matched':
			return c.why === 'disabled' ? t.words.grayedOut : c.why === 'inert' ? t.words.inert : '';
	}
	return '';
}

// failing reports whether a check means the step won't replay.
export function failing(c: Check): boolean {
	return c.status === 'not-found' || c.status === 'not-allowed' || c.status === 'mod-missing';
}

// sentence is a check said in full, for the pane.
export function sentence(c: Check, s: Step): string {
	switch (c.status) {
		case 'matched':
			return c.why === 'disabled' ? t.says.disabled : c.why === 'inert' ? t.says.inert : t.says.matched;
		case 'not-allowed':
			return c.why === 'range' ? t.says.range({ value: s.value }) : t.says.choice({ value: s.value });
		case 'mod-missing':
			return c.why === 'inactive' ? t.says.inactive({ mod: s.mod }) : t.says.modMissing({ mod: s.mod });
		case 'unchecked':
			return s.kind === 'click' && s.side.toLowerCase() === 'right' ? t.says.right : t.says.unchecked;
	}
	switch (c.why) {
		case 'page':
			return t.says.page({ page: s.page });
		case 'index':
			return t.says.index({ index: s.index });
		case 'value':
			return t.says.value({ value: s.option });
	}
	return t.says.option;
}

// counts is a recording's checks in a few words, or "" when every step
// matched.
export function countsText(c: Counts): string {
	const failing = c.notFound + c.notAllowed + c.modMissing;
	const parts: string[] = [];
	if (failing) parts.push(t.wontReplay({ count: failing }));
	if (c.unchecked) parts.push(t.uncheckedCount({ count: c.unchecked }));
	return parts.join(' · ');
}

// fileHeading is a step file's name as its group heading: 0001 · Dirt & Blood.
export function fileHeading(name: string): string {
	const base = name.replace(/\.json$/i, '');
	const at = base.indexOf('_');
	return at > 0 ? `${base.slice(0, at)} · ${base.slice(at + 1)}` : base;
}

// The kind of step that plays each option type.
export const typeKind: Record<Option['type'], Kind> = {
	toggle: 'toggle',
	slider: 'slider',
	menu: 'choose',
	keymap: 'shortcut',
	color: 'color',
	input: 'text',
	text: 'click'
};

// known is the value an option has as far as the catalog knows: the
// player's saved one (MCM Helper), else what a new game shows (from the
// script and plugin), else the mod's default.
export function known(o: Option): Option['default'] {
	return o.saved ?? o.start ?? o.default;
}

// recordable reports whether a recording can name the option exactly: a
// label built at runtime can't be matched without a wildcard. A text option
// without a label is clicked on its right side, by its value, so that has
// to be known instead.
export function recordable(o: Option): boolean {
	if (o.disabled === 'always') return false; // grayed out: the mod means it not to be set
	if (o.inert) return false; // a click on it does nothing
	if (o.type === 'text' && !o.label) return !!o.value && o.valueCertainty === 'exact';
	return o.certainty === 'exact';
}

// stepFor is a new step for an option, its value the one the option has
// (known), where the catalog knows it: what recording it starts with.
export function stepFor(menu: string, page: string, o: Option): Step {
	const kind = typeKind[o.type];
	const d = known(o);
	let value = '';
	switch (kind) {
		case 'toggle':
			value = d?.kind === 'bool' && !d.bool ? 'off' : 'on';
			break;
		case 'slider':
			value = String(d?.kind === 'number' ? d.number : (o.slider?.min ?? 0));
			break;
		case 'choose':
			value = (d?.kind === 'number' ? o.choices?.[d.number]?.raw : undefined) ?? o.choices?.[0]?.raw ?? '';
			break;
		case 'shortcut':
		case 'color':
			value = d?.kind === 'number' && d.number >= 0 ? String(d.number) : '';
			break;
		case 'text':
			value = d?.kind === 'string' ? d.string : '';
			break;
	}
	// A text option without a label is clicked on its right side, which
	// MCM Recorder finds by the value shown there.
	const right = kind === 'click' && !o.label;
	return {
		file: 0,
		mod: menu,
		page,
		option: right ? o.value : o.label,
		index: right ? o.valueIndex : o.index,
		kind,
		value,
		side: right ? 'right' : '',
		state: '',
		wildcard: false
	};
}

// sameOption reports whether two steps set the same option, compared as MCM
// Recorder compares them.
export function sameOption(a: Step, b: Step): boolean {
	return a.mod.toLowerCase() === b.mod.toLowerCase() && a.page.toLowerCase() === b.page.toLowerCase() && a.option.toLowerCase() === b.option.toLowerCase() && a.index === b.index && a.kind === b.kind;
}
