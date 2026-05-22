---
name: new-api-image-build
description: Build and package Docker images for the new-api project on the dev server. Use when the user asks to build a new-api image, package an image, publish a versioned Docker image, deploy/build from the online dev branch, or run build.sh for a version such as 0.1.4. The fixed source of truth is origin/dev on root@43.153.35.31:/home/work/new-api, and the fixed build command is ./build.sh version.
---

# New API Image Build

## Defaults

- Default build target: `root@43.153.35.31:/home/work/new-api`.
- Source branch: `origin/dev`.
- Remote build script: `/home/work/new-api/build.sh <version>`.
- Git tag policy: do not create or delete Git tags unless the user explicitly asks for tag work.

## Workflow

1. Build on the server by default: SSH to `root@43.153.35.31` and use `/home/work/new-api`.
2. Confirm the requested version. If missing, ask for it.
3. Synchronize to `origin/dev` before building:

   ```bash
   git fetch origin dev --tags
   git checkout dev
   git status --short
   git pull --ff-only origin dev
   ```

4. If `git status --short` shows local changes, stop and report them. Do not reset, clean, stash, or overwrite unless the user explicitly asks.
5. Run the build on the server:

   ```bash
   ./build.sh <version>
   ```

6. Report the image tags printed by the script and whether the push succeeded.

## Useful Script

For server builds, prefer the bundled helper:

```bash
/Users/sherx/Code/new-api/.agents/skills/new-api-image-build/scripts/build-new-api-image-on-server.sh 0.1.4
```

The branch and build command are fixed. Do not add alternate branch, remote, registry, local-build, or no-push behavior unless the user explicitly asks to change the skill.

## Server Build Command

When building from the online dev deployment, use this shape:

```bash
ssh root@43.153.35.31 'cd /home/work/new-api && git fetch origin dev --tags && git checkout dev && test -z "$(git status --short)" && git pull --ff-only origin dev && ./build.sh <version>'
```

If the command fails at the clean-worktree check, inspect with:

```bash
ssh root@43.153.35.31 'cd /home/work/new-api && git status --short --branch'
```

Then ask before changing server files.
