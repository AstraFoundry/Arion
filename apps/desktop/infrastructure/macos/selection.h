#ifndef ARION_SELECTION_H
#define ARION_SELECTION_H

// The selection watch is a listen-only mouse-up tap. It is not the
// dictation hold-key tap.
enum {
    ARION_SELECTION_OK = 0,
    ARION_SELECTION_DENIED = 1,
    ARION_SELECTION_UNAVAILABLE = 2,
};

int arion_selection_watch_start(void);
void arion_selection_watch_stop(void);

#endif
