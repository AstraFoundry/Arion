#ifndef ARION_NATIVE_H
#define ARION_NATIVE_H

enum {
    ARION_AUTH_UNDETERMINED = 0,
    ARION_AUTH_RESTRICTED = 1,
    ARION_AUTH_DENIED = 2,
    ARION_AUTH_GRANTED = 3,
};

enum {
    ARION_TARGET_OK = 0,
    ARION_TARGET_UNTRUSTED = 1,
    ARION_TARGET_NONE = 2,
};

typedef struct {
    void *element;
    int pid;
    int secure_input;
    int selected_text_settable;
    int value_readable;
    char role[64];
    char subrole[64];
    char bundle[256];
    char bundle_path[1024];
} arion_target;

int arion_mic_status(void);
void arion_mic_request(void);
int arion_ax_trusted(int prompt);
void arion_open_url(const char *url);
int arion_capture_target(arion_target *out);
void arion_release(void *element);
int arion_focus_matches(void *element, int pid);
int arion_secure_input(void);
long arion_value_length(void *element);
int arion_ax_insert(void *element, const char *utf8);
int arion_type_unicode(const char *utf8);
long arion_paste(const char *utf8);
int arion_restore_clipboard(long owned_change);
long arion_clipboard_change_count(void);
int arion_ime_register(const char *bundle_path);
int arion_ime_enable(const char *source_id);
int arion_ime_disable(const char *source_id);
int arion_ime_state(const char *source_id);

#endif
