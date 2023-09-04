package azure

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/klog/v2"
)

type FakeSender struct {
	statusCode int
	body       *n.ApplicationGateway
}

func (fs *FakeSender) Do(request *http.Request) (response *http.Response, err error) {
	response = &http.Response{
		StatusCode: fs.statusCode,
	}
	if fs.statusCode != 200 {
		err = errors.New("Error while making a GET for the gateway")
	} else {
		if fs.body != nil {
			b, err := json.Marshal(fs.body)
			if err == nil {
				response.Body = io.NopCloser(bytes.NewReader(b))
			}
		}
	}
	return response, err
}

var _ = DescribeTable("Az Application Gateway failures using authorizer", func(statusCodeArg int, errorExpected bool) {
	var azClient = NewAzClient("", "", "", "", "")
	var fakeSender = &FakeSender{
		statusCode: statusCodeArg,
		body:       &n.ApplicationGateway{},
	}
	retryDuration, err := time.ParseDuration("2ms")
	if err != nil {
		klog.Error("Invalid retry duration value")
	}
	azClient.SetDuration(retryDuration)
	azClient.SetSender(fakeSender)
	err = azClient.WaitForGetAccessOnGateway(3)
	if errorExpected {
		Expect(err).To(HaveOccurred())
	} else {
		Expect(err).To(BeNil())
	}
},
	Entry("200 Error", 200, false),
	Entry("400 Error", 400, true),
	Entry("401 Error", 401, true),
	Entry("403 Error", 403, true),
	Entry("404 Error", 404, true),
)
