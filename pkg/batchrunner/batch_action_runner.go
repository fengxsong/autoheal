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

package batchrunner

import (
	"context"
	"fmt"

	alertmanager "github.com/openshift/autoheal/pkg/alertmanager"
	"github.com/openshift/autoheal/pkg/apis/autoheal"
	batch "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type Builder struct {
	k8sClient kubernetes.Interface
}

type Runner struct {
	k8sClient kubernetes.Interface
}

func NewBuilder() *Builder {
	return new(Builder)
}

func (b *Builder) KubernetesClient(k8sClient kubernetes.Interface) *Builder {
	b.k8sClient = k8sClient
	return b
}

func (b *Builder) Build() (*Runner, error) {
	runner := &Runner{
		k8sClient: b.k8sClient,
	}
	return runner, nil
}

func (r *Runner) RunAction(rule *autoheal.HealingRule, action interface{}, alert *alertmanager.Alert) error {
	batchJob := action.(*batch.Job)

	klog.Infof(
		"Running batch job '%s' to heal alert '%s'",
		batchJob.ObjectMeta.Name,
		alert.Labels["alertname"],
	)

	// The name of the job is mandatory:
	name := batchJob.ObjectMeta.Name
	generateName := batchJob.ObjectMeta.GenerateName
	if name == "" && generateName == "" {
		return fmt.Errorf(
			"can't create job for rule '%s', the name or generateName hasn't been specified",
			rule.ObjectMeta.Name,
		)
	}

	// The namespace of the job is optional, the default is the namespace of the rule:
	namespace := batchJob.ObjectMeta.Namespace
	if namespace == "" {
		namespace = rule.ObjectMeta.Namespace
	}

	// Get the resource that manages the collection of batch jobs:
	resource := r.k8sClient.BatchV1().Jobs(namespace)

	// Try to create the job:
	batchJob = batchJob.DeepCopy()
	batchJob.ObjectMeta.Name = name
	batchJob.ObjectMeta.GenerateName = generateName
	batchJob.ObjectMeta.Namespace = namespace
	_, err := resource.Create(context.Background(), batchJob, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		klog.Warningf(
			"Batch job '%s' already exists, will do nothing to heal alert '%s'",
			batchJob.ObjectMeta.Name,
			alert.Labels["alertname"],
		)
	} else if err != nil {
		return err
	} else {
		klog.Infof(
			"Batch job '%s' to heal alert '%s' has been created",
			batchJob.ObjectMeta.Name,
			alert.Labels["alertname"],
		)
	}

	return nil
}
