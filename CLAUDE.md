# Git Setup

Two remotes:
- `origin` - private repo, daily commits to `main`
- `release` - public repo, weekly squashed snapshots

Run `./sync-release.sh` to push a single squashed commit to public repo (script auto-excludes itself).
