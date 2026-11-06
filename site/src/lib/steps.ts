// A recording's step file, written the way MCM Lexicon writes it
// (internal/recordings/write.go): lowercase keys in the order JContainers
// sorts them, two-space indents, and a slider's value always with a
// decimal point. The hero's menu builds its file with this, and the page
// renders the first one with it too.

export type Kind = 'toggle' | 'slider' | 'choose' | 'click';

export interface Step {
	page: string;
	option: string;
	kind: Kind;
	value: string | number;
}

// sliderNumber writes a slider's value as MCM Recorder does: 25.0, not 25.
export function sliderNumber(n: number): string {
	const s = String(n);
	return /[.eE]/.test(s) ? s : `${s}.0`;
}

function pairs(mod: string, s: Step): [string, string, 'string' | 'number'][] {
	const out: [string, string, 'string' | 'number'][] = [['mod', mod, 'string']];
	if (s.kind === 'click') {
		out.push(['click', s.option, 'string']); // a click's selector is what it clicks
	} else {
		out.push(['option', s.option, 'string']);
		if (s.kind === 'slider') out.push(['slider', sliderNumber(Number(s.value)), 'number']);
		else out.push([s.kind, String(s.value), 'string']);
	}
	out.push(['page', s.page, 'string']);
	return out.sort((a, b) => (a[0] < b[0] ? -1 : a[0] > b[0] ? 1 : 0));
}

const esc = (t: string) => t.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
const span = (cls: string, t: string) => `<span class="${cls}">${esc(t)}</span>`;

// stepFile is the file as HTML: keys, values, and punctuation set apart as
// the app sets a file apart while it's typed, and each step in a span of
// its own, so the one just changed can be found.
export function stepFile(mod: string, steps: Step[]): string {
	if (steps.length === 0) return span('p', '[]');
	const objects = steps.map((s, i) => {
		const lines = pairs(mod, s).map(([k, v, type], j, all) => {
			const value = type === 'number' ? span('v', v) : span('v', JSON.stringify(v));
			return `    ${span('k', JSON.stringify(k))}${span('p', ':')} ${value}${j < all.length - 1 ? span('p', ',') : ''}`;
		});
		const close = i < steps.length - 1 ? span('p', '},') : span('p', '}');
		return `<span class="step" data-step="${i}">  ${span('p', '{')}\n${lines.join('\n')}\n  ${close}</span>`;
	});
	return `${span('p', '[')}\n${objects.join('\n')}\n${span('p', ']')}`;
}
