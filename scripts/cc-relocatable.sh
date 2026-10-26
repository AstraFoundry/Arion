#!/bin/sh
# Use the selected Xcode toolchain and its macOS SDK together. A bare cc can
# otherwise use a newer Command Line Tools SDK that its linker cannot read.
# Drop absolute build-machine rpaths so packaged binaries remain relocatable.
for arg; do
	shift
	case $arg in
	-Wl,-rpath,/*) ;;
	*) set -- "$@" "$arg" ;;
	esac
done
exec xcrun --sdk macosx clang "$@"
