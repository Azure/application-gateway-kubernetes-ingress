// +build !ignore_autogenerated

/*
Copyright The Kubernetes Authors.

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureApplicationGatewayLoadDistributionPolicy) DeepCopyInto(out *AzureApplicationGatewayLoadDistributionPolicy) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureApplicationGatewayLoadDistributionPolicy.
func (in *AzureApplicationGatewayLoadDistributionPolicy) DeepCopy() *AzureApplicationGatewayLoadDistributionPolicy {
	if in == nil {
		return nil
	}
	out := new(AzureApplicationGatewayLoadDistributionPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AzureApplicationGatewayLoadDistributionPolicy) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureApplicationGatewayLoadDistributionPolicyList) DeepCopyInto(out *AzureApplicationGatewayLoadDistributionPolicyList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AzureApplicationGatewayLoadDistributionPolicy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureApplicationGatewayLoadDistributionPolicyList.
func (in *AzureApplicationGatewayLoadDistributionPolicyList) DeepCopy() *AzureApplicationGatewayLoadDistributionPolicyList {
	if in == nil {
		return nil
	}
	out := new(AzureApplicationGatewayLoadDistributionPolicyList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AzureApplicationGatewayLoadDistributionPolicyList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureApplicationGatewayLoadDistributionPolicySpec) DeepCopyInto(out *AzureApplicationGatewayLoadDistributionPolicySpec) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Targets != nil {
		in, out := &in.Targets, &out.Targets
		*out = make([]Backend, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureApplicationGatewayLoadDistributionPolicySpec.
func (in *AzureApplicationGatewayLoadDistributionPolicySpec) DeepCopy() *AzureApplicationGatewayLoadDistributionPolicySpec {
	if in == nil {
		return nil
	}
	out := new(AzureApplicationGatewayLoadDistributionPolicySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Backend) DeepCopyInto(out *Backend) {
	*out = *in
	out.Service = in.Service
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Backend.
func (in *Backend) DeepCopy() *Backend {
	if in == nil {
		return nil
	}
	out := new(Backend)
	in.DeepCopyInto(out)
	return out
}
