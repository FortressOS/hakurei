#include "wayland-client-helper.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>

#include "security-context-v1-protocol.h"
#include <wayland-client.h>

static void registry_handle_global(
    void *data,
    struct wl_registry *registry,
    uint32_t name,
    const char *interface,
    uint32_t version) {
  struct wp_security_context_manager_v1 **out = data;

  if (strcmp(interface, wp_security_context_manager_v1_interface.name) == 0)
    *out = wl_registry_bind(registry, name, &wp_security_context_manager_v1_interface, 1);
}

static void registry_handle_global_remove(
    void *data,
    struct wl_registry *registry,
    uint32_t name) {} /* no-op */

static const struct wl_registry_listener registry_listener = {
    .global = registry_handle_global,
    .global_remove = registry_handle_global_remove,
};

hakurei_wayland_res hakurei_bind_wayland_fd(
    char *socket_path,
    int fd,
    const char *app_id,
    const char *instance_id,
    int close_fd) {
  hakurei_wayland_res res = HAKUREI_WAYLAND_SUCCESS; /* see wayland.go for handling */

  struct wl_display *display = NULL;
  struct wl_registry *registry;
  struct wp_security_context_manager_v1 *security_context_manager = NULL;
  int event_cnt;
  int listen_fd = -1;
  struct sockaddr_un sockaddr = {0};
  struct wp_security_context_v1 *security_context;

  display = wl_display_connect_to_fd(fd);
  if (display == NULL) {
    res = HAKUREI_WAYLAND_CONNECT;
    goto out;
  };

  registry = wl_display_get_registry(display);
  if (wl_registry_add_listener(registry, &registry_listener, &security_context_manager) < 0) {
    res = HAKUREI_WAYLAND_LISTENER;
    goto out;
  }
  event_cnt = wl_display_roundtrip(display);
  wl_registry_destroy(registry);
  if (event_cnt < 0) {
    res = HAKUREI_WAYLAND_ROUNDTRIP;
    goto out;
  }

  if (security_context_manager == NULL) {
    res = HAKUREI_WAYLAND_NOT_AVAIL;
    goto out;
  }

  listen_fd = socket(AF_UNIX, SOCK_STREAM, 0);
  if (listen_fd < 0) {
    res = HAKUREI_WAYLAND_SOCKET;
    goto out;
  }

  sockaddr.sun_family = AF_UNIX;
  snprintf(sockaddr.sun_path, sizeof(sockaddr.sun_path), "%s", socket_path);
  if (bind(listen_fd, (struct sockaddr *)&sockaddr, sizeof(sockaddr)) != 0) {
    res = HAKUREI_WAYLAND_BIND;
    goto out;
  }

  if (listen(listen_fd, 0) != 0) {
    res = HAKUREI_WAYLAND_LISTEN;
    goto out;
  }

  security_context = wp_security_context_manager_v1_create_listener(security_context_manager, listen_fd, close_fd);
  if (security_context == NULL) { /* not reached */
    res = HAKUREI_WAYLAND_NOT_AVAIL;
    goto out;
  }
  wp_security_context_v1_set_sandbox_engine(security_context, "app.hakurei");
  wp_security_context_v1_set_app_id(security_context, app_id);
  wp_security_context_v1_set_instance_id(security_context, instance_id);
  wp_security_context_v1_commit(security_context);
  wp_security_context_v1_destroy(security_context);
  if (wl_display_roundtrip(display) < 0) {
    res = HAKUREI_WAYLAND_ROUNDTRIP;
    goto out;
  }

out:
  if (listen_fd >= 0)
    close(listen_fd);
  if (security_context_manager != NULL)
    wp_security_context_manager_v1_destroy(security_context_manager);
  if (display != NULL)
    wl_display_disconnect(display);

  free((void *)socket_path);
  free((void *)app_id);
  free((void *)instance_id);
  return res;
}
