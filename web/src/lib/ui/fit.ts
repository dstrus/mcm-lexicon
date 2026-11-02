// fitToViewport is a Svelte action that makes an element exactly as tall as
// the space between its top edge and the hint strip, so a pane that sticks to
// the top as the page scrolls never runs off the bottom of the window, and
// what it keeps at its own bottom (a form's Save) stays in view.
export function fitToViewport(node: HTMLElement) {
	function fit() {
		if (getComputedStyle(node).position !== 'sticky') {
			node.style.height = ''; // stacked under the list on narrow screens
			return;
		}
		const top = Math.max(0, node.getBoundingClientRect().top);
		const hints = document.querySelector<HTMLElement>('.hints')?.offsetHeight ?? 0;
		node.style.height = `${window.innerHeight - top - hints}px`;
	}
	fit();
	window.addEventListener('scroll', fit, { passive: true });
	window.addEventListener('resize', fit);
	const watch = new ResizeObserver(fit);
	watch.observe(document.body);
	return {
		destroy() {
			window.removeEventListener('scroll', fit);
			window.removeEventListener('resize', fit);
			watch.disconnect();
		}
	};
}

// aboveHints is a Svelte action for a footer that sticks to the bottom of the
// window as the page scrolls (a long form's Save): it sticks just above the
// hint strip, whose height changes as its hints wrap.
export function aboveHints(node: HTMLElement) {
	function place() {
		node.style.bottom = `${document.querySelector<HTMLElement>('.hints')?.offsetHeight ?? 0}px`;
	}
	place();
	window.addEventListener('resize', place);
	const watch = new ResizeObserver(place);
	const hints = document.querySelector<HTMLElement>('.hints');
	if (hints) watch.observe(hints);
	return {
		destroy() {
			window.removeEventListener('resize', place);
			watch.disconnect();
		}
	};
}
