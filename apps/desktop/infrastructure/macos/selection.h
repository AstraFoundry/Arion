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

enum {
    ARION_WEB_OK = 0,
    ARION_WEB_NONE = 1, // no enclosing web area, or it has no selection
    ARION_WEB_FAILED = 2,
};

// arion_web_selection reads the selection of the web area enclosing
// element through its text markers: page text that is not a text field.
// It refuses more than max*2 UTF-16 units. *text is malloc'd UTF-8.
int arion_web_selection(void *element, long max, char **text);

// arion_web_selection_bounds is that selection's screen rectangle in points.
int arion_web_selection_bounds(void *element, double *x, double *y, double *width, double *height);

#endif
