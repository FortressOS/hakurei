#include "acl-update.h"
#include <acl/libacl.h>
#include <stdbool.h>
#include <stdlib.h>
#include <sys/acl.h>

int hakurei_acl_update_file_by_uid(const char *path_p, uid_t uid, acl_perm_t *perms,
                             size_t plen) {
  int ret = -1;
  bool v;
  int i;
  acl_t acl;
  acl_entry_t entry;
  acl_tag_t tag_type;
  void *qualifier_p;
  acl_permset_t permset;

  acl = acl_get_file(path_p, ACL_TYPE_ACCESS);
  if (acl == NULL)
    goto out;

  // prune entries by uid
  for (i = acl_get_entry(acl, ACL_FIRST_ENTRY, &entry); i == 1;
       i = acl_get_entry(acl, ACL_NEXT_ENTRY, &entry)) {
    if (acl_get_tag_type(entry, &tag_type) != 0)
      return -1;
    if (tag_type != ACL_USER)
      continue;

    qualifier_p = acl_get_qualifier(entry);
    if (qualifier_p == NULL)
      return -1;
    v = *(uid_t *)qualifier_p == uid;
    acl_free(qualifier_p);

    if (!v)
      continue;

    acl_delete_entry(acl, entry);
  }

  if (plen == 0)
    goto set;

  if (acl_create_entry(&acl, &entry) != 0)
    goto out;
  if (acl_get_permset(entry, &permset) != 0)
    goto out;
  for (i = 0; i < plen; i++) {
    if (acl_add_perm(permset, perms[i]) != 0)
      goto out;
  }
  if (acl_set_tag_type(entry, ACL_USER) != 0)
    goto out;
  if (acl_set_qualifier(entry, (void *)&uid) != 0)
    goto out;

set:
  if (acl_calc_mask(&acl) != 0)
    goto out;
  if (acl_valid(acl) != 0)
    goto out;
  if (acl_set_file(path_p, ACL_TYPE_ACCESS, acl) == 0)
    ret = 0;

out:
  free((void *)path_p);
  if (acl != NULL)
    acl_free((void *)acl);
  return ret;
}
