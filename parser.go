package main

import (
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Parser struct {
	doc *goquery.Document
}

// NewParser return a new DOM tree parser
func NewParser(reader io.Reader) *Parser {
	doc, _ := goquery.NewDocumentFromReader(reader)
	return &Parser{
		doc: doc,
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

// AllVersions return all all versions
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
		})
	})
	return
}
