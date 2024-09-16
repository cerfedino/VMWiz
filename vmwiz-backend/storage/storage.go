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
	ID             int       `db:"id"`
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
