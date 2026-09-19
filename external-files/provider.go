package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	sdk "github.com/kumbuka-me/sdk"
)

const (
	maxFileBytes       = 128 << 10
	maxLines           = 10000
	maxAnnotations     = 32
	maxAnnotationBytes = 2048
	maxSourceFetches   = 30
)

var errUnavailable = errors.New("external file unavailable")

// source contains one validated administrator-managed repository connection.
type source struct {
	// Provider selects the GitHub or GitLab API adapter.
	Provider string
	// Endpoint is the HTTPS API base URL, including any installation path prefix.
	Endpoint string
	// Repository is the owner/repository path or GitLab project path.
	Repository string
	// Ref selects the branch, tag, or commit fetched by the adapter.
	Ref string
	// Token authenticates requests and is never included in rendered output.
	Token string
	// PrivateIPs lists exact private addresses authorized for this connection.
	PrivateIPs []string
	// InsecureSkipVerify permits the host to skip TLS certificate verification for this source.
	InsecureSkipVerify bool
}

// selectedFile contains a validated line selection with its original first line number.
type selectedFile struct {
	// Content contains only the selected lines with normalized line endings.
	Content string
	// Start is the one-based line number in the original file.
	Start int
}

// sourceRate tracks one source's bounded fixed fetch window.
type sourceRate struct {
	// Since is the start of the current one-minute window.
	Since time.Time
	// Count is the number of fetch attempts charged to that window.
	Count int
}

// sourceRateState serializes per-source request accounting across renders.
type sourceRateState struct {
	// Mutex protects the request counters and expired-window cleanup.
	sync.Mutex
	// Values maps source names to their active fetch windows.
	Values map[string]sourceRate
}

var sourceFetchRates = sourceRateState{Values: make(map[string]sourceRate)}

// loadSource reads and validates one manifest-declared source resource.
func loadSource(name string, read resourceReader) (source, error) {
	record, err := read("sources", name)
	if err != nil || !strings.EqualFold(record.Key, name) {
		return source{}, errUnavailable
	}
	values := record.Values
	enabled, err := strconv.ParseBool(values["enabled"])
	if err != nil || !enabled {
		return source{}, errUnavailable
	}
	insecure, err := strconv.ParseBool(values["insecure_skip_verify"])
	if err != nil {
		return source{}, errUnavailable
	}

	result := source{
		Provider:           values["provider"],
		Endpoint:           strings.TrimRight(values["endpoint"], "/"),
		Repository:         values["repository"],
		Ref:                values["ref"],
		Token:              values["token"],
		PrivateIPs:         splitPrivateIPs(values["private_ips"]),
		InsecureSkipVerify: insecure,
	}
	if !validSource(result) {
		return source{}, errUnavailable
	}
	return result, nil
}

// validSource validates provider-specific semantics that do not belong in Kumbuka core.
func validSource(value source) bool {
	if value.Provider != "github" && value.Provider != "gitlab" {
		return false
	}
	if !validProviderEndpoint(value.Endpoint) || !validRepository(value.Provider, value.Repository) {
		return false
	}
	if value.Ref == "" || len(value.Ref) > 256 || strings.IndexFunc(value.Ref, unicode.IsControl) >= 0 || len(value.PrivateIPs) > 16 {
		return false
	}
	for _, raw := range value.PrivateIPs {
		address, parseErr := netip.ParseAddr(raw)
		if parseErr != nil || address.Zone() != "" || !address.IsPrivate() || address.Is4In6() {
			return false
		}
	}
	return len(value.Token) <= 4096 && strings.IndexFunc(value.Token, unicode.IsControl) < 0
}

// validProviderEndpoint accepts an unambiguous HTTPS API base without credentials, queries, or fragments.
func validProviderEndpoint(value string) bool {
	if len(value) > 512 {
		return false
	}
	endpoint, err := url.Parse(value)
	if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" {
		return false
	}
	if endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
		return false
	}
	if endpoint.RawPath != "" || strings.ContainsAny(endpoint.Host, "%\\") {
		return false
	}
	return endpoint.Path == "" || (validPath(strings.Trim(endpoint.Path, "/")) && !strings.Contains(endpoint.Path, "//"))
}

// validRepository checks a repository path and GitHub's exact owner/name requirement.
func validRepository(provider, repository string) bool {
	if !validPath(repository) || len(repository) > 256 {
		return false
	}
	return provider != "github" || strings.Count(repository, "/") == 1
}

// splitPrivateIPs parses comma- or whitespace-separated exact private address exceptions.
func splitPrivateIPs(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
}

