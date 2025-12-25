package main

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// CSS selectors for parsing go.dev/dl page
const (
	selectorArchive      = "#archive"
	selectorArchiveItems = "div.toggle"
	selectorStable       = "#stable"
	selectorTable        = "table"
	selectorTableHead    = "thead"
	selectorTableBody    = "tbody tr"
	selectorTableData    = "td"
	selectorLink         = "a"
	selectorArticle      = "article"
)

type Parser struct {
	doc      *goquery.Document
	releases map[string]string
}

// NewParser creates a new DOM tree parser
func NewParser(reader io.Reader) (*Parser, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML document: %w", err)
	}
	return &Parser{
		doc:      doc,
		releases: make(map[string]string),
	}, nil
}

// Archived returns all archived versions
func (p *Parser) Archived() map[string]*Version {
	result := make(map[string]*Version)
	p.doc.Find(selectorArchive).Find(selectorArchiveItems).Each(func(i int, selection *goquery.Selection) {
		version, ok := selection.Attr("id")
		if !ok {
			return
		}
		result[version] = &Version{
			Name:     version,
			Packages: p.findPackages(version, selection),
		}
	})
	return result
}

// Stable returns all stable versions
func (p *Parser) Stable() map[string]*Version {
	result := make(map[string]*Version)
	p.doc.Find(selectorStable).NextUntil("#archive,#unstable").Each(func(i int, selection *goquery.Selection) {
		version, ok := selection.Attr("id")
		if !ok {
			return
		}
		result[version] = &Version{
			Name:     version,
			Packages: p.findPackages(version, selection.Find(selectorTable).First()),
		}
	})
	return result
}

// AllVersions returns all versions (stable + archived)
func (p *Parser) AllVersions() map[string]*Version {
	result := p.Stable()
	for k, v := range p.Archived() {
		result[k] = v
	}
	return result
}

func (p *Parser) findPackages(tag string, table *goquery.Selection) (pkgs []*Package) {
	alg := strings.TrimSuffix(table.Find(selectorTableHead).Find("th").Last().Text(), " Checksum")
	released := p.releases[tag]

	table.Find(selectorTableBody).Each(func(i int, tr *goquery.Selection) {
		td := tr.Find(selectorTableData)
		name := td.Eq(0).Find(selectorLink).Text()
		if name == "" {
			return
		}
		pkgs = append(pkgs, &Package{
			Name:      name,
			Tag:       tag,
			URL:       td.Eq(0).Find(selectorLink).AttrOr("href", ""),
			Kind:      td.Eq(1).Text(),
			OS:        td.Eq(2).Text(),
			Arch:      td.Eq(3).Text(),
			Size:      td.Eq(4).Text(),
			Checksum:  td.Eq(5).Text(),
			Algorithm: alg,
			released:  released,
		})
	})
	return
}

func (p *Parser) setReleases(r map[string]string) {
	p.releases = r
}

// DateParser parses release dates from go.dev/doc/devel/release
type DateParser struct {
	doc *goquery.Document
}

// NewDateParser creates a new date parser
func NewDateParser(reader io.Reader) (*DateParser, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse release date document: %w", err)
	}
	return &DateParser{doc: doc}, nil
}

// releasePattern matches "go1.21.5 (released 2023-12-05)"
var releasePattern = regexp.MustCompile(`(go\d+\.\d+(?:\.\d+)?)\s+\(released\s+(\d{4}-\d{2}-\d{2})\)`)

func (p *DateParser) findReleaseDate() map[string]string {
	releases := make(map[string]string)

	// Parse from paragraphs containing "released"
	p.doc.Find(selectorArticle).Find("p").Each(func(i int, selection *goquery.Selection) {
		text := selection.Text()
		matches := releasePattern.FindAllStringSubmatch(text, -1)
		for _, m := range matches {
			if len(m) == 3 {
				releases[m[1]] = m[2]
			}
		}
	})

	// Parse from h2 headers (format: "go1.21 (released 2023-08-08)")
	p.doc.Find(selectorArticle).Find("h2").Each(func(i int, selection *goquery.Selection) {
		text := selection.Text()
		if matches := releasePattern.FindStringSubmatch(text); len(matches) == 3 {
			releases[matches[1]] = matches[2]
		}
	})

	return releases
}
