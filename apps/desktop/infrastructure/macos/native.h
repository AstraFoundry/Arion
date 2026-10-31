#ifndef ARION_NATIVE_H
#define ARION_NATIVE_H
int arion_ime_maintenance_version(const char *path);
char *arion_ime_removal_observation(void);
char *arion_ime_current_id(void);
int arion_ime_exchange(const char *first, const char *second, int existing);
int arion_ime_swap_supported(const char *path);

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
    void *selection;
    int pid;
    int secure_input;
    int value_readable;
    int editable;
    // Edit signals are facts. focusclass.DecideMac decides insertion.
    int value_settable;
    int selected_text_settable;
    int selected_range_settable;
    int editable_ancestor;
    int no_input;
    char role[64];
    char subrole[64];
    char bundle[256];
    char site[256];
    char app_name[256];
} arion_target;

int arion_mic_status(void);
void arion_mic_request(void);
int arion_ax_trusted(int prompt);
void arion_open_url(const char *url);
int arion_capture_target(arion_target *out, int read_site);
int arion_frontmost_pid(void);
int arion_peek_target(int pid, arion_target *out, int read_site);
void *arion_retain(void *element);
void arion_release(void *element);
int arion_focus_matches(void *element, int pid);
int arion_secure_input(void);
enum {
    ARION_PASTE_OK = 0,
    ARION_PASTE_FAILED = 1,
    ARION_PASTE_CHANGED = 2,
    ARION_PASTE_TARGET_CHANGED = 3,
    ARION_PASTE_SECURE = 4,
};
void *arion_paste_begin(const char *utf8);
int arion_paste_stage(void *transaction);
int arion_paste_post(void *transaction, void *element, void *selection, int pid);
int arion_paste_finish(void *transaction);
int arion_selection_matches(void *element, void *selection);
int arion_clipboard_publish(const char *utf8);
long long arion_clipboard_revision(void);
int arion_clipboard_excluded(void);
int arion_history_activate(int pid);
int arion_history_post(int pid, void *element, long long revision);
void *arion_history_focus(int pid, int *secure, int *no_input);
int arion_history_check(int pid, void *element);
// arion_enable_web_accessibility asks Chromium and Electron to publish the
// web accessibility tree. Native apps ignore the attributes.
void arion_enable_web_accessibility(int pid);
int arion_history_anchor(void *element, double *x, double *y, double *width, double *height);
int arion_ime_register(const char *bundle_path);
int arion_ime_validate_payload(const char *bundle_path, const char *bundle_id);
int arion_ime_bundle_matches(const char *bundle_path, const char *source_id);
int arion_ime_enable(const char *source_id);
int arion_ime_state(const char *source_id);
typedef struct {
    int mode_flags;
    int parent_flags;
    int selected_kind;
} arion_ime_snapshot;
// Flags: found=1, enabled=2, selected=4, selectable=8, matching ID=16.
// Selected kind: unavailable=0, Arion mode=1, Arion parent=2, other=3.
void arion_ime_inspect(const char *source_id, const char *parent_id, arion_ime_snapshot *out);
int arion_ime_open_settings(void);
// Select/switch without disabling an already authorized input source.
int arion_ime_select(const char *source_id);
void *arion_ime_watch_create(void);
void arion_ime_watch_cancel(void *watch);
void arion_ime_watch_wait(void *watch, int milliseconds);
void arion_ime_watch_close(void *watch);
int arion_ime_watch_select(void *watch, const char *source_id);
int arion_ime_switch_away(const char *source_id);
// action: 0 counts running hosts, 1 requests exit, 2 forces exit.
// A negative result reports a failed forced termination request.
int arion_ime_processes(const char *bundle_id, int action);

#endif
