#ifndef ARION_HOLD_H
#define ARION_HOLD_H

// Input Monitoring consent for listen-only event taps.
int arion_listen_access(void);
int arion_listen_request(void);

// arion_hold_start installs a listen-only session event tap for key-down and
// modifier changes on its own run-loop thread; 0 means it is running.
int arion_hold_start(void);
void arion_hold_stop(void);

#endif
