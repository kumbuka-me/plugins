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
	Provider           string
	Endpoint           string
	Repository         string
	Ref                string
	Token              string
	PrivateIPs         []string
	InsecureSkipVerify bool
}

// selectedFile contains a validated line selection with its original first line number.
type selectedFile struct {
	Content string
	Start   int
}

// sourceRate tracks one source's bounded rolling fetch window.
type sourceRate struct {
	Since time.Time
	Count int
}

var sourceFetchRates = struct {
	sync.Mutex
	Values map[string]sourceRate
}{Values: make(map[string]sourceRate)}

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
	endpoint, err := url.Parse(value.Endpoint)
	if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" || endpoint.RawPath != "" || strings.ContainsAny(endpoint.Host, "%\\") || len(value.Endpoint) > 512 {
		return false
	}
	if endpoint.Path != "" && (!validPath(strings.Trim(endpoint.Path, "/")) || strings.Contains(endpoint.Path, "//")) {
		return false
	}
	if !validPath(value.Repository) || len(value.Repository) > 256 || (value.Provider == "github" && len(strings.Split(value.Repository, "/")) != 2) {
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
			Type     string `json:"type"`
			Encoding string `json:"encoding"`
			Content  string `json:"content"`
			Size     int    `json:"size"`
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

// allowSourceFetch enforces the plugin's per-source rolling request limit.
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
