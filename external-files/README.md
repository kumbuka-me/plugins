# External Files

Display whole files or selected lines from administrator-configured GitHub and GitLab repositories. File content is displayed as escaped text and is never interpreted as Markdown.

The plugin is disabled by default.

## Setup

1. Configure `KUMBUKA__ENCRYPTION_KEY` before saving repository credentials.
2. Open **Administration → External Files** in the plugin settings section below **Recycle bin**.
3. Add a source with a unique name, `github` or `gitlab` provider, HTTPS API endpoint, repository, and explicit branch/tag/commit.
4. Optionally add a dedicated read-only access token. Secret fields are encrypted by Kumbuka and are available only to the owning plugin at runtime.
5. Enable the source, then enable the External Files plugin.

Typical API endpoints are `https://api.github.com`, GitHub Enterprise `/api/v3`, GitLab `https://gitlab.com/api/v4`, or a self-hosted GitLab `/api/v4` endpoint.

Use **Skip TLS certificate verification** only for a source that cannot use normal certificate validation. A trusted certificate is preferable. The plugin must explicitly declare the `network:insecure-tls` capability before Kumbuka accepts such a request.

## Usage

Whole file:

```markdown
{{external-file source="engineering" path="src/main.go"}}
```

Single original line:

```markdown
{{external-file source="engineering" path="src/main.go" lines="12"}}
```

Inclusive line range with annotations:

```markdown
{{external-file source="engineering" path="src/main.go" lines="10-25" note="12:Initialize the client." note="19:Handle errors before continuing."}}
```

Repeat `note` to annotate multiple lines. Annotation text is always HTML-escaped.

## Architecture and security

External Files owns its repository configuration and provider protocol. It reads its structured `sources` settings through `sdk.Resources()` and builds provider requests itself. Kumbuka core has no External Files URL, token, TLS switch, provider model, approval model, or repository-specific HTTP code.

Network I/O still crosses Kumbuka's generic `sdk.HTTP()` host capability. Kumbuka validates bounded HTTP requests, enforces the plugin's declared network permissions, blocks special-use destinations, limits concurrency and response sizes, and performs TLS. `network:private` is required before a plugin may connect to private address space, while loopback, link-local, metadata, and other special-use destinations remain blocked. `network:insecure-tls` is required before a plugin can disable certificate verification for a request.

Tokens are stored as `secret` fields in the generic plugin settings framework. Kumbuka encrypts them at rest, masks them in administrator forms, and decrypts them only when returning the owning plugin's resource records. Plugins never receive Kumbuka's encryption key.

Files are limited to 128 KiB of valid UTF-8 text and 10,000 lines. Control and Unicode formatting characters are rejected. Provider errors and credentials are never rendered into page output.

## Screenshots

Rendered file with an inline annotation:

![External Files renders annotated source excerpts inside Kumbuka.](assets/screenshots/external-files-annotated-readme.png)

The screenshot above shows the current renderer before the right-side annotation gutter redesign. It still demonstrates the plugin's core workflow: stable line selection, escaped source text, copy support, and a human-readable explanation anchored to original file lines.

## Build

External Files requires the SDK release that provides `Resources()` and the generic `HTTP()` capability.

```sh
./scripts/build-plugin.sh external-files dist
```
