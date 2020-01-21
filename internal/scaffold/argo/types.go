
package argo

import (
	"github.com/argoproj/argo/workflow/common"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"log"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	argo_util "github.com/argoproj/argo/workflow/util"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	argoJson "github.com/argoproj/pkg/json"
)

// Types is the input needed to generate a pkg/apis/<group>/<version>/<kind>_types.go file
type ArgoTypes struct {
	input.Input

	// Resource defines the inputs for the new types file
	Resource *scaffold.Resource
	ArgoWorkflowPath string
	WorkflowParams []string
}

func (s *ArgoTypes) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = filepath.Join(scaffold.ApisDir,
			s.Resource.GoImportGroup,
			strings.ToLower(s.Resource.Version),
			s.Resource.LowerKind+"_types.go")
	}

	fileContents, err := argo_util.ReadManifest(s.ArgoWorkflowPath)
	if err != nil {
		log.Fatal(err)
	}

	var workflows []wfv1.Workflow
	for _, body := range fileContents {
		wfs := unmarshalWorkflows(body, true)
		workflows = append(workflows, wfs...)
	}

	s.WorkflowParams = make([]string,0)

	if len(workflows) > 0 {
		for _, param := range workflows[0].Spec.Arguments.Parameters {
			s.WorkflowParams = append(s.WorkflowParams, strings.Title(strings.ToLower(param.Name)))
		}
	}
	// Error if this file exists.
	s.IfExistsAction = input.Error
	s.TemplateBody = typesTemplate
	return s.Input, nil
}

func unmarshalWorkflows(wfBytes []byte, strict bool) []wfv1.Workflow {
	var wf wfv1.Workflow
	var jsonOpts []argoJson.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, argoJson.DisallowUnknownFields)
	}
	err := argoJson.Unmarshal(wfBytes, &wf, jsonOpts...)
	if err == nil {
		return []wfv1.Workflow{wf}
	}
	yamlWfs, err := common.SplitWorkflowYAMLFile(wfBytes, strict)
	if err == nil {
		return yamlWfs
	}
	log.Fatalf("Failed to parse workflow: %v", err)
	return nil
}

const typesTemplate = `package {{ .Resource.Version }}

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// {{.Resource.Kind}}Spec defines the desired state of {{.Resource.Kind}}
type {{.Resource.Kind}}Spec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	{{range $val := .WorkflowParams}}
    	{{$val}} ` + "`" + `json:"{{$val}},omitempty"` + "`" + `
	{{end}}
}

// {{.Resource.Kind}}Status defines the observed state of {{.Resource.Kind}}
type {{.Resource.Kind}}Status struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	WorkflowName   string   ` + "`" + `json:"workflowName,omitempty"` + "`" + `
	WorkflowHash   string   ` + "`" + `json:"workflowHash,omitempty"` + "`" + `
	WorkflowStatus  string   ` + "`" + `json:"workflowStatus,omitempty"` + "`" + `
	CollisionCount   string   ` + "`" + `json:"collisionCount,omitempty"` + "`" + `
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// {{.Resource.Kind}} is the Schema for the {{ .Resource.Resource }} API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path={{.Resource.Resource}},scope=Namespaced
type {{.Resource.Kind}} struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
	metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

	Spec   {{.Resource.Kind}}Spec   ` + "`" + `json:"spec,omitempty"` + "`" + `
	Status {{.Resource.Kind}}Status ` + "`" + `json:"status,omitempty"` + "`" + `
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// {{.Resource.Kind}}List contains a list of {{.Resource.Kind}}
type {{.Resource.Kind}}List struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`" + `
	metav1.ListMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `
	Items           []{{ .Resource.Kind }} ` + "`" + `json:"items"` + "`" + `
}

func init() {
	SchemeBuilder.Register(&{{.Resource.Kind}}{}, &{{.Resource.Kind}}List{})
}
`
