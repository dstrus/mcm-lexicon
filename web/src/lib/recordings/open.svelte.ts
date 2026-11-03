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

	// dirty reports whether the draft's steps for step's option aren't on
	// disk yet: the saved recording has none, sets the option to something
	// else, or clicks a stepper another number of times.
	dirty(step: Step): boolean {
		const mine = this.steps.filter((s) => sameOption(s, step));
		const was = this.saved.filter((s) => sameOption(s, step));
		return mine.length !== was.length || mine.some((s, i) => s.value !== was[i]!.value || s.side !== was[i]!.side);
	}

	// clicks is how many times the recording clicks step's option: its
	// identical click steps. On a landing page, the default page's count.
	clicks(step: Step, landing = false): number {
		const n = this.steps.filter((s) => sameOption(s, step)).length;
		return n || !landing ? n : this.steps.filter((s) => sameOption(s, { ...step, page: '' })).length;
	}

	// find is the recording's step for the same option as step, with its
	// number (from 1), or null. On a landing page, the page the menu opens
	// on a copy of, a step on the default page is the same option.
	find(step: Step, landing = false): { n: number; step: Step } | null {
		let i = this.steps.findIndex((s) => sameOption(s, step));
		if (i < 0 && landing) i = this.steps.findIndex((s) => sameOption(s, { ...step, page: '' }));
		return i < 0 ? null : { n: i + 1, step: this.steps[i]! };
	}
}
