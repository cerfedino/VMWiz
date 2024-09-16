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

// Almost like a ternary operator
func If(condition bool, trueVal any, falseVal any) any {
	if condition {
		return trueVal
	}
	return falseVal
}

type SQLRequest struct {
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
	Init(dataSourceName string)
	StoreVMRequest(req *form.Form) error
}

type sqlstorage struct {
	db *sql.DB
}

type postgresstorage sqlstorage

var DB postgresstorage

func (s *postgresstorage) Init(dataSourceName string) {
	POSTGRES_USER := os.Getenv("POSTGRES_USER")
	POSTGRES_PASSWORD := os.Getenv("POSTGRES_PASSWORD")
	POSTGRES_DB := os.Getenv("POSTGRES_DB")

	if dataSourceName == "" {
		dataSourceName = fmt.Sprintf("postgres://%v:%v@vmwiz-db/%v?sslmode=disable", POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB)
	}

	db, _ := sql.Open("postgres", dataSourceName)
	if err := db.Ping(); err != nil {
		log.Fatalf("Could not ping SQL data source '%s':\n%s", dataSourceName, err)
	} else {
		log.Printf("Successfully connected to SQL data source '%s'\n", dataSourceName)
		s.db = db
	}

	m, err := migrate.New("file://migrations/", dataSourceName)
	if err != nil {
		log.Fatal(err)
	}
	m.Up()
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

func (s *postgresstorage) GetVMRequest(id int64) (*SQLRequest, error) {
	var req SQLRequest
	err := DB.db.QueryRow(`SELECT 
	ID, created, email, personalEmail, isOrganization, orgName, hostname, image, cores, ramGB, diskGB, sshPubkeys, comments
	FROM request WHERE id=$1`, id).Scan(&req.ID, &req.CreatedAt, &req.Email, &req.PersonalEmail, &req.IsOrganization, &req.OrgName, &req.Hostname, &req.Image, &req.Cores, &req.RamGB, &req.DiskGB, pq.Array(&req.SshPubkeys), &req.Comments)
	if err != nil {
		log.Printf("Error getting from SQL: \n%s", err)
		return nil, err
	}
	return &req, nil
}
