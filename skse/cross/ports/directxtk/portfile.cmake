# A stand-in for vcpkg's directxtk, for the compile check only: the same
# source, but just its headers and SimpleMath, the one part CommonLibSSE NG
# uses. The real package compiles shaders with the Windows SDK's fxc.exe,
# which xwin doesn't download. REF and SHA512 are the real package's at
# the baseline in vcpkg-configuration.json; update them together.
vcpkg_from_github(
    OUT_SOURCE_PATH SOURCE_PATH
    REPO Microsoft/DirectXTK
    REF may2026
    SHA512 9306774b06f52b4c37938fe4b3a10df8c7a85652188a25dc25e60ae9ff6fbf9d2cf920b114de3fc3945c564054cbb166cc45fca073021129e75ce282a51636e7
    HEAD_REF main
)
vcpkg_cmake_configure(SOURCE_PATH "${CMAKE_CURRENT_LIST_DIR}/stub" OPTIONS "-DDXTK_SOURCE=${SOURCE_PATH}")
vcpkg_cmake_install()
vcpkg_cmake_config_fixup(CONFIG_PATH share/directxtk)
file(REMOVE_RECURSE "${CURRENT_PACKAGES_DIR}/debug/include" "${CURRENT_PACKAGES_DIR}/debug/share")
file(INSTALL "${SOURCE_PATH}/LICENSE" DESTINATION "${CURRENT_PACKAGES_DIR}/share/${PORT}" RENAME copyright)
