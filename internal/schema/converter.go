// Package schema provides conversion from OpenAPI v3 schemas to an internal model.
package schema

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bmurray/pkl-gateway-api/internal/crd"
)

// Resource represents a Gateway API resource.
type Resource struct {
	Group       string
	Kind        string
	APIVersion  string
	Scope       string
	Description string
	Properties  []Property
	Classes     []Class
}

// Property represents a property of a resource or class.
type Property struct {
	Name        string
	Type        Type
	Required    bool
	Description string
}

// Class represents a nested class within a resource.
type Class struct {
	Name        string
	Description string
	Properties  []Property
	Classes     []Class // Nested classes within this class
}

// Type represents a Pkl type.
type Type interface {
	PklType() string
}

// StringType represents a Pkl String.
type StringType struct{}

func (StringType) PklType() string { return "String" }

// IntType represents a Pkl Int.
type IntType struct{}

func (IntType) PklType() string { return "Int" }

// Int32Type represents a Pkl Int32.
type Int32Type struct{}

func (Int32Type) PklType() string { return "Int32" }

// BoolType represents a Pkl Boolean.
type BoolType struct{}

func (BoolType) PklType() string { return "Boolean" }

// DynamicType represents a Pkl Dynamic.
type DynamicType struct{}

func (DynamicType) PklType() string { return "Dynamic" }

// EnumType represents a Pkl enum - using String for compatibility with optional rendering.
// The allowed values are documented in comments.
type EnumType struct {
	Values []string
}

func (e EnumType) PklType() string {
	// Use String type for compatibility with optional properties and YAML rendering.
	// Allowed values: documented in property description
	return "String"
}

// ListingType represents a Pkl Listing<T>.
type ListingType struct {
	ElementType Type
}

func (l ListingType) PklType() string {
	return fmt.Sprintf("Listing<%s>", l.ElementType.PklType())
}

// MappingType represents a Pkl Mapping<String, T>.
type MappingType struct {
	ValueType Type
}

func (m MappingType) PklType() string {
	return fmt.Sprintf("Mapping<String, %s>", m.ValueType.PklType())
}

// ClassRefType represents a reference to a nested class.
type ClassRefType struct {
	ClassName string
}

func (c ClassRefType) PklType() string { return c.ClassName }

// OptionalType wraps a type to make it optional.
type OptionalType struct {
	Inner Type
}

func (o OptionalType) PklType() string {
	return o.Inner.PklType() + "?"
}

// IntOrStringType represents the Kubernetes IntOrString type.
type IntOrStringType struct{}

func (IntOrStringType) PklType() string { return "Int|String" }

// K8sObjectMetaType represents a reference to k8s ObjectMeta.
type K8sObjectMetaType struct{}

func (K8sObjectMetaType) PklType() string { return "ObjectMeta" }

// Converter converts CRDs to Resources.
type Converter struct {
	classCounter int
	classes      []Class
	seenClasses  map[string]bool // Track class names to avoid duplicates
}

// NewConverter creates a new Converter.
func NewConverter() *Converter {
	return &Converter{
		seenClasses: make(map[string]bool),
	}
}

// Convert converts a CRD and version to a Resource.
func (c *Converter) Convert(crdDef crd.CRD, version crd.CRDVersion) (*Resource, error) {
	c.classCounter = 0
	c.classes = nil
	c.seenClasses = make(map[string]bool)

	schema := version.Schema.OpenAPIV3Schema
	if schema == nil {
		return nil, fmt.Errorf("no OpenAPIV3Schema found for %s %s", crdDef.Spec.Names.Kind, version.Name)
	}

	// Get spec and status properties
	var properties []Property

	// Skip apiVersion, kind, metadata - they're handled by the base class
	for name, propSchema := range schema.Properties {
		if name == "apiVersion" || name == "kind" || name == "metadata" {
			continue
		}

		prop, err := c.convertProperty(name, propSchema, schema.IsRequired(name))
		if err != nil {
			return nil, fmt.Errorf("failed to convert property %s: %w", name, err)
		}
		properties = append(properties, prop)
	}

	// Sort properties for deterministic output
	sort.Slice(properties, func(i, j int) bool {
		// Put spec first, then status, then others alphabetically
		order := map[string]int{"spec": 0, "status": 1}
		oi, oki := order[properties[i].Name]
		oj, okj := order[properties[j].Name]
		if oki && okj {
			return oi < oj
		}
		if oki {
			return true
		}
		if okj {
			return false
		}
		return properties[i].Name < properties[j].Name
	})

	return &Resource{
		Group:       crdDef.Spec.Group,
		Kind:        crdDef.Spec.Names.Kind,
		APIVersion:  version.Name,
		Scope:       crdDef.Spec.Scope,
		Description: schema.Description,
		Properties:  properties,
		Classes:     c.classes,
	}, nil
}

func (c *Converter) convertProperty(name string, schema *crd.OpenAPISchema, required bool) (Property, error) {
	typ, err := c.convertType(name, schema)
	if err != nil {
		return Property{}, err
	}

	// Wrap in optional if not required
	if !required {
		typ = OptionalType{Inner: typ}
	}

	return Property{
		Name:        name,
		Type:        typ,
		Required:    required,
		Description: schema.Description,
	}, nil
}

