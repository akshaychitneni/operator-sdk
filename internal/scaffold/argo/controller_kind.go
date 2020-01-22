package argo

import (
	"fmt"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"log"
	"path"
	"path/filepath"
	"strings"
	"unicode"
	"encoding/json"
	argo_util "github.com/argoproj/argo/workflow/util"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)




// ControllerKind is the input needed to generate a pkg/controller/<kind>/<kind>_controller.go file
type ArgoControllerKind struct {
	input.Input

	// Resource defines the inputs for the controller's primary resource
	Resource *scaffold.Resource
	// CustomImport holds the import path for a built-in or custom Kubernetes
	// API that this controller reconciles, if specified by the scaffold invoker.
	CustomImport string

	// The following fields will be overwritten by GetInput().
	//
	// ImportMap maps all imports destined for the scaffold to their import
	// identifier, if any.
	ImportMap map[string]string
	// GoImportIdent is the import identifier for the API reconciled by this
	// controller.
	GoImportIdent string

	ArgoWorkflowPath string
	WorkflowJsonStr string
}

func (s *ArgoControllerKind) GetInput() (input.Input, error) {
	if s.Path == "" {
		fileName := s.Resource.LowerKind + "_controller.go"
		s.Path = filepath.Join(scaffold.ControllerDir, s.Resource.LowerKind, fileName)
	}
	// Error if this file exists.
	s.IfExistsAction = input.Error
	s.TemplateBody = controllerKindTemplate

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
		b, err := json.Marshal(workflows[0])
		if err != nil {
			log.Fatal(err)
		}
		s.WorkflowJsonStr = "`" + string(b) + "`"
	}

	// Set imports.
	if err := s.setImports(); err != nil {
		return input.Input{}, err
	}
	return s.Input, nil
}

func (s *ArgoControllerKind) setImports() (err error) {
	s.ImportMap = controllerKindImports
	importPath := ""
	if s.CustomImport != "" {
		importPath, s.GoImportIdent, err = getCustomAPIImportPathAndIdent(s.CustomImport)
		if err != nil {
			return err
		}
	} else {
		importPath = path.Join(s.Repo, "pkg", "apis", s.Resource.GoImportGroup, s.Resource.Version)
		s.GoImportIdent = s.Resource.GoImportGroup + s.Resource.Version
	}
	// Import identifiers must be unique within a file.
	for p, id := range s.ImportMap {
		if s.GoImportIdent == id && importPath != p {
			// Append "api" to the conflicting import identifier.
			s.GoImportIdent = s.GoImportIdent + "api"
			break
		}
	}
	s.ImportMap[importPath] = s.GoImportIdent
	return nil
}

func getCustomAPIImportPathAndIdent(m string) (p string, id string, err error) {
	sm := strings.Split(m, "=")
	for i, e := range sm {
		if i == 0 {
			p = strings.TrimSpace(e)
		} else if i == 1 {
			id = strings.TrimSpace(e)
		}
	}
	if p == "" {
		return "", "", fmt.Errorf(`custom import "%s" path is empty`, m)
	}
	if id == "" {
		if len(sm) == 2 {
			return "", "", fmt.Errorf(`custom import "%s" identifier is empty, remove "=" from passed string`, m)
		}
		sp := strings.Split(p, "/")
		if len(sp) > 1 {
			id = sp[len(sp)-2] + sp[len(sp)-1]
		} else {
			id = sp[0]
		}
		id = strings.ToLower(id)
	}
	idb := &strings.Builder{}
	// By definition, all package identifiers must be comprised of "_", unicode
	// digits, and/or letters.
	for _, r := range id {
		if unicode.IsDigit(r) || unicode.IsLetter(r) || r == '_' {
			if _, err := idb.WriteRune(r); err != nil {
				return "", "", err
			}
		}
	}
	return p, idb.String(), nil
}

var controllerKindImports = map[string]string{
	"k8s.io/api/core/v1":                                           "corev1",
	"k8s.io/apimachinery/pkg/api/errors":                           "",
	"k8s.io/apimachinery/pkg/apis/meta/v1":                         "metav1",
	"k8s.io/apimachinery/pkg/runtime":                              "",
	"k8s.io/apimachinery/pkg/types":                                "",
	"sigs.k8s.io/controller-runtime/pkg/client":                    "",
	"sigs.k8s.io/controller-runtime/pkg/controller":                "",
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil": "",
	"sigs.k8s.io/controller-runtime/pkg/handler":                   "",
	"sigs.k8s.io/controller-runtime/pkg/manager":                   "",
	"sigs.k8s.io/controller-runtime/pkg/reconcile":                 "",
	"sigs.k8s.io/controller-runtime/pkg/log":                       "logf",
	"sigs.k8s.io/controller-runtime/pkg/source":                    "",
	"k8s.io/kubernetes/pkg/controller":								"k8s_controller",
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1":			"argov1",
	"github.com/argoproj/pkg/json":									"argoJson",
}

