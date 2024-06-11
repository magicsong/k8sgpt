/*
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

package manifest

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/k8sgpt-ai/k8sgpt/pkg/ai"
	"github.com/k8sgpt-ai/k8sgpt/pkg/cache"
	"github.com/k8sgpt-ai/k8sgpt/pkg/common"
	"github.com/k8sgpt-ai/k8sgpt/pkg/kubernetes"
	"github.com/k8sgpt-ai/k8sgpt/pkg/util"
	"github.com/spf13/viper"
)

type Manifester struct {
	Context        context.Context
	AIClient       ai.IAI
	Cache          cache.ICache
	Results        []common.Result
	Errors         []string
	MaxConcurrency int
	Namespace      string
	Language       string
	KubeClient     *kubernetes.Client
}

type (
	ManifestStatus string
	ManifestErrors []string
)

const (
	StatusGenerated ManifestStatus = "Generated"
	StatusFailed    ManifestStatus = "Failed"
)

type YamlOutput struct {
	Status ManifestStatus `json:"status"`
	Errors ManifestErrors `json:"errors"`
	Yaml   string         `json:"yaml"`
}

func NewManifester(namespace string, maxConcurrency int, noCache bool) (*Manifester, error) {
	var configAI ai.AIConfiguration
	if err := viper.UnmarshalKey("ai", &configAI); err != nil {
		return nil, err
	}

	if len(configAI.Providers) == 0 {
		return nil, errors.New("AI provider not specified in configuration. Please run the setup")
	}

	defaultProvider := configAI.DefaultProvider
	if defaultProvider == "" {
		return nil, errors.New("default AI provider not specified")
	}

	var aiProvider ai.AIProvider
	for _, provider := range configAI.Providers {
		if defaultProvider == provider.Name {
			aiProvider = provider
			break
		}
	}

	if aiProvider.Name == "" {
		return nil, fmt.Errorf("AI provider %s not found in configuration", defaultProvider)
	}

	aiClient := ai.NewClient(aiProvider.Name)
	if err := aiClient.Configure(&aiProvider); err != nil {
		return nil, err
	}

	cacheConfig, err := cache.GetCacheConfiguration()
	if err != nil {
		return nil, err
	}

	// Get kubernetes client from viper.
	kubecontext := viper.GetString("kubecontext")
	kubeconfig := viper.GetString("kubeconfig")
	client, err := kubernetes.NewClient(kubecontext, kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("initialising kubernetes client: %w", err)
	}

	// Load remote cache if it is configured.
	cache, err := cache.GetCacheConfiguration()
	if err != nil {
		return nil, err
	}

	if noCache {
		cache.DisableCache()
	}

	return &Manifester{
		Context:        context.Background(),
		AIClient:       aiClient,
		Cache:          cacheConfig,
		MaxConcurrency: maxConcurrency,
		Namespace:      namespace,
		KubeClient:     client,
	}, nil
}

func (m *Manifester) GenerateManifest(requirements string, anonymize bool) (string, error) {
	cacheKey := util.GetCacheKey(m.AIClient.GetName(), m.Language, requirements)

	if !m.Cache.IsCacheDisabled() && m.Cache.Exists(cacheKey) {
		response, err := m.Cache.Load(cacheKey)
		if err != nil {
			return "", err
		}

		if response != "" {
			output, err := base64.StdEncoding.DecodeString(response)
			if err == nil {
				return string(output), nil
			}
		}
	}

	promptTemplate := ai.PromptMap["k8s_manifest"]
	prompt := fmt.Sprintf(strings.TrimSpace(promptTemplate), m.Language, requirements)
	response, err := m.AIClient.GetCompletion(m.Context, prompt)
	if err != nil {
		return "", err
	}

	if err := m.Cache.Store(cacheKey, base64.StdEncoding.EncodeToString([]byte(response))); err != nil {
		return "", err
	}
	return response, nil
}

func (m *Manifester) ApplyManifest(answer string) error {
	return m.KubeClient.ApplyManifest(answer)
}

func (m *Manifester) Close() {
	if m.AIClient != nil {
		m.AIClient.Close()
	}
}
