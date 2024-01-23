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

package main

import (
	"context"
	"reflect"
	"regexp"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"

	"github.com/openshift/autoheal/pkg/alertmanager"
	"github.com/openshift/autoheal/pkg/apis/autoheal"
	"github.com/openshift/autoheal/pkg/metrics"
)

func (h *Healer) runAlertsWorker() {
	for h.pickAlert() {
		// Nothing.
	}
}

func (h *Healer) pickAlert() bool {
	// Get the next item and end the work loop if asked to stop:
	item, stop := h.alertsQueue.Get()
	if stop {
		return false
	}

	// Process the item and make sure to always tell the queue that we are done with this item:
	err := func(item interface{}) error {
		h.alertsQueue.Done(item)

		// Check that the item we got from the queue is really an alert, and discard it otherwise:
		alert, ok := item.(*alertmanager.Alert)
		if !ok {
			h.alertsQueue.Forget(item)
		}

		// Process and then forget the alert:
		err := h.processAlert(alert)
		if err != nil {
			return err
		}
		h.alertsQueue.Forget(alert)

		return nil
	}(item)
	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

func (h *Healer) processAlert(alert *alertmanager.Alert) error {
	switch alert.Status {
	case alertmanager.AlertStatusFiring:
		return h.startHealing(context.Background(), alert)
	case alertmanager.AlertStatusResolved:
		return h.cancelHealing(context.Background(), alert)
	default:
		klog.Warningf(
			"Unknown status '%s' reported by alert manager, will ignore it",
			alert.Status,
		)
		return nil
	}
}

func (h *Healer) filterActivated(alert *alertmanager.Alert) []*autoheal.HealingRule {
	// Find the rules that are activated for the alert
	activated := make([]*autoheal.HealingRule, 0)
	h.rulesCache.Range(func(_, value interface{}) bool {
		rule := value.(*autoheal.HealingRule)
		matches, err := h.checkRule(rule, alert)
		if err != nil {
			klog.Errorf(
				"Error while checking if rule '%s' matches alert '%s': %s",
				rule.ObjectMeta.Name,
				alert.Name(),
				err,
			)
		} else if matches {
			klog.Infof(
				"Rule '%s' matches alert '%s'",
				rule.ObjectMeta.Name,
				alert.Name(),
			)
			activated = append(activated, rule)
		}
		return true
	})
	return activated
}

// startHealing starts the healing process for the given alert.
func (h *Healer) startHealing(ctx context.Context, alert *alertmanager.Alert) error {
	activated := h.filterActivated(alert)
	if len(activated) == 0 {
		klog.Infof("No rule matches alert '%s'", alert.Name())
		return nil
	}
	// Execute the activated rules
	for _, rule := range activated {
		err := h.runRule(ctx, rule, alert)
		if err != nil {
			return err
		}
	}

	return nil
}

// cancelHealing cancels the healing process for the given alert.
func (h *Healer) cancelHealing(_ context.Context, alert *alertmanager.Alert) error {
	return nil
}

func (h *Healer) checkRule(rule *autoheal.HealingRule, alert *alertmanager.Alert) (matches bool, err error) {
	klog.Infof(
		"Checking rule '%s' for alert '%s'",
		rule.ObjectMeta.Name,
		alert.Name(),
	)
	matches, err = h.checkMap(alert.Labels, rule.Labels)
	if !matches || err != nil {
		return
	}
	matches, err = h.checkMap(alert.Annotations, rule.Annotations)
	if !matches || err != nil {
		return
	}
	for i := range rule.MatchExpressions {
		match := rule.MatchExpressions[i].Match(alert.Labels)
		if !match {
			return false, nil
		}
	}
	return true, nil
}

func (h *Healer) checkMap(values, patterns map[string]string) (result bool, err error) {
	if len(patterns) > 0 {
		if len(values) == 0 {
			return
		}
		for key, pattern := range patterns {
			value, present := values[key]
			if !present {
				return
			}
			var matches bool
			matches, err = regexp.MatchString(pattern, value)
			if !matches || err != nil {
				return
			}
		}
	}
	result = true
	return
}

func (h *Healer) runRule(ctx context.Context, rule *autoheal.HealingRule, alert *alertmanager.Alert) error {
	klog.Infof(
		"Running rule '%s' for alert '%s'",
		rule.ObjectMeta.Name,
		alert.Name(),
	)

	rule = rule.DeepCopy()
	// Process the templates inside the action:
	template, err := NewObjectTemplateBuilder().
		Variable("alert", ".").
		Variable("labels", ".Labels").
		Variable("annotations", ".Annotations").
		Build()
	if err != nil {
		return err
	}
	err = template.Process(rule, alert)
	if err != nil {
		return err
	}

	// Execute the action
	// TODO: refactor to factory builder
	for _, runner := range h.actionRunners {
		action := runner.OnAction(rule, alert)
		// can't simply assert that action == nil cause action is a typed pointer
		if v := reflect.ValueOf(action); v.Kind() == reflect.Ptr && !v.IsNil() {
			// Increment the metric of requested heales.
			metrics.ActionRequested(
				reflect.TypeOf(action).Elem().Name(),
				rule.ObjectMeta.Name,
				alert.Labels["alertname"],
			)

			// Discard the action if it has been executed recently
			if h.actionMemory.Has(action) {
				klog.Infof(
					"Action for rule '%s' and alert '%s' has been executed recently, it will be ignored",
					rule.ObjectMeta.Name,
					alert.Name(),
				)
				return nil
			}

			if err = runner.RunAction(ctx, rule, alert); err != nil {
				return err
			}
			// Remember that the action was executed recently, even if the execution failed
			h.actionMemory.Add(action)
		}
	}

	return nil
}
