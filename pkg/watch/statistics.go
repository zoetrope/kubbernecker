package watch

type Statistics struct {
	GroupVersionKind string
	Namespaces       map[string]*NamespaceStatistics
}

type NamespaceStatistics struct {
	Resources        map[string]*ResourceStatistics
	AddCount         int
	DeleteCount      int
	UpdateTotalCount int
}

type ResourceStatistics struct {
	UpdateCount int
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
