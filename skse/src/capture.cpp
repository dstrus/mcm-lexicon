// The capture: each menu SkyUI lists, each of its pages run as the Journal
// runs it, and what the page added read back from SkyUI's option buffers
// (PLAN.md §11). Script calls and reads happen on the game's main thread,
// through SKSE's task queue, or in the callback of the call that ran the
// page; the thread that drives it only waits.

#include "capture.h"

using json = nlohmann::ordered_json;
using RE::BSScript::Object;
using RE::BSScript::Variable;
using ObjectPtr = RE::BSTSmartPointer<Object>;

namespace capture
{
	namespace
	{
		// How long the menus have to settle after a load, and how long one
		// page or one main-thread task may take.
		constexpr auto settleFirst = 10s;
		constexpr auto settleQuiet = 5s;
		constexpr auto settleMost = 120s;
		constexpr auto pageMost = 10s;
		constexpr auto taskMost = 10s;

		std::atomic_bool running{ false };

		// The plugin's folder, MCM Lexicon beside the SKSE logs: outside a
		// mod manager's virtual files, so the app finds it without the list.
		std::optional<std::filesystem::path> Folder()
		{
			auto dir = logger::log_directory();
			if (!dir) {
				return std::nullopt;
			}
			return *dir / "MCM Lexicon";
		}

		// OnMain runs a_fn on the main thread and waits for its result, or
		// gives up after taskMost.
		template <class F>
		auto OnMain(F a_fn) -> std::optional<std::invoke_result_t<F>>
		{
			using R = std::invoke_result_t<F>;
			auto done = std::make_shared<std::promise<R>>();
			auto result = done->get_future();
			SKSE::GetTaskInterface()->AddTask([done, a_fn]() mutable { done->set_value(a_fn()); });
			if (result.wait_for(taskMost) != std::future_status::ready) {
				return std::nullopt;
			}
			return result.get();
		}

		// SkyUI's option types, by their numbers in SKI_ConfigBase.
		std::string_view TypeName(std::int32_t a_type)
		{
			switch (a_type) {
			case 1:
				return "header"sv;
			case 2:
				return "text"sv;
			case 3:
				return "toggle"sv;
			case 4:
				return "slider"sv;
			case 5:
				return "menu"sv;
			case 6:
				return "color"sv;
			case 7:
				return "keymap"sv;
			case 8:
				return "input"sv;
			default:
				return "empty"sv;
			}
		}

		// The upper half of Windows-1252 that isn't Latin-1, from 0x80.
		constexpr char32_t cp1252[32] = {
			0x20AC, 0x0081, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021, 0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008D, 0x017D, 0x008F,
			0x0090, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014, 0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x009D, 0x017E, 0x0178
		};

		// Text decodes a script's string as the catalog does (windows1252 in
		// internal/plugin): its bytes are Windows-1252, as the Creation Kit
		// writes them. JSON needs UTF-8, and the catalog's labels must match.
		std::string Text(std::string_view a_bytes)
		{
			std::string out;
			out.reserve(a_bytes.size());
			for (unsigned char c : a_bytes) {
				char32_t r = c >= 0x80 && c < 0xA0 ? cp1252[c - 0x80] : c;
				if (r < 0x80) {
					out += static_cast<char>(r);
				} else if (r < 0x800) {
					out += static_cast<char>(0xC0 | (r >> 6));
					out += static_cast<char>(0x80 | (r & 0x3F));
				} else {
					out += static_cast<char>(0xE0 | (r >> 12));
					out += static_cast<char>(0x80 | ((r >> 6) & 0x3F));
					out += static_cast<char>(0x80 | (r & 0x3F));
				}
			}
			return out;
		}

		std::string Lower(std::string_view a_text)
		{
			std::string s(a_text);
			std::ranges::transform(s, s.begin(), [](unsigned char c) { return static_cast<char>(std::tolower(c)); });
			return s;
		}

		// QuestKey names the menu's quest as the catalog does
		// (plugin.FormKey): its plugin, lowercased, and its ID within it.
		std::string QuestKey(const ObjectPtr& a_menu)
		{
			auto vm = RE::BSScript::Internal::VirtualMachine::GetSingleton();
			auto quest = vm->GetObjectHandlePolicy()->GetObjectForHandle(RE::FormType::Quest, a_menu->GetHandle());
			if (!quest) {
				return {};
			}
			auto file = quest->GetFile(0);
			if (!file) {
				return {};
			}
			return std::format("{}:{:x}", Lower(file->GetFilename()), quest->GetLocalFormID() & 0xFFFFFF);
		}

		struct Menu
		{
			std::string name;    // as SkyUI lists it
			std::string script;  // the menu's script
			std::string quest;
			std::vector<std::string> pages;
			ObjectPtr object;
		};

