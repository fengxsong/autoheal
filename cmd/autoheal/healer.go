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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/syncmap"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"github.com/openshift/autoheal/pkg/alertmanager"
	"github.com/openshift/autoheal/pkg/apis/autoheal"
	"github.com/openshift/autoheal/pkg/awxrunner"
	"github.com/openshift/autoheal/pkg/batchrunner"
	"github.com/openshift/autoheal/pkg/config"
	"github.com/openshift/autoheal/pkg/generated/clientset/versioned"
	informers "github.com/openshift/autoheal/pkg/generated/informers/externalversions"
	"github.com/openshift/autoheal/pkg/memory"
	"github.com/openshift/autoheal/pkg/metrics"
	"github.com/openshift/autoheal/pkg/webhookrunner"
)

// HealerBuilder is used to create new healers.
type HealerBuilder struct {
	// Configuration files.
	configFiles []string

	// Kubernetes client.
	k8sClient    kubernetes.Interface
	healerClient versioned.Interface
}

// Healer contains the information needed to receive notifications about changes in the
// Prometheus configuration and to start or reload it when there are changes.
type Healer struct {
	// The configuration.
	config *config.Config
	// autoheal custom resource clientset
	client versioned.Interface

	// Kubernetes client.
	k8sClient kubernetes.Interface

	// The current set of healing rules.
	rulesCache *sync.Map

	// We use two queues, one to process updates to the rules and another to process incoming
	// notifications from the alert manager:
	rulesQueue  workqueue.RateLimitingInterface
	alertsQueue workqueue.RateLimitingInterface

	// Executed actions will be stored here in order to prevent repeated execution.
	actionMemory *memory.ShortTermMemory

	// a map of ActionRunner which run awx/batch/etc actions.
	actionRunners map[ActionRunnerType]ActionRunner

	informerFactory informers.SharedInformerFactory
	hrInformer      cache.SharedIndexInformer
}

// NewHealerBuilder creates a new builder for healers.
func NewHealerBuilder() *HealerBuilder {
	b := new(HealerBuilder)
	return b
}

// ConfigFile adds one configuration file.
func (b *HealerBuilder) ConfigFile(path string) *HealerBuilder {
	b.configFiles = append(b.configFiles, path)
	return b
}

// ConfigFiles adds one or more configuration files or directories. They will be loaded in the order
// given. For directories all the contained files will be loaded, in alphabetical order.
func (b *HealerBuilder) ConfigFiles(paths []string) *HealerBuilder {
	if len(paths) > 0 {
		for _, path := range paths {
			b.ConfigFile(path)
		}
	}
	return b
}

// KubernetesClient sets the Kubernetes client that will be used by the healer.
func (b *HealerBuilder) KubernetesClient(client kubernetes.Interface) *HealerBuilder {
	b.k8sClient = client
	return b
}

func (b *HealerBuilder) HealerClient(client versioned.Interface) *HealerBuilder {
	b.healerClient = client
	return b
}

// Build creates the healer using the configuration stored in the builder.
func (b *HealerBuilder) Build() (h *Healer, err error) {
	var cfg *config.Config

	// Create new config and load the configuration files:
	if len(b.configFiles) == 0 {
		err = fmt.Errorf("no configuration file has been provided")
		return
	}
	cfg, err = config.NewBuilder().
		Client(b.k8sClient).
		Files(b.configFiles).
		Build()
	if err != nil {
		return
	}

	// Create the actions memory:
	actionMemory, err := memory.NewShortTermMemoryBuilder().
		Duration(cfg.Throttling().Interval()).
		Build()
	if err != nil {
		return
	}

	// Allocate the healer:
	h = new(Healer)
	h.client = b.healerClient
	h.k8sClient = b.k8sClient
	h.config = cfg
	h.actionMemory = actionMemory

	// Initialize the map of rules:
	h.rulesCache = new(syncmap.Map)

	// Create the queues:
	h.rulesQueue = workqueue.NewRateLimitingQueueWithConfig(
		workqueue.DefaultControllerRateLimiter(),
		workqueue.RateLimitingQueueConfig{Name: "rules"},
	)
	h.alertsQueue = workqueue.NewRateLimitingQueueWithConfig(
		workqueue.DefaultControllerRateLimiter(),
		workqueue.RateLimitingQueueConfig{Name: "alerts"},
	)

	// allocate new action runners
	h.actionRunners = make(map[ActionRunnerType]ActionRunner)
	h.informerFactory = informers.NewSharedInformerFactory(h.client, 0)
	h.hrInformer = h.informerFactory.Autoheal().V1alpha2().HealingRules().Informer()
	h.hrInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { h.enqueue(watch.Added, obj) },
		UpdateFunc: func(_, newObj interface{}) { h.enqueue(watch.Modified, newObj) },
		DeleteFunc: func(obj interface{}) { h.enqueue(watch.Deleted, obj) },
	})

	return
}

