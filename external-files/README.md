# External Files

External Files embeds repository files from GitHub and GitLab directly into Kumbuka pages. Show a whole file or selected lines, add numbered annotations, and pin content to a branch, tag, or commit. GitHub Enterprise and self-hosted GitLab are supported.

Repository content is displayed as escaped text and is never executed or interpreted as Markdown.

This plugin is disabled by default. It uses Kumbuka's generic plugin resources for connection settings, a bounded in-memory plugin cache, and the generic host-mediated HTTP capability for network access.

## Setup

1. Configure `KUMBUKA__ENCRYPTION_KEY` before storing repository credentials.
2. Install and enable External Files.
3. Open **Administration → Plugin settings → External Files**.
4. Add a source with a unique name, provider, API endpoint, repository, and explicit branch, tag, or commit.
5. Under **Appearance**, choose the default reference side and color and whether line numbers, referenced-line highlighting, provider, and revision metadata are shown.
6. Under **Cache**, choose the cache TTL. The default is **1 hour**.
7. For private repositories, add a dedicated read-only token restricted to that repository.
8. For an internal server, add the exact RFC1918 or IPv6 ULA addresses the provider hostname is allowed to resolve to.

Examples of API endpoints:

- GitHub.com: `https://api.github.com`
- GitHub Enterprise: `https://git.example.com/api/v3`
- GitLab.com: `https://gitlab.com/api/v4`
- Self-hosted GitLab: `https://git.example.com/api/v4`

GitHub repositories use `owner/repository`. GitLab repositories may contain nested group paths. Use a commit SHA when annotations must remain stable.

The source setting **Skip TLS certificate verification** is off by default. Prefer configuring a trusted CA with `SSL_CERT_FILE` or `SSL_CERT_DIR`; use the insecure switch only for a source whose transport you explicitly trust.

## Usage

Place each macro on its own line, outside a code fence.

Whole file:

```markdown
{{external-file source="engineering" path="src/main.go"}}
```

Single original line:

```markdown
{{external-file source="engineering" path="src/main.go" lines="12"}}
```

Inclusive line range with numbered annotations:

```markdown
{{external-file source="engineering" path="src/main.go" lines="10-25" note="12:Initialize the client." note="19:Handle errors before continuing."}}
```

Repeat `note` to annotate more lines, including multiple notes on one line. Descriptions are plain text. Annotation line numbers refer to the original file and must be inside the displayed range. References are rendered in a dedicated gutter instead of being inserted into the source text.

### Presentation overrides

Appearance defaults are configured under **Administration → Plugin settings → External Files → Appearance**. The defaults are right-side references, accent color, referenced-line highlighting, line numbers, provider label, and branch/tag/commit metadata enabled.

A single embed can override those defaults:

```markdown
{{external-file source="engineering" path="src/main.go" lines="10-25" note="12:Initialize the client." reference-position="left" reference-color="yellow" highlight-references="false" line-numbers="true" show-provider="true" show-branch="false"}}
```

Supported `reference-position` values are `right` and `left`. Supported colors are `accent`, `blue`, `green`, `yellow`, `orange`, `red`, `purple`, and `gray`. Boolean overrides accept `true` or `false`.

The rendered header shows provider, repository, file path, and revision metadata according to those settings. Line numbers remain separate from annotation markers so the source stays visually aligned.

## Cache behavior

External Files caches the complete validated provider file in a bounded in-memory cache owned by the plugin runtime. The default TTL is **1 hour** and can be changed under **Cache** in the plugin settings.

The cache is lazy: there is no background polling. A file inside its TTL is served directly from the cache. The first page visit after expiry fetches the file again and replaces the cached value. If that refresh fails, the last valid cached copy is served instead. Files that are never viewed create no refresh traffic.

The **Refresh cache** action in the plugin settings marks all cached files stale. It does not fetch every repository immediately; each file is fetched again on its next page visit. Changing a source connection also invalidates cached content for that source automatically. The cache is intentionally ephemeral and starts empty after a Kumbuka restart or plugin reload.

## Settings ownership

All source, appearance, and cache settings belong to this plugin. Kumbuka does not have External Files-specific URL, token, provider, TLS, presentation, TTL, or refresh configuration.

Kumbuka generically renders and stores the plugin's manifest-declared typed settings and resource fields. `secret` fields are encrypted at rest and masked in administration; the plugin receives the decrypted value through `sdk.Resources()` when it reads its own source record.

External Files then creates an `sdk.HTTPRequest`. Kumbuka performs the actual network I/O and applies generic host security policy. The plugin requests these permissions:

- `settings:read` to read its typed settings and source records;
- `network:http` for outbound HTTP(S);
- `network:private` so explicitly configured exact private addresses can be used;
- `network:insecure-tls` so a source can explicitly disable origin certificate verification.

## Security and limits

- Tokens never appear in Markdown, URLs, browser storage, or rendered plugin output.
- Only authenticated Kumbuka invocation contexts can use the generic HTTP capability. Public share rendering cannot fetch external files.
- External Files accepts HTTPS provider endpoints only and never follows redirects because the host HTTP client disables them.
- Kumbuka validates and pins resolved destination addresses before dialing. Private destinations require exact addresses supplied by this plugin from the administrator-managed source. Loopback, link-local, shared-address space, metadata, documentation, and other special-use destinations remain blocked.
- External Files validates provider responses and accepts at most 128 KiB of UTF-8 text and 10,000 lines. Binary/control content and Unicode formatting controls are rejected. Git LFS objects are not expanded.
- A maximum of 32 annotations is allowed per embed, with 2,048 bytes per description.
- The plugin permits at most 30 fetch attempts per source per minute. Kumbuka separately applies generic HTTP request/response bounds, timeouts, concurrency limits, and the plugin invocation deadline.
- Provider errors and credentials are never copied into page error boxes.
- External content is not stored in rendered-page artifacts, browser storage, or Kumbuka's database by the cache. Cached files live only in the plugin runtime's bounded memory cache.

An administrator-configured source grants this plugin access to the repository at the selected revision. It is not a per-reader repository ACL. Use a dedicated repository when only a subset of content should be disclosed.

## Proxy and CA environment settings

Kumbuka's generic plugin HTTP client honors conventional deployment networking settings:

- `HTTPS_PROXY` / `https_proxy`
- `HTTP_PROXY` / `http_proxy`
- `NO_PROXY` / `no_proxy`
- `SSL_CERT_FILE`
- `SSL_CERT_DIR`

Proxy configuration belongs to the Kumbuka process, not to this plugin. Proxy TLS verification is never disabled by a plugin source's **Skip TLS certificate verification** setting.

## Visual editor

In Visual mode, external-file macros render as file cards showing the configured source, path, line range, and presentation options without fetching repository content. Select the card to edit the embed settings or its Markdown source.

## References

- [GitHub repository contents API](https://docs.github.com/en/rest/repos/contents)
- [GitLab repository files API](https://docs.gitlab.com/api/repository_files/)
- [OWASP SSRF prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
