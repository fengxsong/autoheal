package webhookrunner

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	"k8s.io/klog/v2"

	"github.com/openshift/autoheal/pkg/alertmanager"
	"github.com/openshift/autoheal/pkg/apis/autoheal"
)

type Runner struct {
	client *http.Client
}

func (r *Runner) OnAction(rule *autoheal.HealingRule, _ *alertmanager.Alert) any {
	return rule.Webhook
}

func (r *Runner) RunAction(ctx context.Context, rule *autoheal.HealingRule, alert *alertmanager.Alert) error {
	klog.Infof("Trigger webhook to heal alert '%s'", alert.Labels["alertname"])
	action := rule.Webhook.DeepCopy()
	if r.client == nil {
		r.client = http.DefaultClient
	}
	if action.Proxy != "" {
		proxy, err := url.Parse(action.URL)
		if err != nil {
			return err
		}
		r.client.Transport = &http.Transport{Proxy: http.ProxyURL(proxy)}
		// restore to default
		defer func() {
			r.client.Transport = nil
		}()
	}
	method := http.MethodPost
	if action.Method != "" {
		method = action.Method
	}
	klog.V(3).InfoS("http request", "method", method, "url", action.URL, "body", action.Template)
	req, err := http.NewRequest(method, action.URL, bytes.NewBuffer([]byte(action.Template)))
	if err != nil {
		return err
	}
	if action.Headers != nil {
		for k, v := range action.Headers {
			for i := range v {
				req.Header.Add(k, v[i])
			}
		}
	}
	if auth := action.BasicAuth; auth != nil {
		req.SetBasicAuth(auth.Username, auth.Password)
	}
	req = req.WithContext(ctx)
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()
	if resp.StatusCode/100 > 3 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		klog.Errorf("unexpected webhook response: %s %v", resp.Status, string(body))
	}
	return nil
}
