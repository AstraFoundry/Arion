#include <stdint.h>

typedef struct {
    int on_active_space;
    int on_screen;
    int above_fixture;
    int overlaps_fixture;
    int frontmost_pid;
    unsigned long behavior;
} arion_overlay_observation;

void arion_observe_overlay(uintptr_t handle, int fixture_pid, arion_overlay_observation *out);
