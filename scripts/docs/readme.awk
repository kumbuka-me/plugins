# Drop the README title and resolve relative links against the plugin repository.
function normalize(path, parts, n, i, out, depth, stack) {
  n = split(path, parts, "/"); depth = 0
  for (i = 1; i <= n; i++) {
    if (parts[i] == "." || parts[i] == "") continue
    if (parts[i] == "..") { if (depth > 0) depth--; continue }
    stack[++depth] = parts[i]
  }
  out = ""
  for (i = 1; i <= depth; i++) out = out (i > 1 ? "/" : "") stack[i]
  return out
}
NR == 1 && /^# / { next }
!started && /^[[:space:]]*$/ { next }
{
  started = 1
  if (/^[[:space:]]*(```|~~~)/) fenced = !fenced
  line = $0; out = ""
  while (!fenced && match(line, /!?\[[^]]*\]\([^[:space:])]+\)/)) {
    out = out substr(line, 1, RSTART - 1)
    link = substr(line, RSTART, RLENGTH)
    line = substr(line, RSTART + RLENGTH)
    pos = index(link, "](")
    target = substr(link, pos + 2, length(link) - pos - 2)
    if (target !~ /:/ && target !~ /^[#\/]/) {
      base = link ~ /^!/ ? "https://raw.githubusercontent.com/kumbuka-me/plugins/main/" : "https://github.com/kumbuka-me/plugins/blob/main/"
      link = substr(link, 1, pos + 1) base normalize(slug "/" target) ")"
    }
    out = out link
  }
  print out line
}
