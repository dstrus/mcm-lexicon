import { request } from './client.ts';
import type { EditingRequest, SaveFolder, SourceRequest, Status } from './types/api.ts';

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
