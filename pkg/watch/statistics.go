package watch

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Statistics struct {
	GroupVersionKind metav1.GroupVersionKind         `json:"gvk"`
	Namespaces       map[string]*NamespaceStatistics `json:"namespaces"`
}

type NamespaceStatistics struct {
	Resources   map[string]*ResourceStatistics `json:"resources"`
	AddCount    int                            `json:"add"`
	DeleteCount int                            `json:"delete"`
	UpdateCount int                            `json:"update"`
}

type ResourceStatistics struct {
	UpdateCount int `json:"update"`
}

func (in *Statistics) DeepCopy() *Statistics {
	if in == nil {
		return nil
	}
	out := new(Statistics)
	in.DeepCopyInto(out)
	return out
}

func (in *Statistics) DeepCopyInto(out *Statistics) {
	*out = *in

	if in.Namespaces != nil {
		in, out := &in.Namespaces, &out.Namespaces
		*out = make(map[string]*NamespaceStatistics, len(*in))
		for key, val := range *in {
			var outVal *NamespaceStatistics
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(NamespaceStatistics)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
	return
}

func (in *NamespaceStatistics) DeepCopy() *NamespaceStatistics {
	if in == nil {
		return nil
	}
	out := new(NamespaceStatistics)
	in.DeepCopyInto(out)
	return out
}

func (in *NamespaceStatistics) DeepCopyInto(out *NamespaceStatistics) {
	*out = *in

	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make(map[string]*ResourceStatistics, len(*in))
		for key, val := range *in {
			var outVal *ResourceStatistics
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(ResourceStatistics)
				*out = *in
			}
			(*out)[key] = outVal
		}
	}
	return
}
