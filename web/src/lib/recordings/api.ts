import { request } from '#lib/api/client.ts';
import type { OpenRequest, Status } from '#lib/api/types/api.ts';
import type { AddAllReply, AddReply, BackupsReply, Changes, CompareReply, CompareRequest, CreateRequest, DeletedReply, Draft, Editor, ListReply, SaveReply, Step } from '#lib/api/types/recordings.ts';

export const recordingsDependency = 'app:recordings';

const at = (name: string) => `/api/recordings/${encodeURIComponent(name)}`;

export function getRecordings(fetchFn?: typeof fetch): Promise<ListReply> {
	return request<ListReply>('GET', '/api/recordings', undefined, fetchFn);
}

// getRecording is a recording as the editor shows it: its draft, or the
// saved recording when there's none.
export function getRecording(name: string, fetchFn?: typeof fetch): Promise<Editor> {
	return request<Editor>('GET', at(name), undefined, fetchFn);
}

// create makes a new recording, held as a draft until it's saved.
export function create(req: CreateRequest): Promise<Editor> {
	return request<Editor>('POST', '/api/recordings', req);
}

export function putDraft(name: string, draft: Draft): Promise<Editor> {
	return request<Editor>('PUT', `${at(name)}/draft`, draft);
}

// putFileText replaces one of a recording's files with its text as typed:
// file 0 is the info file, and file n the nth step file Save would write.
export function putFileText(name: string, file: number, text: string): Promise<Editor> {
	return request<Editor>('PUT', `${at(name)}/files/${file}`, { text });
}

// getBackups is every backup of a recording, newest first, each with its
// files against what's on disk.
export function getBackups(name: string, fetchFn?: typeof fetch): Promise<BackupsReply> {
	return request<BackupsReply>('GET', `${at(name)}/backups`, undefined, fetchFn);
}

// restoreFrom puts a backup in the draft: Save writes its files exactly.
export function restoreFrom(name: string, id: string): Promise<Editor> {
	return request<Editor>('PUT', `${at(name)}/backups/${encodeURIComponent(id)}`);
}

// getDeletedBackups is each recording the list no longer has that has a
// backup.
export function getDeletedBackups(fetchFn?: typeof fetch): Promise<DeletedReply> {
	return request<DeletedReply>('GET', '/api/backups', undefined, fetchFn);
}

export function discardDraft(name: string): Promise<Editor> {
	return request<Editor>('DELETE', `${at(name)}/draft`);
}

// addStep appends a step to a recording's draft.
export function addStep(name: string, step: Step): Promise<AddReply> {
	return request<AddReply>('POST', `${at(name)}/steps`, { step });
}

// setClicks makes the recording click a stepper n times: that many
// identical click steps, in place; 0 takes them out.
export function setClicks(name: string, step: Step, clicks: number): Promise<AddReply> {
	return request<AddReply>('POST', `${at(name)}/steps`, { step, clicks });
}

// addAll records many options at once; ones the recording has keep their
// steps.
export function addAll(name: string, steps: Step[]): Promise<AddAllReply> {
	return request<AddAllReply>('POST', `${at(name)}/steps/all`, { steps });
}

// deleteRecording puts a recording's deletion in the draft; Save does it.
export function deleteRecording(name: string): Promise<Editor> {
	return request<Editor>('DELETE', at(name));
}

// restore takes a recording's deletion out of the draft.
export function restore(name: string): Promise<Editor> {
	return request<Editor>('POST', `${at(name)}/restore`);
}

// getChanges is the whole draft, for reviewing before Save.
export function getChanges(fetchFn?: typeof fetch): Promise<Changes> {
	return request<Changes>('GET', '/api/changes', undefined, fetchFn);
}

// saveAll writes the whole draft.
export function saveAll(): Promise<SaveReply> {
	return request<SaveReply>('POST', '/api/changes/save');
}

// discardAll drops the whole draft.
// undo takes back the draft's last change; redo puts back the last undone.
export function undo(): Promise<Changes> {
	return request<Changes>('POST', '/api/changes/undo');
}

export function redo(): Promise<Changes> {
	return request<Changes>('POST', '/api/changes/redo');
}

export function discardAll(): Promise<Changes> {
	return request<Changes>('DELETE', '/api/changes');
}

// setOpen makes a recording the one new steps go to; "" closes it.
export function setOpen(name: string): Promise<Status> {
	return request<Status>('PUT', '/api/recordings/open', { name } satisfies OpenRequest);
}

export function compare(path: string): Promise<CompareReply> {
	return request<CompareReply>('POST', '/api/compare', { path } satisfies CompareRequest);
}

export function recordingHref(name: string, tab = ''): string {
	return `/recordings/${encodeURIComponent(name)}${tab ? `/${tab}` : ''}`;
}