func (c *Converter) convertType(name string, schema *crd.OpenAPISchema) (Type, error) {
	// Handle x-kubernetes-int-or-string
	if schema.XKubernetesIntOrString != nil && *schema.XKubernetesIntOrString {
		return IntOrStringType{}, nil
	}

	// Handle x-kubernetes-preserve-unknown-fields
	if schema.XKubernetesPreserveUnknownFields != nil && *schema.XKubernetesPreserveUnknownFields {
		return DynamicType{}, nil
	}

	// Handle enums
	if len(schema.Enum) > 0 {
		return EnumType{Values: schema.Enum}, nil
	}

	switch schema.Type {
	case "string":
		return StringType{}, nil

	case "integer":
		if schema.Format == "int32" {
			return Int32Type{}, nil
		}
		return IntType{}, nil

	case "boolean":
		return BoolType{}, nil

	case "array":
		if schema.Items == nil {
			return ListingType{ElementType: DynamicType{}}, nil
		}
		elemType, err := c.convertType(name+"Item", schema.Items)
		if err != nil {
			return nil, err
		}
		return ListingType{ElementType: elemType}, nil

	case "object":
		// Map type (additionalProperties without properties)
		if schema.AdditionalProperties != nil && len(schema.Properties) == 0 {
			valueType, err := c.convertType(name+"Value", schema.AdditionalProperties)
			if err != nil {
				return nil, err
			}
			return MappingType{ValueType: valueType}, nil
		}

		// Nested object with properties - create a class
		if len(schema.Properties) > 0 {
			className := toPklClassName(name)
			// Only add the class if we haven't seen it before
			if !c.seenClasses[className] {
				class, err := c.convertClass(className, schema)
				if err != nil {
					return nil, err
				}
				c.classes = append(c.classes, class)
				c.seenClasses[className] = true
			}
			return ClassRefType{ClassName: className}, nil
		}

		// Empty object or additionalProperties: true
		return DynamicType{}, nil

	default:
		// Unknown type, use Dynamic
		return DynamicType{}, nil
	}
}

func (c *Converter) convertClass(name string, schema *crd.OpenAPISchema) (Class, error) {
	return c.convertClassWithParent(name, schema)
}

func (c *Converter) convertClassWithParent(name string, schema *crd.OpenAPISchema) (Class, error) {
	var properties []Property
	var nestedClasses []Class

	for propName, propSchema := range schema.Properties {
		// For nested objects, we need to create nested classes with unique names
		// Use current class name as parent context
		typ, err := c.convertTypeForClass(name, propName, propSchema)
		if err != nil {
			return Class{}, fmt.Errorf("failed to convert property %s: %w", propName, err)
		}

		required := schema.IsRequired(propName)
		if !required {
			typ = OptionalType{Inner: typ}
		}

		properties = append(properties, Property{
			Name:        propName,
			Type:        typ,
			Required:    required,
			Description: propSchema.Description,
		})
	}

	// Sort properties alphabetically
	sort.Slice(properties, func(i, j int) bool {
		return properties[i].Name < properties[j].Name
	})

	return Class{
		Name:        name,
		Description: schema.Description,
		Properties:  properties,
		Classes:     nestedClasses,
	}, nil
}

func (c *Converter) convertTypeForClass(parentClass, propName string, schema *crd.OpenAPISchema) (Type, error) {
	// Handle x-kubernetes-int-or-string
	if schema.XKubernetesIntOrString != nil && *schema.XKubernetesIntOrString {
		return IntOrStringType{}, nil
	}

	// Handle x-kubernetes-preserve-unknown-fields
	if schema.XKubernetesPreserveUnknownFields != nil && *schema.XKubernetesPreserveUnknownFields {
		return DynamicType{}, nil
	}

	// Handle enums
	if len(schema.Enum) > 0 {
		return EnumType{Values: schema.Enum}, nil
	}

	switch schema.Type {
	case "string":
		return StringType{}, nil

	case "integer":
		if schema.Format == "int32" {
			return Int32Type{}, nil
		}
		return IntType{}, nil

	case "boolean":
		return BoolType{}, nil

	case "array":
		if schema.Items == nil {
			return ListingType{ElementType: DynamicType{}}, nil
		}
		// Use parent+propName for array items to maintain context
		elemType, err := c.convertTypeForClass(parentClass+toPklClassName(propName), "Item", schema.Items)
		if err != nil {
			return nil, err
		}
		return ListingType{ElementType: elemType}, nil

	case "object":
		// Map type
		if schema.AdditionalProperties != nil && len(schema.Properties) == 0 {
			valueType, err := c.convertTypeForClass(parentClass, propName+"Value", schema.AdditionalProperties)
			if err != nil {
				return nil, err
			}
			return MappingType{ValueType: valueType}, nil
		}

		// Nested object - create a class with parent context to avoid name collisions
		if len(schema.Properties) > 0 {
			className := toPklClassName(parentClass + toPklClassName(propName))
			// Only add the class if we haven't seen it before
			if !c.seenClasses[className] {
				class, err := c.convertClassWithParent(className, schema)
				if err != nil {
					return nil, err
				}
				c.classes = append(c.classes, class)
				c.seenClasses[className] = true
			}
			return ClassRefType{ClassName: className}, nil
		}

		return DynamicType{}, nil

	default:
		return DynamicType{}, nil
	}
}

// toPklClassName converts a property name to a Pkl class name.
func toPklClassName(name string) string {
	if len(name) == 0 {
		return name
	}
	// Capitalize first letter
	return strings.ToUpper(name[:1]) + name[1:]
}
