package storage

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/form"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/proxmox"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

const (
	REQUEST_STATUS_PENDING  = "pending"
	REQUEST_STATUS_ACCEPTED = "accepted"
	REQUEST_STATUS_REJECTED = "rejected"
)

type SQLVMRequest struct {
	ID               int64     `db:"requestId"`
	RequestCreatedAt time.Time `db:"requestCreatedAt"`
	RequestStatus    string    `db:"requestStatus"`
	Email            string    `db:"email"`
	PersonalEmail    string    `db:"personalEmail"`
	IsOrganization   bool      `db:"isOrganization"`
	OrgName          string    `db:"orgName"`
	Hostname         string    `db:"hostname"`
	Image            string    `db:"image"`
	Cores            int       `db:"cores"`
	RamGB            int       `db:"ramGB"`
	DiskGB           int       `db:"diskGB"`
	SshPubkeys       []string  `db:"sshPubkeys"`
	Comments         string    `db:"comments"`
}

func (f *SQLVMRequest) ToString() string {
	return `
ID: ` + fmt.Sprintf("%v", f.ID) + `
RequestCreatedAt: ` + fmt.Sprintf("%v", f.RequestCreatedAt) + `
RequestStatus: ` + fmt.Sprintf("%v", f.RequestStatus) + `
Email: ` + fmt.Sprintf("%v", f.Email) + `
PersonalEmail: ` + fmt.Sprintf("%v", f.PersonalEmail) + `
IsOrganization: ` + fmt.Sprintf("%v", f.IsOrganization) + `
OrgName: ` + fmt.Sprintf("%v", f.OrgName) + `
Hostname: ` + fmt.Sprintf("%v", f.Hostname) + `
Image: ` + fmt.Sprintf("%v", f.Image) + `
Cores: ` + fmt.Sprintf("%v", f.Cores) + `
RamGB: ` + fmt.Sprintf("%v", f.RamGB) + `
DiskGB: ` + fmt.Sprintf("%v", f.DiskGB) + `
SshPubkeys: ` + fmt.Sprintf("%v", f.SshPubkeys) + `
Comments: ` + fmt.Sprintf("%v", f.Comments) + `
`
}

func (s *SQLVMRequest) ToVMOptions() *proxmox.VMCreationOptions {
	return &proxmox.VMCreationOptions{
		Template:   s.Image,
		FQDN:       s.Hostname,
		Reinstall:  false,
		Cores_CPU:  s.Cores,
		RAM_MB:     int64(s.RamGB * 1024),
		Disk_GB:    int64(s.DiskGB),
		SSHPubkeys: s.SshPubkeys,
		// TODO: Proper handling of notes
		Notes: fmt.Sprintf("nethz=TODO  uni_contact=%s  contact=%s", s.PersonalEmail, s.Email),
		Tags:  []string{"created-by-vmwiz"},

		UseQemuAgent: false,
	}
}

