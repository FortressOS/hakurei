#include <sys/acl.h>

int f_acl_update_file_by_uid(const char *path_p, uid_t uid, acl_perm_t *perms, size_t plen);
