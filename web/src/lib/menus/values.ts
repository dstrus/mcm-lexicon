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
	return o.control === 'stepper' ? t.clickToCycle : t.click;
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
