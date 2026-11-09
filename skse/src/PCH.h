#pragma once

#include <RE/Skyrim.h>
#include <SKSE/SKSE.h>

// Windows's headers come in partway through CommonLib's, and their macros
// rename Variable::GetObject (to GetObjectA) wherever it's called after
// them. NOMINMAX (CMakeLists.txt) keeps them from defining min and max.
#undef GetObject

#include <nlohmann/json.hpp>

#include <algorithm>
#include <atomic>
#include <cctype>
#include <chrono>
#include <filesystem>
#include <format>
#include <fstream>
#include <functional>
#include <future>
#include <memory>
#include <optional>
#include <string>
#include <thread>
#include <vector>

namespace logger = SKSE::log;
using namespace std::literals;
