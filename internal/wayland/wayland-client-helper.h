#include <stdbool.h>
#include <sys/un.h>

typedef enum {
    HAKUREI_WAYLAND_SUCCESS,
    /* wl_display_connect_to_fd failed, errno */
    HAKUREI_WAYLAND_CONNECT,
    /* wl_registry_add_listener failed, errno */
    HAKUREI_WAYLAND_LISTENER,
    /* wl_display_roundtrip failed, errno */
    HAKUREI_WAYLAND_ROUNDTRIP,
    /* compositor does not implement wp_security_context_v1 */
    HAKUREI_WAYLAND_NOT_AVAIL,
    /* socket failed, errno */
    HAKUREI_WAYLAND_SOCKET,
    /* bind failed, errno */
    HAKUREI_WAYLAND_BIND,
    /* listen failed, errno */
    HAKUREI_WAYLAND_LISTEN,

    /* ensure pathname failed, implemented in conn.go */
    HAKUREI_WAYLAND_CREAT,
    /* socket for host server failed, implemented in conn.go */
    HAKUREI_WAYLAND_HOST_SOCKET,
    /* connect for host server failed, implemented in conn.go */
    HAKUREI_WAYLAND_HOST_CONNECT,
    /* cleanup failed, implemented in conn.go */
    HAKUREI_WAYLAND_CLEANUP,
} hakurei_wayland_res;

hakurei_wayland_res hakurei_security_context_bind(
    char *socket_path,
    int server_fd,
    const char *app_id,
    const char *instance_id,
    int close_fd);

/* returns whether the specified size fits in the sun_path field of sockaddr_un */
static inline bool hakurei_is_valid_size_sun_path(size_t sz) {
    struct sockaddr_un sockaddr;
    return sz <= sizeof(sockaddr.sun_path);
};
