// Package crd provides parsing for Kubernetes CustomResourceDefinition YAML files.
package crd

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// CRD represents a Kubernetes CustomResourceDefinition.
type CRD struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       CRDSpec  `yaml:"spec"`
}

// Metadata contains CRD metadata.
type Metadata struct {
	Name        string            `yaml:"name"`
	Annotations map[string]string `yaml:"annotations"`
}

// CRDSpec contains the CRD specification.
type CRDSpec struct {
	Group    string       `yaml:"group"`
	Names    CRDNames     `yaml:"names"`
	Scope    string       `yaml:"scope"`
	Versions []CRDVersion `yaml:"versions"`
}

// CRDNames contains the naming information for a CRD.
type CRDNames struct {
	Kind       string   `yaml:"kind"`
	ListKind   string   `yaml:"listKind"`
	Plural     string   `yaml:"plural"`
	Singular   string   `yaml:"singular"`
	ShortNames []string `yaml:"shortNames"`
	Categories []string `yaml:"categories"`
}

// CRDVersion represents a version of a CRD.
type CRDVersion struct {
	Name    string    `yaml:"name"`
	Served  bool      `yaml:"served"`
	Storage bool      `yaml:"storage"`
	Schema  CRDSchema `yaml:"schema"`
}

// CRDSchema contains the OpenAPI v3 schema for a CRD version.
type CRDSchema struct {
	OpenAPIV3Schema *OpenAPISchema `yaml:"openAPIV3Schema"`
}

// OpenAPISchema represents an OpenAPI v3 schema.
type OpenAPISchema struct {
	Type                 string                   `yaml:"type"`
	Format               string                   `yaml:"format"`
	Description          string                   `yaml:"description"`
	Properties           map[string]*OpenAPISchema `yaml:"properties"`
	Items                *OpenAPISchema           `yaml:"items"`
	AdditionalProperties *OpenAPISchema           `yaml:"additionalProperties"`
	Required             []string                 `yaml:"required"`
	Enum                 []string                 `yaml:"enum"`
	Default              interface{}              `yaml:"default"`
	Minimum              *float64                 `yaml:"minimum"`
	Maximum              *float64                 `yaml:"maximum"`
	MinLength            *int                     `yaml:"minLength"`
	MaxLength            *int                     `yaml:"maxLength"`
	Pattern              string                   `yaml:"pattern"`
	MinItems             *int                     `yaml:"minItems"`
	MaxItems             *int                     `yaml:"maxItems"`

	// Composition
	OneOf []OpenAPISchema `yaml:"oneOf"`
	AnyOf []OpenAPISchema `yaml:"anyOf"`
	AllOf []OpenAPISchema `yaml:"allOf"`

	// Kubernetes extensions
	XKubernetesPreserveUnknownFields *bool                    `yaml:"x-kubernetes-preserve-unknown-fields"`
	XKubernetesValidations           []KubernetesValidation   `yaml:"x-kubernetes-validations"`
	XKubernetesListMapKeys           []string                 `yaml:"x-kubernetes-list-map-keys"`
	XKubernetesListType              string                   `yaml:"x-kubernetes-list-type"`
	XKubernetesMapType               string                   `yaml:"x-kubernetes-map-type"`
	XKubernetesIntOrString           *bool                    `yaml:"x-kubernetes-int-or-string"`
}

// KubernetesValidation represents a CEL validation rule.
type KubernetesValidation struct {
	Rule    string `yaml:"rule"`
	Message string `yaml:"message"`
}

// IsRequired returns true if the given property name is in the required list.
func (s *OpenAPISchema) IsRequired(name string) bool {
	for _, r := range s.Required {
		if r == name {
			return true
		}
	}
	return false
}

// ParseCRDs parses a multi-document YAML containing CRDs.
// The standard-install.yaml file contains multiple CRDs separated by "---".
func ParseCRDs(data []byte) ([]CRD, error) {
	var crds []CRD

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var doc map[string]interface{}
		err := decoder.Decode(&doc)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decode YAML document: %w", err)
		}

		// Skip non-CRD documents (like namespace definitions)
		kind, ok := doc["kind"].(string)
		if !ok || kind != "CustomResourceDefinition" {
			continue
		}

		// Re-encode and decode into our CRD struct
		docBytes, err := yaml.Marshal(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to re-encode document: %w", err)
		}

		var crd CRD
		if err := yaml.Unmarshal(docBytes, &crd); err != nil {
			return nil, fmt.Errorf("failed to parse CRD: %w", err)
		}

		crds = append(crds, crd)
	}

	return crds, nil
}

// ParseCRDFile is a convenience function that parses a single CRD YAML file.
func ParseCRDFile(data []byte) (*CRD, error) {
	var crd CRD
	if err := yaml.Unmarshal(data, &crd); err != nil {
		return nil, fmt.Errorf("failed to parse CRD: %w", err)
	}
	return &crd, nil
}
