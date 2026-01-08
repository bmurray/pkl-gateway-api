// Command generate fetches Gateway API CRDs from GitHub and generates Pkl modules.
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkl-community/pkl-gateway-api/internal/crd"
	"github.com/pkl-community/pkl-gateway-api/internal/generator"
	"github.com/pkl-community/pkl-gateway-api/internal/schema"
)

func main() {
	version := flag.String("version", "v1.4.1", "Gateway API version to fetch")
	k8sVersion := flag.String("k8s-version", "1.3.0", "pkl-k8s version to depend on")
	outputDir := flag.String("output", "generated-package", "Output directory for generated files")
	templatesDir := flag.String("templates", "templates", "Directory containing base templates")
	experimental := flag.Bool("experimental", false, "Include experimental resources")
	flag.Parse()

	if err := run(*version, *k8sVersion, *outputDir, *templatesDir, *experimental); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(version, k8sVersion, outputDir, templatesDir string, experimental bool) error {
	fmt.Printf("Generating Pkl modules for Gateway API %s\n", version)

	// Ensure output directory exists and is clean
	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("failed to clean output directory: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Fetch CRDs
	fmt.Printf("Fetching standard CRDs from GitHub...\n")
	standardData, err := fetchCRDs(version, "standard")
	if err != nil {
		return fmt.Errorf("failed to fetch standard CRDs: %w", err)
	}

	var allCRDs []crd.CRD

	// Parse standard CRDs
	crds, err := crd.ParseCRDs(standardData)
	if err != nil {
		return fmt.Errorf("failed to parse standard CRDs: %w", err)
	}
	allCRDs = append(allCRDs, crds...)
	fmt.Printf("  Parsed %d standard CRDs\n", len(crds))

	// Optionally fetch experimental CRDs
	if experimental {
		fmt.Printf("Fetching experimental CRDs from GitHub...\n")
		experimentalData, err := fetchCRDs(version, "experimental")
		if err != nil {
			return fmt.Errorf("failed to fetch experimental CRDs: %w", err)
		}

		crds, err := crd.ParseCRDs(experimentalData)
		if err != nil {
			return fmt.Errorf("failed to parse experimental CRDs: %w", err)
		}
		// Filter out duplicates (experimental includes standard)
		seen := make(map[string]bool)
		for _, c := range allCRDs {
			seen[c.Spec.Names.Kind] = true
		}
		for _, c := range crds {
			if !seen[c.Spec.Names.Kind] {
				allCRDs = append(allCRDs, c)
			}
		}
		fmt.Printf("  Parsed %d experimental CRDs (after dedup)\n", len(crds)-len(allCRDs)+len(crds))
	}

	// Convert CRDs to Resources and generate Pkl modules
	gen := generator.NewGenerator(outputDir, strings.TrimPrefix(version, "v")+"0", k8sVersion)
	converter := schema.NewConverter()

	var allResources []*schema.Resource
	for _, crdDef := range allCRDs {
		for _, ver := range crdDef.Spec.Versions {
			if !ver.Served {
				continue
			}

			fmt.Printf("Generating %s %s...\n", crdDef.Spec.Names.Kind, ver.Name)

			resource, err := converter.Convert(crdDef, ver)
			if err != nil {
				return fmt.Errorf("failed to convert %s %s: %w", crdDef.Spec.Names.Kind, ver.Name, err)
			}

			if err := gen.GenerateResource(resource); err != nil {
				return fmt.Errorf("failed to generate %s %s: %w", crdDef.Spec.Names.Kind, ver.Name, err)
			}

			allResources = append(allResources, resource)
		}
	}

	// Generate schema file
	fmt.Printf("Generating gatewaySchema.pkl...\n")
	if err := gen.GenerateSchema(allResources); err != nil {
		return fmt.Errorf("failed to generate schema: %w", err)
	}

	// Copy base templates
	fmt.Printf("Copying base templates...\n")
	absTemplatesDir, err := filepath.Abs(templatesDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for templates: %w", err)
	}
	if err := gen.CopyBaseTemplates(absTemplatesDir); err != nil {
		return fmt.Errorf("failed to copy base templates: %w", err)
	}

	fmt.Printf("Done! Generated files in %s\n", outputDir)
	return nil
}

func fetchCRDs(version, channel string) ([]byte, error) {
	url := fmt.Sprintf(
		"https://github.com/kubernetes-sigs/gateway-api/releases/download/%s/%s-install.yaml",
		version, channel,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, url)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}
