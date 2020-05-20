# AGIC E2E
AGIC E2E consists of two scenarios, MFU, Most Frequently Use, and LFU, Least Frequently Use.
For each of the scenario, test cases are organized in a way that how ingress is defined with namespace:
- One Namespace One Ingress, 1N1I
- One Namespace Many Ingresses, 1NMI

One scenario can have multiple test suites, and one test suite can have multiple test cases.

for example, Test Suite or context "One Namespace One Ingress" defines a Test case or Spec "ssl-e2e-redirect", the test case deploys one ingress in a namespace. 
```bash
// scenario
var _ = Describe("MFU", func() {
	var (
		clientset *kubernetes.Clientset
		err       error
	)
    // test suite, 1N1I
	Context("One Namespace One Ingress", func() {
		BeforeEach(func() {
			clientset, err = getClient()
			Expect(err).To(BeNil())
			cleanUp(clientset)
		})

        // test case
		It("[ssl-e2e-redirect] ssl termination and ssl redirect to https backend should work", func() {
            ...
        })
        ...
    })
}       
```