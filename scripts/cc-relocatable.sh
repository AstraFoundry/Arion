#!/bin/sh
# Compiles and links like cc, minus -Wl,-rpath flags that name an absolute
# directory. The sherpa-onnx module links with an rpath into the Go module
# cache of the build machine; dropping it at link time lets `mygo build`
# sign, archive and sign for updates the very binary that ships, which then
# finds its libraries only through @executable_path/../Resources/lib.
for arg; do
	shift
	case $arg in
	-Wl,-rpath,/*) ;;
	*) set -- "$@" "$arg" ;;
	esac
done
exec cc "$@"
