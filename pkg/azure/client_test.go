// +build unittest

package azure

import (
	"errors"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type FakeSender struct {
	statusCode int
}

func (fs FakeSender) Do(request *http.Request) (response *http.Response, err error) {
	response = &http.Response{
		StatusCode: fs.statusCode,
	}
	if fs.statusCode != 200 {
		err = errors.New("Error while making a GET for the gateway")
	}
	return response, err
}

var _ = DescribeTable("Az Application Gateway failures using authorizer", func(statusCodeArg int) {
	var azClient = NewAzClient("", "", "", "", "")
	var fakeSender = FakeSender{
		statusCode: statusCodeArg,
	}
	azClient.SetDuration("4ms")
	azClient.SetSender(fakeSender)
	err := azClient.WaitForGetAccessOnGateway(3)
	Expect(err).NotTo(Equal(nil))
},
	Entry("400 Error", 400),
	Entry("401 Error", 401),
	Entry("403 Error", 403),
	Entry("403 Error", 404),
)
