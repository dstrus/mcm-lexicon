import { request } from './client.ts';
import type { SourceRequest, Status } from './types/api.ts';

export function getStatus(fetchFn?: typeof fetch): Promise<Status> {
	return request<Status>('GET', '/api/status', undefined, fetchFn);
}

// setSource chooses the mod list (and profile) to read, or a catalog file
// to open instead; MCM Lexicon remembers the choice.
export function setSource(req: SourceRequest): Promise<Status> {
	return request<Status>('PUT', '/api/source', req);
}
