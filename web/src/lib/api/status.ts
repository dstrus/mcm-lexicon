import { request } from './client.ts';
import type { EditingRequest, SaveFolder, SourceRequest, Status } from './types/api.ts';
import type { Listing, Looking, Source } from './types/modlist.ts';

export function getStatus(fetchFn?: typeof fetch): Promise<Status> {
	return request<Status>('GET', '/api/status', undefined, fetchFn);
}

// setSource chooses the mod list (and profile) to read, or a catalog file
// to open instead; MCM Lexicon remembers the choice.
export function setSource(req: SourceRequest): Promise<Status> {
	return request<Status>('PUT', '/api/source', req);
}

// setEditing turns editing on or off; off with changes in the draft needs
// discard.
export function setEditing(editing: boolean, discard = false): Promise<Status> {
	return request<Status>('PUT', '/api/editing', { editing, discard } satisfies EditingRequest);
}

// setSaveFolder names the mod folder recordings are saved to in an MO2
// instance, and whether [NoDelete] goes before the name.
export function setSaveFolder(name: string, noDelete: boolean): Promise<Status> {
	return request<Status>('PUT', '/api/settings/save-folder', { name, noDelete } satisfies SaveFolder);
}

// browseFolders lists a folder for the picker, marked for what it's
// looking for: a mod list's folders, or catalog files. With no path, it
// lists the folder MCM Lexicon runs from.
export function browseFolders(path = '', looking: Looking = 'list'): Promise<Listing> {
	return request<Listing>('GET', `/api/folders?for=${looking}&path=${encodeURIComponent(path)}`);
}

// checkSource is the mod list at path, without choosing it: its kind, and
// for an MO2 instance its profiles and the one MO2 has selected.
export function checkSource(path: string): Promise<Source> {
	return request<Source>('GET', `/api/source?path=${encodeURIComponent(path)}`);
}
