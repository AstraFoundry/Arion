#ifndef ARION_GUIDE_H
#define ARION_GUIDE_H

#include <stddef.h>

typedef struct {
    int running;
    int visible;
    int frontmost;
    double x, y, width, height;
} arion_settings_window;

void arion_find_settings_window(arion_settings_window *out);
int arion_app_bundle(char *out, size_t size);

#endif
