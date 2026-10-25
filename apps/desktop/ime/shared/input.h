#ifndef ARION_SHARED_INPUT_H
#define ARION_SHARED_INPUT_H
#ifdef __cplusplus
extern "C" {
#endif

typedef struct ArionShiftTap {
    int held;
    int pending;
    unsigned int side;
    double pressed_at;
} ArionShiftTap;

/* Platforms normalize left/right Shift to nonzero side IDs, or zero when
 * unavailable. No key conversion or engine mutation occurs in these rules. */
int arion_shift_transition(ArionShiftTap *state, int shift_key, int down,
    unsigned int side, int other_modifiers, int unchanged_flags, double time);
void arion_shift_interrupt(ArionShiftTap *state);
void arion_shift_reset(ArionShiftTap *state);

typedef enum ArionKeyAction {
    ARION_KEY_PASS,
    ARION_KEY_PROCESS,
    ARION_KEY_TOGGLE_ASCII,
    ARION_KEY_TOGGLE_PUNCTUATION
} ArionKeyAction;
/* modifiers use Rime/X11 Shift=1, Control=4, Alt=8, Super=1<<26. */
ArionKeyAction arion_key_action(int key, int modifiers, int ascii);
/* Map direct umlaut spelling only for Chinese input; host events stay intact. */
int arion_key_normalize(int key, int modifiers, int ascii);

#ifdef __cplusplus
}
#endif
#endif
