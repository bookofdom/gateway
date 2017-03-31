package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gateway/config"
	"gateway/db/oracle"
	aperrors "gateway/errors"
	"gateway/logreport"
	"gateway/model"
	"io"
	"net"
)

// OracleRequest encapsulates a request made to a Oracle endpoint.
type OracleRequest struct {
	sqlRequest
	Config     *oracle.OracleSpec `json:"config"`
	oracleConf config.Oracle
}

// OracleResponse encapsulates a response from a OracleRequest
type OracleResponse struct {
	Body *json.RawMessage `json:"body"`
}

func (r *OracleRequest) Log(devMode bool) string {
	s := r.sqlRequest.Log(devMode)
	if devMode {
		s += fmt.Sprintf("\nConnection: %+v", r.Config)
	}
	return s
}

func (r *OracleRequest) JSON() ([]byte, error) {
	return json.Marshal(r)
}

// JSON marshals the OracleResponse to JSON
func (oracleResponse *OracleResponse) JSON() ([]byte, error) {
	logreport.Printf("Attempting to marshal oracle response")
	bytes, err := json.Marshal(&oracleResponse)
	if err != nil {
		logreport.Printf("FOUND AN ERROR %s", err)
	}
	return bytes, err
}

func NewOracleRequest(endpoint *model.RemoteEndpoint, data *json.RawMessage) (Request, error) {
	request := &OracleRequest{}

	if err := json.Unmarshal(*data, request); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal request json: %v", err)
	}

	endpointData := &OracleRequest{}
	if err := json.Unmarshal(endpoint.Data, endpointData); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal endpoint configuration: %v", err)
	}
	request.updateWith(endpointData)

	if endpoint.SelectedEnvironmentData != nil {
		if err := json.Unmarshal(*endpoint.SelectedEnvironmentData, endpointData); err != nil {
			return nil, err
		}
		request.updateWith(endpointData)
	}

	return request, nil
}

func (r *OracleRequest) Perform() Response {
	// response := &OracleResponse{Type: "oracle"}
	requestBytes, err := json.Marshal(&r)
	if err != nil {
		return NewErrorResponse(aperrors.NewWrapped("[oracle] Unmarshaling request data", err))
	}

	hostPort := fmt.Sprintf("%s:%d", r.oracleConf.OracleHost, r.oracleConf.OraclePort)
	conn, err := net.Dial("tcp", hostPort)
	if err != nil {
		return NewErrorResponse(aperrors.NewWrapped("[oracle] Connecting to gateway oracle", err))
	}

	defer conn.Close()

	message := fmt.Sprintf("%s\n\n", string(requestBytes))
	_, err = conn.Write([]byte(message))

	if err != nil {
		return NewErrorResponse(aperrors.NewWrapped("[oracle] Sending data to gateway oracle", err))
	}

	buf := bytes.NewBuffer([]byte{})
	done := false
	for !done {
		var responseBytes = make([]byte, 1024)
		readlen, err := conn.Read(responseBytes)
		if err != nil {
			if err != io.EOF {
				logreport.Printf("Error when reading from socket: %s", err)
				return NewErrorResponse(aperrors.NewWrapped("[oracle] Reading data from gateway oracle", err))
			}
			done = true
		}
		if readlen == 0 {
			break
		}
		buf.Write(responseBytes[:readlen])
	}

	rawMessage := new(json.RawMessage)
	decoder := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	err = decoder.Decode(rawMessage)
	if err != nil {
		return NewErrorResponse(aperrors.NewWrapped("[gateway-oracle] Marshaling response", err))
	}

	return &OracleResponse{Body: rawMessage}
}

// TODO - refactor to DRY this code up across different data sources
func (r *OracleRequest) updateWith(endpointData *OracleRequest) {
	if endpointData.Config != nil {
		if r.Config == nil {
			r.Config = &oracle.OracleSpec{}
		}
		r.Config.UpdateWith(endpointData.Config)
	}
	r.sqlRequest.updateWith(endpointData.sqlRequest)
}

// Log returns a string containing the deatils to be logged pertaining to the SoapResponse
func (oracleResponse *OracleResponse) Log() string {
	var buffer bytes.Buffer
	bytes := []byte(*oracleResponse.Body)
	buffer.Write(bytes)
	return buffer.String()
}
