// Tokens of a JSON file, for showing one with its parts set apart: the
// keys MCM Recorder reads, the values it fills in, and the punctuation
// between. Nothing here checks the file; the server does that when the
// text is applied, so half-typed text tokenizes too (a string left open
// runs to the end of its line).

export type Kind = 'text' | 'key' | 'string' | 'number' | 'literal' | 'punct';

export interface Token {
	kind: Kind;
	text: string;
}

const token = /\s+|"(?:[^"\\\n]|\\.)*"?|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?|true|false|null|[{}[\]:,]|[^\s"{}[\]:,]+/gy;

// tokens splits a file into its parts. A string is a key when a colon
// follows it, with only white space between.
export function tokens(text: string): Token[] {
	const out: Token[] = [];
	token.lastIndex = 0;
	let m: RegExpExecArray | null;
	while ((m = token.exec(text))) {
		const t = m[0];
		let kind = kindOf(t);
		if (kind === 'string') {
			const rest = text.slice(token.lastIndex);
			if (/^\s*:/.test(rest)) kind = 'key';
		}
		out.push({ kind, text: t });
	}
	return out;
}

function kindOf(t: string): Kind {
	const c = t[0]!;
	if (/\s/.test(c)) return 'text';
	if (c === '"') return 'string';
	if (c === '-' || /\d/.test(c)) return 'number';
	if (t === 'true' || t === 'false' || t === 'null') return 'literal';
	if (t.length === 1 && '{}[]:,'.includes(t)) return 'punct';
	return 'text';
}