// validPath validates repository and file paths without interpreting traversal or encoded separators.
func validPath(value string) bool {
	if value == "" || len(value) > 1024 || strings.ContainsAny(value, "\\%?#") || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

// providerFileURL constructs the provider API URL for one validated source and file path.
func providerFileURL(value source, path string) string {
	if value.Provider == "gitlab" {
		return value.Endpoint + "/projects/" + url.PathEscape(value.Repository) + "/repository/files/" + url.PathEscape(path) + "/raw?ref=" + url.QueryEscape(value.Ref) + "&lfs=false"
	}
	segments := strings.Split(value.Repository+"/contents/"+path, "/")
	for index := range segments {
		segments[index] = url.PathEscape(segments[index])
	}
	return value.Endpoint + "/repos/" + strings.Join(segments, "/") + "?ref=" + url.QueryEscape(value.Ref)
}

// fetchFile requests one provider file through Kumbuka's generic HTTP capability.
func fetchFile(value source, path string, do httpDoer) (string, error) {
	if !validPath(path) {
		return "", errUnavailable
	}
	headers := map[string]string{"User-Agent": "Kumbuka-External-Files/1"}
	if value.Provider == "github" {
		headers["Accept"] = "application/vnd.github+json"
		headers["X-GitHub-Api-Version"] = "2022-11-28"
		if value.Token != "" {
			headers["Authorization"] = "Bearer " + value.Token
		}
	} else {
		headers["Accept"] = "application/octet-stream"
		if value.Token != "" {
			headers["PRIVATE-TOKEN"] = value.Token
		}
	}

	response, err := do(sdk.HTTPRequest{
		Method:             http.MethodGet,
		URL:                providerFileURL(value, path),
		Headers:            headers,
		AllowedPrivateIPs:  value.PrivateIPs,
		InsecureSkipVerify: value.InsecureSkipVerify,
	})
	if err != nil || response.StatusCode != http.StatusOK || responseHasContentEncoding(response.Headers) {
		return "", errUnavailable
	}
	body := response.Body
	if value.Provider == "github" {
		var file struct {
			// Type distinguishes regular files from directories and links.
			Type string `json:"type"`
			// Encoding must identify the supported base64 representation.
			Encoding string `json:"encoding"`
			// Content contains the encoded provider file bytes.
			Content string `json:"content"`
			// Size is the provider-reported decoded byte count.
			Size int `json:"size"`
		}
		if json.Unmarshal(body, &file) != nil || file.Type != "file" || file.Encoding != "base64" || file.Size > maxFileBytes {
			return "", errUnavailable
		}
		body, err = base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			return "", errUnavailable
		}
	}
	content := string(body)
	if !validContent(content) {
		return "", errUnavailable
	}
	return content, nil
}

// responseHasContentEncoding rejects provider responses that still carry transfer encodings.
func responseHasContentEncoding(headers map[string][]string) bool {
	for name, values := range headers {
		if strings.EqualFold(name, "Content-Encoding") && len(values) > 0 && strings.TrimSpace(strings.Join(values, "")) != "" {
			return true
		}
	}
	return false
}

// validContent accepts bounded UTF-8 text without executable formatting controls.
func validContent(content string) bool {
	if len(content) > maxFileBytes || !utf8.ValidString(content) {
		return false
	}
	for _, r := range content {
		if (unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t') || unicode.In(r, unicode.Cf) {
			return false
		}
	}
	return true
}

// selectLines validates and extracts an inclusive one-based source range.
func selectLines(content string, start, end int) (selectedFile, error) {
	if !validContent(content) {
		return selectedFile{}, errUnavailable
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSuffix(content, "\n")
	lines := strings.Split(content, "\n")
	if len(lines) > maxLines {
		return selectedFile{}, errUnavailable
	}
	if start == 0 && end == 0 {
		start, end = 1, len(lines)
	}
	if start < 1 || end < start || end > len(lines) {
		return selectedFile{}, errUnavailable
	}
	return selectedFile{Content: strings.Join(lines[start-1:end], "\n"), Start: start}, nil
}

// allowSourceFetch enforces the plugin's per-source fixed-window request limit.
func allowSourceFetch(name string) bool {
	now := time.Now()
	sourceFetchRates.Lock()
	defer sourceFetchRates.Unlock()

	if len(sourceFetchRates.Values) >= 1024 {
		for key, value := range sourceFetchRates.Values {
			if now.Sub(value.Since) >= time.Minute {
				delete(sourceFetchRates.Values, key)
			}
		}
	}
	rate, exists := sourceFetchRates.Values[name]
	if !exists && len(sourceFetchRates.Values) >= 1024 {
		return false
	}
	if !exists || now.Sub(rate.Since) >= time.Minute {
		rate = sourceRate{Since: now}
	}
	if rate.Count >= maxSourceFetches {
		return false
	}
	rate.Count++
	sourceFetchRates.Values[name] = rate
	return true
}
