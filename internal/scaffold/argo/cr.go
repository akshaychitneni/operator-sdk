package argo

import (
	"fmt"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"log"
	"path/filepath"
	"strings"
	"text/template"
	argo_util "github.com/argoproj/argo/workflow/util"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"

)

// CR is the input needed to generate a deploy/crds/<full group>_<version>_<kind>_cr.yaml file
type CR struct {
	input.Input

	// Resource defines the inputs for the new custom resource
	Resource *scaffold.Resource

	// Spec is a custom spec for the CR. It will be automatically indented. If
	// unset, a default spec will be created.
	Spec string
	WorkflowParamMap map[string]string
	ArgoWorkflowPath string
}

func (s *CR) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = crPathForResource(scaffold.CRDsDir, s.Resource)
	}
	s.TemplateBody = crTemplate
	if s.TemplateFuncs == nil {
		s.TemplateFuncs = template.FuncMap{}
	}
	s.TemplateFuncs["indent"] = indent
	fileContents, err := argo_util.ReadManifest(s.ArgoWorkflowPath)
	if err != nil {
		log.Fatal(err)
	}

	var workflows []wfv1.Workflow
	for _, body := range fileContents {
		wfs := unmarshalWorkflows(body, true)
		workflows = append(workflows, wfs...)
	}
	if len(workflows) > 0 {
		//TODO: validate workflow
		s.WorkflowParamMap = map[string]string{}
		for _, param := range workflows[0].Spec.Arguments.Parameters {
			s.WorkflowParamMap[strings.Title(strings.ToLower(param.Name))] = *param.Value
		}

	}
	return s.Input, nil
}

func crPathForResource(dir string, r *scaffold.Resource) string {
	file := fmt.Sprintf("%s_%s_%s_cr.yaml", r.FullGroup, r.Version, r.LowerKind)
	return filepath.Join(dir, file)
}

func indent(spaces int, v string) string {
	pad := strings.Repeat(" ", spaces)
	return pad + strings.Replace(v, "\n", "\n"+pad, -1)
}

const crTemplate = `apiVersion: {{ .Resource.APIVersion }}
kind: {{ .Resource.Kind }}
metadata:
  name: example-{{ .Resource.LowerKind }}
spec:
{{- range $key, $value := .WorkflowParamMap }}
{{ $key | indent 2 }}: "{{ $value }}"
{{- end }}
`
