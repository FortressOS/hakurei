package wl

//go:generate sh -c "wayland-scanner client-header `pkg-config --variable=datarootdir wayland-protocols`/wayland-protocols/staging/security-context/security-context-v1.xml security-context-v1-protocol.h"
//go:generate sh -c "wayland-scanner private-code `pkg-config --variable=datarootdir wayland-protocols`/wayland-protocols/staging/security-context/security-context-v1.xml security-context-v1-protocol.c"

/*
#cgo linux pkg-config: wayland-client
#cgo freebsd openbsd LDFLAGS: -lwayland-client

#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>

#include <unistd.h>
#include <sys/socket.h>
#include <sys/un.h>

#include <wayland-client.h>
#include "security-context-v1-protocol.h"

static void registry_handle_global(void *data, struct wl_registry *registry, uint32_t name, const char *interface, uint32_t version) {
  struct wp_security_context_manager_v1 **out = data;

  if (strcmp(interface, wp_security_context_manager_v1_interface.name) == 0)
      *out = wl_registry_bind(registry, name, &wp_security_context_manager_v1_interface, 1);
}

static void registry_handle_global_remove(void *data, struct wl_registry *registry, uint32_t name) { } // no-op

static const struct wl_registry_listener registry_listener = {
  .global = registry_handle_global,
  .global_remove = registry_handle_global_remove,
};

static int32_t bind_wayland_fd(char *socket_path, int fd, const char *app_id, const char *instance_id, int sync_fd) {
  int32_t res = 0; // refer to resErr for meaning

  struct wl_display *display;
  display = wl_display_connect_to_fd(fd);
  if (!display) {
    res = 1;
    goto out;
  };

  struct wl_registry *registry;
  registry = wl_display_get_registry(display);

  struct wp_security_context_manager_v1 *security_context_manager = NULL;
  wl_registry_add_listener(registry, &registry_listener, &security_context_manager);
  int ret;
  ret = wl_display_roundtrip(display);
  wl_registry_destroy(registry);
  if (ret < 0)
    goto out;

  if (!security_context_manager) {
    res = 2;
    goto out;
  }

  int listen_fd = -1;
  listen_fd = socket(AF_UNIX, SOCK_STREAM, 0);
  if (listen_fd < 0)
    goto out;

  struct sockaddr_un sockaddr = {0};
  sockaddr.sun_family = AF_UNIX;
  snprintf(sockaddr.sun_path, sizeof(sockaddr.sun_path), "%s", socket_path);
  if (bind(listen_fd, (struct sockaddr *)&sockaddr, sizeof(sockaddr)) != 0)
    goto out;

  if (listen(listen_fd, 0) != 0)
    goto out;

  struct wp_security_context_v1 *security_context;
  security_context = wp_security_context_manager_v1_create_listener(security_context_manager, listen_fd, sync_fd);
  wp_security_context_v1_set_sandbox_engine(security_context, "moe.ophivana.fortify");
  wp_security_context_v1_set_app_id(security_context, app_id);
  wp_security_context_v1_set_instance_id(security_context, instance_id);
  wp_security_context_v1_commit(security_context);
  wp_security_context_v1_destroy(security_context);
  if (wl_display_roundtrip(display) < 0)
    goto out;

out:
  if (listen_fd >= 0)
    close(listen_fd);
  if (security_context_manager)
    wp_security_context_manager_v1_destroy(security_context_manager);
  if (display)
    wl_display_disconnect(display);

  free((void *)socket_path);
  free((void *)app_id);
  free((void *)instance_id);
  return res;
}
*/
import "C"
import "errors"

var resErr = [...]error{
	0: nil,
	1: errors.New("wl_display_connect_to_fd() failed"),
	2: errors.New("wp_security_context_v1 not available"),
}

func bindWaylandFd(socketPath string, fd uintptr, appID, instanceID string, syncFD uintptr) error {
	res := C.bind_wayland_fd(C.CString(socketPath), C.int(fd), C.CString(appID), C.CString(instanceID), C.int(syncFD))
	return resErr[int32(res)]
}
