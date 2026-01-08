// Package generator provides Pkl module generation from Gateway API resources.
package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/bmurray/pkl-gateway-api/internal/schema"
)

const licenseHeader = `//===----------------------------------------------------------------------===//
// Copyright © 2024-2025 The Pkl Gateway API Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//===----------------------------------------------------------------------===//

`

// Generator generates Pkl modules from Gateway API resources.
type Generator struct {
	outputDir  string
	version    string
	k8sVersion string
	releaseTag string
}

// NewGenerator creates a new Generator.
func NewGenerator(outputDir, version, k8sVersion, releaseTag string) *Generator {
	return &Generator{
		outputDir:  outputDir,
		version:    version,
		k8sVersion: k8sVersion,
		releaseTag: releaseTag,
	}
}

// GenerateResource generates a Pkl module for a resource.
func (g *Generator) GenerateResource(r *schema.Resource) error {
	// Create output directory: gateway.networking.k8s.io/v1/
	dir := filepath.Join(g.outputDir, r.Group, r.APIVersion)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Generate module content
	var buf bytes.Buffer
	buf.WriteString(licenseHeader)

	// Doc comment
	if r.Description != "" {
		buf.WriteString(formatDocComment(r.Description, ""))
	}

	// Module declaration
	moduleName := fmt.Sprintf("gateway.%s.%s.%s", strings.ReplaceAll(r.Group, ".", "_"), r.APIVersion, r.Kind)
	buf.WriteString("@ModuleInfo { minPklVersion = \"0.25.0\" }\n")
	buf.WriteString(fmt.Sprintf("open module %s\n\n", moduleName))

	// Extends and imports
	buf.WriteString("extends \".../GatewayResource.pkl\"\n\n")
	buf.WriteString("import \"@k8s/apimachinery/pkg/apis/meta/v1/ObjectMeta.pkl\"\n\n")

	// Fixed apiVersion and kind
	buf.WriteString(fmt.Sprintf("fixed apiVersion: \"%s/%s\"\n\n", r.Group, r.APIVersion))
	buf.WriteString(fmt.Sprintf("fixed kind: \"%s\"\n\n", r.Kind))

	// Metadata property
	buf.WriteString("/// Standard object's metadata.\n")
	buf.WriteString("metadata: ObjectMeta?\n\n")

	// Properties
	for _, prop := range r.Properties {
		if prop.Description != "" {
			buf.WriteString(formatDocComment(prop.Description, ""))
		}
		buf.WriteString(fmt.Sprintf("%s: %s\n\n", prop.Name, prop.Type.PklType()))
	}

	// Classes
	for _, class := range r.Classes {
		g.generateClass(&buf, class, "")
	}

	// Write file
	filename := filepath.Join(dir, r.Kind+".pkl")
	if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	return nil
}

func (g *Generator) generateClass(buf *bytes.Buffer, class schema.Class, indent string) {
	// Doc comment
	if class.Description != "" {
		buf.WriteString(formatDocComment(class.Description, indent))
	}

	buf.WriteString(fmt.Sprintf("%sclass %s {\n", indent, class.Name))

	// Properties
	for _, prop := range class.Properties {
		if prop.Description != "" {
			buf.WriteString(formatDocComment(prop.Description, indent+"  "))
		}
		buf.WriteString(fmt.Sprintf("%s  %s: %s\n", indent, prop.Name, prop.Type.PklType()))
		buf.WriteString("\n")
	}

	buf.WriteString(fmt.Sprintf("%s}\n\n", indent))
}

// GenerateSchema generates the gatewaySchema.pkl file.
func (g *Generator) GenerateSchema(resources []*schema.Resource) error {
	var buf bytes.Buffer
	buf.WriteString(licenseHeader)

	buf.WriteString("@ModuleInfo { minPklVersion = \"0.25.0\" }\n")
	buf.WriteString("module gateway.gatewaySchema\n\n")

	buf.WriteString("/// Resource modules keyed by `kind` and `apiVersion`.\n")
	buf.WriteString("resourceTemplates: Mapping<String, Mapping<String, unknown>> = new {\n")

	// Group resources by kind
	byKind := make(map[string][]*schema.Resource)
	for _, r := range resources {
		byKind[r.Kind] = append(byKind[r.Kind], r)
	}

	// Sort kinds
	var kinds []string
	for k := range byKind {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)

	for _, kind := range kinds {
		buf.WriteString(fmt.Sprintf("  [\"%s\"] {\n", kind))
		rs := byKind[kind]
		// Sort by apiVersion
		sort.Slice(rs, func(i, j int) bool {
			return rs[i].APIVersion < rs[j].APIVersion
		})
		for _, r := range rs {
			fullAPIVersion := fmt.Sprintf("%s/%s", r.Group, r.APIVersion)
			importPath := fmt.Sprintf("%s/%s/%s.pkl", r.Group, r.APIVersion, r.Kind)
			buf.WriteString(fmt.Sprintf("    [\"%s\"] = import(\"%s\")\n", fullAPIVersion, importPath))
		}
		buf.WriteString("  }\n")
	}

	buf.WriteString("}\n")

	filename := filepath.Join(g.outputDir, "gatewaySchema.pkl")
	if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	return nil
}

// CopyBaseTemplates copies the base template files to the output directory.
func (g *Generator) CopyBaseTemplates(templatesDir string) error {
	// Copy GatewayResource.pkl
	src := filepath.Join(templatesDir, "GatewayResource.pkl")
	dst := filepath.Join(g.outputDir, "GatewayResource.pkl")
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", src, err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", dst, err)
	}

	// Generate PklProject from template
	src = filepath.Join(templatesDir, "PklProject.tmpl")
	data, err = os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", src, err)
	}

	tmpl, err := template.New("PklProject").Parse(string(data))
	if err != nil {
		return fmt.Errorf("failed to parse PklProject template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]string{
		"Version":    g.version,
		"K8sVersion": g.k8sVersion,
		"ReleaseTag": g.releaseTag,
	})
	if err != nil {
		return fmt.Errorf("failed to execute PklProject template: %w", err)
	}

	dst = filepath.Join(g.outputDir, "PklProject")
	if err := os.WriteFile(dst, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", dst, err)
	}

	return nil
}

// formatDocComment formats a description as a Pkl doc comment.
func formatDocComment(desc, indent string) string {
	if desc == "" {
		return ""
	}

	var buf bytes.Buffer
	lines := strings.Split(desc, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			buf.WriteString(fmt.Sprintf("%s///\n", indent))
		} else {
			buf.WriteString(fmt.Sprintf("%s/// %s\n", indent, line))
		}
	}
	return buf.String()
}
