// Command schemagen generates typed Go entity accessors from CS2 schema definitions.
//
// Usage:
//
//	go run ./cmd/schemagen -input configs/schema/cs2_schema.json -output pkg/gostrike/entities/generated.go
//
// The generated code provides typed wrappers around Entity.GetPropInt/SetPropInt etc.,
// so plugin authors can use e.g. pawn.Health() instead of pawn.GetPropInt("CCSPlayerPawn", "m_iHealth").
//
// Schema approach inspired by CounterStrikeSharp (github.com/roflmuffin/CounterStrikeSharp).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

// SchemaFile represents the top-level schema JSON
type SchemaFile struct {
	Classes map[string]SchemaClass `json:"classes"`
}

// SchemaClass represents a single entity class
type SchemaClass struct {
	Parent string                 `json:"parent,omitempty"`
	Fields map[string]SchemaField `json:"fields"`
}

// SchemaField represents a single field in a class
type SchemaField struct {
	Type      string `json:"type"`
	Networked bool   `json:"networked"`
	Comment   string `json:"comment,omitempty"`
}

// GeneratedClass holds data for code generation
type GeneratedClass struct {
	GoName    string
	ClassName string
	Parent    string
	GoParent  string
	Fields    []GeneratedField
}

// GeneratedField holds field data for code generation
type GeneratedField struct {
	GoName     string // e.g., "Health"
	FieldName  string // e.g., "m_iHealth"
	ClassName  string // e.g., "CCSPlayerPawn"
	Type       string // e.g., "int32"
	GoType     string // e.g., "int32"
	Networked  bool
	HasSetter  bool
	GetterFunc string // e.g., "GetPropInt"
	SetterFunc string // e.g., "SetPropInt"
}

func main() {
	inputPath := flag.String("input", "configs/schema/cs2_schema.json", "Path to schema JSON file")
	outputPath := flag.String("output", "pkg/gostrike/entities/generated.go", "Path to output Go file")
	flag.Parse()

	data, err := os.ReadFile(*inputPath)
	if err != nil {
		log.Fatalf("Failed to read schema file: %v", err)
	}

	var schema SchemaFile
	if err := json.Unmarshal(data, &schema); err != nil {
		log.Fatalf("Failed to parse schema JSON: %v", err)
	}

	classes := processClasses(schema)

	if err := generateCode(classes, *outputPath); err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}

	fmt.Printf("Generated %d entity types in %s\n", len(classes), *outputPath)
}

func processClasses(schema SchemaFile) []GeneratedClass {
	// Sort class names for deterministic output
	classNames := make([]string, 0, len(schema.Classes))
	for name := range schema.Classes {
		classNames = append(classNames, name)
	}
	sort.Strings(classNames)

	var classes []GeneratedClass
	for _, className := range classNames {
		cls := schema.Classes[className]
		gc := GeneratedClass{
			GoName:    className,
			ClassName: className,
			Parent:    cls.Parent,
			GoParent:  cls.Parent,
		}

		// Sort field names for deterministic output
		fieldNames := make([]string, 0, len(cls.Fields))
		for name := range cls.Fields {
			fieldNames = append(fieldNames, name)
		}
		sort.Strings(fieldNames)

		for _, fieldName := range fieldNames {
			field := cls.Fields[fieldName]
			gf := GeneratedField{
				GoName:    schemaFieldToGoName(fieldName),
				FieldName: fieldName,
				ClassName: className,
				Type:      field.Type,
				Networked: field.Networked,
			}

			switch field.Type {
			case "int32":
				gf.GoType = "int32"
				gf.GetterFunc = "GetPropInt"
				gf.SetterFunc = "SetPropInt"
				gf.HasSetter = true
			case "uint64":
				// uint64 fields are read as two int32s or via special handling
				// For now, treat as int32 (lower 32 bits) - schemagen limitation
				gf.GoType = "int32"
				gf.GetterFunc = "GetPropInt"
				gf.SetterFunc = "SetPropInt"
				gf.HasSetter = true
			case "float32":
				gf.GoType = "float32"
				gf.GetterFunc = "GetPropFloat"
				gf.SetterFunc = "SetPropFloat"
				gf.HasSetter = true
			case "bool":
				gf.GoType = "bool"
				gf.GetterFunc = "GetPropBool"
				gf.SetterFunc = "SetPropBool"
				gf.HasSetter = true
			case "string":
				gf.GoType = "string"
				gf.GetterFunc = "GetPropString"
				gf.HasSetter = false // String setters need special handling
			case "vector3":
				gf.GoType = "gostrike.Vector3"
				gf.GetterFunc = "GetPropVector"
				gf.SetterFunc = "SetPropVector"
				gf.HasSetter = true
			default:
				continue // Skip unknown types
			}

			gc.Fields = append(gc.Fields, gf)
		}

		classes = append(classes, gc)
	}

	return classes
}

// schemaFieldToGoName converts m_iHealth -> Health, m_bIsScoped -> IsScoped, etc.
func schemaFieldToGoName(field string) string {
	// Remove m_ prefix
	name := field
	if strings.HasPrefix(name, "m_") {
		name = name[2:]
	}

	// Remove common type prefixes
	prefixes := []string{"i", "b", "fl", "n", "sz", "s", "vec", "ang", "p", "e", "h"}
	for _, prefix := range prefixes {
		if len(name) > len(prefix) && strings.HasPrefix(name, prefix) {
			next := rune(name[len(prefix)])
			if unicode.IsUpper(next) {
				name = name[len(prefix):]
				break
			}
		}
	}

	// Ensure first letter is uppercase
	if len(name) > 0 {
		runes := []rune(name)
		runes[0] = unicode.ToUpper(runes[0])
		name = string(runes)
	}

	return name
}

var codeTemplate = `// Code generated by schemagen. DO NOT EDIT.
//
// Generated from CS2 entity schema definitions.
// Schema system approach inspired by CounterStrikeSharp
// (github.com/roflmuffin/CounterStrikeSharp).

package entities

import (
	"github.com/corrreia/gostrike/pkg/gostrike"
)

{{range $cls := .}}
// {{$cls.GoName}} wraps an Entity with typed accessors for {{$cls.ClassName}} schema fields.
type {{$cls.GoName}} struct {
	*gostrike.Entity
}

// New{{$cls.GoName}} wraps an existing Entity as a {{$cls.GoName}}.
// Returns nil if the entity is nil.
func New{{$cls.GoName}}(e *gostrike.Entity) *{{$cls.GoName}} {
	if e == nil {
		return nil
	}
	return &{{$cls.GoName}}{Entity: e}
}
{{range $f := $cls.Fields}}
// {{$f.GoName}} returns the {{$f.ClassName}}::{{$f.FieldName}} property.
func (e *{{$cls.GoName}}) {{$f.GoName}}() {{$f.GoType}} {
	v, _ := e.{{$f.GetterFunc}}("{{$f.ClassName}}", "{{$f.FieldName}}")
	return v
}
{{if $f.HasSetter}}
// Set{{$f.GoName}} sets the {{$f.ClassName}}::{{$f.FieldName}} property.
func (e *{{$cls.GoName}}) Set{{$f.GoName}}(v {{$f.GoType}}) {
	e.{{$f.SetterFunc}}("{{$f.ClassName}}", "{{$f.FieldName}}", v)
}
{{end}}{{end}}{{end}}`

func generateCode(classes []GeneratedClass, outputPath string) error {
	tmpl, err := template.New("entities").Parse(codeTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Ensure output directory exists
	dir := outputPath[:strings.LastIndex(outputPath, "/")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	return tmpl.Execute(f, classes)
}
