// clipped is a Svelte action for text that may be cut short with an
// ellipsis: when it is, resting the pointer on it, or focusing the link or
// button it's in, shows the whole text in a tip beneath it, in the app's
// own materials rather than the browser's, and in the text's own type, so
// it reads as the same words continued. Text that fits shows nothing,
// unless a note is given (a menu's raw $KEY name): the tip then shows the
// note under the text, in the file font, whether the text is cut short or
// not. On a select, the text is the entry it shows.

const delay = 250; // ms before the tip appears, so passing over rows doesn't flicker

// A select doesn't report its shown entry's width, so the entry is
// measured in the select's own type against the room inside its padding,
// which leaves out the chevron's.
let ruler: CanvasRenderingContext2D | null | undefined;
function selectCut(node: HTMLSelectElement, text: string): boolean {
	ruler ??= document.createElement('canvas').getContext('2d');
	if (!ruler) return false;
	const type = getComputedStyle(node);
	// Built from its parts: the font shorthand can read back empty.
	ruler.font = `${type.fontStyle} ${type.fontWeight} ${type.fontSize} ${type.fontFamily}`;
	ruler.letterSpacing = type.letterSpacing === 'normal' ? '0px' : type.letterSpacing;
	const room = node.clientWidth - parseFloat(type.paddingLeft) - parseFloat(type.paddingRight);
	return ruler.measureText(text).width > room + 1;
}

export function clipped(node: HTMLElement, note?: string) {
	let tip: HTMLElement | undefined;
	let timer: ReturnType<typeof setTimeout> | undefined;

	function show() {
		const text = (node instanceof HTMLSelectElement ? node.selectedOptions[0]?.text : node.textContent)?.trim();
		if (!text) return;
		const cut = node instanceof HTMLSelectElement ? selectCut(node, text) : node.scrollWidth > node.clientWidth + 1;
		if (!cut && !note) return;
		tip = document.createElement('div');
		tip.className = 'tip';
		tip.setAttribute('role', 'tooltip');
		tip.textContent = text;
		if (note) {
			const raw = document.createElement('span');
			raw.className = 'tip-note';
			raw.textContent = note;
			tip.appendChild(raw);
		}
		const type = getComputedStyle(node);
		tip.style.fontFamily = type.fontFamily;
		tip.style.fontSize = type.fontSize;
		tip.style.fontWeight = type.fontWeight;
		tip.style.letterSpacing = type.letterSpacing;
		tip.style.textTransform = type.textTransform;
		document.body.appendChild(tip);
		const r = node.getBoundingClientRect();
		const t = tip.getBoundingClientRect();
		const gap = 4;
		// Set in from the text's start by four of its letters' height, so
		// the first word or so of the row beneath, in a list, still shows
		// past the tip's left edge.
		const indent = 4 * parseFloat(type.fontSize);
		let left = Math.min(r.left + indent, window.innerWidth - t.width - 8);
		let top = r.bottom + gap;
		if (top + t.height > window.innerHeight - 8) top = r.top - gap - t.height;
		tip.style.left = `${Math.max(8, left)}px`;
		tip.style.top = `${top}px`;
		window.addEventListener('scroll', hide, { capture: true, passive: true });
	}
	function hide() {
		clearTimeout(timer);
		timer = undefined;
		tip?.remove();
		tip = undefined;
		window.removeEventListener('scroll', hide, { capture: true });
	}
	function enter() {
		clearTimeout(timer);
		timer = setTimeout(show, delay);
	}

	const control = node.closest<HTMLElement>('a, button');
	node.addEventListener('mouseenter', enter);
	node.addEventListener('mouseleave', hide);
	// A select's list opening, or a new entry, takes the tip away.
	node.addEventListener('mousedown', hide);
	node.addEventListener('change', hide);
	control?.addEventListener('focus', enter);
	control?.addEventListener('blur', hide);
	return {
		update(next?: string) {
			note = next;
		},
		destroy() {
			hide();
			node.removeEventListener('mouseenter', enter);
			node.removeEventListener('mouseleave', hide);
			node.removeEventListener('mousedown', hide);
			node.removeEventListener('change', hide);
			control?.removeEventListener('focus', enter);
			control?.removeEventListener('blur', hide);
		}
	};
}
