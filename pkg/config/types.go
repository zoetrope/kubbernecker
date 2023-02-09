package config

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/yaml"
)

// Config represents the configuration file of kubbernecker.
type Config struct {
	TargetResources        []TargetResource      `json:"TargetResources,omitempty"`
	NamespaceSelector      *metav1.LabelSelector `json:"NamespaceSelector,omitempty"`
	EnableClusterResources bool                  `json:"EnableClusterResources,omitempty"`
}

type TargetResource struct {
	metav1.GroupVersionKind `json:",inline"`
	NamespaceSelector       *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
	ResourceSelector        *metav1.LabelSelector `json:"resourceSelector,omitempty"`
}

func (c *Config) SelectorFor(gvk metav1.GroupVersionKind) (nsSelector labels.Selector, resSelector labels.Selector, err error) {
	for _, target := range c.TargetResources {
		if target.GroupVersionKind == gvk {
			if target.NamespaceSelector != nil {
				nsSelector, err = metav1.LabelSelectorAsSelector(target.NamespaceSelector)
				if err != nil {
					return
				}
			}
			if target.ResourceSelector != nil {
				resSelector, err = metav1.LabelSelectorAsSelector(target.ResourceSelector)
				if err != nil {
					return
				}
			}
		}
	}
	if nsSelector == nil {
		if c.NamespaceSelector != nil {
			nsSelector, err = metav1.LabelSelectorAsSelector(c.NamespaceSelector)
			if err != nil {
				return
			}
		} else {
			nsSelector = labels.Everything()
		}
	}
	if resSelector == nil {
		resSelector = labels.Everything()
	}

	return
}

// Validate validates the configurations.
func (c *Config) Validate() error {

	return nil
}

// Load loads configurations.
func (c *Config) Load(data []byte) error {
	return yaml.Unmarshal(data, c, yaml.DisallowUnknownFields)
}
