cmake_minimum_required(VERSION 3.20)
project(ArionTSF LANGUAGES NONE)

# Keep one compiler/SDK/layout/manifest pipeline. CMake is only a launcher.
if(NOT CMAKE_HOST_SYSTEM_NAME STREQUAL "Windows")
  message(FATAL_ERROR "The canonical Windows input-method build requires native Windows.")
endif()
foreach(parameter ARION_OUTPUT ARION_KEYBOARD_SDK ARION_KEYBOARD_MANIFEST_SHA256)
  if(DEFINED ${parameter})
    message(FATAL_ERROR "${parameter} is no longer supported. Use node scripts/build-windows-input-method.mjs with its fixed output, pinned SDK and computed manifest binding.")
  endif()
endforeach()
get_filename_component(ARION_REPOSITORY_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/../../../.." ABSOLUTE)
find_program(ARION_NODE_EXECUTABLE NAMES node REQUIRED)
add_custom_target(ArionTSF ALL
  COMMAND "${ARION_NODE_EXECUTABLE}" "${ARION_REPOSITORY_ROOT}/scripts/build-windows-input-method.mjs"
  WORKING_DIRECTORY "${ARION_REPOSITORY_ROOT}"
  COMMENT "Building the Windows input method through its canonical Node pipeline"
  USES_TERMINAL
  VERBATIM)
