package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gateway/config"
	aperrors "gateway/errors"
	"gateway/model"
	"gateway/soap"
	"io"
	"log"
	"net"
	"strings"
)

// SoapRequest encapsulates a request made via SOAP
type SoapRequest struct {
	ServiceName             string                   `json:"serviceName"`
	EndpointName            string                   `json:"endpointName"`
	OperationName           string                   `json:"operationName,omitempty"`
	ActionName              string                   `json:"actionName,omitempty"`
	Params                  *json.RawMessage         `json:"params"`
	URL                     string                   `json:"url,omitempty"`
	JarURL                  string                   `json:"jarUrl"`
	WssePasswordCredentials *WssePasswordCredentials `json:"wssePasswordCredentials,omitempty"`

	soapConf config.Soap
}

// WssePasswordCredentials represents credentials for a SOAP request as specified
// by the WS-Security spec
type WssePasswordCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SoapResponse encapsulates a response from a SoapRequest
type SoapResponse struct {
	Body *json.RawMessage `json:"body"`
}

// NewSoapRequest constructs a new SoapRequest
func NewSoapRequest(endpoint *model.RemoteEndpoint, data *json.RawMessage, soapConf config.Soap) (Request, error) {
	request := new(SoapRequest)

	request.soapConf = soapConf

	if err := json.Unmarshal(*data, request); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal request json: %v", err)
	}

	endpointData := new(SoapRequest)
	if err := json.Unmarshal(endpoint.Data, endpointData); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal endpoint data: %v", err)
	}
	request.updateWith(endpointData)

	if endpoint.SelectedEnvironmentData != nil {
		if err := json.Unmarshal(*endpoint.SelectedEnvironmentData, endpointData); err != nil {
			return nil, err
		}
		request.updateWith(endpointData)
	}

	var err error
	request.JarURL, err = soap.JarURLForRemoteEndpointID(endpoint.ID)
	if err != nil {
		return nil, fmt.Errorf("Unable to determine jar URL: %v", err)
	}

	return request, nil
}

func (soapRequest *SoapRequest) updateWith(other *SoapRequest) {
	if other.ServiceName != "" {
		soapRequest.ServiceName = other.ServiceName
	}

	if other.EndpointName != "" {
		soapRequest.EndpointName = other.EndpointName
	}

	if other.OperationName != "" {
		soapRequest.OperationName = other.OperationName
	}

	if other.ActionName != "" {
		soapRequest.ActionName = other.ActionName
	}

	if other.URL != "" {
		soapRequest.URL = other.URL
	}

	if other.WssePasswordCredentials != nil {
		soapRequest.WssePasswordCredentials = other.WssePasswordCredentials
	}
}

// Log returns the SOAP request basics, e.g. 'ServiceName, EndpointName, OperationName, ActionName, URL' when in server mode.
// When in dev mode the Params and WssePasswordCredentials (sans password) are also returned.
func (soapRequest *SoapRequest) Log(devMode bool) string {
	var buffer bytes.Buffer
	if devMode {
		buffer.WriteString(fmt.Sprintf("ServiceName: %s\nEndpointName: %s\nOperationName: %s\nActionName: %s\nURL: %s", soapRequest.ServiceName, soapRequest.EndpointName, soapRequest.OperationName, soapRequest.ActionName, soapRequest.URL))
		if soapRequest.WssePasswordCredentials != nil {
			passwordStr := strings.Replace(soapRequest.WssePasswordCredentials.Password, "", "*", len(soapRequest.WssePasswordCredentials.Password))
			buffer.WriteString(fmt.Sprintf("\nWssePasswordCredentials:\n  Username:  %s\n  Password:  %s", soapRequest.WssePasswordCredentials.Username, passwordStr))
		}
		buffer.WriteString(fmt.Sprintf("\nParams: %v\n", soapRequest.Params))
	} else {
		buffer.WriteString(fmt.Sprintf("%s, %s, %s, %s, %s", soapRequest.ServiceName, soapRequest.EndpointName, soapRequest.OperationName, soapRequest.ActionName, soapRequest.URL))
	}
	return buffer.String()
}

// Perform executes the SoapRequest
func (soapRequest *SoapRequest) Perform() Response {
	requestBytes, err := json.Marshal(&soapRequest)
	if err != nil {
		return NewErrorResponse(aperrors.NewWrapped("[soap] Unmarshaling request data", err))
	}

	hostPort := fmt.Sprintf("%s:%d", soapRequest.soapConf.SoapClientHost, soapRequest.soapConf.SoapClientPort)
	conn, err := net.Dial("tcp", hostPort)
	if err != nil {
		return NewErrorResponse(aperrors.NewWrapped("[soap] Connecting to soapclient", err))
	}

	defer conn.Close()

	message := fmt.Sprintf("%s\n\n", string(requestBytes))
	_, err = conn.Write([]byte(message))

	if err != nil {
		return NewErrorResponse(aperrors.NewWrapped("[soap] Sending data to soapclient", err))
	}

	buf := bytes.NewBuffer([]byte{})
	done := false
	for !done {

		var responseBytes = make([]byte, 1024)
		readlen, err := conn.Read(responseBytes)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error when reading from socket: %s", err)
				return NewErrorResponse(aperrors.NewWrapped("[soap] Reading data from soapclient", err))
			}
			done = true
		}
		if readlen == 0 {
			break
		}

		buf.Write(responseBytes)
	}

	rawMessage := new(json.RawMessage)
	decoder := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	err = decoder.Decode(rawMessage)
	if err != nil {
		return NewErrorResponse(aperrors.NewWrapped("[soap] Marshaling response", err))
	}

	return &SoapResponse{Body: rawMessage}
}

// JSON marshals the SoapResponse to JSON
func (soapResponse *SoapResponse) JSON() ([]byte, error) {
	log.Printf("Attempting to marshal soap response")
	bytes, err := json.Marshal(&soapResponse)
	if err != nil {
		log.Printf("FOUND AN ERROR %s", err)
	}
	return bytes, err
}

// Log returns a string containing the deatils to be logged pertaining to the SoapResponse
func (soapResponse *SoapResponse) Log() string {
	var buffer bytes.Buffer
	bytes := []byte(*soapResponse.Body)
	buffer.Write(bytes)
	return buffer.String()
}
