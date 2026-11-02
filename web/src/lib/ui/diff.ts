// What changed between two versions of a text, word by word, for reading
// a file that will be written against the one it replaces.

export interface Op {
	kind: 'same' | 'added' | 'removed';
	text: string;
}

// Words and the whitespace between them, each its own token, so a diff
// keeps the original spacing.
function tokens(s: string): string[] {
	return s.split(/(\s+)/).filter((t) => t !== '');
}

// Past this many tokens a side, the quadratic comparison would stall, and
// such a text is shown as wholly replaced.
const most = 2500;

// diffWords is the edit from a to b, as runs of kept, added, and removed text.
export function diffWords(a: string, b: string): Op[] {
	if (a === b) return a ? [{ kind: 'same', text: a }] : [];
	const x = tokens(a);
	const y = tokens(b);
	if (x.length > most || y.length > most) {
		return [...(a ? [{ kind: 'removed', text: a } as Op] : []), ...(b ? [{ kind: 'added', text: b } as Op] : [])];
	}
	// Longest common subsequence, then walk it back.
	const w = y.length + 1;
	const L = new Uint16Array((x.length + 1) * w);
	for (let i = x.length - 1; i >= 0; i--) {
		for (let j = y.length - 1; j >= 0; j--) {
			L[i * w + j] = x[i] === y[j] ? L[(i + 1) * w + j + 1]! + 1 : Math.max(L[(i + 1) * w + j]!, L[i * w + j + 1]!);
		}
	}
	const out: Op[] = [];
	const push = (kind: Op['kind'], text: string) => {
		const last = out[out.length - 1];
		if (last && last.kind === kind) last.text += text;
		else out.push({ kind, text });
	};
	let i = 0;
	let j = 0;
	while (i < x.length && j < y.length) {
		if (x[i] === y[j]) {
			push('same', x[i]!);
			i++;
			j++;
		} else if (L[(i + 1) * w + j]! >= L[i * w + j + 1]!) {
			push('removed', x[i]!);
			i++;
		} else {
			push('added', y[j]!);
			j++;
		}
	}
	for (; i < x.length; i++) push('removed', x[i]!);
	for (; j < y.length; j++) push('added', y[j]!);
	return out;
}
