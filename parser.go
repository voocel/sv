package main

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

// CSS selectors for parsing go.dev/dl page
// These can be updated if go.dev changes their HTML structure
const (
	selectorArchive      = "#archive"
	selectorArchiveItems = "div.toggle"
	selectorStable       = "#stable"
	selectorTable        = "table"
	selectorTableHead    = "thead"
	selectorTableRow     = "tr"
	selectorTableData    = "td"
	selectorLink         = "a"
	selectorArticle      = "article"
)

type Parser struct {
	doc      *goquery.Document
	releases map[string]string
}

// NewParser return a new DOM tree parser
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

// Archived return all archived versions
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

// Stable return all Stable versions
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

// AllVersions return all versions
func (p *Parser) AllVersions() map[string]*Version {
	stables := p.Stable()
	archives := p.Archived()
	for s, version := range archives {
		stables[s] = version
	}
	return stables
}

func (p *Parser) findPackages(tag string, table *goquery.Selection) (pkgs []*Package) {
	alg := strings.TrimSuffix(table.Find(selectorTableHead).Find("th").Last().Text(), " Checksum")
	table.Find(selectorTableRow).Not("first").Each(func(i int, tr *goquery.Selection) {
		td := tr.Find(selectorTableData)
		released := p.releases[tag]
		pkgs = append(pkgs, &Package{
			Name:      td.Eq(0).Find(selectorLink).Text(),
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

type DateParser struct {
	doc      *goquery.Document
	releases map[string]string
}

// NewDateParser return a new DOM tree parser
func NewDateParser(reader io.Reader) (*DateParser, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse release date document: %w", err)
	}
	return &DateParser{
		doc:      doc,
		releases: make(map[string]string),
	}, nil
}

func (p *DateParser) findReleaseDate() map[string]string {
	releaseRegex := regexp.MustCompile(`go[\s\S]*\)`)
	p.doc.Find(selectorArticle).Find("p:contains(released)").Each(func(i int, selection *goquery.Selection) {
		result := releaseRegex.FindString(selection.Text())
		if result == "" {
			return
		}
		tmp := strings.Split(result, " ")
		var version, date string
		if len(tmp) == 3 {
			version, date = tmp[0], strings.TrimRight(tmp[2], ")")
		} else if len(tmp) == 2 {
			version = strings.FieldsFunc(tmp[0], unicode.IsSpace)[0]
			date = strings.TrimRight(tmp[1], ")")
		}
		if version != "" && date != "" {
			p.releases[version] = date
		}
	})

	p.doc.Find(selectorArticle).Find("h2").Each(func(i int, selection *goquery.Selection) {
		tmp := strings.Split(selection.Text(), " ")
		if len(tmp) == 3 {
			p.releases[tmp[0]] = strings.TrimRight(tmp[2], ")")
		}
	})
	return p.releases
}
