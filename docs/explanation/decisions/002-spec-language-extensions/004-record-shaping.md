---
title: "E4 to E7: nested records, first items, merged lists, and trees"
description: The four closed ways a spec may shape a reply record, each with its syntax, limits, threats, and tests.
category: explanation
tags:
  - explanation
  - security
  - decision
  - planned
---

# E4 to E7: nested records, first items, merged lists, and trees

This page belongs to [ADR 002](../002-spec-language-extensions.md). E4 is built by pull request 10, E5 by pull
request 11, and E6 by pull request 12; E7 is still planned. A response holds at most one of `items`, `first`, and
`merge`. `tree` needs `items`. All four keep the caps of S9: 64 levels and 1,048,576 tokens.

## E4: nested record with an allow-list

`auth.whoami` returns a user and a team. The user also holds an e-mail address, and a pointer to the whole object would
expose it.

```yaml
fields:
  user:
    object:
      id: { pointer: /data/user/id }
      name: { pointer: /data/user/name }
```

- **Limits.** At most 32 fields in an object, and one level of `object`. A pointer that reaches an object or a list is
  refused, so a whole object is never copied.
- **L5. A nested record passes a withheld field.** Control: the mapper reads declared pointers only, and the reply tree
  is dropped after mapping. Errors name paths, never values (R1). Proof: `TestNestedRecordWithholdsUndeclaredFields`
  plants an e-mail in the reply and finds it in no output, error, or log. `TestObjectPointerToWholeObjectIsRefused` and
  `TestNestedObjectDepthLimit` hold the limits.
- **Cannot do.** Wildcards, pass-through, or renaming by a rule.

## E5: first item or not found

`shares.get` takes the first share of a list.

```yaml
response:
  first: /data
  fields:
    id: { pointer: /id }
```

- **Rule.** The engine takes item 0 and maps its fields with pointers relative to the item. An empty list ends with
  `retriever.<kind>.not_found`. A value that is not a list is `unexpected_response`.
- **Threat L6 (first item).** The first item may not be the one the author meant. Control: only index 0, and the review
  text says "first of /data". Proof: `TestFirstTakesIndexZeroOnly`, `TestFirstOfEmptyListIsNotFound`, and
  `TestFirstOfNonListIsUnexpected`.
- **Cannot do.** Pick another index, filter, sort, or take the last.

## E6: merged lists with constant labels

`suggestions.mention` merges users, groups, and documents.

```yaml
response:
  merge:
    label_field: kind
    lists:
      - { items: /data/users, label: user, fields: { id: { pointer: /id } } }
      - { items: /data/documents, label: document, fields: { id: { pointer: /id } } }
```

- **Limits.** Two to eight lists. A label is a constant of lowercase letters and hyphens, at most 32 characters. The
  output keeps the declared list order, then the reply order. It shares the output cap of any list. The label beats a
  reply field of the same name.
- **Threat L6 (merge).** Control: the limits above. Proof: `TestMergeKeepsDeclaredOrderAndLabels`,
  `TestMergeLabelBeatsReplyField`, `TestMergeListCountLimit`, and `TestMergeOutputCap`.
- **Cannot do.** Remove duplicates, sort, join lists, or build a label from reply data.

## E7: computed tree fields

A tree read returns nested nodes. The engine flattens them depth first and computes two fields, and only these two.

```yaml
response:
  items: /data
  fields:
    id: { pointer: /id, required: true }
  tree:
    children: /children
    parent_field: parentId
    depth_field: depth
    root_parent_param: document_id # optional: the parent of the top nodes
    max_depth: 64
```

- **Rule.** `parentId` is the ID of the parent node. For a top node it is null, or the value of `root_parent_param`.
  `depth` counts from 0. A node deeper than `max_depth`, and its subtree, is left out, as the Go connector does today.
  `max_depth` is at most 64, and a spec may lower it.
- **A cap to raise.** A tree level takes two JSON levels, an array and an object, but S9 stops decoding at 64 levels.
  So a reply with `tree` is decoded with a fixed higher cap of 136 levels, which covers 65 tree levels. No spec field
  sets it, and no other reply changes. It is a proposal that the owner can still veto at review.
- **L7. A deep or wide tree makes the engine compute without bound.** Control: the depth cap, a flat loop with no
  recursion, and the token cap. Proof: `TestTreeFlattensDepthFirstWithParentAndDepth`,
  `TestTreeDropsNodesPastMaxDepth`, `TestTreeRefusesMaxDepthAbove64`, `TestTreeNeedsRequiredId`, and
  `TestOnlyTreeRepliesGetTheHigherDecodeCap`.
- **Cannot do.** Count, sum, sort, build a path string, or compute any other field.
