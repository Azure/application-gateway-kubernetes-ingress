package azure

import (
	"net/http"
	"testing"
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type FakeSender struct{
	statusCode int
}

func (fs FakeSender) Do(request *http.Request) (response *http.Response, err error) {
	response = &http.Response{
		StatusCode:fs.statusCode,
	}
	err = errors.New("Error while making a GET for the gateway")
	return response, err
}

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Azure Client Suite")
}

var _ = Describe("Azure Client", func() {
	Context("Az Client Application Gateway Client not received successfully (authorizer failure)", func() {
		It("should handle 400 errors from Azure gracefully", func() {

			var azClient = NewAzClient("", "", "", "", "")
			var fakeSender = FakeSender{
				statusCode:400,
			}
			azClient.SetSender(fakeSender)
			err := azClient.WaitForGetAccessOnGateway(10)
			Expect(err).NotTo(Equal(nil))
		})
		It("should handle 401 errors from Azure gracefully", func() {

			var azClient = NewAzClient("", "", "", "", "")
			var fakeSender = FakeSender{
				statusCode:401,
			}
			azClient.SetSender(fakeSender)
			err := azClient.WaitForGetAccessOnGateway(10)
			Expect(err).NotTo(Equal(nil))
		})
	})
})
