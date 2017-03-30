package remote_endpoint

import (
	"encoding/json"
	"fmt"
	"gateway/db"
	"gateway/db/oracle"
	"github.com/jmoiron/sqlx/types"
)

type Oracle struct {
	Config *oracle.OracleSpec `json:"config"`
}

func OracleConfig(data types.JsonText) (db.Specifier, error) {
	var conf Oracle
	err := json.Unmarshal(data, &conf)
	if err != nil {
		return nil, fmt.Errorf("bad JSON for Oracle config: %s", err.Error())
	}

	spec, err := oracle.Config(
		oracle.Connection(conf.Config),
	)

	if err != nil {
		return nil, err
	}

	return spec, nil
}
