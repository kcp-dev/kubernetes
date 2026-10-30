package mutating

import v1 "k8s.io/api/admissionregistration/v1"

func CompilePolicy(policy *v1.MutatingAdmissionPolicy) PolicyEvaluator {
	return compilePolicy(policy)
}
