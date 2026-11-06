/* Arion capture-reset-v1. The caller owns a stopped capture-only device. */
#ifndef MALGO_RESET_CAPTURE_H
#define MALGO_RESET_CAPTURE_H
#include "miniaudio.h"
#include <string.h>
#if defined(MA_SUPPORT_COREAUDIO)
#include <AudioToolbox/AudioToolbox.h>
#endif

static ma_result malgo_reset_capture(ma_device* device) {
    if (device == NULL) return MA_INVALID_ARGS;
    if (device->type != ma_device_type_capture || ma_device_get_state(device) != ma_device_state_stopped) {
        return MA_INVALID_OPERATION;
    }
#if defined(MA_SUPPORT_COREAUDIO)
    if (device->pContext != NULL && device->pContext->backend == ma_backend_coreaudio && device->coreaudio.audioUnitCapture != NULL) {
        if (AudioUnitReset((AudioUnit)device->coreaudio.audioUnitCapture, kAudioUnitScope_Global, 0) != noErr) {
            return MA_ERROR;
        }
    }
#endif
    ma_result result = ma_data_converter_reset(&device->capture.converter);
    if (result != MA_SUCCESS) return result;
    if (device->capture.pIntermediaryBuffer != NULL) {
        memset(device->capture.pIntermediaryBuffer, 0,
            device->capture.intermediaryBufferCap * ma_get_bytes_per_frame(device->capture.format, device->capture.channels));
    }
    device->capture.intermediaryBufferLen = 0;
#if defined(MA_SUPPORT_COREAUDIO)
    if (device->pContext != NULL && device->pContext->backend == ma_backend_coreaudio) {
        AudioBufferList* buffers = (AudioBufferList*)device->coreaudio.pAudioBufferList;
        if (buffers != NULL) {
            for (UInt32 i = 0; i < buffers->mNumberBuffers; ++i) {
                if (buffers->mBuffers[i].mData != NULL) {
                    memset(buffers->mBuffers[i].mData, 0,
                        device->coreaudio.audioBufferCapInFrames * ma_get_bytes_per_frame(
                            device->capture.internalFormat, buffers->mBuffers[i].mNumberChannels));
                }
            }
        }
    }
#endif
    return MA_SUCCESS;
}
#endif
