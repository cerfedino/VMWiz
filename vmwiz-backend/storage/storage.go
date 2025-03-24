package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/form"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type SQLVMRequest struct {
	ID             int64     `db:"id"`
	CreatedAt      time.Time `db:"created"`
	Email          string    `db:"email"`
	PersonalEmail  string    `db:"personalEmail"`
	IsOrganization bool      `db:"isOrganization"`
	OrgName        string    `db:"orgName"`
	Hostname       string    `db:"hostname"`
	Image          string    `db:"image"`
	Cores          int       `db:"cores"`
	RamGB          int       `db:"ramGB"`
	DiskGB         int       `db:"diskGB"`
	SshPubkeys     []string  `db:"sshPubkeys"`
	Comments       string    `db:"comments"`
}

type Storage interface {
	CreateConnection() error
	InitMigrations() error
	Init(dataSourceName string)
	StoreVMRequest(req *form.Form) error
}

type postgresstorage struct {
	db        *sql.DB
	migration *migrate.Migrate
}

var DB postgresstorage

func buildConnectionString(POSTGRES_USER string, POSTGRES_PASSWORD string, POSTGRES_DB string) string {
	return fmt.Sprintf("postgres://%v:%v@vmwiz-db/%v?sslmode=disable", POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB)
}

func (s *postgresstorage) CreateConnection() error {
	POSTGRES_USER := os.Getenv("POSTGRES_USER")
	POSTGRES_PASSWORD := os.Getenv("POSTGRES_PASSWORD")
	POSTGRES_DB := os.Getenv("POSTGRES_DB")

	if s.db != nil {
		s.db.Close()
		s.db = nil
	}
	db, err := sql.Open("postgres", buildConnectionString(POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB))
	if err != nil {
		return fmt.Errorf("Test DB connection failed: %v", err.Error())
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("Test DB connection failed: %v", err.Error())
	} else {
		s.db = db
		return nil
	}
}

func (s *postgresstorage) InitMigrations() error {
	if s.migration != nil {
		s.migration.Close()
		s.migration = nil
	}
	m, err := migrate.New("file://migrations/", buildConnectionString(os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_DB")))
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

func (s *postgresstorage) StoreVMRequest(req *form.Form) error {

	_, err := DB.db.Exec(`INSERT INTO request
		(email, personalEmail, isOrganization, orgName, hostname, image, cores, ramGB, diskGB, sshPubkeys, comments)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		req.Email, req.PersonalEmail, req.IsOrganization, req.OrgName, req.Hostname, req.Image, req.Cores, req.RamGB, req.DiskGB, pq.Array(req.SshPubkeys), req.Comments)
	if err != nil {
		log.Printf("Error inserting into SQL: \n%s", err)
	}
	return nil
}

func (s *postgresstorage) GetVMRequest(id int64) (*SQLVMRequest, error) {
	var req SQLVMRequest
	err := DB.db.QueryRow(`SELECT 
	ID, created, email, personalEmail, isOrganization, orgName, hostname, image, cores, ramGB, diskGB, sshPubkeys, comments
	FROM request WHERE id=$1`, id).Scan(&req.ID, &req.CreatedAt, &req.Email, &req.PersonalEmail, &req.IsOrganization, &req.OrgName, &req.Hostname, &req.Image, &req.Cores, &req.RamGB, &req.DiskGB, pq.Array(&req.SshPubkeys), &req.Comments)
	if err != nil {
		log.Printf("Error getting from SQL: \n%s", err)
		return nil, err
	}
	return &req, nil
}

func (s *postgresstorage) GetAllVMsRequest(id int64) (*SQLVMRequest[], error) {
	var ids []int64
	err := DB.db.QueryRow(`SELECT ID FROM request`).Scan(ids)
	if err != nil {
		log.Printf("Error getting from SQL: \n%s", err)
		return nil, err
	}
	//for id in ids
	var reqs []SQLVMRequest
	for _, id := range ids {
		var req SQLVMRequest
		req, err := s.GetVMRequest(id)
		if err != nil {
			log.Printf("Error getting from SQL: \n%s", err)
			return nil, err
		}
		reqs = append(reqs, req)
	}

	return &reqs, nil
}
