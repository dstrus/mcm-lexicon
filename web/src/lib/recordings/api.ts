import { request } from '#lib/api/client.ts';
import type { OpenRequest, Status } from '#lib/api/types/api.ts';
import type { CompareReply, CompareRequest, Detail, ListReply } from '#lib/api/types/recordings.ts';

export const recordingsDependency = 'app:recordings';

export function getRecordings(fetchFn?: typeof fetch): Promise<ListReply> {
	return request<ListReply>('GET', '/api/recordings', undefined, fetchFn);
}

export function getRecording(name: string, fetchFn?: typeof fetch): Promise<Detail> {
	return request<Detail>('GET', `/api/recordings/${encodeURIComponent(name)}`, undefined, fetchFn);
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
