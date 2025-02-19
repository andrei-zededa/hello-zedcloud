#!/usr/bin/env bash

set -ue;

_git="$(command -v git)";

latest_tag="$($_git describe --tags --abbrev=0 2>/dev/null || echo "0.0.0")";
latest_commit="$($_git rev-parse --short HEAD || echo "0")";
commit_count="$($_git rev-list "${latest_tag}"..HEAD --count 2>/dev/null || echo "0")";
dirty_status="$($_git status -s | wc -l || echo "0")";

version="${latest_tag}";
[ "$commit_count" -gt "0" ] && version="${version}+${latest_commit}.ahead${commit_count}";
[ "$dirty_status" -gt "0" ] && {
	if [[ "$version" =~ \++ ]]; then
		version="${version}.dirty";
	else
		version="${version}+dirty";
	fi
}

echo "$version";
