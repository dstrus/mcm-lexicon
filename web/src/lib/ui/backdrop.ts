// closeOnBackdrop gives a modal dialog the handlers that close it when its
// backdrop is clicked, to spread onto the <dialog>. A click whose press began
// inside the dialog doesn't count, though the browser sends it to the dialog
// itself when the button is let go outside, as it is when selecting text
// past the dialog's edge.
export function closeOnBackdrop(close: (e: Event) => void) {
	let pressedOutside = false;
	return {
		onpointerdown: (e: PointerEvent) => {
			pressedOutside = e.target === e.currentTarget;
		},
		onclick: (e: MouseEvent) => {
			if (pressedOutside && e.target === e.currentTarget) close(e);
			pressedOutside = false;
		}
	};
}
