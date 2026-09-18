# External Files

Display whole files or selected lines from administrator-approved GitHub and GitLab repositories, including self-hosted instances. File content is displayed as text, never executed or interpreted as Markdown.

Requires the accompanying Kumbuka host and SDK changes that add `external:read`. Older hosts reject this permission. This plugin is disabled by default.

## Setup

1. Configure Kumbuka's existing application encryption key before adding credentials.
2. Open **Administration → Plugins → Manage approved external file sources**.
3. Enter a source name, provider API endpoint, repository, and explicit revision. Use a commit SHA when annotations must remain stable.
4. For private repositories, enter a dedicated read-only token restricted to that repository. GitHub fine-grained tokens need repository Contents read access. GitLab tokens can use `read_repository` for the repository files API.
5. Confirm that the repository's files at this revision may be disclosed to Kumbuka users. Save the approval, then install and enable this plugin.

Examples of API endpoints:

- GitHub.com: `https://api.github.com`
- GitHub Enterprise: `https://git.example.com/api/v3`
- GitLab.com: `https://gitlab.com/api/v4`
- Self-hosted GitLab: `https://git.example.com/api/v4`, including any installation prefix

For an internal Git server, also approve its exact private IPv4 or IPv6 ULA addresses. Public DNS addresses require no IP exception. All resolved addresses must be permitted. HTTPS certificate validation is enabled by default. Configure your CA using the settings below when necessary. Loopback, link-local, shared-address space, and metadata destinations remain blocked.

An approval covers the **entire repository at the specified revision**, including GitHub's resolution of in-repository symlinks. It does not grant repository permissions to individual readers. Authors cannot change the endpoint, repository, revision, credential, or network exceptions through Markdown. Use a dedicated repository when only a subset of content should be disclosed.

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

Repeat `note` to annotate more lines, including multiple notes on one line. Descriptions appear below the content box and use plain text. Escape quotes as `\"`. Annotation line numbers refer to the original file and must be inside the displayed range. Files use their original line numbers even when only a range is selected.

## Security and limits

- Tokens are encrypted at rest using Kumbuka's deployment-managed encryption key. They are never placed in Markdown, plugin settings, browser storage, URLs, or plugin responses.
- Source records live in a core-owned storage namespace unavailable to plugin settings/storage capabilities. Replacing an approval requires supplying credentials again, preventing accidental forwarding of an old token to a new endpoint.
- Only authenticated host contexts can fetch. Public share links do not fetch external content. In Kumbuka's no-auth deployment mode, everyone is treated as the local administrator; approval therefore exposes content to everyone who can reach that deployment.
- Fetching uses HTTPS with hostname verification, no redirects, and validated DNS answers pinned to the actual dialed address, including proxy CONNECT tunnels. Guests cannot supply URLs, HTTP headers or request bodies.
- Provider responses, errors, and tokens are never echoed into error boxes. All content and annotation descriptions are HTML-escaped; Kumbuka also applies its normal sanitizer.
- Maximum file size: 128 KiB of UTF-8 text and 10,000 lines. Selecting a range still fetches and validates the whole file. Binary files, control characters other than tab/newline/carriage return, and Unicode formatting controls are rejected. Git LFS objects are not expanded.
- Maximum 32 annotations per embed and 2,048 bytes per description. Maximum 30 fetch attempts per source per minute per server process and 8 concurrent fetches. Requests obey the host's invocation deadline (normally 2 seconds), with an additional 8-second HTTP ceiling. A slow provider may show an unavailable box.
- External content is not stored in rendered-page artifacts or a shared content cache. Revocation applies to subsequent requests, not already-running requests, copies or completed exports.

This plugin deliberately does not support arbitrary URLs, executable content, personal OAuth connections, or browser-side access tokens.

## Proxy and TLS environment settings

These are **Kumbuka server environment variables**; they cannot be set by a plugin or page author.

- `HTTPS_PROXY` / `https_proxy`: an HTTP or HTTPS proxy for provider HTTPS requests. Proxy URL credentials are supported and sent only to the proxy.
- `HTTP_PROXY` / `http_proxy`: recognized by the conventional proxy selector; providers themselves must use HTTPS, so their requests use `HTTPS_PROXY`.
- `NO_PROXY` / `no_proxy`: bypass rules using Go's conventional host/domain/IP/CIDR matching. Uppercase variables take precedence when both cases are set.
- `SSL_CERT_FILE`: PEM CA certificate file. `SSL_CERT_DIR`: platform-separated list of CA directories (colon-separated on Unix). Custom CAs are added to system trust for providers and HTTPS proxies.
- `KUMBUKA_EXTERNAL_FILES_INSECURE_SKIP_VERIFY=true`: explicitly disable provider TLS certificate and hostname verification. Default is `false`. The CLI equivalent is `--external-files-insecure-skip-verify`. The administration page displays a warning when enabled. This does **not** disable HTTPS proxy certificate verification or SSRF destination checks. Disabling verification permits interception of content and provider credentials; custom CA trust is preferable.

Example:

```sh
HTTPS_PROXY=http://proxy.internal:3128
NO_PROXY=git.internal.example
SSL_CERT_FILE=/run/secrets/company-ca.pem
```

The proxy endpoint is trusted deployment infrastructure and may reside on a private network. The provider destination is still resolved and checked by Kumbuka, and the proxy receives a numeric CONNECT destination to prevent a second DNS lookup from bypassing the address policy. Proxies must support CONNECT to numeric destinations. Unsupported proxy schemes fail closed. `ALL_PROXY`, SOCKS proxies, and unrelated tools' TLS bypass variables are not interpreted.

## Build

The plugin and host pull requests temporarily pin the companion SDK fork commit while [the SDK capability](https://github.com/kumbuka-me/sdk/pull/5) is under review:

```sh
./scripts/build-plugin.sh external-files dist
```

Build the Kumbuka application normally from its checkout. Before merging the dependent pull requests, publish the updated upstream SDK, update both repositories' SDK dependency, and remove the temporary fork replacements; the existing released SDK does not contain this capability.

## References

- [GitHub repository contents API](https://docs.github.com/en/rest/repos/contents)
- [GitLab repository files API](https://docs.gitlab.com/api/repository_files/)
- [OWASP SSRF prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
