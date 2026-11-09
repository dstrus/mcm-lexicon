// MCMLexicon.dll: captures what each MCM shows, when MCM Lexicon asks
// (PLAN.md §11). It has no game plugin and writes nothing to the save.

#include "capture.h"

SKSEPluginLoad(const SKSE::LoadInterface* a_skse)
{
	SKSE::Init(a_skse);
	logger::info("MCM Lexicon capture {}", LEXICON_VERSION_TEXT);

	SKSE::GetMessagingInterface()->RegisterListener([](SKSE::MessagingInterface::Message* a_msg) {
		if (a_msg->type == SKSE::MessagingInterface::kPostLoadGame) {
			capture::OnGameLoaded();
		}
	});
	return true;
}
