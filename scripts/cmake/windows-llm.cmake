# Apply MinGW's static compiler runtime policy to executable targets outside
# ggml's CMake directory too. MSVC receives /MT through the staging flags.
if(MINGW)
  add_link_options(-static -static-libgcc -static-libstdc++)
endif()
