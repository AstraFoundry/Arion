#define MINIAUDIO_IMPLEMENTATION
#include "../reset_capture.h"
#undef NDEBUG
#include <assert.h>

static void check_rate(ma_uint32 source_rate) {
    ma_device device;
    memset(&device, 0, sizeof(device));
    device.type = ma_device_type_capture;
    ma_atomic_device_state_set(&device.state, ma_device_state_stopped);
    ma_data_converter_config config = ma_data_converter_config_init(
        ma_format_f32, ma_format_f32, 1, 1, source_rate, 16000);
    config.resampling.algorithm = ma_resample_algorithm_linear;
    assert(ma_data_converter_init(&config, NULL, &device.capture.converter) == MA_SUCCESS);

    float previous[1024], converted[2048];
    for (int i = 0; i < 1024; ++i) previous[i] = 1.0f;
    ma_uint64 input_count = 1024, output_count = 2048;
    assert(ma_data_converter_process_pcm_frames(&device.capture.converter,
        previous, &input_count, converted, &output_count) == MA_SUCCESS);

    float buffered[320];
    for (int i = 0; i < 320; ++i) buffered[i] = 0.5f;
    device.capture.format = ma_format_f32;
    device.capture.channels = 1;
    device.capture.pIntermediaryBuffer = buffered;
    device.capture.intermediaryBufferCap = 320;
    device.capture.intermediaryBufferLen = 17;

    ma_atomic_device_state_set(&device.state, ma_device_state_started);
    assert(malgo_reset_capture(&device) == MA_INVALID_OPERATION);
    assert(device.capture.intermediaryBufferLen == 17 && buffered[0] == 0.5f);
    ma_atomic_device_state_set(&device.state, ma_device_state_stopped);
    assert(malgo_reset_capture(&device) == MA_SUCCESS);
    assert(device.capture.intermediaryBufferLen == 0);
    for (int i = 0; i < 320; ++i) assert(buffered[i] == 0.0f);

    // A new silent session must not inherit even a filter tail from the old one.
    float silence[1024] = {0};
    input_count = 1024; output_count = 2048;
    assert(ma_data_converter_process_pcm_frames(&device.capture.converter,
        silence, &input_count, converted, &output_count) == MA_SUCCESS);
    assert(output_count > 0);
    for (ma_uint64 i = 0; i < output_count; ++i) assert(converted[i] == 0.0f);
    assert(malgo_reset_capture(&device) == MA_SUCCESS);
    ma_data_converter_uninit(&device.capture.converter, NULL);
}

int main(void) {
    assert(malgo_reset_capture(NULL) == MA_INVALID_ARGS);
    check_rate(16000);
    check_rate(44100);
    check_rate(48000);
    check_rate(96000);
    return 0;
}
