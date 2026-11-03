// The recording new steps go to, as a page sees it: its name, if it's still
// there, and its steps, read again whenever the recordings change, so a
// page can show what's recorded already, and which of it isn't saved yet.
import { page } from '$app/state';
import type { Status } from '#lib/api/types/api.ts';
import type { ListReply, Step } from '#lib/api/types/recordings.ts';
import { getRecording } from './api.ts';
import { sameOption } from './steps.ts';

export class OpenRecording {
	steps = $state<Step[]>([]);
	// The steps as the game sees them now: what Save has written. Empty for
	// a recording never saved.
	saved = $state<Step[]>([]);

	// Whether changes can be made at all.
	editing = $derived(!!(page.data.status as Status | undefined)?.editing);
	// The name the top bar chose, if that recording is still there: a new one
	// that was discarded counts as none.
	name = $derived.by(() => {
		const want = ((page.data.status as Status | undefined)?.openRecording ?? '').toLowerCase();
		const list = page.data.recordings as ListReply | null | undefined;
		return list?.recordings.find((r) => r.name.toLowerCase() === want)?.name ?? '';
	});

	constructor() {
		$effect(() => {
			const name = this.name;
			void page.data.recordings; // reloaded after every change
			if (!this.editing || !name) {
				this.steps = [];
				this.saved = [];
				return;
			}
			getRecording(name)
				.then((ed) => {
					this.steps = ed.draft.steps;
					this.saved = ed.recording?.steps ?? [];
				})
				.catch(() => {
					this.steps = [];
					this.saved = [];
				});
		});
	}

	// dirty reports whether a step of the draft isn't on disk yet: the saved
	// recording has no step for its option, or sets it to something else.
	dirty(step: Step): boolean {
		const was = this.saved.find((s) => sameOption(s, step));
		return !was || was.value !== step.value || was.side !== step.side;
	}

	// find is the recording's step for the same option as step, with its
	// number (from 1), or null.
	find(step: Step): { n: number; step: Step } | null {
		const i = this.steps.findIndex((s) => sameOption(s, step));
		return i < 0 ? null : { n: i + 1, step: this.steps[i]! };
	}
}
