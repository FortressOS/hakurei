#include "pipewire-helper.h"
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/socket.h>

#include <spa/utils/result.h>
#include <spa/utils/string.h>
#include <spa/utils/ansi.h>
#include <spa/debug/pod.h>
#include <spa/debug/format.h>
#include <spa/debug/types.h>
#include <spa/debug/file.h>

#include <pipewire/pipewire.h>
#include <pipewire/extensions/security-context.h>

/* contains most of the state used by hakurei_pw_security_context_bind,
 * not ideal, but it is too painful to separate state with the abysmal
 * API of pipewire */
struct hakurei_pw_security_context_state {
    struct pw_main_loop *loop;
    struct pw_context *context;

    struct pw_core *core;
    struct spa_hook core_listener;

    struct pw_registry *registry;
    struct spa_hook registry_listener;

    struct pw_properties *props;

    struct pw_security_context *sec;

    int pending_create;
    int create_result;
    int pending;
    int done;
};

/* for field global of registry_events */
static void registry_event_global(
    void *data, uint32_t id,
    uint32_t permissions, const char *type, uint32_t version,
    const struct spa_dict *props) {
    struct hakurei_pw_security_context_state *state = data;

    if (spa_streq(type, PW_TYPE_INTERFACE_SecurityContext))
        state->sec = pw_registry_bind(state->registry, id, type, version, 0);
}

/* for field global_remove of registry_events */
static void registry_event_global_remove(void *data, uint32_t id) {} /* no-op */

static const struct pw_registry_events registry_events = {
    PW_VERSION_REGISTRY_EVENTS,
    .global = registry_event_global,
    .global_remove = registry_event_global_remove,
};

/* for field error of core_events */
static void on_core_error(void *data, uint32_t id, int seq, int res, const char *message) {
    struct hakurei_pw_security_context_state *state = data;

    pw_log_error("error id:%u seq:%d res:%d (%s): %s",
                 id, seq, res, spa_strerror(res), message);

    if (seq == SPA_RESULT_ASYNC_SEQ(state->pending_create))
        state->create_result = res;

    if (id == PW_ID_CORE && res == -EPIPE) {
        state->done = true;
        pw_main_loop_quit(state->loop);
    }
}

static const struct pw_core_events core_events = {
    PW_VERSION_CORE_EVENTS,
    .error = on_core_error,
};

/* for field done of stack allocated core_events in roundtrip */
static void core_event_done(void *data, uint32_t id, int seq) {
    struct hakurei_pw_security_context_state *state = data;
    if (id == PW_ID_CORE && seq == state->pending) {
        state->done = true;
        pw_main_loop_quit(state->loop);
    }
}

static void roundtrip(struct hakurei_pw_security_context_state *state) {
    struct spa_hook core_listener;
    static const struct pw_core_events core_events = {
        PW_VERSION_CORE_EVENTS,
        .done = core_event_done,
    };
    spa_zero(core_listener);
    pw_core_add_listener(state->core, &core_listener, &core_events, state);

    state->done = false;
    state->pending = pw_core_sync(state->core, PW_ID_CORE, 0);

    while (!state->done)
        pw_main_loop_run(state->loop);

    spa_hook_remove(&core_listener);
}