		// FindManager is SkyUI's SKI_ConfigManager, found by the script bound
		// to a quest rather than by a form ID, which SkyUI's ports may move.
		ObjectPtr FindManager()
		{
			auto vm = RE::BSScript::Internal::VirtualMachine::GetSingleton();
			auto policy = vm->GetObjectHandlePolicy();
			for (auto quest : RE::TESDataHandler::GetSingleton()->GetFormArray<RE::TESQuest>()) {
				if (!quest) {
					continue;
				}
				ObjectPtr found;
				auto handle = policy->GetHandleForObject(RE::FormType::Quest, quest);
				if (handle && vm->FindBoundObject(handle, "SKI_ConfigManager", found) && found) {
					return found;
				}
			}
			return {};
		}

		// Menus lists the menus SkyUI has registered, in its order. Main
		// thread only.
		std::vector<Menu> Menus(const ObjectPtr& a_manager)
		{
			std::vector<Menu> menus;
			auto configs = a_manager->GetVariable("_modConfigs");
			auto names = a_manager->GetVariable("_modNames");
			if (!configs || !names || !configs->IsObjectArray()) {
				return menus;
			}
			auto c = configs->GetArray();
			auto n = names->GetArray();
			for (std::uint32_t i = 0; c && n && i < c->size() && i < n->size(); ++i) {
				auto object = (*c)[i].GetObject();
				if (!object) {
					continue;
				}
				Menu m{ .name = Text((*n)[i].GetString()), .object = object };
				if (auto type = object->GetTypeInfo()) {
					m.script = type->GetName();
				}
				m.quest = QuestKey(object);
				if (auto pages = object->GetProperty("Pages"); pages && pages->IsArray()) {
					if (auto a = pages->GetArray()) {
						for (std::uint32_t p = 0; p < a->size(); ++p) {
							m.pages.emplace_back((*a)[p].GetString());  // raw: SetPage gets them back
						}
					}
				}
				menus.push_back(std::move(m));
			}
			return menus;
		}

		// ReadPage reads the option buffers a page just filled: each index a
		// position on the page's grid, the type in the flags' low byte.
		json ReadPage(const ObjectPtr& a_menu)
		{
			json options = json::array();
			auto array = [&](std::string_view a_name) -> RE::BSTSmartPointer<RE::BSScript::Array> {
				auto v = a_menu->GetVariable(a_name);
				return v && v->IsArray() ? v->GetArray() : nullptr;
			};
			auto flags = array("_optionFlagsBuf");
			auto text = array("_textBuf");
			auto str = array("_strValueBuf");
			auto num = array("_numValueBuf");
			auto state = array("_stateOptionMap");  // SkyUI 4 has none
			if (!flags || !text || !str || !num) {
				return options;
			}
			auto size = std::min({ flags->size(), text->size(), str->size(), num->size() });
			for (std::uint32_t i = 0; i < size; ++i) {
				auto f = (*flags)[i].GetSInt();
				if ((f & 0xFF) == 0) {
					continue;  // empty: nothing there, or an empty option, which SkyUI can't tell apart
				}
				options.push_back({
					{ "position", i },
					{ "type", TypeName(f & 0xFF) },
					{ "flags", f >> 8 },
					{ "text", Text((*text)[i].GetString()) },
					{ "str", Text((*str)[i].GetString()) },
					{ "num", (*num)[i].GetFloat() },
					{ "state", state && i < state->size() ? Text((*state)[i].GetString()) : ""s },
				});
			}
			return options;
		}

		// Done calls back when a page's SetPage returns.
		class Done : public RE::BSScript::IStackCallbackFunctor
		{
		public:
			explicit Done(std::function<void()> a_fn) :
				fn(std::move(a_fn)) {}
			void operator()(Variable) override { fn(); }
			void SetObject(const ObjectPtr&) override {}

		private:
			std::function<void()> fn;
		};

		// RunPage runs one page as the Journal would, SetPage(page, index),
		// and returns what it added, or nullopt after pageMost. A call that
		// returns late finds no one waiting and changes nothing.
		std::optional<json> RunPage(const Menu& a_menu, const std::string& a_page, std::int32_t a_index)
		{
			auto done = std::make_shared<std::promise<json>>();
			auto result = done->get_future();
			auto object = a_menu.object;
			auto sent = OnMain([object, a_page, a_index, done]() mutable {
				auto vm = RE::BSScript::Internal::VirtualMachine::GetSingleton();
				RE::BSTSmartPointer<RE::BSScript::IStackCallbackFunctor> callback{ new Done([object, done]() {
					done->set_value(ReadPage(object));
				}) };
				auto args = RE::MakeFunctionArguments(RE::BSFixedString(a_page), std::move(a_index));
				return vm->DispatchMethodCall(object, "SetPage", args, callback);
			});
			if (!sent || !*sent || result.wait_for(pageMost) != std::future_status::ready) {
				return std::nullopt;
			}
			return result.get();
		}

		bool JournalOpen()
		{
			return OnMain([]() { return RE::UI::GetSingleton()->IsMenuOpen(RE::JournalMenu::MENU_NAME); }).value_or(true);
		}

