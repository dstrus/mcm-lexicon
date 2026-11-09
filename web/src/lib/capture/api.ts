import { request } from '#lib/api/client.ts';
import type { SettingsRequest, Status } from '#lib/api/types/capture.ts';

// getCapture is what Settings shows of the capture from the game: the
// plugin's folder, a request waiting, and the capture added.
export function getCapture(): Promise<Status> {
	return request<Status>('GET', '/api/capture');
}

// requestCapture asks the plugin to capture at the next load of a save.
export function requestCapture(): Promise<Status> {
	return request<Status>('POST', '/api/capture/request');
}

// cancelRequest takes back a request the game hasn't acted on.
export function cancelRequest(): Promise<Status> {
	return request<Status>('DELETE', '/api/capture/request');
}

// setCaptureSettings chooses whether a capture is added, whether requests
// include trackers, and a capture file to add in place of the game's.
export function setCaptureSettings(req: SettingsRequest): Promise<Status> {
	return request<Status>('PUT', '/api/capture/settings', req);
}