const controllerKindTemplate = `package {{ .Resource.LowerKind }}

import (
	"context"

	{{range $p, $i := .ImportMap -}}
	{{$i}} "{{$p}}"
	{{end}}
)

var log = logf.Log.WithName("controller_{{ .Resource.LowerKind }}")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new {{ .Resource.Kind }} Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
    workflowJson := {{ .WorkflowJsonStr}}
	workflowJsonBytes := []byte(workflowJson)
	workflows := unmarshalWorkflows(workflowJsonBytes, true)
	if len(workflows) == 0 {
		return nil
	}
	return &ReconcileAppAService{client: mgr.GetClient(), scheme: mgr.GetScheme(), workflowTemplate: &workflows[0]}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("{{ .Resource.LowerKind }}-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource {{ .Resource.Kind }}
	err = c.Watch(&source.Kind{Type: &{{ .GoImportIdent }}.{{ .Resource.Kind }}{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &argov1.Workflow{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(workflow handler.MapObject) []reconcile.Request {
			workflowObj := workflow.Object.(*argov1.Workflow)
			if workflowObj.Status.Phase == argov1.NodeSucceeded || workflowObj.Status.Phase == argov1.NodeFailed {
				if cr, ok := workflowObj.Labels["cr"]; !ok {
					return nil
				} else {
					return []reconcile.Request{
						{NamespacedName: types.NamespacedName{
							Name: cr,
							Namespace: workflowObj.Namespace,
						}},
					}
				}
			}
			return nil
		}),
	})

	return nil
}

// blank assignment to verify that Reconcile{{ .Resource.Kind }} implements reconcile.Reconciler
var _ reconcile.Reconciler = &Reconcile{{ .Resource.Kind }}{}

// Reconcile{{ .Resource.Kind }} reconciles a {{ .Resource.Kind }} object
type Reconcile{{ .Resource.Kind }} struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	workflowTemplate *argov1.Workflow
}

// Reconcile reads that state of the cluster for a {{ .Resource.Kind }} object and makes changes based on the state read
// and what is in the {{ .Resource.Kind }}.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *Reconcile{{ .Resource.Kind }}) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling {{ .Resource.Kind }}")

	// Fetch the {{ .Resource.Kind }} instance
	instance := &{{ .GoImportIdent }}.{{ .Resource.Kind }}{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.Status.WorkflowHash == computeHash(instance) {
		workflow := &argov1.Workflow{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Status.WorkflowName, Namespace: instance.Namespace}, workflow)
		if err != nil && errors.IsNotFound(err) {
			err = r.creatNewWorkflow(instance)
			if err != nil {
				return reconcile.Result{}, err
			} else {
				return reconcile.Result{}, nil
			}
		} else if err != nil {
			return reconcile.Result{}, err
		}
		if workflow.Status.Phase == argov1.NodePending || workflow.Status.Phase == argov1.NodeRunning {
			return reconcile.Result{ RequeueAfter: 1 * time.Minute }, nil
		} else if workflow.Status.Phase == argov1.NodeSucceeded {
			return reconcile.Result{}, nil
		} else if workflow.Status.Phase == argov1.NodeError ||  workflow.Status.Phase == argov1. NodeFailed {
			return reconcile.Result{}, fmt.Errorf("Failed workflow %s", workflow.Name)
		}
	} else {
		err = r.creatNewWorkflow(instance)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func (r *Reconcile{{ .Resource.Kind }}) newWorkflowForCR(cr *{{ .GoImportIdent }}.{{ .Resource.Kind }}) (w *argov1.Workflow) {
	params := make([]argov1.Parameter,0)
	vals := getSpecValsUsingReflection(cr)
	for k, v := range vals {
		params = append(params, argov1.Parameter{Name: k, Value: &v})
	}
	w = r.workflowTemplate.DeepCopy()
	w.Name = fmt.Sprintf("%s-workflow-%s", cr.Name, computeHash(cr))
	w.Labels = map[string]string{"cr": cr.Name}
	w.Namespace = cr.Namespace
	w.Spec.Arguments = argov1.Arguments{Parameters: params}
	return
}

func (r *Reconcile{{ .Resource.Kind }}) creatNewWorkflow(instance *{{ .GoImportIdent }}.{{ .Resource.Kind }}) error {
	if instance.Status.CollisionCount == nil {
		instance.Status.CollisionCount = new(int32)
	} else{
		*instance.Status.CollisionCount++
	}
	workflow := r.newWorkflowForCR(instance)
	if err := controllerutil.SetControllerReference(instance, workflow, r.scheme); err != nil {
		return err
	}
	err := r.client.Create(context.TODO(), workflow)
	if err != nil {
		return err
	}
	instance.Status.WorkflowHash = computeHash(instance)
	instance.Status.WorkflowName = workflow.Name
	err = r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		return err
	}
	return nil
}

func computeHash(cr *{{ .GoImportIdent }}.{{ .Resource.Kind }}) string {
    labels := getSpecValsUsingReflection(cr)
	hash := k8s_controller.ComputeHash(&corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}}, cr.Status.CollisionCount)
	return hash
}

func getSpecValsUsingReflection(cr *{{ .GoImportIdent }}.{{ .Resource.Kind }}) map[string]string {
	vals := make(map[string]string)
	val := reflect.ValueOf(cr).Elem()
	specVal := val.FieldByName("Spec").Interface().({{ .GoImportIdent }}.{{ .Resource.Kind }}Spec)
	rspecVal := reflect.ValueOf(&specVal).Elem()
	for i := 0; i < rspecVal.NumField(); i++ {
		valueField := rspecVal.Field(i)
		value := valueField.Interface().(string)
		typeField := rspecVal.Type().Field(i)
		vals[typeField.Name] = strings.ToLower(value)
	}
	return vals
}

func unmarshalWorkflows(wfBytes []byte, strict bool) []argov1.Workflow {
	var wf argov1.Workflow
	var jsonOpts []argoJson.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, argoJson.DisallowUnknownFields)
	}
	err := argoJson.Unmarshal(wfBytes, &wf, jsonOpts...)
	if err == nil {
		return []argov1.Workflow{wf}
	}
	yamlWfs, err := common.SplitWorkflowYAMLFile(wfBytes, strict)
	if err == nil {
		return yamlWfs
	}
	return nil
}
`
