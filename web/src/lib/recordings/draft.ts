// Changing a recording's draft: each change sends the whole draft, and the
// server answers with the editor as it now stands.
import { invalidate } from '$app/navigation';
import type { Draft, Editor, Step } from '#lib/api/types/recordings.ts';
import { putDraft, recordingsDependency } from './api.ts';

// change applies edit to a copy of the editor's draft, sends it, and
// reloads what depends on recordings.
export async function change(ed: Editor, edit: (d: Draft) => void): Promise<Editor> {
	const d: Draft = { ...ed.draft, steps: ed.draft.steps.map((s) => ({ ...s })) };
	edit(d);
	const next = await putDraft(d.name, d);
	await invalidate(recordingsDependency);
	return next;
}

// move is the steps with the one at i moved by one place (-1 up, 1 down).
export function move(steps: Step[], i: number, by: -1 | 1): void {
	const j = i + by;
	if (j < 0 || j >= steps.length) return;
	[steps[i], steps[j]] = [steps[j]!, steps[i]!];
}
