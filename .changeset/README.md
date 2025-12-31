# Changesets

This folder is used by `@changesets/cli` to track version changes.

## Adding a changeset

To add a changeset, run:

```bash
npx changeset
```

Or manually create a markdown file in this folder with:

```markdown
---
"eratemanager": patch
---

Description of your change
```

Valid bump types are: `patch`, `minor`, `major`
