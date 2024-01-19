//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by defaulter-gen. DO NOT EDIT.

package v1alpha2

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&HealingRule{}, func(obj interface{}) { SetObjectDefaults_HealingRule(obj.(*HealingRule)) })
	return nil
}

func SetObjectDefaults_HealingRule(in *HealingRule) {
	if in.BatchJob != nil {
		for i := range in.BatchJob.Template.Spec.InitContainers {
			a := &in.BatchJob.Template.Spec.InitContainers[i]
			for j := range a.Ports {
				b := &a.Ports[j]
				if b.Protocol == "" {
					b.Protocol = "TCP"
				}
			}
			if a.LivenessProbe != nil {
				if a.LivenessProbe.ProbeHandler.GRPC != nil {
					if a.LivenessProbe.ProbeHandler.GRPC.Service == nil {
						var ptrVar1 string = ""
						a.LivenessProbe.ProbeHandler.GRPC.Service = &ptrVar1
					}
				}
			}
			if a.ReadinessProbe != nil {
				if a.ReadinessProbe.ProbeHandler.GRPC != nil {
					if a.ReadinessProbe.ProbeHandler.GRPC.Service == nil {
						var ptrVar1 string = ""
						a.ReadinessProbe.ProbeHandler.GRPC.Service = &ptrVar1
					}
				}
			}
			if a.StartupProbe != nil {
				if a.StartupProbe.ProbeHandler.GRPC != nil {
					if a.StartupProbe.ProbeHandler.GRPC.Service == nil {
						var ptrVar1 string = ""
						a.StartupProbe.ProbeHandler.GRPC.Service = &ptrVar1
					}
				}
			}
		}
		for i := range in.BatchJob.Template.Spec.Containers {
			a := &in.BatchJob.Template.Spec.Containers[i]
			for j := range a.Ports {
				b := &a.Ports[j]
				if b.Protocol == "" {
					b.Protocol = "TCP"
				}
			}
			if a.LivenessProbe != nil {
				if a.LivenessProbe.ProbeHandler.GRPC != nil {
					if a.LivenessProbe.ProbeHandler.GRPC.Service == nil {
						var ptrVar1 string = ""
						a.LivenessProbe.ProbeHandler.GRPC.Service = &ptrVar1
					}
				}
			}
			if a.ReadinessProbe != nil {
				if a.ReadinessProbe.ProbeHandler.GRPC != nil {
					if a.ReadinessProbe.ProbeHandler.GRPC.Service == nil {
						var ptrVar1 string = ""
						a.ReadinessProbe.ProbeHandler.GRPC.Service = &ptrVar1
					}
				}
			}
			if a.StartupProbe != nil {
				if a.StartupProbe.ProbeHandler.GRPC != nil {
					if a.StartupProbe.ProbeHandler.GRPC.Service == nil {
						var ptrVar1 string = ""
						a.StartupProbe.ProbeHandler.GRPC.Service = &ptrVar1
					}
				}
			}
		}
		for i := range in.BatchJob.Template.Spec.EphemeralContainers {
			a := &in.BatchJob.Template.Spec.EphemeralContainers[i]
			for j := range a.EphemeralContainerCommon.Ports {
				b := &a.EphemeralContainerCommon.Ports[j]
				if b.Protocol == "" {
					b.Protocol = "TCP"
				}
			}
			if a.EphemeralContainerCommon.LivenessProbe != nil {
				if a.EphemeralContainerCommon.LivenessProbe.ProbeHandler.GRPC != nil {
					if a.EphemeralContainerCommon.LivenessProbe.ProbeHandler.GRPC.Service == nil {
						var ptrVar1 string = ""
						a.EphemeralContainerCommon.LivenessProbe.ProbeHandler.GRPC.Service = &ptrVar1
					}
				}
			}
			if a.EphemeralContainerCommon.ReadinessProbe != nil {
				if a.EphemeralContainerCommon.ReadinessProbe.ProbeHandler.GRPC != nil {
					if a.EphemeralContainerCommon.ReadinessProbe.ProbeHandler.GRPC.Service == nil {
						var ptrVar1 string = ""
						a.EphemeralContainerCommon.ReadinessProbe.ProbeHandler.GRPC.Service = &ptrVar1
					}
				}
			}
			if a.EphemeralContainerCommon.StartupProbe != nil {
				if a.EphemeralContainerCommon.StartupProbe.ProbeHandler.GRPC != nil {
					if a.EphemeralContainerCommon.StartupProbe.ProbeHandler.GRPC.Service == nil {
						var ptrVar1 string = ""
						a.EphemeralContainerCommon.StartupProbe.ProbeHandler.GRPC.Service = &ptrVar1
					}
				}
			}
		}
	}
	if in.DisableForResolvedMessage == nil {
		var ptrVar1 bool = true
		in.DisableForResolvedMessage = &ptrVar1
	}
}
