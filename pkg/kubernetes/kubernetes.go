/*
Copyright 2023 The K8sGPT Authors.
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

package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *Client) GetConfig() *rest.Config {
	return c.Config
}

func (c *Client) GetClient() kubernetes.Interface {
	return c.Client
}

func (c *Client) GetCtrlClient() ctrl.Client {
	return c.CtrlClient
}

func NewClient(kubecontext string, kubeconfig string) (*Client, error) {
	var config *rest.Config
	config, err := rest.InClusterConfig()
	if kubeconfig != "" || err != nil {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()

		if kubeconfig != "" {
			loadingRules.ExplicitPath = kubeconfig
		}

		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules,
			&clientcmd.ConfigOverrides{
				CurrentContext: kubecontext,
			})
		// create the clientset
		config, err = clientConfig.ClientConfig()
		if err != nil {
			return nil, err
		}
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	ctrlClient, err := ctrl.New(config, ctrl.Options{})
	if err != nil {
		return nil, err
	}

	serverVersion, err := clientSet.ServerVersion()
	if err != nil {
		return nil, err
	}

	return &Client{
		Client:        clientSet,
		CtrlClient:    ctrlClient,
		Config:        config,
		ServerVersion: serverVersion,
	}, nil
}

const defaultNamespace = "default"

func (c *Client) ApplyManifest(completion string) error {

	dd, err := dynamic.NewForConfig(c.GetConfig())
	if err != nil {
		return err
	}
	cl, err := discovery.NewDiscoveryClientForConfig(c.GetConfig())
	if err != nil {
		return err
	}
	namespace := defaultNamespace
	preprocessCompletion(&completion)
	print(completion)
	manifest := []byte(completion)
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(manifest), 100)
	for {
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			// 如果ERROR 是EOF，那么就是没有需要apply的资源
			if strings.Contains(err.Error(), "EOF") {
				break
			}
			return fmt.Errorf("error decoding: %w", err)
		}

		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			return err
		}
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return err
		}

		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

		gr, err := restmapper.GetAPIGroupResources(cl)
		if err != nil {
			return err
		}

		mapper := restmapper.NewDiscoveryRESTMapper(gr)
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		var dri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace(namespace)
			}
			dri = dd.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = dd.Resource(mapping.Resource)
		}

		if _, err := dri.Apply(context.Background(), unstructuredObj.GetName(), unstructuredObj, metav1.ApplyOptions{FieldManager: "application/apply-patch"}); err != nil {
			return err
		}
	}
	return nil
}

func preprocessCompletion(completion *string) {
	// Remove markdown if present
	if strings.HasPrefix(*completion, "```") {
		// remove first line
		lines := strings.Split(*completion, "\n")
		*completion = strings.Join(lines[1:len(lines)-1], "\n")
	}
}
