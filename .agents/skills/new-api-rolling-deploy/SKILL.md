---
name: new-api-rolling-deploy
description: Deploy a new-api Docker image with a CLB-aware rolling update across the Beijing (bj) or Silicon Valley (sv) node pair. Use when the user asks to deploy, roll out, or update new-api on either regional cluster.
---

# New API rolling deployment

This skill updates one node at a time behind Tencent CLB. It removes a node from
the selected region's CLB listener, waits for existing connections to drain,
updates `/home/work/new-api`'s `new-api` container, checks
`http://127.0.0.1:3000/api/status` and the Docker health state, then registers the
node again before moving to the next node. CLB mutations are asynchronous; the
script polls `DescribeTaskStatus` and does not continue until each mutation
reports success.

## Mandatory region confirmation

Before any mutating action, ask the user to confirm exactly one target:

- `bj`: Beijing CLB `lb-k2xs6db9`; nodes `82.157.29.200 (172.21.32.13)` and
  `49.232.247.197 (172.21.32.9)`.
- `sv`: Silicon Valley nodes `43.153.32.67 (172.26.0.31)` and
  `43.172.117.230 (172.26.0.32)`; load balancer identifiers come from the SV
  environment file.

Do not infer the region from the image tag, current SSH host, or previous turn.
If the user has not confirmed `bj` or `sv`, stop before SSH, CLB calls, image
pulls, container restarts, or any other mutation. If the user names both, ask
which one should be updated first and whether the second should follow.

## Configuration and execution

1. Ensure the selected environment file exists and contains its CLB listener ID:
   - `bj`: `scripts/rolling-update-bj.env`
   - `sv`: `scripts/rolling-update.env`
   The committed `*.env.example` files are templates and must not be used as
   live configuration until identifiers have been filled in.
2. Ensure `tccli` is installed and authenticated for the Tencent Cloud account
   that owns the selected CLB. Never put credentials in the repository or env
   files.
3. Run a dry run first when configuration has changed:

   ```bash
   DRY_RUN=1 ./scripts/rolling-update-bj.sh <image>
   # or
   DRY_RUN=1 ./scripts/rolling-update.sh <image>
   ```

4. After the user has confirmed the target region and the configuration passes
   dry run, execute the corresponding script. Use a versioned image when one is
   available, for example `uswccr.ccs.tencentyun.com/floatai/newapi:1.1.4`.

The scripts stop on a failed CLB operation, failed remote update, or health
timeout. On an update/health failure they attempt to register the node back
into CLB before stopping. Do not skip the health check or update both nodes in
parallel. A successful deployment must report the selected region, image,
both node results, and the final health state.
