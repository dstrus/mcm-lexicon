// MCM Recorder's keymap steps hold DirectX scan codes (A is 30, F1 is 59),
// as the game does, with mouse buttons from 256 up and a gamepad's from
// 266 (SkyUI's own: Back is 271, LB 274). In VR the game maps the motion
// controllers' buttons onto the gamepad's. These name them, and turn a key
// pressed in the browser, or a gamepad button, into one.
import { m } from '#lib/copy/messages.ts';

const k = m.keys;

const names: Record<number, string> = {
	1: k.esc,
	12: '-',
	13: '=',
	14: k.backspace,
	15: k.tab,
	26: '[',
	27: ']',
	28: k.enter,
	29: k.leftCtrl,
	39: ';',
	40: "'",
	41: '`',
	42: k.leftShift,
	43: '\\',
	51: ',',
	52: '.',
	53: '/',
	54: k.rightShift,
	55: k.numpad({ key: '*' }),
	56: k.leftAlt,
	57: k.space,
	58: k.capsLock,
	69: k.numLock,
	70: k.scrollLock,
	71: k.numpad({ key: '7' }),
	72: k.numpad({ key: '8' }),
	73: k.numpad({ key: '9' }),
	74: k.numpad({ key: '-' }),
	75: k.numpad({ key: '4' }),
	76: k.numpad({ key: '5' }),
	77: k.numpad({ key: '6' }),
	78: k.numpad({ key: '+' }),
	79: k.numpad({ key: '1' }),
	80: k.numpad({ key: '2' }),
	81: k.numpad({ key: '3' }),
	82: k.numpad({ key: '0' }),
	83: k.numpad({ key: '.' }),
	87: 'F11',
	88: 'F12',
	156: k.numpadEnter,
	157: k.rightCtrl,
	181: k.numpad({ key: '/' }),
	183: k.printScreen,
	184: k.rightAlt,
	197: k.pause,
	199: k.home,
	200: k.up,
	201: k.pageUp,
	203: k.left,
	205: k.right,
	207: k.end,
	208: k.down,
	209: k.pageDown,
	210: k.insert,
	211: k.delete,
	256: k.leftMouse,
	257: k.rightMouse,
	258: k.middleMouse,
	264: k.wheelUp,
	265: k.wheelDown
};

// A gamepad's buttons, in SkyUI's order from 266, with the index each has
// in the browser's standard gamepad layout.
export const gamepadButtons: { code: number; name: string; index: number }[] = [
	{ code: 276, name: k.padA, index: 0 },
	{ code: 277, name: k.padB, index: 1 },
	{ code: 278, name: k.padX, index: 2 },
	{ code: 279, name: k.padY, index: 3 },
	{ code: 274, name: k.padLB, index: 4 },
	{ code: 275, name: k.padRB, index: 5 },
	{ code: 280, name: k.padLT, index: 6 },
	{ code: 281, name: k.padRT, index: 7 },
	{ code: 270, name: k.padStart, index: 9 },
	{ code: 271, name: k.padBack, index: 8 },
	{ code: 272, name: k.padLeftStick, index: 10 },
	{ code: 273, name: k.padRightStick, index: 11 },
	{ code: 266, name: k.padUp, index: 12 },
	{ code: 267, name: k.padDown, index: 13 },
	{ code: 268, name: k.padLeft, index: 14 },
	{ code: 269, name: k.padRight, index: 15 }
];
for (const b of gamepadButtons) names[b.code] = b.name;

// In Skyrim VR the game presents the motion controllers to the menus as a
// gamepad, so each controller button is one of the gamepad's codes. Which
// button is which code is the engine's doing, per controller family, and
// this table is provisional until confirmed in game: an entry that's wrong
// mislabels a button, but the code it writes is still a real one.
// The Quest-style controllers' pictures, in static/controller/, each at
// twice its size too (touch-a@2x.png).
export type Glyph = `touch-${'a' | 'b' | 'x' | 'y' | 'left-stick' | 'right-stick' | 'left-grip' | 'right-grip' | 'left-trigger' | 'right-trigger'}`;
export interface VRButton {
	code: number;
	name: string; // the controller's own name for the button
	glyph?: Glyph;
}
export interface VRFamily {
	name: string;
	buttons: VRButton[];
}
export const vrFamilies: VRFamily[] = [
	{
		name: k.vrTouch,
		buttons: [
			{ code: 276, name: k.vrA, glyph: 'touch-a' },
			{ code: 277, name: k.vrB, glyph: 'touch-b' },
			{ code: 278, name: k.vrX, glyph: 'touch-x' },
			{ code: 279, name: k.vrY, glyph: 'touch-y' },
			{ code: 274, name: k.vrLeftGrip, glyph: 'touch-left-grip' },
			{ code: 275, name: k.vrRightGrip, glyph: 'touch-right-grip' },
			{ code: 280, name: k.vrLeftTrigger, glyph: 'touch-left-trigger' },
			{ code: 281, name: k.vrRightTrigger, glyph: 'touch-right-trigger' },
			{ code: 272, name: k.vrLeftStick, glyph: 'touch-left-stick' },
			{ code: 273, name: k.vrRightStick, glyph: 'touch-right-stick' },
			{ code: 270, name: k.vrMenu }
		]
	},
	{
		name: k.vrWands,
		buttons: [
			{ code: 280, name: k.vrLeftTrigger },
			{ code: 281, name: k.vrRightTrigger },
			{ code: 274, name: k.vrLeftGrip },
			{ code: 275, name: k.vrRightGrip },
			{ code: 272, name: k.vrLeftPad },
			{ code: 273, name: k.vrRightPad },
			{ code: 270, name: k.vrLeftMenu },
			{ code: 271, name: k.vrRightMenu }
		]
	}
];

