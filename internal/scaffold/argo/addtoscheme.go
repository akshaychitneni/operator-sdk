package argo

import (
	"fmt"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"path/filepath"
	"strings"
)


// AddToScheme is the input needed to generate an addtoscheme_<group>_<version>.go file
type AddToScheme struct {
	input.Input

	// Resource defines the inputs for the new api
	Resource *scaffold.Resource
}

func (s *AddToScheme) GetInput() (input.Input, error) {
	if s.Path == "" {
		fileName := fmt.Sprintf("addtoscheme_%s_%s.go",
			s.Resource.GoImportGroup,
			strings.ToLower(s.Resource.Version))
		s.Path = filepath.Join(scaffold.ApisDir, fileName)
	}
	s.TemplateBody = addToSchemeTemplate
	return s.Input, nil
}

const addToSchemeTemplate = `package apis

import (
	"{{ .Repo }}/pkg/apis/{{ .Resource.GoImportGroup}}/{{ .Resource.Version }}"
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, {{ .Resource.Version }}.SchemeBuilder.AddToScheme, argov1.SchemeBuilder.AddToScheme)
}
`
