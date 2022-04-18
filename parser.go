package main

import (
	"github.com/PuerkitoBio/goquery"
	"io"
)

type Parser struct {
	doc *goquery.Document
}

func NewParser(reader io.Reader) *Parser {
	doc, _ := goquery.NewDocumentFromReader(reader)
	return &Parser{
		doc: doc,
	}
}

func (p *Parser) ArchivedVersions() []string {
	versions := make([]string, 0)
	p.doc.Find("#archive").Find("div.toggle").Each(func(i int, selection *goquery.Selection) {
		version, _ := selection.Attr("id")
		versions = append(versions, version)
	})
	return versions
}
