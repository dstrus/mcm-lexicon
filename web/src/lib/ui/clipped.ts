// clipped is a Svelte action for text that may be cut short with an
// ellipsis: when it is, resting the pointer on it shows the whole text in
// a tip beneath it, in the app's own materials rather than the browser's.
// Text that fits shows nothing.

const delay = 250; // ms before the tip appears, so passing over rows doesn't flicker

export function clipped(node: HTMLElement) {
	let tip: HTMLElement | undefined;
	let timer: ReturnType<typeof setTimeout> | undefined;

	function show() {
		if (node.scrollWidth <= node.clientWidth + 1) return;
		const text = node.textContent?.trim();
		if (!text) return;
		tip = document.createElement('div');
		tip.className = 'tip';
		tip.setAttribute('role', 'tooltip');
		tip.textContent = text;
		document.body.appendChild(tip);
		const r = node.getBoundingClientRect();
		const t = tip.getBoundingClientRect();
		const gap = 4;
		let left = Math.min(r.left, window.innerWidth - t.width - 8);
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

	node.addEventListener('mouseenter', enter);
	node.addEventListener('mouseleave', hide);
	return {
		destroy() {
			hide();
			node.removeEventListener('mouseenter', enter);
			node.removeEventListener('mouseleave', hide);
		}
	};
}
