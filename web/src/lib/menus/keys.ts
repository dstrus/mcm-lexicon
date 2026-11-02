// MCM Recorder's keymap steps hold DirectX scan codes (A is 30, F1 is 59),
// as the game does, with mouse buttons from 256 up. These name them.
// Writing them from a key pressed in the browser comes with the recording
// editor.
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
