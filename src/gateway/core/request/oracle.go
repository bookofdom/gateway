package request

import (
	"encoding/json"
	"errors"
	"fmt"

	"gateway/config"
	"gateway/db/pools"
	sql "gateway/db/sql"
	"gateway/model"
)

// OracleRequest encapsulates a request made to a Oracle endpoint.
type OracleRequest struct {
	sqlRequest
	Config     *sql.OracleSpec `json:"config"`
	oracleConf config.Oracle
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
	response := &OracleResponse{Type: "oracle"}

	defer func() {
		if r := recover(); r != nil {
			response.Error = fmt.Sprintf("%v", r)
		}
	}()

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

	return response
}

// TODO - refactor to DRY this code up across different data sources
func (r *OracleRequest) updateWith(endpointData *OracleRequest) {
	if endpointData.Config != nil {
		if r.Config == nil {
			r.Config = &sql.PostgresSpec{}
		}
		r.Config.UpdateWith(endpointData.Config)
	}
	r.sqlRequest.updateWith(endpointData.sqlRequest)
}
