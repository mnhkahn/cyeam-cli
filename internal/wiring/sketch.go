package wiring

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const wireModule = "WireModuleID"

type instance struct {
	Index      string          `xml:"modelIndex,attr"`
	Module     string          `xml:"moduleIdRef,attr"`
	Title      string          `xml:"title"`
	Breadboard *breadboardView `xml:"views>breadboardView"`
}

type breadboardView struct {
	Geometry   geometry    `xml:"geometry"`
	Connectors []connector `xml:"connectors>connector"`
}

type geometry struct {
	X     float64 `xml:"x,attr"`
	Y     float64 `xml:"y,attr"`
	X1    float64 `xml:"x1,attr"`
	Y1    float64 `xml:"y1,attr"`
	X2    float64 `xml:"x2,attr"`
	Y2    float64 `xml:"y2,attr"`
	Flags int     `xml:"wireFlags,attr"`
}

type connector struct {
	ID    string `xml:"connectorId,attr"`
	Links []link `xml:"connects>connect"`
}

type link struct {
	Index string `xml:"modelIndex,attr"`
	Pin   string `xml:"connectorId,attr"`
	Layer string `xml:"layer,attr"`
}

type part struct {
	Module string `xml:"moduleId,attr"`
	Title  string `xml:"title"`
	Pins   []struct {
		ID   string `xml:"id,attr"`
		Name string `xml:"name,attr"`
	} `xml:"connectors>connector"`
}

type point [2]float64

type wire struct {
	Index string
	Ends  [2]point
	Links [2][]link
}

type terminal struct {
	Part string `json:"part"`
	Pin  string `json:"pin,omitempty"`
	At   point  `json:"at"`
}

type route struct {
	Wires []int      `json:"wires"`
	Pins  []terminal `json:"pins"`
}

type sketch struct {
	Wires     []wire
	Parts     map[string]part
	Instances map[string]instance
}

func readSketch(path string) (sketch, error) {
	var result sketch
	result.Parts = make(map[string]part)
	result.Instances = make(map[string]instance)

	var data []byte
	var err error
	if strings.EqualFold(filepath.Ext(path), ".fzz") {
		data, err = readArchive(path, result.Parts)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return result, err
	}
	var diagram struct {
		Instances []instance `xml:"instances>instance"`
	}
	if err := xml.Unmarshal(data, &diagram); err != nil {
		return result, fmt.Errorf("read Fritzing sketch: %w", err)
	}
	for _, inst := range diagram.Instances {
		result.Instances[inst.Index] = inst
		view := inst.Breadboard
		if inst.Module != wireModule || view == nil {
			continue
		}
		// PCB traces, ratsnest lines and schematic traces are not drawn in breadboard view.
		if view.Geometry.Flags&(4|16|128) != 0 {
			continue
		}
		g := view.Geometry
		wire := wire{Index: inst.Index, Ends: [2]point{{g.X + g.X1, g.Y + g.Y1}, {g.X + g.X2, g.Y + g.Y2}}}
		visible := true
		for _, c := range view.Connectors {
			end := 0
			if c.ID != "connector0" {
				end = 1
			}
			for _, connection := range c.Links {
				visible = visible && strings.HasPrefix(connection.Layer, "breadboard")
			}
			wire.Links[end] = append(wire.Links[end], c.Links...)
		}
		if visible {
			result.Wires = append(result.Wires, wire)
		}
	}
	if len(result.Wires) == 0 {
		return result, fmt.Errorf("the sketch has no drawn breadboard wires")
	}
	return result, nil
}

func readArchive(path string, parts map[string]part) ([]byte, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer archive.Close()

	var diagram []byte
	for _, f := range archive.File {
		if f.FileInfo().IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Name))
		if ext != ".fz" && ext != ".fzp" {
			continue
		}
		file, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			return nil, err
		}
		if ext == ".fz" {
			if diagram != nil {
				return nil, fmt.Errorf("the .fzz contains multiple .fz sketches")
			}
			diagram = data
			continue
		}
		var metadata part
		if err := xml.Unmarshal(data, &metadata); err != nil {
			return nil, fmt.Errorf("read part %s: %w", f.Name, err)
		}
		parts[metadata.Module] = metadata
	}
	if diagram == nil {
		return nil, fmt.Errorf("the .fzz contains no .fz sketch")
	}
	return diagram, nil
}

type pinKey struct{ Part, Pin string }

func (s sketch) label(key pinKey, at point) terminal {
	inst := s.Instances[key.Part]
	part := s.Parts[inst.Module]
	title := strings.TrimSpace(inst.Title)
	if title == "" {
		title = part.Title
	}
	if title == "" {
		title = inst.Module
	}
	if title == "" {
		title = "Part " + key.Part
	}
	pin := key.Pin
	for _, p := range part.Pins {
		if p.ID == key.Pin && strings.TrimSpace(p.Name) != "" {
			pin = p.Name
			break
		}
	}
	return terminal{Part: title, Pin: pin, At: at}
}

func (s sketch) routes() []route {
	index := make(map[string]int, len(s.Wires))
	parent := make([]int, len(s.Wires))
	for i, wire := range s.Wires {
		index[wire.Index] = i
		parent[i] = i
	}
	find := func(i int) int {
		for i != parent[i] {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	join := func(a, b int) { parent[find(a)] = find(b) }

	type endpoint struct {
		Terminal terminal
		Key      pinKey
	}
	endpoints := make([][]endpoint, len(s.Wires))
	wireAtPin := make(map[pinKey]int)
	for i, w := range s.Wires {
		for end, links := range w.Links {
			attached := false
			for _, l := range links {
				if other, ok := index[l.Index]; ok {
					join(i, other)
					attached = true
					continue
				}
				if s.Instances[l.Index].Module == wireModule {
					continue
				}
				attached = true
				key := pinKey{l.Index, l.Pin}
				if other, ok := wireAtPin[key]; ok {
					join(i, other)
				} else {
					wireAtPin[key] = i
				}
				endpoints[i] = append(endpoints[i], endpoint{s.label(key, w.Ends[end]), key})
			}
			if !attached {
				key := pinKey{w.Index, fmt.Sprint(end)}
				endpoints[i] = append(endpoints[i], endpoint{terminal{Part: "Open end", At: w.Ends[end]}, key})
			}
		}
	}
	nets := make(map[int]int)
	seen := make(map[pinKey]bool)
	var routes []route
	for i, ends := range endpoints {
		root := find(i)
		group, ok := nets[root]
		if !ok {
			group = len(routes)
			nets[root] = group
			routes = append(routes, route{})
		}
		routes[group].Wires = append(routes[group].Wires, i)
		for _, end := range ends {
			if !seen[end.Key] {
				routes[group].Pins = append(routes[group].Pins, end.Terminal)
				seen[end.Key] = true
			}
		}
	}
	return routes
}
