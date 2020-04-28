// Copyright 2019 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package objectbucket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ghodss/yaml"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	chnv1alpha1 "github.com/open-cluster-management/multicloud-operators-channel/pkg/apis/apps/v1"
	dplv1alpha1 "github.com/open-cluster-management/multicloud-operators-deployable/pkg/apis/apps/v1"
	appv1alpha1 "github.com/open-cluster-management/multicloud-operators-subscription/pkg/apis/apps/v1"
)

var c client.Client

var id = types.NamespacedName{
	Name:      "endpoint",
	Namespace: "default",
}

var (
	sharedkey = types.NamespacedName{
		Name:      "test",
		Namespace: "default",
	}
	objchn = &chnv1alpha1.Channel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sharedkey.Name,
			Namespace: sharedkey.Namespace,
		},
		Spec: chnv1alpha1.ChannelSpec{},
	}
	objsub = &appv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sharedkey.Name,
			Namespace: sharedkey.Namespace,
		},
		Spec: appv1alpha1.SubscriptionSpec{
			Channel: sharedkey.String(),
		},
	}
	subitem = &appv1alpha1.SubscriberItem{
		Subscription: objsub,
		Channel:      objchn,
	}
	deployableStr = `apiVersion: apps.open-cluster-management.io/v1
kind: Deployable
metadata:
  annotations:
    apps.open-cluster-management.io/is-local-deployable: "false"
  name: sample-cr-configmap
  namespace: dev2
spec:
  template:
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: testconfigmap
      namespace: default
      labels:
        test: true
      annotations:
        key1: "false"
    data:
      purpose: for test`
)

// Handler handles connections to aws
type Handler struct {
	*s3.Client
}

func TestObjectSubscriber(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c = mgr.GetClient()

	g.Expect(Add(mgr, cfg, &id, 2)).NotTo(gomega.HaveOccurred())
	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	g.Expect(defaultSubscriber.SubscribeItem(subitem)).NotTo(gomega.HaveOccurred())

	time.Sleep(1 * time.Second)

	g.Expect(defaultSubscriber.UnsubscribeItem(sharedkey)).NotTo(gomega.HaveOccurred())
}

func TestDoSubscribeDeployable1(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	obsi := &SubscriberItem{}
	dpl := &dplv1alpha1.Deployable{}

	err := yaml.Unmarshal([]byte(deployableStr), &dpl)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	template := &unstructured.Unstructured{}
	err = json.Unmarshal(dpl.Spec.Template.Raw, template)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	labels := make(map[string]string)
	labels["test"] = "true"
	template.SetLabels(labels)

	annots := make(map[string]string)
	annots["key1"] = "val1"
	template.SetAnnotations(annots)

	dpl.Spec.Template.Raw, err = json.Marshal(template)

	pkgFilter := &appv1alpha1.PackageFilter{}
	pkgFilter.Version = "someversion"

	matchLabels := make(map[string]string)
	matchLabels["test"] = "true_nogood"
	lblSelector := &metav1.LabelSelector{}
	lblSelector.MatchLabels = matchLabels
	pkgFilter.LabelSelector = lblSelector

	annotations := make(map[string]string)
	annotations["key1"] = "val1"
	pkgFilter.Annotations = annotations
	objsub.Spec.PackageFilter = pkgFilter

	objsub.Spec.Package = "sample-cr-configmap"

	obsi.Subscription = objsub
	obsi.Channel = objchn

	_, _, err = obsi.doSubscribeDeployable(dpl, nil, nil)
	g.Expect(err).To(gomega.HaveOccurred())
}

func TestDoSubscribeDeployable2(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	obsi := &SubscriberItem{}
	dpl := &dplv1alpha1.Deployable{}

	err := yaml.Unmarshal([]byte(deployableStr), &dpl)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	template := &unstructured.Unstructured{}
	err = json.Unmarshal(dpl.Spec.Template.Raw, template)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	labels := make(map[string]string)
	labels["test"] = "true"
	template.SetLabels(labels)

	annots := make(map[string]string)
	annots["key1"] = "val1"
	template.SetAnnotations(annots)

	dpl.Spec.Template.Raw, err = json.Marshal(template)

	pkgFilter := &appv1alpha1.PackageFilter{}
	pkgFilter.Version = "someversion"

	matchLabels := make(map[string]string)
	matchLabels["test"] = "true"
	lblSelector := &metav1.LabelSelector{}
	lblSelector.MatchLabels = matchLabels
	pkgFilter.LabelSelector = lblSelector

	annotations := make(map[string]string)
	annotations["key1"] = "val1_nogood"
	pkgFilter.Annotations = annotations
	objsub.Spec.PackageFilter = pkgFilter

	objsub.Spec.Package = "sample-cr-configmap"

	obsi.Subscription = objsub
	obsi.Channel = objchn

	_, _, err = obsi.doSubscribeDeployable(dpl, nil, nil)
	g.Expect(err).To(gomega.HaveOccurred())
}

func TestDoSubscribeDeployable3(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	obsi := &SubscriberItem{}
	dpl := &dplv1alpha1.Deployable{}

	err := yaml.Unmarshal([]byte(deployableStr), &dpl)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	template := &unstructured.Unstructured{}
	err = json.Unmarshal(dpl.Spec.Template.Raw, template)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	labels := make(map[string]string)
	labels["test"] = "true"
	template.SetLabels(labels)

	annots := make(map[string]string)
	annots["key1"] = "val1"
	template.SetAnnotations(annots)

	dpl.Spec.Template.Raw, err = json.Marshal(template)

	pkgFilter := &appv1alpha1.PackageFilter{}

	matchLabels := make(map[string]string)
	matchLabels["test"] = "true"
	lblSelector := &metav1.LabelSelector{}
	lblSelector.MatchLabels = matchLabels
	pkgFilter.LabelSelector = lblSelector

	annotations := make(map[string]string)
	annotations["key1"] = "val1"
	pkgFilter.Annotations = annotations
	objsub.Spec.PackageFilter = pkgFilter

	objsub.Spec.Package = "sample-cr-configmap"

	obsi.Subscription = objsub
	obsi.Channel = objchn

	_, _, err = obsi.doSubscribeDeployable(dpl, nil, nil)
	g.Expect(err).To(gomega.HaveOccurred())
}