type Storage interface {
	CreateConnection() error
	InitMigrations() error
	Init(dataSourceName string)

	StoreVMRequest(req *form.Form) error
	GetVMRequest(id int64) (*SQLVMRequest, error)
	UpdateVMRequest(req SQLVMRequest) error
	UpdateVMRequestStatus(id int64, status string) error
	GetAllVMRequests() ([]*SQLVMRequest, error)
	SurveyStore(vmid int, hostname string, uuid string) (int64, error)
	SurveyResponseUpdate(uuid string, response bool) error
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
	POSTGRES_USER := config.AppConfig.POSTGRES_USER
	POSTGRES_PASSWORD := config.AppConfig.POSTGRES_PASSWORD
	POSTGRES_DB := config.AppConfig.POSTGRES_DB

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
	m, err := migrate.New("file://migrations/", buildConnectionString(config.AppConfig.POSTGRES_USER, config.AppConfig.POSTGRES_PASSWORD, config.AppConfig.POSTGRES_DB))
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

func (s *postgresstorage) StoreVMRequest(req *form.Form) (*int64, error) {

	res := s.db.QueryRow(`INSERT INTO request
		(email, personalEmail, isOrganization, orgName, hostname, image, cores, ramGB, diskGB, sshPubkeys, comments)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING requestID`,
		req.Email, req.PersonalEmail, req.IsOrganization, req.OrgName, fmt.Sprintf("%v.vsos.ethz.ch", req.Hostname), req.Image, req.Cores, req.RamGB, req.DiskGB, pq.Array(req.SshPubkeys), req.Comments)
	// Get the last inserted ID
	var id int64
	err := res.Scan(&id)
	if err != nil {
		log.Printf("StoreVMRequest: Error getting last insert ID: %s", err)
		return nil, err
	}

	return &id, nil
}

func (s *postgresstorage) GetVMRequest(id int64) (*SQLVMRequest, error) {
	var req SQLVMRequest
	err := s.db.QueryRow(`SELECT 
	requestID,requestCreatedAt, requestStatus, email, personalEmail, isOrganization, orgName, hostname, image, cores, ramGB, diskGB, sshPubkeys, comments
	FROM request WHERE requestID=$1`, id).Scan(&req.ID, &req.RequestCreatedAt, &req.RequestStatus, &req.Email, &req.PersonalEmail, &req.IsOrganization, &req.OrgName, &req.Hostname, &req.Image, &req.Cores, &req.RamGB, &req.DiskGB, pq.Array(&req.SshPubkeys), &req.Comments)
	if err != nil {
		log.Printf("GetVMRequest: Error when executing query: %s", err)
		return nil, err
	}
	return &req, nil
}

func (s *postgresstorage) UpdateVMRequest(req SQLVMRequest) error {
	_, err := s.db.Exec(`UPDATE request SET requestCreatedAt=$1, requestStatus=$2, email=$3, personalEmail=$4, isOrganization=$5, orgName=$6, hostname=$7, image=$8, cores=$9, ramGB=$10, diskGB=$11, sshPubkeys=$12, comments=$13 WHERE requestID=$14`,
		req.RequestCreatedAt, req.RequestStatus, req.Email, req.PersonalEmail, req.IsOrganization, req.OrgName, req.Hostname, req.Image, req.Cores, req.RamGB, req.DiskGB, pq.Array(req.SshPubkeys), req.Comments, req.ID)
	if err != nil {
		log.Printf("UpdateVMRequest: Error updating SQL: %s", err)
		return err
	}

	return nil
}

func (s *postgresstorage) UpdateVMRequestStatus(id int64, status string) error {
	_, err := s.db.Exec(`UPDATE request SET requestStatus=$1 WHERE requestID=$2`, status, id)
	if err != nil {
		log.Printf("UpdateVMRequestStatus: Error updating SQL: %s", err)
	}
	return nil
}

func (s *postgresstorage) GetAllVMRequests() ([]*SQLVMRequest, error) {
	rows, err := s.db.Query(`SELECT requestID FROM request`)
	if err != nil {
		log.Printf("GetAllVMRequests: Error when executing query: %s", err)
		return nil, err
	}
	// Store all IDs
	var ids []*int64
	for rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil {
			log.Printf("GetAllVMRequests: Error while scanning rows: %s", err)
			return nil, err
		}
		ids = append(ids, &id)
	}

	//for id in ids
	var reqs []*SQLVMRequest
	for _, id := range ids {
		var req *SQLVMRequest
		req, err := s.GetVMRequest(*id)
		if err != nil {
			log.Printf("GetAllVMRequests: %s", err)
			return nil, err
		}
		reqs = append(reqs, req)
	}

	return reqs, nil
}

type SQLUsageSurveyEmail struct {
	Id         int64  `db:"id"`
	Recipient  string `db:"recipient"`
	SurveyId   int64  `db:"surveyId"`
	Vmid       int    `db:"vmid"`
	Hostname   string `db:"hostname"`
	Uuid       string `db:"uuid"`
	Email_sent bool   `db:"email_sent"`
	Still_used *bool  `db:"still_used"`
}

type SQLUsageSurvey struct {
	id   int64     `db:"id"`
	date time.Time `db:"date"`
}

func (s *postgresstorage) SurveyCreateNew() (int64, error) {
	res := s.db.QueryRow(`INSERT INTO survey DEFAULT VALUES RETURNING id`)
	// Get the last inserted ID
	var id int64
	err := res.Scan(&id)
	if err != nil {
		return -1, fmt.Errorf("SurveyCreateNew: Error getting last insert ID: %s", err)
	}
	return id, nil
}

func (s *postgresstorage) SurveyEmailStore(recipient string, surveyId int64, vmid int, hostname string, uuid string, email_sent bool, still_used *bool) (int64, error) {
	res := s.db.QueryRow(`INSERT INTO survey_email (recipient, vmid, surveyId, hostname, uuid, email_sent, still_used) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`, recipient, vmid, surveyId, hostname, uuid, email_sent, still_used)
	// Get the last inserted ID
	var insertedID int64
	err := res.Scan(&insertedID)
	if err != nil {
		return -1, fmt.Errorf("SurveyEmailStore: Error getting last insert ID: %s", err)
	}
	return insertedID, nil
}

