# MCMLexicon.dll, the SKSE plugin that captures what each MCM shows in a
# running game (PLAN.md §11). One DLL for Skyrim SE, AE, and VR, built on
# CommonLibSSE-NG, which is GPL-3.0-or-later: the DLL as distributed is too
# (skse/LICENSE.md). Builds on Windows with MSVC and vcpkg, and the Release
# workflow builds it; the cross preset checks it compiles on macOS or Linux
# (cross/). Nothing else in the repository needs it.
cmake_minimum_required(VERSION 3.25)

# A release passes its version without the v and any pre-release part
# (0.2.0 for v0.2.0-beta.1), since a DLL's version is numbers only, and
# the whole tag as LEXICON_VERSION_TEXT, which the capture file records.
set(LEXICON_VERSION "0.0.0" CACHE STRING "The plugin's version, numbers only")
set(LEXICON_VERSION_TEXT "dev" CACHE STRING "The release tag the capture file names")

project(MCMLexicon VERSION ${LEXICON_VERSION} LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 23)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# CommonLibSSE-NG at a release tag: in CI, its CMake links that tag's
# prebuilt library rather than compiling it (about 16 minutes).
include(FetchContent)
set(BUILD_TESTS OFF CACHE BOOL "" FORCE)
FetchContent_Declare(
	CommonLibSSE
	GIT_REPOSITORY https://github.com/alandtse/CommonLibSSE-NG.git
	GIT_TAG v16.0.1
	GIT_SHALLOW TRUE
)
FetchContent_MakeAvailable(CommonLibSSE)
# Its CMake defines the plugin helper only when it links the prebuilt
# library; a source build needs it included here.
if(NOT COMMAND add_commonlibsse_plugin)
	include("${commonlibsse_SOURCE_DIR}/cmake/CommonLibSSE.cmake")
endif()

find_package(nlohmann_json CONFIG REQUIRED)

add_commonlibsse_plugin(
	${PROJECT_NAME}
	AUTHOR "Davey Strus"
	USE_ADDRESS_LIBRARY
	SOURCES
		src/capture.cpp
		src/capture.h
		src/main.cpp
)
target_compile_definitions(${PROJECT_NAME} PRIVATE LEXICON_VERSION_TEXT="${LEXICON_VERSION_TEXT}" NOMINMAX WIN32_LEAN_AND_MEAN)
target_link_libraries(${PROJECT_NAME} PRIVATE nlohmann_json::nlohmann_json)
target_precompile_headers(${PROJECT_NAME} PRIVATE src/PCH.h)
