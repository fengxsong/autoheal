/*
Copyright (c) 2018 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file contains the definitions of the unversioned types used internally by the auto-heal
// service.

package autoheal

import (
	"slices"

	batch "k8s.io/api/batch/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/autoheal/pkg/apis/extension"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HealingRule is the description of an healing rule.
type HealingRule struct {
	meta.TypeMeta

	// Standard object metadata.
	// +optional
	meta.ObjectMeta

	// Labels is map containing the names of the labels and the regular expressions that they should
	// match in order to activate the rule.
	// +optional
	Labels map[string]string

	// Annotations is map containing the names of the annotations and the regular expressions that
	// they should match in order to activate the rule.
	// +optional
	Annotations map[string]string

	// A list of selector requirements by alert's labels.
	// +optional
	MatchExpressions []SelectorRequirement `json:"matchExpressions,omitempty"`

	// AWXJob is the AWX job that will be executed when the rule is activated.
	// +optional
	AWXJob *AWXJobAction

	// BatchJob is the batch job that will be executed when the rule is activated.
	// +optional
	BatchJob *batch.Job
}

// AWXJobAction describes how to run an Ansible AWX job.
type AWXJobAction struct {
	// Template is the name of the AWX job template that will be launched.
	// +optional
	Template string

	// ExtraVars are the extra variables that will be passed to job.
	// +optional
	ExtraVars *extension.AnyConfig

	// Limit is a pattern that will be passed to the job to constrain
	// the hosts that will be affected by the playbook.
	// +optional
	Limit string
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HealingRuleList is a list of healing rules.
type HealingRuleList struct {
	meta.TypeMeta
	meta.ListMeta

	Items []HealingRule
}

type SelectorOperator string

const (
	SelectorOpIn           SelectorOperator = "In"
	SelectorOpNotIn        SelectorOperator = "NotIn"
	SelectorOpExists       SelectorOperator = "Exists"
	SelectorOpDoesNotExist SelectorOperator = "DoesNotExist"
	SelectorOpGt           SelectorOperator = "Gt"
	SelectorOpLt           SelectorOperator = "Lt"
)

type SelectorRequirement struct {
	// The label key that the selector applies to.
	Key string `json:"key"`
	// Represents a key's relationship to a set of values.
	// Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
	Operator SelectorOperator `json:"operator"`
	// An array of string values. If the operator is In or NotIn,
	// the values array must be non-empty. If the operator is Exists or DoesNotExist,
	// the values array must be empty. If the operator is Gt or Lt, the values
	// array must have a single element, which will be interpreted as an integer.
	// This array is replaced during a strategic merge patch.
	// +optional
	Values []string `json:"values,omitempty"`
}

func (s SelectorRequirement) Match(labels map[string]string) bool {
	switch s.Operator {
	case SelectorOpIn:
		v, ok := labels[s.Key]
		return ok && slices.Contains(s.Values, v)
	case SelectorOpNotIn:
		v, ok := labels[s.Key]
		return !ok || !slices.Contains(s.Values, v)
	case SelectorOpExists:
		_, ok := labels[s.Key]
		return ok
	case SelectorOpGt:
	case SelectorOpLt:
	}
	return false
}
