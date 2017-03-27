package sql

import (
	"fmt"
	"gateway/config"
	"gateway/db"
	aperrors "gateway/errors"
	"log"
	"os/exec"
	"path"
)

type OracleSpec struct {
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DbName   string `json:"dbname"`
	Host     string `json:"host"`
	SSLMode  string `json:"sslmode"`
}

const (
	bin = "gateway-oracle"
)

var (
	pathToCmd       string
	oracleAvailable = false

	oracleCmd *exec.Cmd
)

func Available() bool {
	return oracleAvailable
}

// Configure initializes the oracle package
func Configure(oracle config.Oracle) error {
	pathToCmd = oracle.FullPath

	var err error

	fullOracleCommandPath := path.Join(path.Clean(pathToCmd))

	// ensure that we have valid full paths to each executable
	fullOracleCommandPath, err = exec.LookPath(fullOracleCommandPath)
	if err != nil {
		return fmt.Errorf("Received error attempting to execute LookPath for java")
	}

	cmd := exec.Command(fullOracleCommandPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Received error from gateway oracle command.  Output is %s", output)
		return fmt.Errorf("Received error checking for existence of gateway oracle command: %s", err)
	}

	oracleAvailable = true

	err = cmd.Start()
	if err != nil {
		return aperrors.NewWrapped("[gateway-oracle] Error creating command for running oracle client", err)
	}

	return nil
}

func (p *OracleSpec) validate() error {
	return validate(p, []validation{
		{kw: "port", errCond: p.Port < 0, val: p.Port},
		{kw: "user", errCond: p.User == "", val: p.User},
		{kw: "password", errCond: p.Password == "", val: p.Password},
		{kw: "dbname", errCond: p.DbName == "", val: p.DbName},
		{kw: "host", errCond: p.Host == "", val: p.Host},
		{kw: "sslmode", errCond: !sslModes[sslMode(p.SSLMode)], val: p.SSLMode},
	})
}

func (p *OracleSpec) ConnectionString() string {
	return fmt.Sprintf("%s/%s@%s:%d/%s",
		p.User,
		p.Password,
		p.Host,
		p.Port,
		p.DbName,
	)
}

func (p *OracleSpec) UniqueServer() string {
	return p.ConnectionString()
}

func (m *OracleSpec) NeedsUpdate(s db.Specifier) bool {
	return false
}
