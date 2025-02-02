package provider

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"sort"
	"time"

	"github.com/GoogleCloudPlatform/magic-modules/mmv1/api"
	"github.com/GoogleCloudPlatform/magic-modules/mmv1/google"
)

type TFLint struct {
	Terraform
}

func NewTFLint(product *api.Product, versionName string, startTime time.Time) TFLint {
	return TFLint{
		Terraform: NewTerraform(product, versionName, startTime),
	}
}

func (t TFLint) Generate(outputFolder, productPath, resourceToGenerate string, generateCode, generateDocs bool) {
	if err := os.MkdirAll(outputFolder, os.ModePerm); err != nil {
		log.Println(fmt.Errorf("error creating output directory %v: %v", outputFolder, err))
	}

	t.GenerateObjects(outputFolder, resourceToGenerate, generateCode, generateDocs)
}

func (t *TFLint) GenerateObjects(outputFolder, resourceToGenerate string, generateCode, generateDocs bool) {
	for _, object := range t.Product.Objects {
		object.ExcludeIfNotInVersion(&t.Version)

		if resourceToGenerate != "" && object.Name != resourceToGenerate {
			log.Printf("Excluding %s per user request", object.Name)
			continue
		}

		t.GenerateObject(*object, outputFolder, t.TargetVersionName, generateCode, generateDocs)
	}
}

func (t *TFLint) GenerateObject(object api.Resource, outputFolder, productPath string, generateCode, generateDocs bool) {
	templateData := NewTemplateData(outputFolder, t.TargetVersionName)

	if !object.IsExcluded() {
		log.Printf("Generating %s rules", object.Name)
		t.GenerateResource(object, *templateData, outputFolder, generateCode, generateDocs)
	}
}

func (t *TFLint) GenerateResource(object api.Resource, templateData TemplateData, outputFolder string, generateCode, generateDocs bool) {
	for _, prop := range object.AllUserProperties() {
		if !t.isTargetProp(prop) {
			continue
		}

		ruleName := t.generateRuleName(object, prop)
		targetFilePath := path.Join(outputFolder, fmt.Sprintf("%s.go", ruleName))

		input := struct {
			RuleName      string
			Rule_name     string
			ResourceType  string
			AttributeName string
			Prop          *api.Type
		}{
			RuleName:      google.Camelize(ruleName, "upper"),
			Rule_name:     ruleName,
			ResourceType:  object.TerraformName(),
			AttributeName: google.Underscore(prop.Name),
			Prop:          prop,
		}
		templateData.GenerateFile(targetFilePath, "templates/tflint/rule.go.tmpl", input, true, "templates/tflint/rule.go.tmpl")
	}
}

func (t *TFLint) isTargetProp(prop *api.Type) bool {
	if prop.Output {
		return false
	}
	if prop.IsA("Enum") {
		return true
	}
	if (prop.IsA("String") || prop.IsA("Integer")) && prop.Validation.Regex != "" {
		return true
	}
	return false
}

func (t *TFLint) generateRuleName(object api.Resource, prop *api.Type) string {
	return fmt.Sprintf("%s_invalid_%s", object.TerraformName(), google.Underscore(prop.Name))
}

func (t TFLint) CopyCommonFiles(outputFolder string, generateCode, generateDocs bool) {
	log.Printf("Copying common files for %s", ProviderName(t))

	files := map[string]string{
		"verify/validation.go": "third_party/terraform/verify/validation.go",
	}
	t.CopyFileList(outputFolder, files, generateCode)
}

func (t TFLint) CompileCommonFiles(outputFolder string, products []*api.Product, overridePath string) {
	templateData := NewTemplateData(outputFolder, t.TargetVersionName)

	type resourceURL struct {
		Name string
		URL  string
	}
	input := struct {
		RuleNames    []string
		ResourceURLs []resourceURL
	}{
		RuleNames:    []string{},
		ResourceURLs: []resourceURL{},
	}

	for _, product := range products {
		var productURL string
		u, err := url.Parse(product.BaseUrl)
		if err != nil {
			log.Println(fmt.Errorf("cannot parse product.BaseUrl: %v", err))
		}
		if u != nil {
			productURL = u.Host
		}

		for _, object := range product.Objects {
			object.ExcludeIfNotInVersion(&t.Version)
			if object.IsExcluded() {
				continue
			}

			if productURL != "" {
				input.ResourceURLs = append(input.ResourceURLs, resourceURL{Name: object.TerraformName(), URL: productURL})
			}

			for _, prop := range object.AllUserProperties() {
				if !t.isTargetProp(prop) {
					continue
				}

				ruleName := t.generateRuleName(*object, prop)
				input.RuleNames = append(input.RuleNames, google.Camelize(ruleName, "upper"))
			}
		}
	}
	sort.Strings(input.RuleNames)
	sort.Slice(input.ResourceURLs, func(i, j int) bool { return input.ResourceURLs[i].Name < input.ResourceURLs[j].Name })

	templateData.GenerateFile(
		path.Join(outputFolder, "provider.go"),
		"templates/tflint/provider.go.tmpl",
		input,
		true,
		"templates/tflint/provider.go.tmpl",
	)
	templateData.GenerateFile(
		path.Join(outputFolder, "api_definition.go"),
		"templates/tflint/api_definition.go.tmpl",
		input,
		true,
		"templates/tflint/api_definition.go.tmpl",
	)
}