		// Settle waits for SkyUI's menus to register after a load: settleFirst,
		// then until the count stops changing for settleQuiet, at most
		// settleMost in all. It returns the manager, or none if SkyUI isn't
		// there.
		ObjectPtr Settle()
		{
			std::this_thread::sleep_for(settleFirst);
			auto manager = OnMain(FindManager).value_or(ObjectPtr{});
			if (!manager) {
				return {};
			}
			auto count = [&]() { return OnMain([manager]() { return Menus(manager).size(); }).value_or(0); };
			auto start = std::chrono::steady_clock::now();
			auto last = count();
			auto quietSince = std::chrono::steady_clock::now();
			while (std::chrono::steady_clock::now() - start < settleMost - settleFirst) {
				std::this_thread::sleep_for(1s);
				auto now = count();
				if (now != last) {
					last = now;
					quietSince = std::chrono::steady_clock::now();
				} else if (std::chrono::steady_clock::now() - quietSince >= settleQuiet) {
					break;
				}
			}
			return manager;
		}

		std::string Now()
		{
			return std::format("{:%FT%TZ}", std::chrono::floor<std::chrono::seconds>(std::chrono::system_clock::now()));
		}

		// Write writes the capture beside the request, to a temporary name
		// first, so MCM Lexicon never reads half a file.
		bool Write(const std::filesystem::path& a_folder, const json& a_capture)
		{
			auto target = a_folder / "capture.json";
			auto temp = a_folder / "capture.json.tmp";
			{
				std::ofstream out(temp, std::ios::binary | std::ios::trunc);
				if (!out) {
					return false;
				}
				out << a_capture.dump(1, '\t');
				if (!out) {
					return false;
				}
			}
			std::error_code err;
			std::filesystem::rename(temp, target, err);
			return !err;
		}

		void Run(std::filesystem::path a_folder, std::vector<std::string> a_skip)
		{
			auto manager = Settle();
			json file = {
				{ "format", 1 },
				{ "plugin", LEXICON_VERSION_TEXT },
				{ "game", REL::Module::IsVR() ? "SkyrimVR" : "SkyrimSE" },
				{ "runtime", REL::Module::get().version().string(".") },
				{ "captured", Now() },
				{ "menus", json::array() },
				{ "skipped", json::array() },
			};
			if (!manager) {
				file["stopped"] = "SkyUI's menus weren't found";
			}

			auto menus = manager ? OnMain([manager]() { return Menus(manager); }).value_or(std::vector<Menu>{}) : std::vector<Menu>{};
			logger::info("Capturing {} menus", menus.size());
			for (const auto& m : menus) {
				if (std::ranges::any_of(a_skip, [&](const std::string& s) { return Lower(s) == Lower(m.script); })) {
					file["skipped"].push_back({ { "script", m.script }, { "reason", "asked to skip" } });
					continue;
				}
				json menu = {
					{ "name", m.name },
					{ "script", m.script },
					{ "quest", m.quest },
					{ "pages", json::array() },
				};
				auto pages = m.pages.empty() ? std::vector<std::string>{ "" } : m.pages;
				for (std::int32_t i = 0; i < static_cast<std::int32_t>(pages.size()); ++i) {
					// SetPage draws into the Journal when it's open.
					if (JournalOpen()) {
						file["stopped"] = "the Journal menu opened";
						break;
					}
					json page = { { "name", Text(pages[i]) } };
					if (auto options = RunPage(m, pages[i], m.pages.empty() ? -1 : i)) {
						page["options"] = std::move(*options);
					} else {
						page["error"] = "timed out";
						logger::warn("{}: page \"{}\" timed out", m.name, Text(pages[i]));
					}
					menu["pages"].push_back(std::move(page));
				}
				file["menus"].push_back(std::move(menu));
				if (file.contains("stopped")) {
					break;
				}
			}

			if (Write(a_folder, file)) {
				std::error_code err;
				std::filesystem::remove(a_folder / "capture-request.json", err);
				logger::info("Wrote the capture of {} menus", file["menus"].size());
			} else {
				logger::error("Couldn't write {}", (a_folder / "capture.json").string());
			}
			running = false;
		}
	}

	void OnGameLoaded()
	{
		auto folder = Folder();
		if (!folder) {
			return;
		}
		auto request = *folder / "capture-request.json";
		std::error_code err;
		if (!std::filesystem::exists(request, err) || running.exchange(true)) {
			return;
		}

		// The request names the menus to skip, by script.
		std::vector<std::string> skip;
		try {
			std::ifstream in(request, std::ios::binary);
			auto r = json::parse(in);
			if (r.contains("skip")) {
				skip = r["skip"].get<std::vector<std::string>>();
			}
		} catch (const std::exception& e) {
			logger::warn("Reading the request: {}; capturing every menu", e.what());
		}

		logger::info("A capture was requested");
		std::thread(Run, *folder, std::move(skip)).detach();
	}
}
