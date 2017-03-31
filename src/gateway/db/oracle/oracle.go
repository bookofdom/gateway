package oracle

import (
	"errors"
	"fmt"
	"gateway/config"
	"gateway/db"
	aperrors "gateway/errors"
	"log"
	"os/exec"
	"path"
	// apsql "gateway/sql"
)

type oraSpec interface {
	db.Specifier
}

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

func Config(confs ...db.Configurator) (db.Specifier, error) {
	var spec oraSpec
	var ok bool

	for _, conf := range confs {
		s, err := conf(spec)
		if err != nil {
			return nil, err
		}
		spec, ok = s.(oraSpec)
		if !ok {
			return nil, fmt.Errorf("oracle Config requires Oracle Specifier, got %T", s)
		}

	}
	return spec, nil
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
	return nil
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

type DB struct {
	conf *OracleSpec
}

func (d *DB) Spec() db.Specifier {
	return d.conf
}

func Connection(c oraSpec) db.Configurator {
	return func(s db.Specifier) (db.Specifier, error) {
		if c == nil {
			return nil, errors.New("can't validate nil specifier")
		}

		spec, ok := c.(*OracleSpec)
		if !ok {
			return nil, fmt.Errorf("invalid type %T", c)
		}

		err := spec.validate()
		if err != nil {
			return nil, err
		}

		return spec, nil
	}
}

func (r *OracleSpec) UpdateWith(spec *OracleSpec) error {
	if spec == nil {
		return errors.New("cannot update Oracle with a nil Specifier")
	}

	if err := spec.validate(); err != nil {
		return err
	}

	*r = *spec
	return nil
}

func (d *DB) Update(s db.Specifier) error {
	spec, ok := s.(*OracleSpec)
	if !ok {
		return fmt.Errorf("can't update Oracle with %T", spec)
	}

	if err := spec.validate(); err != nil {
		return err
	}

	return nil
}

func (m *OracleSpec) NewDB() (db.DB, error) {
	db := &DB{m}
	return db, nil
}
