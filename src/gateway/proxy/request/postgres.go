package request

import (
	"encoding/json"
	"errors"
	"fmt"

	"gateway/db/pools"
	sql "gateway/db/sql"
	"gateway/model"
)

// PostgresRequest encapsulates a request made to a Postgres endpoint.
type PostgresRequest struct {
	sqlRequest
	Config *sql.PostgresSpec `json:"config"`
}

func (r *PostgresRequest) Log(devMode bool) string {
	s := r.sqlRequest.Log(devMode)
	if devMode {
		s += fmt.Sprintf("\nConnection: %+v", r.Config)
	}
	return s
}

func NewPostgresRequest(pools *pools.Pools, endpoint *model.RemoteEndpoint, data *json.RawMessage) (Request, error) {
	request := &PostgresRequest{}
	if err := json.Unmarshal(*data, request); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal request json: %v", err)
	}

	endpointData := &PostgresRequest{}
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

	if pools == nil {
		return nil, errors.New("database pools not set up")
	}

	conn, err := pools.Connect(sql.Config(
		sql.Connection(request.Config),
		sql.MaxOpenIdle(request.MaxOpenConn, request.MaxIdleConn),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	if pqConn, ok := conn.(*sql.DB); ok {
		if pqConn.DB == nil {
			return nil, fmt.Errorf("got nil database connection")
		}
		request.conn = pqConn.DB
		return request, nil
	}

	return nil, fmt.Errorf("need Postgres connection, got %T", conn)
}

// TODO - refactor to DRY this code up across different data sources
func (r *PostgresRequest) updateWith(endpointData *PostgresRequest) {
	if endpointData.Config != nil {
		if r.Config == nil {
			r.Config = &sql.PostgresSpec{}
		}
		r.Config.UpdateWith(endpointData.Config)
	}
	r.sqlRequest.updateWith(endpointData.sqlRequest)
}
