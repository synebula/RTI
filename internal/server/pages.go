package server

import (
	"bytes"
	"embed"
	htmpl "html/template"
)

//go:embed web/*
var webPages embed.FS

type PageRenderer struct {
	rootHTML string
	qrJS     string
	pairTpl  *htmpl.Template
}

type pairPageData struct {
	LocalURL  string
	PhoneURL  string
}

func LoadPages() (*PageRenderer, error) {
	rootHTML, err := webPages.ReadFile("web/root.html")
	if err != nil {
		return nil, err
	}
	qrJS, _ := webPages.ReadFile("web/qrcode.min.js") // Optional, ignore error if missing

	pairTpl, err := htmpl.ParseFS(webPages, "web/pair.html")
	if err != nil {
		return nil, err
	}
	return &PageRenderer{
		rootHTML: string(rootHTML),
		qrJS:     string(qrJS),
		pairTpl:  pairTpl,
	}, nil
}

func (p *PageRenderer) RenderPairPage(localURL, phoneURL string) (string, error) {
	var out bytes.Buffer
	err := p.pairTpl.Execute(&out, pairPageData{LocalURL: localURL, PhoneURL: phoneURL})
	return out.String(), err
}
