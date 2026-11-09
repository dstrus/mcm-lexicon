// How an option's values read: in the option's own terms (On, 25 cm, a
// menu entry's text, a key's name), and what it allows.
import type { Option, Value } from '#lib/api/types/catalog.ts';
import { m } from '#lib/copy/messages.ts';
import { keyName } from './keys.ts';

const t = m.values;

// number shows n as a slider's format asks: "{1} cm" is one decimal place
// and "cm" after it. Without a format, trailing zeros go.
export function number(n: number, format = ''): string {
	const places = /\{(\d)\}/.exec(format);
	if (places) return format.replace(places[0], n.toFixed(Number(places[1])));
	return Number.isInteger(n) ? n.toLocaleString() : n.toLocaleString(undefined, { maximumFractionDigits: 4 });
}

// snapped is n on a slider's range: held to its ends, and moved to the
// nearest step from its start, with no more decimals than the step and the
// start have, so 0.1 steps don't drift to 0.30000000000000004.
export function snapped(n: number, r: { min: number; max: number; step: number }): number {
	const held = Math.min(r.max, Math.max(r.min, n));
	if (!(r.step > 0)) return held;
	const decimals = (x: number) => (String(x).split('.')[1] ?? '').length;
	const v = r.min + Math.round((held - r.min) / r.step) * r.step;
	return Number(Math.min(r.max, v).toFixed(Math.max(decimals(r.step), decimals(r.min))));
}

// plain is a slider's value as its number field shows it: with the
// decimal places the option's format gives ({1} is one), without the
// format's words, which the field isn't for.
export function plain(n: number, format = ''): string {
	const places = /\{(\d)\}/.exec(format);
	return places ? n.toFixed(Number(places[1])) : String(n);
}

// short is a stored value as a field first shows it: plain, but without a
// format no more than four decimals, as values read elsewhere, since a float
// a plugin wrote reads 6.8938999. A field left showing it saves the value
// it came from, not this.
export function short(n: number, format = ''): string {
	return /\{\d\}/.test(format) ? plain(n, format) : String(Number(n.toFixed(4)));
}

// hex is a color's value as the game writes it, #RRGGBB.
export function hex(n: number): string {
	return '#' + Math.max(0, Math.round(n)).toString(16).padStart(6, '0').slice(-6).toUpperCase();
}

// show is what a value reads as for an option.
export function show(o: Option, v: Value): string {
	if (v.kind === 'bool') return v.bool ? t.on : t.off;
	if (v.kind === 'string') return v.string === '' ? t.empty : `“${v.string}”`;
	const n = v.number;
	switch (o.type) {
		case 'keymap':
			return keyName(n);
		case 'color':
			return hex(n);
		case 'menu':
		case 'text':
			if (o.choices?.length) return o.choices[n]?.text ?? t.choiceNumber({ n });
			return number(n);
		case 'slider':
			return number(n, o.slider?.format);
	}
	return number(n);
}

// same reports whether two values are the same setting.
export function same(a: Value, b: Value): boolean {
	if (a.kind !== b.kind) return false;
	if (a.kind === 'bool') return a.bool === b.bool;
	if (a.kind === 'string') return a.string === b.string;
	return Math.abs(a.number - b.number) < 1e-6;
}

// startSays reports whether a new game's value says more than the
// default: there's no default, or it differs.
export function startSays(o: Option): boolean {
	return !!o.start && !(o.default && same(o.start, o.default));
}

// changed reports whether the player's saved value differs from the
// option's default: the Changed Rule's copper.
export function changed(o: Option): boolean {
	return !!o.saved && !!o.default && !same(o.saved, o.default);
}

// allowed is what an option accepts, in a few words.
export function allowed(o: Option): string {
	switch (o.type) {
		case 'toggle':
			return t.onOrOff;
		case 'slider':
			return o.slider ? t.range({ min: number(o.slider.min), max: number(o.slider.max), step: number(o.slider.step) }) : t.aNumber;
		case 'menu':
			return choicesInBrief(o);
		case 'keymap':
			return t.aKey;
		case 'color':
			return t.aColor;
		case 'input':
			return t.text;
	}
	if (o.control === 'stepper') return o.choices?.length ? t.clickCycles({ values: o.choices.map((c) => c.text).join(', ') }) : t.clickToCycle;
	// A text option showing a value: a right-side click finds it by that.
	if (o.value && o.valueCertainty === 'exact') return t.clickShowing({ value: o.valueText || o.value });
	return t.click;
}

function choicesInBrief(o: Option): string {
	const choices = o.choices ?? [];
	if (!choices.length) return t.aChoice;
	const shown = choices.slice(0, 2).map((c) => c.text).join(', ');
	return choices.length > 2 ? t.andMore({ shown, count: choices.length - 2 }) : shown;
}

// edgeSpace reports whether a selector starts or ends with a space, which
// no font shows and MCM Recorder matches exactly.
export function edgeSpace(label: string): boolean {
	return label !== label.trim();
}
