// MCM Recorder's keymap steps hold DirectX scan codes (A is 30, F1 is 59),
// as the game does, with mouse buttons from 256 up. These name them, and
// turn a key pressed in the browser into one.
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
