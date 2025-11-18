#include <stdbool.h>
#include <sys/un.h>

typedef enum {
    HAKUREI_PIPEWIRE_SUCCESS,
    /* pw_main_loop_new failed, errno */
    HAKUREI_PIPEWIRE_MAINLOOP,
    /* pw_context_new failed, errno */
    HAKUREI_PIPEWIRE_CTX,
    /* pw_context_connect failed, errno */
    HAKUREI_PIPEWIRE_CONNECT,
    /* pw_core_get_registry failed */
    HAKUREI_PIPEWIRE_REGISTRY,
    /* no security context object found */
    HAKUREI_PIPEWIRE_NOT_AVAIL,
    /* socket failed, errno */
    HAKUREI_PIPEWIRE_SOCKET,
    /* bind failed, errno */
    HAKUREI_PIPEWIRE_BIND,
    /* listen failed, errno */
    HAKUREI_PIPEWIRE_LISTEN,
    /* pw_security_context_create failed, translated errno */
    HAKUREI_PIPEWIRE_ATTACH,

    /* ensure pathname failed, implemented in conn.go */
    HAKUREI_PIPEWIRE_CREAT,
    /* cleanup failed, implemented in conn.go */
    HAKUREI_PIPEWIRE_CLEANUP,
} hakurei_pipewire_res;

hakurei_pipewire_res hakurei_pw_security_context_bind(
    char *socket_path,
    char *remote_path,
    int close_fd);

/* returns whether the specified size fits in the sun_path field of sockaddr_un */
static inline bool hakurei_pw_is_valid_size_sun_path(size_t sz) {
    struct sockaddr_un sockaddr;
    return sz <= sizeof(sockaddr.sun_path);
};
