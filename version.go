package main

import (
	"cmp"
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// Pre-release kinds in ascending precedence: alpha < beta < rc < stable.
const (
	preAlpha = iota
	preBeta
	preRC
	preStable
)

var versionRe = regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?(?:(alpha|beta|rc)(\d+)?)?$`)

// goVersion is a parsed Go version such as "go1.22.3" or "go1.23rc1".
type goVersion struct {
	major, minor, patch int
	pre                 int // preAlpha, preBeta, preRC or preStable
	preNum              int
}

// parseGoVersion accepts "go1.22.3", "v1.22.3", "1.22rc1" and friends.
func parseGoVersion(s string) (goVersion, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "go")
	s = strings.TrimPrefix(s, "v")

	m := versionRe.FindStringSubmatch(s)
	if m == nil {
		return goVersion{}, false
	}

	v := goVersion{pre: preStable}
	v.major, _ = strconv.Atoi(m[1])
	v.minor, _ = strconv.Atoi(m[2])
	if m[3] != "" {
		v.patch, _ = strconv.Atoi(m[3])
	}
	switch m[4] {
	case "alpha":
		v.pre = preAlpha
	case "beta":
		v.pre = preBeta
	case "rc":
		v.pre = preRC
	}
	if m[5] != "" {
		v.preNum, _ = strconv.Atoi(m[5])
	}
	return v, true
}

func (v goVersion) compare(o goVersion) int {
	if d := cmp.Compare(v.major, o.major); d != 0 {
		return d
	}
	if d := cmp.Compare(v.minor, o.minor); d != 0 {
		return d
	}
	if d := cmp.Compare(v.patch, o.patch); d != 0 {
		return d
	}
	if d := cmp.Compare(v.pre, o.pre); d != 0 {
		return d
	}
	return cmp.Compare(v.preNum, o.preNum)
}

// compareTags compares two version strings; unparsable tags sort lowest.
func compareTags(a, b string) int {
	va, oka := parseGoVersion(a)
	vb, okb := parseGoVersion(b)
	switch {
	case oka && okb:
		return va.compare(vb)
	case oka:
		return 1
	case okb:
		return -1
	default:
		return strings.Compare(a, b)
	}
}

// sortVersionsDesc sorts version tags newest-first.
func sortVersionsDesc(tags []string) {
	sort.SliceStable(tags, func(i, j int) bool {
		return compareTags(tags[i], tags[j]) > 0
	})
}

// normalizeTag converts "1.22.3" / "v1.22.3" / "go1.22.3" to "go1.22.3".
func normalizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	switch {
	case strings.HasPrefix(tag, "go"):
		return tag
	case strings.HasPrefix(tag, "v"):
		return "go" + tag[1:]
	default:
		return "go" + tag
	}
}

// archiveName returns the official archive filename for a version tag on
// the current platform, e.g. "go1.22.3.windows-amd64.zip".
func archiveName(tag string) string {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("%s.%s-%s%s", tag, runtime.GOOS, runtime.GOARCH, ext)
}
