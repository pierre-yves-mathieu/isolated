#!/bin/bash
set -e

# Guard - only this script can push to release
export LXC_RELEASE_PUSH=1

# Fetch latest from release remote
git fetch release main:release-branch 2>/dev/null || {
    # First time: create orphan branch
    git checkout --orphan release-branch
    git reset --hard
}

# Switch to release branch
git checkout release-branch

# Replace all content with current main
git checkout main -- .

# Remove files that shouldn't be in public release
rm -f sync-release.sh

# Stage all changes
git add -A

# Check if there are changes to commit
if git diff --cached --quiet; then
    echo "No changes to release"
    git checkout main
    exit 0
fi

# Commit with Sunday's date
GIT_AUTHOR_DATE="$(date -d 'last sunday' -I)T12:00:00" \
GIT_COMMITTER_DATE="$(date -d 'last sunday' -I)T12:00:00" \
git commit --no-verify -m "Weekly snapshot $(date -d 'last sunday' +%Y-%m-%d)"

# Push to release remote (not force, builds history)
git push release release-branch:main

# Return to main
git checkout main

echo "✓ Released to public repo"
