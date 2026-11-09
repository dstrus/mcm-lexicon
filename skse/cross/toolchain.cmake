# clang-cl and lld-link targeting 64-bit Windows and the MSVC ABI, against
# a Windows SDK and C runtime that xwin downloaded, for checking that the
# plugin compiles on macOS or Linux. Releases are built with MSVC (DEVELOPING.md).
set(CMAKE_SYSTEM_NAME Windows)
set(CMAKE_SYSTEM_PROCESSOR AMD64)

# Homebrew keeps LLVM and lld out of PATH, so look where it puts them too.
set(_llvm_hints
	/opt/homebrew/opt/llvm/bin /opt/homebrew/opt/lld/bin
	/usr/local/opt/llvm/bin /usr/local/opt/lld/bin)
find_program(CMAKE_C_COMPILER clang-cl HINTS ${_llvm_hints} REQUIRED)
find_program(CMAKE_CXX_COMPILER clang-cl HINTS ${_llvm_hints} REQUIRED)
find_program(CMAKE_LINKER lld-link HINTS ${_llvm_hints} REQUIRED)
# Every link runs a resource compiler and a manifest tool, CMake's own
# compiler checks included; without these it looks for MSVC's rc and mt.
find_program(CMAKE_RC_COMPILER llvm-rc HINTS ${_llvm_hints})
find_program(CMAKE_MT llvm-mt HINTS ${_llvm_hints})

# The sysroot from xwin splat --use-winsysroot-style
# --preserve-ms-arch-notation: XWIN_SYSROOT, else ~/xwin-out.
if(DEFINED XWIN_SYSROOT)
	set(_sysroot "${XWIN_SYSROOT}")
elseif(DEFINED ENV{XWIN_SYSROOT})
	set(_sysroot "$ENV{XWIN_SYSROOT}")
else()
	set(_sysroot "$ENV{HOME}/xwin-out")
endif()

# vcpkg loads this file several times per configure, once for each package
# it builds too, so the flags are added only once. /EHsc is here rather
# than in the project because each package's build sees only this file,
# and clang-cl turns exceptions off without it.
if(NOT CMAKE_CXX_FLAGS MATCHES "winsysroot")
	set(_target "--target=x86_64-pc-windows-msvc /winsysroot${_sysroot}")
	set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} ${_target}" CACHE STRING "" FORCE)
	set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} ${_target} /EHsc" CACHE STRING "" FORCE)
	set(CMAKE_RC_FLAGS "${CMAKE_RC_FLAGS} ${_target}" CACHE STRING "" FORCE)
	set(CMAKE_EXE_LINKER_FLAGS "${CMAKE_EXE_LINKER_FLAGS} /winsysroot:${_sysroot}" CACHE STRING "" FORCE)
	set(CMAKE_SHARED_LINKER_FLAGS "${CMAKE_SHARED_LINKER_FLAGS} /winsysroot:${_sysroot}" CACHE STRING "" FORCE)
endif()