hakurei_pipewire_res hakurei_pw_security_context_bind(
    char *socket_path,
    char *remote_path,
    int close_fd) {
    hakurei_pipewire_res res = HAKUREI_PIPEWIRE_SUCCESS; /* see pipewire.go for handling */

    struct hakurei_pw_security_context_state state = {0};
    struct pw_loop *l;
    struct spa_error_location loc;
    int listen_fd;
    struct sockaddr_un sockaddr = {0};

    /* stack allocated because pw_deinit is always called before returning,
     * in the implementation it actually does nothing with these addresses
     * and I have no idea why it would even need these, still it is safe to
     * do this to not risk a future version of pipewire clobbering strings */
    int fake_argc = 1;
    char *fake_argv[] = {"hakurei", NULL};
    /* this makes multiple getenv calls, caller must ensure to NOT setenv
     * before this function returns */
    pw_init(&fake_argc, (char ***)&fake_argv);

    /* as far as I can tell, setting engine to "org.flatpak" gets special
     * treatment, and should never be used here because the .flatpak-info
     * hack is vulnerable to a confused deputy attack */
    state.props = pw_properties_new(
        PW_KEY_SEC_ENGINE, "app.hakurei",
        PW_KEY_ACCESS, "restricted",
        NULL);

    /* this is unfortunately required to do ANYTHING with pipewire */
    state.loop = pw_main_loop_new(NULL);
    if (state.loop == NULL) {
        res = HAKUREI_PIPEWIRE_MAINLOOP;
        goto out;
    }
    l = pw_main_loop_get_loop(state.loop);

    /* boilerplate from src/tools/pw-container.c */
    state.context = pw_context_new(l, NULL, 0);
    if (state.context == NULL) {
        res = HAKUREI_PIPEWIRE_CTX;
        goto out;
    }

    /* boilerplate from src/tools/pw-container.c;
     * this does not unsetenv, so special handling is not required
     * unlike for libwayland-client */
    state.core = pw_context_connect(
        state.context,
        pw_properties_new(
            PW_KEY_REMOTE_INTENTION, "manager",
            PW_KEY_REMOTE_NAME, remote_path,
            NULL),
        0);
    if (state.core == NULL) {
        res = HAKUREI_PIPEWIRE_CONNECT;
        goto out;
    }

    /* obtains the security context */
    pw_core_add_listener(state.core, &state.core_listener, &core_events, &state);
    state.registry = pw_core_get_registry(state.core, PW_VERSION_REGISTRY, 0);
    if (state.registry == NULL) {
        res = HAKUREI_PIPEWIRE_REGISTRY;
        goto out;
    }
    /* undocumented, this ends up calling registry_method_marshal_add_listener,
     * which is hard-coded to return 0, note that the function pointer this calls
     * is uninitialised for some pw_registry objects so if you are using this code
     * as an example you must keep that in mind */
    pw_registry_add_listener(state.registry, &state.registry_listener, &registry_events, &state);
    roundtrip(&state);
    if (state.sec == NULL) {
        res = HAKUREI_PIPEWIRE_NOT_AVAIL;
        goto out;
    }

    /* socket to attach security context */
    listen_fd = socket(AF_UNIX, SOCK_STREAM, 0);
    if (listen_fd < 0) {
        res = HAKUREI_PIPEWIRE_SOCKET;
        goto out;
    }

    /* similar to libwayland, pipewire requires bind and listen to be called
     * on the socket before being passed to pw_security_context_create */
    sockaddr.sun_family = AF_UNIX;
    snprintf(sockaddr.sun_path, sizeof(sockaddr.sun_path), "%s", socket_path);
    if (bind(listen_fd, (struct sockaddr *)&sockaddr, sizeof(sockaddr)) != 0) {
        res = HAKUREI_PIPEWIRE_BIND;
        goto out;
    }
    if (listen(listen_fd, 0) != 0) {
        res = HAKUREI_PIPEWIRE_LISTEN;
        goto out;
    }


    /* attach security context to socket */
    state.create_result = 0;
    state.pending_create = pw_security_context_create(state.sec, listen_fd, close_fd, &state.props->dict);
    if (SPA_RESULT_IS_ASYNC(state.pending_create)) {
        pw_log_debug("create: %d", state.pending_create);
        roundtrip(&state);
    }
    pw_log_debug("create result: %d", state.create_result);
    if (state.create_result < 0) {
        /* spa_strerror */
        if (SPA_RESULT_IS_ASYNC(-state.create_result))
            errno = EINPROGRESS;
        else
            errno = -state.create_result;

        res = HAKUREI_PIPEWIRE_ATTACH;
        goto out;
    }

out:
    if (listen_fd >= 0)
        close(listen_fd);
    if (state.sec != NULL)
        pw_proxy_destroy((struct pw_proxy *)state.sec);
    if (state.registry != NULL)
        pw_proxy_destroy((struct pw_proxy *)state.registry);
    if (state.core != NULL) {
        /* these happen after core is checked non-NULL and always succeeds */
        spa_hook_remove(&state.registry_listener);
        spa_hook_remove(&state.core_listener);

        pw_core_disconnect(state.core);
    }
    if (state.context != NULL)
        pw_context_destroy(state.context);
    if (state.loop != NULL)
        pw_main_loop_destroy(state.loop);
    pw_properties_free(state.props);
    pw_deinit();

    free((void *)socket_path);
    if (remote_path != NULL)
        free((void *)remote_path);
    return res;
}