func (s *postgresstorage) SurveyEmailUpdateResponse(uuid string, response bool) error {
	_, err := s.db.Exec(`UPDATE survey_email SET still_used = $1 WHERE uuid = $2`, response, uuid)
	if err != nil {
		return fmt.Errorf("SurveyEmailUpdate: Error updating survey response:\n%s", err)
	}
	return nil
}

func (s *postgresstorage) SurveyEmailMarkAsSent(uuid string) error {
	_, err := s.db.Exec(`UPDATE survey_email SET email_sent = $1 WHERE uuid = $2`, true, uuid)
	if err != nil {
		return fmt.Errorf("SurveyEmailMarkAsSent: Error updating survey email sent status: %s", err)
	}
	return nil
}

func (s *postgresstorage) SurveyGetAllIDs() ([]int64, error) {
	res, err := s.db.Query(`SELECT id FROM survey`)
	if err != nil {
		return nil, fmt.Errorf("SurveyGetAllIDs: Error executing query: %s", err)
	}
	defer res.Close()

	var ids []int64
	for res.Next() {
		if err = res.Err(); err == sql.ErrNoRows {
			return ids, nil
		}

		var id int64
		err = res.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("Error while scanning rows: %s", err)
		}

		ids = append(ids, id)
	}

	return ids, nil
}

func (s *postgresstorage) SurveyEmailGetAllBySurveyID(surveyId int64) (*[]SQLUsageSurveyEmail, error) {
	var surveyEmails []SQLUsageSurveyEmail
	rows, err := s.db.Query(`SELECT id, recipient, surveyId, vmid, hostname, uuid, email_sent, still_used FROM survey_email WHERE surveyId=$1`, surveyId)
	if err != nil {
		return nil, fmt.Errorf("SurveyEmailGetAllBySurveyID: Error while executing query: %s", err)
	}
	defer rows.Close()
	for rows.Next() {
		var surveyEmail SQLUsageSurveyEmail
		err = rows.Scan(&surveyEmail.Id, &surveyEmail.Recipient, &surveyEmail.SurveyId, &surveyEmail.Vmid, &surveyEmail.Hostname, &surveyEmail.Uuid, &surveyEmail.Email_sent, &surveyEmail.Still_used)
		if err != nil {
			return nil, fmt.Errorf("SurveyEmailGetAllBySurveyID: Error while scanning rows: %s", err)
		}
		surveyEmails = append(surveyEmails, surveyEmail)
	}

	return &surveyEmails, nil
}

func (s *postgresstorage) SurveyGetLastId() (int, error) {
	// Get the last inserted ID
	var id int
	err := s.db.QueryRow(`SELECT id FROM survey ORDER BY date DESC LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		id = 0
		err = nil
	}
	if err != nil {
		return -1, fmt.Errorf("SurveyGetLastId: Error getting last insert ID: %s", err)
	}
	return id, nil
}

func (s *postgresstorage) SurveyEmailNegative(surveyId int) ([]string, error) {
	res, err := s.db.Query(`SELECT hostname FROM survey_email WHERE still_used = false AND surveyId = $1`, surveyId)
	if err != nil {
		return nil, fmt.Errorf("SurveyEmailNegative: Error executing query: %s", err)
	}
	defer res.Close()
	var hostnames []string
	for res.Next() {
		var hostname string
		err = res.Scan(&hostname)
		if err != nil {
			return nil, fmt.Errorf("SurveyEmailNegative: Error while scanning rows: %s", err)
		}
		hostnames = append(hostnames, hostname)
	}
	if err = res.Err(); err != nil {
		return nil, fmt.Errorf("SurveyEmailNegative: Error while scanning rows: %s", err)
	}
	if hostnames == nil {
		hostnames = []string{}
	}
	return hostnames, nil
}

func (s *postgresstorage) SurveyEmailNotResponded(surveyId int) ([]string, error) {
	res, err := s.db.Query(`SELECT hostname FROM survey_email WHERE still_used IS NULL AND surveyId = $1`, surveyId)
	if err != nil {
		return nil, fmt.Errorf("SurveyEmailNotResponded: Error executing query: %s", err)
	}
	defer res.Close()
	var hostnames []string
	for res.Next() {
		var hostname string
		err = res.Scan(&hostname)
		if err != nil {
			return nil, fmt.Errorf("SurveyEmailNotResponded: Error while scanning rows: %s", err)
		}
		hostnames = append(hostnames, hostname)
	}
	if err = res.Err(); err != nil {
		return nil, fmt.Errorf("SurveyEmailNotResponded: Error while scanning rows: %s", err)
	}
	if hostnames == nil {
		hostnames = []string{}
	}
	return hostnames, nil
}
