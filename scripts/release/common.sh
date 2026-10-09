# Shared release lookup helpers for POSIX shell scripts.

# latest_plugin_tags prints the newest valid stable tag for each plugin.
latest_plugin_tags() {
  awk '
    function valid_component(value) {
      return value ~ /^(0|[1-9][0-9]*)$/
    }

    function newer(plugin, major, minor, patch) {
      return !(plugin in tags) ||
        major > majors[plugin] ||
        (major == majors[plugin] && minor > minors[plugin]) ||
        (major == majors[plugin] && minor == minors[plugin] && patch > patches[plugin])
    }

    {
      separator = index($0, "/v")
      if (separator <= 1) next

      plugin = substr($0, 1, separator - 1)
      version = substr($0, separator + 2)
      if (plugin !~ /^[a-z0-9][a-z0-9-]*$/) next
      if (split(version, components, ".") != 3) next
      if (!valid_component(components[1]) || !valid_component(components[2]) || !valid_component(components[3])) next

      major = components[1] + 0
      minor = components[2] + 0
      patch = components[3] + 0
      if (newer(plugin, major, minor, patch)) {
        tags[plugin] = $0
        majors[plugin] = major
        minors[plugin] = minor
        patches[plugin] = patch
      }
    }

    END {
      for (plugin in tags) print plugin "\t" tags[plugin]
    }
  ' | sort
}

# published_release reports whether a tag appears in a newline-delimited release list.
published_release() {
  printf '%s\n' "$1" | grep -Fqx -- "$2"
}

# plugin_release_tags lists the newest published or unpublished tag for each plugin.
plugin_release_tags() {
  tags=$("$GH" api --paginate "repos/$GITHUB_REPOSITORY/tags" --jq '.[].name') || return $?
  printf '%s\n' "$tags" | latest_plugin_tags
}

# plugin_published_releases lists GitHub release tag names.
plugin_published_releases() {
  "$GH" api --paginate "repos/$GITHUB_REPOSITORY/releases" --jq '.[].tag_name'
}
