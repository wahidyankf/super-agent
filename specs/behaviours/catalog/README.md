# Catalog behaviours

The sources catalog: which sources exist, how a command finds the catalog file, and how a bad catalog is refused.

## Files

- [sources-catalog.feature](sources-catalog.feature): the source list and its filters, the catalog file resolution order
  (flag, then environment variable, then the current directory), and refusals for a missing catalog, duplicate IDs, an
  access level above the kind's ceiling, and an unknown source ID.
