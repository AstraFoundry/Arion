#ifndef ARION_HOLD_H
#define ARION_HOLD_H

// Input Monitoring consent for listen-only event taps.
int arion_listen_access(void);
int arion_listen_request(void);

enum { ARION_HOLD_PERMISSION_DENIED = -1, ARION_HOLD_UNAVAILABLE = -2 };

// arion_hold_start installs a listen-only session event tap for key-down and
// modifier changes on its own run-loop thread. System-defined events, which
// carry volume and media keys, are not in the mask: the tap neither delays
// nor consumes them. 0 means it is running; negative values distinguish
// consent from adapter failure.
int arion_hold_start(void);
void arion_hold_stop(void);

#endif
