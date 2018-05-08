package slb

//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
// Code generated by Alibaba Cloud SDK Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// SetBackendServers invokes the slb.SetBackendServers API synchronously
// api document: https://help.aliyun.com/api/slb/setbackendservers.html
func (client *Client) SetBackendServers(request *SetBackendServersRequest) (response *SetBackendServersResponse, err error) {
	response = CreateSetBackendServersResponse()
	err = client.DoAction(request, response)
	return
}

// SetBackendServersWithChan invokes the slb.SetBackendServers API asynchronously
// api document: https://help.aliyun.com/api/slb/setbackendservers.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) SetBackendServersWithChan(request *SetBackendServersRequest) (<-chan *SetBackendServersResponse, <-chan error) {
	responseChan := make(chan *SetBackendServersResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.SetBackendServers(request)
		if err != nil {
			errChan <- err
		} else {
			responseChan <- response
		}
	})
	if err != nil {
		errChan <- err
		close(responseChan)
		close(errChan)
	}
	return responseChan, errChan
}

// SetBackendServersWithCallback invokes the slb.SetBackendServers API asynchronously
// api document: https://help.aliyun.com/api/slb/setbackendservers.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) SetBackendServersWithCallback(request *SetBackendServersRequest, callback func(response *SetBackendServersResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *SetBackendServersResponse
		var err error
		defer close(result)
		response, err = client.SetBackendServers(request)
		callback(response, err)
		result <- 1
	})
	if err != nil {
		defer close(result)
		callback(nil, err)
		result <- 0
	}
	return result
}

// SetBackendServersRequest is the request struct for api SetBackendServers
type SetBackendServersRequest struct {
	*requests.RpcRequest
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	LoadBalancerId       string           `position:"Query" name:"LoadBalancerId"`
	BackendServers       string           `position:"Query" name:"BackendServers"`
	OwnerAccount         string           `position:"Query" name:"OwnerAccount"`
	AccessKeyId          string           `position:"Query" name:"access_key_id"`
	Tags                 string           `position:"Query" name:"Tags"`
}

// SetBackendServersResponse is the response struct for api SetBackendServers
type SetBackendServersResponse struct {
	*responses.BaseResponse
	RequestId      string                            `json:"RequestId" xml:"RequestId"`
	LoadBalancerId string                            `json:"LoadBalancerId" xml:"LoadBalancerId"`
	BackendServers BackendServersInSetBackendServers `json:"BackendServers" xml:"BackendServers"`
}

// CreateSetBackendServersRequest creates a request to invoke SetBackendServers API
func CreateSetBackendServersRequest() (request *SetBackendServersRequest) {
	request = &SetBackendServersRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Slb", "2014-05-15", "SetBackendServers", "slb", "openAPI")
	return
}

// CreateSetBackendServersResponse creates a response to parse from SetBackendServers response
func CreateSetBackendServersResponse() (response *SetBackendServersResponse) {
	response = &SetBackendServersResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