// inVR reports whether the list's game is Skyrim VR, as Mod Organizer
// names it, so the controller list can speak of motion controllers.
export function inVR(game: string | undefined): boolean {
	return /\bVR\b/i.test(game ?? '');
}

// The rows of letters and digits, which DirectX numbers in keyboard order.
const rows: [number, string][] = [
	[2, '1234567890'],
	[16, 'QWERTYUIOP'],
	[30, 'ASDFGHJKL'],
	[44, 'ZXCVBNM']
];
for (const [start, keys] of rows) [...keys].forEach((key, i) => (names[start + i] = key));
for (let i = 0; i < 10; i++) names[59 + i] = `F${i + 1}`;
for (let i = 0; i < 3; i++) names[259 + i] = k.mouseButton({ n: i + 4 });

// keyName is the key a scan code stands for, or its number when it isn't
// known; 0 and -1 mean no key.
export function keyName(code: number): string {
	if (code <= 0) return k.none;
	return names[code] ?? k.code({ code });
}

// The scan code of each key by its place on the keyboard (KeyboardEvent.code),
// which is what DirectX numbers too, whatever the layout prints on it.
const byPosition: Record<string, number> = {
	Escape: 1,
	Minus: 12,
	Equal: 13,
	Backspace: 14,
	Tab: 15,
	BracketLeft: 26,
	BracketRight: 27,
	Enter: 28,
	ControlLeft: 29,
	Semicolon: 39,
	Quote: 40,
	Backquote: 41,
	ShiftLeft: 42,
	Backslash: 43,
	Comma: 51,
	Period: 52,
	Slash: 53,
	ShiftRight: 54,
	NumpadMultiply: 55,
	AltLeft: 56,
	Space: 57,
	CapsLock: 58,
	NumLock: 69,
	ScrollLock: 70,
	Numpad7: 71,
	Numpad8: 72,
	Numpad9: 73,
	NumpadSubtract: 74,
	Numpad4: 75,
	Numpad5: 76,
	Numpad6: 77,
	NumpadAdd: 78,
	Numpad1: 79,
	Numpad2: 80,
	Numpad3: 81,
	Numpad0: 82,
	NumpadDecimal: 83,
	F11: 87,
	F12: 88,
	NumpadEnter: 156,
	ControlRight: 157,
	NumpadDivide: 181,
	PrintScreen: 183,
	AltRight: 184,
	Pause: 197,
	Home: 199,
	ArrowUp: 200,
	PageUp: 201,
	ArrowLeft: 203,
	ArrowRight: 205,
	End: 207,
	ArrowDown: 208,
	PageDown: 209,
	Insert: 210,
	Delete: 211
};
for (const [start, keys] of rows) [...keys].forEach((key, i) => (byPosition[/\d/.test(key) ? `Digit${key}` : `Key${key}`] = start + i));
for (let i = 0; i < 10; i++) byPosition[`F${i + 1}`] = 59 + i;

// scanCodeOf is the scan code for a key press, or undefined for a key the
// game has no code for.
export function scanCodeOf(e: KeyboardEvent): number | undefined {
	return byPosition[e.code];
}

// mouseCodeOf is the code for a mouse button press: left 256, right 257,
// middle 258, then the side buttons.
export function mouseCodeOf(e: MouseEvent): number | undefined {
	return [256, 258, 257, 259, 260][e.button];
}

// watchGamepads calls onPress with the code of each gamepad button pressed
// (a press, not a hold) on any gamepad the browser sees, until the
// returned function is called. Browsers see a gamepad only after a button
// is pressed on it, and some see none; the list of buttons is the sure way.
export function watchGamepads(onPress: (code: number) => void): () => void {
	if (typeof navigator === 'undefined' || typeof navigator.getGamepads !== 'function') return () => {};
	const was = new Map<number, boolean[]>();
	let frame = 0;
	const poll = () => {
		for (const g of navigator.getGamepads()) {
			if (!g) continue;
			const before = was.get(g.index) ?? [];
			g.buttons.forEach((b, i) => {
				if (!b.pressed || before[i]) return;
				const button = gamepadButtons.find((x) => x.index === i);
				if (button) onPress(button.code);
			});
			was.set(
				g.index,
				g.buttons.map((b) => b.pressed)
			);
		}
		frame = requestAnimationFrame(poll);
	};
	frame = requestAnimationFrame(poll);
	return () => cancelAnimationFrame(frame);
}
