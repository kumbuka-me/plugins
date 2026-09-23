# Shared release-tag parsing helpers for POSIX shell scripts.

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
