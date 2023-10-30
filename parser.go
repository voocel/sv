package main

import (
	"io"
	"regexp"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

const releaseUrl = "https://go.dev/doc/devel/release"

type Parser struct {
	doc      *goquery.Document
	releases map[string]string
}

// NewParser return a new DOM tree parser
func NewParser(reader io.Reader) *Parser {
	doc, _ := goquery.NewDocumentFromReader(reader)
	return &Parser{
		doc:      doc,
		releases: make(map[string]string),
	}
}

// Archived return all archived versions
func (p *Parser) Archived() map[string]*Version {
	result := make(map[string]*Version)
	p.doc.Find("#archive").Find("div.toggle").Each(func(i int, selection *goquery.Selection) {
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
	p.doc.Find("#stable").NextUntil("#archive,#unstable").Each(func(i int, selection *goquery.Selection) {
		version, ok := selection.Attr("id")
		if !ok {
			return
		}

		result[version] = &Version{
			Name:     version,
			Packages: p.findPackages(version, selection.Find("table").First()),
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
	alg := strings.TrimSuffix(table.Find("thead").Find("th").Last().Text(), " Checksum")
	table.Find("tr").Not("first").Each(func(i int, tr *goquery.Selection) {
		td := tr.Find("td")
		released := p.releases[tag]
		pkgs = append(pkgs, &Package{
			Name:      td.Eq(0).Find("a").Text(),
			Tag:       tag,
			URL:       td.Eq(0).Find("a").AttrOr("href", ""),
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
func NewDateParser(reader io.Reader) *DateParser {
	doc, _ := goquery.NewDocumentFromReader(reader)
	return &DateParser{
		doc:      doc,
		releases: make(map[string]string),
	}
}

func (p *DateParser) findReleaseDate() map[string]string {
	p.doc.Find("article").Find("p:contains(released)").Each(func(i int, selection *goquery.Selection) {
		reg, err := regexp.Compile(`go[\s\S]*\)`)
		if err != nil {
			panic(err)
		}
		result := reg.FindString(selection.Text())
		if len(result) == 0 {
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
		p.releases[version] = date
	})
	return p.releases
}
