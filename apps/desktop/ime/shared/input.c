#include "input.h"
#include <string.h>

int arion_shift_transition(ArionShiftTap *state, int shift_key, int down,
    unsigned int side, int other_modifiers, int unchanged_flags, double time) {
    if (down && state->held && unchanged_flags && !other_modifiers && side == state->side) return 0;
    int toggle = shift_key && !down && state->held && state->pending &&
        (!side || !state->side || side == state->side) && !other_modifiers &&
        time >= state->pressed_at && time - state->pressed_at < 0.3;
    if (shift_key && down && !state->held && !other_modifiers) {
        state->pending = 1;
        state->side = side;
        state->pressed_at = time;
    } else state->pending = 0;
    state->held = down;
    return toggle;
}
void arion_shift_interrupt(ArionShiftTap *state) { state->pending = 0; }
void arion_shift_reset(ArionShiftTap *state) { memset(state, 0, sizeof(*state)); }
ArionKeyAction arion_key_action(int key, int modifiers, int ascii) {
    const int shortcuts = 4 | 8 | (1 << 26);
    if (key == ' ' && (modifiers & 1) && !(modifiers & shortcuts)) return ARION_KEY_TOGGLE_ASCII;
    if (!ascii && key == '.' && (modifiers & (shortcuts | 1)) == 4) return ARION_KEY_TOGGLE_PUNCTUATION;
    if (key < 0 || (modifiers & shortcuts)) return ARION_KEY_PASS;
    return ARION_KEY_PROCESS;
}
int arion_key_normalize(int key, int modifiers, int ascii) {
    if (ascii || (modifiers & (4 | 8 | (1 << 26)))) return key;
    if (key == 0x00fc) return 'v';
    if (key == 0x00dc) return 'V';
    return key;
}
