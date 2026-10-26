#pragma once
#include <msctf.h>

// MinGW omits this desktop TSF interface. Match the Windows SDK declaration:
// microsoft/win32metadata, generation/WinSDK/RecompiledIdlHeaders/um/msctf.h.
#ifndef __ITfTextLayoutSink_INTERFACE_DEFINED__
#define __ITfTextLayoutSink_INTERFACE_DEFINED__
enum TfLayoutCode { TF_LC_CREATE=0, TF_LC_CHANGE=1, TF_LC_DESTROY=2 };
DEFINE_GUID(IID_ITfTextLayoutSink,0x2af2d06a,0xdd5b,0x4927,0xa0,0xb4,0x54,0xf1,0x9c,0x91,0xfa,0xde);
struct ITfTextLayoutSink : IUnknown {
    virtual HRESULT STDMETHODCALLTYPE OnLayoutChange(ITfContext*,TfLayoutCode,ITfContextView*)=0;
};
#endif
