#pragma once

namespace capture
{
	// OnGameLoaded starts a capture, on a thread of its own, if MCM Lexicon
	// has asked for one: a capture-request.json in the plugin's folder
	// beside the SKSE logs. Without a request it does nothing.
	void OnGameLoaded();
}
