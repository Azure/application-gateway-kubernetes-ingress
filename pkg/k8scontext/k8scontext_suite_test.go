package k8scontext_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestK8scontext(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "K8scontext Suite")
}
