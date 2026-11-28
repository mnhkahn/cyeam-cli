package wiring

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSketch = `<module><instances>
<instance moduleIdRef="Controller" modelIndex="1"><title>ESP32</title></instance>
<instance moduleIdRef="Ring" modelIndex="2"><title>LED ring</title></instance>
<instance moduleIdRef="Other" modelIndex="3"><title>Separate device</title></instance>
<instance moduleIdRef="WireModuleID" modelIndex="4"><views><breadboardView>
  <geometry x="0" y="0" x2="9" y2="0" wireFlags="64"/>
  <connectors>
    <connector connectorId="connector0"><connects><connect modelIndex="1" connectorId="data" layer="breadboard"/></connects></connector>
    <connector connectorId="connector1"><connects><connect modelIndex="5" connectorId="connector0" layer="breadboardWire"/></connects></connector>
  </connectors>
</breadboardView></views></instance>
<instance moduleIdRef="WireModuleID" modelIndex="5"><views><breadboardView>
  <geometry x="9" y="0" x2="9" y2="0" wireFlags="64"/>
  <connectors>
    <connector connectorId="connector0"><connects><connect modelIndex="4" connectorId="connector1" layer="breadboardWire"/></connects></connector>
    <connector connectorId="connector1"><connects><connect modelIndex="2" connectorId="din" layer="breadboard"/></connects></connector>
  </connectors>
</breadboardView></views></instance>
<instance moduleIdRef="WireModuleID" modelIndex="6"><views><breadboardView>
  <geometry x="9" y="0" x2="0" y2="9" wireFlags="64"/>
  <connectors>
    <connector connectorId="connector0"><connects><connect modelIndex="3" connectorId="signal" layer="breadboard"/></connects></connector>
  </connectors>
</breadboardView></views></instance>
<instance moduleIdRef="WireModuleID" modelIndex="7"><views><breadboardView>
  <geometry x="0" y="0" x2="9" y2="9" wireFlags="16"/>
</breadboardView></views></instance>
</instances></module>`

const testSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="300" height="150" viewBox="0 0 300 150">
<g id="wires">
<line x1="10" y1="20" x2="110" y2="20" />
<path d="M110,20C140,15 180,15 210,20"/>
<line x1="110" y1="20" x2="110" y2="120"/>
<line x1="10" y1="20" x2="110" y2="20" />
<path d="M110,20C140,15 180,15 210,20"/>
<line x1="110" y1="20" x2="110" y2="120"/>
<circle cx="110" cy="20" />
</g></svg>`

func TestRenderUsesFritzingLinksForWireRoutes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "diagram.fzz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	for name, content := range map[string]string{
		"diagram.fz":     testSketch,
		"part.esp32.fzp": `<module moduleId="Controller"><connectors><connector id="data" name="GPIO14 &amp; data"/></connectors></module>`,
		"part.ring.fzp":  `<module moduleId="Ring"><connectors><connector id="din" name="DIN"/></connectors></module>`,
	} {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	svgPath := filepath.Join(dir, "diagram.svg")
	if err := os.WriteFile(svgPath, []byte(testSVG), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := Render(context.Background(), path, svgPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(result, []byte(` data-cyeam-wire="`)) != 3 || !bytes.Contains(result, []byte("<script")) {
		t.Fatalf("unexpected SVG output %s", result)
	}
	var doc struct {
		Metadata string `xml:"metadata"`
	}
	if err := xml.Unmarshal(result, &doc); err != nil {
		t.Fatal(err)
	}
	var routes []route
	if err := json.Unmarshal([]byte(doc.Metadata), &routes); err != nil {
		t.Fatal(err)
	}
	if len(routes) != 2 || len(routes[0].Wires) != 2 || len(routes[0].Pins) != 2 ||
		routes[0].Pins[0].Pin != "GPIO14 & data" || routes[0].Pins[1].Pin != "DIN" ||
		routes[0].Pins[1].At != (point{210, 20}) {
		t.Fatalf("bad joined route: %+v", routes)
	}
	if len(routes[1].Wires) != 1 || len(routes[1].Pins) != 2 || routes[1].Pins[1].Part != "Open end" {
		t.Fatalf("coincident but unconnected wire should have a separate route: %+v", routes[1])
	}
}

func TestRenderRejectsMismatchedSVG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "diagram.fz")
	if err := os.WriteFile(path, []byte(testSketch), 0600); err != nil {
		t.Fatal(err)
	}
	diagram, err := readSketch(path)
	if err != nil {
		t.Fatal(err)
	}
	other := strings.ReplaceAll(testSVG, `x2="110" y2="120"`, `x2="110" y2="130"`)
	if _, err := interactiveSVG([]byte(other), diagram); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected a geometry mismatch, got %v", err)
	}
}
