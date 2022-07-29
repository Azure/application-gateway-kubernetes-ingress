package azure

import (
	"net/http"
	"testing"
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type FakeSender struct{}

func (fs FakeSender) Do(request *http.Request) (response *http.Response, err error) {
	response = &http.Response{
		StatusCode:401,
	}
	err = errors.New("Error while making a GET for the gateway")
	return response, err
}

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Azure Client Suite")
}

var _ = Describe("Azure Client", func() {
	Context("Az Client Application Gateway Client created successfully", func() {
		It("should handle 401 errors from Azure gracefully", func() {

			var azClient = NewAzClient("", "", "", "", "")
			var fakeSender = FakeSender{}
			azClient.SetSender(fakeSender)
			gateway, err := azClient.GetGateway()
			Expect(err).NotTo(Equal(nil))
			Expect(gateway).To(Equal(nil))
		})
	})
})
