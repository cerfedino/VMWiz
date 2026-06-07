package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/proxmox"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const (
	REQUEST_STATUS_PENDING  = "pending"
	REQUEST_STATUS_ACCEPTED = "accepted"
	REQUEST_STATUS_REJECTED = "rejected"
	REQUEST_STATUS_HELD     = "hold"

	// Reserved catch-all log scope id, owns "0.log".
	SCOPE_ROOT = "0"
)

// ToString renders a VM request for CLI output and notifications.
func (r Request) ToString() string {
	return `ID: ` + fmt.Sprintf("%v", r.Requestid) + `
RequestCreatedAt: ` + fmt.Sprintf("%v", r.Requestcreatedat) + `
RequestStatus: ` + fmt.Sprintf("%v", r.Requeststatus) + `
Email: ` + fmt.Sprintf("%v", r.Email) + `
PersonalEmail: ` + fmt.Sprintf("%v", r.Personalemail) + `
IsOrganization: ` + fmt.Sprintf("%v", r.Isorganization) + `
OrgName: ` + fmt.Sprintf("%v", r.Orgname.String) + `
Hostname: ` + fmt.Sprintf("%v", r.Hostname) + `
Image: ` + fmt.Sprintf("%v", r.Image) + `
Cores: ` + fmt.Sprintf("%v", r.Cores) + `
RamGB: ` + fmt.Sprintf("%v", r.Ramgb) + `
DiskGB: ` + fmt.Sprintf("%v", r.Diskgb) + `
SecondaryDiskGB: ` + fmt.Sprintf("%v", r.Secondarydiskgb) + `
SshPubkeys: ` + fmt.Sprintf("%v", r.Sshpubkeys) + `
Comments: ` + fmt.Sprintf("%v", r.Comments.String) + `
`
}

func (r Request) ToVMOptions() *proxmox.VMCreationOptions {
	return &proxmox.VMCreationOptions{
		Template:         r.Image,
		FQDN:             r.Hostname,
		Reinstall:        false,
		Cores_CPU:        int(r.Cores),
		RAM_MB:           int64(r.Ramgb) * 1024,
		Disk_GB:          int64(r.Diskgb),
		SecondaryDisk_GB: int64(r.Secondarydiskgb),
		SSHPubkeys:       r.Sshpubkeys,
		Notes:            "VM is being set up, please wait...",
		Tags:             []string{"created-by-vmwiz"},
		DescriptionKVPairs: map[string]string{
			"nethz":       "TODO",
			"uni_contact": r.Email,
			"contact":     r.Personalemail,
		},

		UseQemuAgent: false,
	}
}

// ToString renders a usage survey for CLI output.
func (s Survey) ToString() string {
	return fmt.Sprintf("Survey ID: %v\nCreated date: %v", s.ID, s.Date)
}

type postgresstorage struct {
	*Queries
	db        *sql.DB
	migration *migrate.Migrate
}

var DB postgresstorage

func buildConnectionString(POSTGRES_USER string, POSTGRES_PASSWORD string, POSTGRES_DB string) string {
	return fmt.Sprintf("postgres://%v:%v@vmwiz-db/%v?sslmode=disable", POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB)
}

func (s *postgresstorage) CreateConnection() error {
	POSTGRES_USER := config.AppConfig.POSTGRES_USER
	POSTGRES_PASSWORD := config.AppConfig.POSTGRES_PASSWORD
	POSTGRES_DB := config.AppConfig.POSTGRES_DB

	if s.db != nil {
		s.db.Close()
		s.db = nil
		s.Queries = nil
	}
	conn, err := sql.Open("postgres", buildConnectionString(POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB))
	if err != nil {
		return fmt.Errorf("Test DB connection failed: %v", err.Error())
	}

	if err := conn.Ping(); err != nil {
		return fmt.Errorf("Test DB connection failed: %v", err.Error())
	}
	s.db = conn
	s.Queries = New(conn)
	return nil
}

func (s *postgresstorage) InitMigrations() error {
	if s.migration != nil {
		s.migration.Close()
		s.migration = nil
	}
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("Couldn't load embedded migrations: %v", err.Error())
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, buildConnectionString(config.AppConfig.POSTGRES_USER, config.AppConfig.POSTGRES_PASSWORD, config.AppConfig.POSTGRES_DB))
	if err != nil {
		return fmt.Errorf("Couldn't initialize migrations: %v", err.Error())
	}
	s.migration = m
	return nil
}

func (s *postgresstorage) Init() error {
	err := s.CreateConnection()
	if err != nil {
		return fmt.Errorf("Initializing DB: %v", err.Error())
	}

	err = s.InitMigrations()
	if err != nil {
		return fmt.Errorf("Initializing DB: %v", err.Error())
	}

	err = s.migration.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("Initializing DB: Running migration UP:%v", err.Error())
	}

	return nil
}

// logger.ScopeStore stuff

func (s *postgresstorage) CreateLogScope(id string, parentID string, rootID string, label string) error {
	parent := sql.NullString{}
	if parentID != "" {
		parent = sql.NullString{String: parentID, Valid: true}
	}
	return s.Queries.CreateLogScope(context.Background(), CreateLogScopeParams{
		ID:       id,
		ParentID: parent,
		RootID:   rootID,
		Label:    label,
	})
}

func (s *postgresstorage) FinishLogScope(id string, failed bool) error {
	return s.Queries.FinishLogScope(context.Background(), FinishLogScopeParams{ID: id, Failed: failed})
}

func (s *postgresstorage) LogScopeFinished(id string) (finished bool, failed bool, err error) {
	row, err := s.Queries.GetLogScopeStatus(context.Background(), id)
	if err != nil {
		return false, false, err
	}
	return row.EndedAt.Valid, row.Failed, nil
}

func (s *postgresstorage) LogScopeRootID(id string) (string, error) {
	return s.Queries.GetLogScopeRootID(context.Background(), id)
}

func (s *postgresstorage) LogScopeSubtreeIDs(id string) ([]string, error) {
	return s.Queries.ListLogScopeSubtreeIDs(context.Background(), id)
}

func (s *postgresstorage) ScopeIDsBefore(cutoff time.Time) ([]string, error) {
	return s.Queries.ListExpiredRootLogScopeIDs(context.Background(), ListExpiredRootLogScopeIDsParams{
		RootScopeID: SCOPE_ROOT,
		Cutoff:      sql.NullTime{Time: cutoff, Valid: true},
	})
}
