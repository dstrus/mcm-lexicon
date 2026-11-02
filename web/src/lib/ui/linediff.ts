// What changed between two texts, line by line, for reading one file's
// version against the one it replaces.
import { diffWords } from './diff.ts';

export interface Line {
	kind: 'same' | 'added' | 'removed';
	text: string;
}

// Past this many lines a side, the quadratic comparison would stall, and
// the texts are shown as wholly replaced.
const most = 4000;

// diffLines is the edit from a to b, one entry per line.
export function diffLines(a: string, b: string): Line[] {
	const x = a === '' ? [] : a.replace(/\n$/, '').split('\n');
	const y = b === '' ? [] : b.replace(/\n$/, '').split('\n');
	if (x.length > most || y.length > most) {
		return [...x.map((text) => ({ kind: 'removed', text }) as Line), ...y.map((text) => ({ kind: 'added', text }) as Line)];
	}
	// Longest common subsequence, then walk it forward.
	const w = y.length + 1;
	const L = new Uint32Array((x.length + 1) * w);
	for (let i = x.length - 1; i >= 0; i--) {
		for (let j = y.length - 1; j >= 0; j--) {
			L[i * w + j] = x[i] === y[j] ? L[(i + 1) * w + j + 1]! + 1 : Math.max(L[(i + 1) * w + j]!, L[i * w + j + 1]!);
		}
	}
	const out: Line[] = [];
	let i = 0;
	let j = 0;
	while (i < x.length && j < y.length) {
		if (x[i] === y[j]) {
			out.push({ kind: 'same', text: x[i]! });
			i++;
			j++;
		} else if (L[(i + 1) * w + j]! >= L[i * w + j + 1]!) {
			out.push({ kind: 'removed', text: x[i++]! });
		} else {
			out.push({ kind: 'added', text: y[j++]! });
		}
	}
	while (i < x.length) out.push({ kind: 'removed', text: x[i++]! });
	while (j < y.length) out.push({ kind: 'added', text: y[j++]! });
	return out;
}

// Hunk is a run of changed lines with a few unchanged ones around it, or a
// count of unchanged lines left out between hunks.
export type Hunk = { lines: Line[] } | { skipped: number };

// hunks keeps the changes and `context` lines either side of each.
export function hunks(lines: Line[], context = 3): Hunk[] {
	const keep = new Uint8Array(lines.length);
	lines.forEach((l, i) => {
		if (l.kind === 'same') return;
		for (let k = Math.max(0, i - context); k <= Math.min(lines.length - 1, i + context); k++) keep[k] = 1;
	});
	const out: Hunk[] = [];
	let run: Line[] = [];
	let skipped = 0;
	lines.forEach((l, i) => {
		if (keep[i]) {
			if (skipped) out.push({ skipped });
			skipped = 0;
			run.push(l);
		} else {
			if (run.length) out.push({ lines: run });
			run = [];
			skipped++;
		}
	});
	if (run.length) out.push({ lines: run });
	if (skipped) out.push({ skipped });
	return out;
}

// A part of a changed line: whether it differs from the line it pairs with.
export interface Part {
	changed: boolean;
	text: string;
}

// markWords pairs each run of removed lines with the run of added lines
// after it, line for line, and marks the words that differ between each
// pair, so a long line with one word changed reads as that word. Lines
// without a partner have no parts: the whole line changed.
export function markWords(lines: Line[]): Map<Line, Part[]> {
	const out = new Map<Line, Part[]>();
	for (let i = 0; i < lines.length; ) {
		if (lines[i]!.kind !== 'removed') {
			i++;
			continue;
		}
		let r = i;
		while (r < lines.length && lines[r]!.kind === 'removed') r++;
		let a = r;
		while (a < lines.length && lines[a]!.kind === 'added') a++;
		for (let k = 0; k < Math.min(r - i, a - r); k++) {
			const was = lines[i + k]!;
			const now = lines[r + k]!;
			const ops = diffWords(was.text, now.text);
			out.set(was, ops.filter((o) => o.kind !== 'added').map((o) => ({ changed: o.kind === 'removed', text: o.text })));
			out.set(now, ops.filter((o) => o.kind !== 'removed').map((o) => ({ changed: o.kind === 'added', text: o.text })));
		}
		i = a;
	}
	return out;
}
