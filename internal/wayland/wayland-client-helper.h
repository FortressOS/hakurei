#include <stdint.h>

int32_t hakurei_bind_wayland_fd(
    char *socket_path,
    int fd,
    const char *app_id,
    const char *instance_id,
    int sync_fd);
