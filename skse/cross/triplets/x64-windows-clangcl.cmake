# vcpkg's packages for the compile check, built with cross/toolchain.cmake:
# static libraries on the DLL C runtime, as the release build's are.
set(VCPKG_TARGET_ARCHITECTURE x64)
set(VCPKG_CRT_LINKAGE dynamic)
set(VCPKG_LIBRARY_LINKAGE static)
# VCPKG_CMAKE_SYSTEM_NAME stays unset: setting it to Windows would unset
# VCPKG_TARGET_IS_WINDOWS. The toolchain does the targeting.
set(VCPKG_CHAINLOAD_TOOLCHAIN_FILE "${CMAKE_CURRENT_LIST_DIR}/../toolchain.cmake")
