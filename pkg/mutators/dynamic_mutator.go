package mutators

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/logging"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/mutation/match"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/mutation/mutators/core"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/mutation/path/parser"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/mutation/schema"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/mutation/types"
	mutationtypes "github.com/open-policy-agent/gatekeeper/v3/pkg/mutation/types"
	"github.com/open-policy-agent/opa/rego"
	runtimeschema "k8s.io/apimachinery/pkg/runtime/schema"
	mutationsv1alpha1 "kubesphere.io/muato/api/mutations/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("mutation").WithValues(logging.Process, "mutation", logging.Mutator, "dynamic")

// Mutator is a mutator object built out of a Dynamic instance.
type Mutator struct {
	id      types.ID
	dynamic *mutationsv1alpha1.Dynamic
}

// Mutator implements mutatorWithSchema.
var _ mutationtypes.Mutator = &Mutator{}

func (m *Mutator) Matches(mutable *types.Mutable) (bool, error) {
	target := &match.Matchable{
		Object:    mutable.Object,
		Namespace: mutable.Namespace,
		Source:    mutable.Source,
	}
	matches, err := match.Matches(&m.dynamic.Spec.Match, target)
	if err != nil {
		return false, err
	}
	return matches, nil
}

func (m *Mutator) TerminalType() parser.NodeType {
	return schema.Unknown
}

func (m *Mutator) Mutate(mutable *types.Mutable) (bool, error) {
	evalQuery, err := rego.New(rego.Query(defaultRegoQuery), rego.Module(defaultRegoFileName, m.dynamic.Spec.Rego)).PrepareForEval(context.Background())
	if err != nil {
		log.Error(err, "Failed to prepare rego query", "mutator", m.id)
		return false, err
	}

	// The policy decision is contained in the results returned by the Eval() call. You can inspect the decision and handle it accordingly.
	results, err := evalQuery.Eval(context.Background(), rego.EvalInput(mutable.Object.Object))
	if err != nil {
		log.Error(err, "Failed to evaluate rego query", "mutator", m.id)
		return false, err
	}

	if len(results) > 0 && len(results[0].Expressions) > 0 {
		if content, ok := results[0].Expressions[0].Value.(map[string]interface{}); ok {
			input, _ := json.Marshal(mutable.Object)
			output, _ := json.Marshal(content)
			mutable.Object.SetUnstructuredContent(content)
			log.Info("Mutating object", "mutator", m.id, "input", string(input), "output", string(output))
			return true, nil
		}
	}
	return false, nil
}

func (m *Mutator) MustTerminate() bool {
	return true
}

func (m *Mutator) ID() types.ID {
	return m.id
}

func (m *Mutator) HasDiff(mutator types.Mutator) bool {
	toCheck, ok := mutator.(*Mutator)
	if !ok { // different types, different
		return true
	}
	if !cmp.Equal(toCheck.id, m.id) {
		return true
	}
	// any difference in spec may be enough
	if !cmp.Equal(toCheck.dynamic.Spec, m.dynamic.Spec) {
		return true
	}

	return false
}

func (m *Mutator) Path() parser.Path {
	return parser.Path{}
}

func (m *Mutator) DeepCopy() types.Mutator {
	res := &Mutator{
		id:      m.id,
		dynamic: m.dynamic.DeepCopy(),
	}
	return res
}

func (m *Mutator) String() string {
	return fmt.Sprintf("%s/%s/%s:%d", m.id.Kind, m.id.Namespace, m.id.Name, m.dynamic.GetGeneration())
}

const (
	defaultRegoQuery    = "data.mutating.modified"
	defaultRegoFileName = "mutating.rego"
)

// MutatorForDynamic returns a mutator built from the given dynamic instance.
func MutatorForDynamic(dynamic *mutationsv1alpha1.Dynamic) (*Mutator, error) {
	log.V(1).Info("Creating mutator", "dynamic", dynamic)
	if err := core.ValidateName(dynamic.Name); err != nil {
		return nil, err
	}
	// This is not always set by the kubernetes API server
	dynamic.SetGroupVersionKind(runtimeschema.GroupVersionKind{Group: mutationsv1alpha1.GroupVersion.Group, Kind: "Dynamic"})
	return &Mutator{
		id:      types.MakeID(dynamic),
		dynamic: dynamic.DeepCopy(),
	}, nil
}