// Run waits for the informers caches to sync, and then starts the workers and the web server.
func (h *Healer) Run(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer h.rulesQueue.ShutDown()
	defer h.alertsQueue.ShutDown()
	defer h.config.ShutDown()

	h.informerFactory.Start(stopCh)
	if ok := cache.WaitForCacheSync(stopCh, h.hrInformer.HasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	// Start the workers:
	go wait.Until(h.runRulesWorker, time.Second, stopCh)
	go wait.Until(h.runAlertsWorker, time.Second, stopCh)

	// initialize runners.
	awxRunner, err := awxrunner.NewBuilder().
		Config(h.config.AWX()).
		StopCh(stopCh).
		Build()

	if err != nil {
		klog.Warningf("Error building AWX runner: %s", err)
	} else {
		h.actionRunners[ActionRunnerTypeAWX] = awxRunner
	}

	batchRunner, err := batchrunner.NewBuilder().
		KubernetesClient(h.k8sClient).
		Build()

	if err != nil {
		klog.Warningf("Error building batch runner: %s", err)
	} else {
		h.actionRunners[ActionRunnerTypeBatch] = batchRunner
	}

	// TODO: create webhook runner with parameters, like http.Client, k8sclient..
	h.actionRunners[ActionRunnerTypeWebhook] = &webhookrunner.Runner{}

	klog.Info("Workers started")

	// Reload the rules cache.
	h.reloadRulesCache()

	// Add a listener that will reload the rules cache
	// on config object change.
	h.config.AddChangeListener(func(_ *config.ChangeEvent) {
		h.reloadRulesCache()
	})

	// Start the web server
	http.HandleFunc("/-/healthy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Healthy"))
	})
	http.Handle("/metrics", metrics.Handler())
	http.HandleFunc("/alerts", h.handleRequest)

	server := &http.Server{Addr: ":9099"}
	go server.ListenAndServe()
	klog.Info("Web server started")

	// Wait till we are requested to stop:
	<-stopCh

	// Shutdown the web server:
	err = server.Shutdown(context.TODO())
	if err != nil {
		return err
	}

	return nil
}

// Reload all rules in rules cache (by sending "Deleted" + "Added" to queue).
func (h *Healer) reloadRulesCache() {
	// Send Delete signal to all rules currently in rules cache:
	h.rulesCache.Range(func(key, value interface{}) bool {
		rule := value.(*autoheal.HealingRule)
		change := &RuleChange{
			Type: watch.Deleted,
			Rule: rule,
		}
		h.rulesQueue.Add(change)

		return true
	})

	// For each rule inside the configuration create a change and add it to the queue:
	rules := h.config.Rules()
	if len(rules) > 0 {
		for _, rule := range rules {
			change := &RuleChange{
				Type: watch.Added,
				Rule: rule,
			}
			h.rulesQueue.Add(change)
		}
		klog.Infof("Loaded %d healing rules from the configuration", len(rules))
	} else {
		klog.Warningf("There are no healing rules in the configuration")
	}
}

func (h *Healer) handleRequest(response http.ResponseWriter, request *http.Request) {
	// Read the request body:
	body, err := io.ReadAll(request.Body)
	if err != nil {
		klog.Warningf("Can't read request body: %s", err)
		http.Error(
			response,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest,
		)
		return
	}

	// Dump the request to the log:
	if klog.V(5).Enabled() {
		klog.Infof("Request body:\n%s", h.indent(body))
	}

	// Parse the JSON request body:
	message := new(alertmanager.Message)
	if err = json.Unmarshal(body, message); err != nil {
		klog.Warningf("Can't parse request body: %s", err)
		http.Error(
			response,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest,
		)
		return
	}

	// Handle the parsed message:
	h.handleMessage(message)
}

func (h *Healer) handleMessage(message *alertmanager.Message) {
	for _, alert := range message.Alerts {
		h.alertsQueue.AddRateLimited(alert)
	}
}

func (h *Healer) indent(data []byte) []byte {
	buffer := new(bytes.Buffer)
	err := json.Indent(buffer, data, "", "  ")
	if err != nil {
		return data
	}
	return buffer.Bytes()
}
