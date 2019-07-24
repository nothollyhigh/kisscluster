package app

import (
	"encoding/xml"
)

var (
	xmlconfig = &XMLConfig{}
)

type XMLConfig struct {
	XMLName xml.Name   `xml:"setting"`
	Options XMLOptions `xml:"options"`
	Proxy   XMLProxy   `xml:"proxy"`
}

type XMLOptions struct {
	XMLName   xml.Name     `xml:"options"`
	Debug     bool         `xml:"debug,attr"`
	LogDir    string       `xml:"logdir,attr"`
	Redirect  bool         `xml:"redirect,attr"`
	Heartbeat XMLHeartbeat `xml:"heartbeat"`
}

type XMLHeartbeat struct {
	XMLName  xml.Name `xml:"heartbeat"`
	Interval int      `xml:"interval,attr"`
	Timeout  int      `xml:"timeout,attr"`
}

type XMLProxy struct {
	XMLName xml.Name  `xml:"proxy"`
	Lines   []XMLLine `xml:"line"`
}

type XMLLine struct {
	XMLName  xml.Name `xml:"line"`
	Name     string   `xml:"name,attr"`
	Addr     string   `xml:"addr,attr"`
	Type     string   `xml:"type,attr"`
	Redirect string   `xml:"redirect,attr"`
	// Maxload  int64     `xml:"maxload,attr"`
	TLS    bool       `xml:"tls,attr"`
	Routes []XMLRoute `xml:"route"`
	Certs  []XMLCert  `xml:"cert"`
	Nodes  []XMLNode  `xml:"node"`
}

type XMLNode struct {
	XMLName   xml.Name `xml:"node"`
	Addr      string   `xml:"addr,attr"`
	Maxload   int64    `xml:"maxload,attr"`
	speed     int
	isDisable bool
}

type XMLCert struct {
	XMLName  xml.Name `xml:"cert"`
	Certfile string   `xml:"certfile,attr"`
	Keyfile  string   `xml:"keyfile,attr"`
}

type XMLRoute struct {
	XMLName xml.Name `xml:"route"`
	Path    string   `xml:"path,attr"`
}
