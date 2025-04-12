package survey

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"strconv"
	"strings"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/notifier"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/proxmox"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/google/uuid"
)

type SurveyVM struct {
	Nethz        string
	Mail         string
	ExternalMail string
	Hostname     string
	VMID         int
}

func CreateVMUsageSurvey() error {
	vms, err := generateSurveys()
	if err != nil {
		return err
	}

	surveyID, err := storage.DB.SurveyStore()
	if err != nil {
		return err
	}

	err = notifier.NotifyVMUsageSurvey(surveyID, fmt.Sprintf("Created new VM usage survey with ID %d", surveyID))
	if err != nil {
		return fmt.Errorf("Failed create VM usage survey: %v", err)
	}

	emails_sent := 0
	for _, vm := range vms {

		uuidString := uuid.New().String()

		_, err := storage.DB.SurveyQuestionStore(surveyID, vm.VMID, vm.Hostname, uuidString)
		if err != nil {
			return fmt.Errorf("Failed create VM usage survey: %v", err)
		}

		url := config.AppConfig.VMWIZ_SCHEME + "://" + config.AppConfig.VMWIZ_HOSTNAME + ":" + strconv.Itoa(config.AppConfig.VMWIZ_PORT) + "/survey?id=" + uuidString

		// Receiver email address.
		receivers := []string{}
		if config.AppConfig.SMTP_RECEIVER_OVERRIDE != "" {
			// Override the receiver email address with the one from the config if present
			receivers = []string{config.AppConfig.SMTP_RECEIVER_OVERRIDE}
		} else {
			receivers = []string{}
		}

		VMUSAGE_SURVEY_TEMPLATE_PATH := "survey/vmusage_survey.tmpl"
		mail_content := new(bytes.Buffer)
		vmusage_survey_template, err := template.ParseFiles(VMUSAGE_SURVEY_TEMPLATE_PATH)
		if err != nil {
			return fmt.Errorf("Failed create VM usage survey: Failed to parse email template: %v", err)
		}
		err = vmusage_survey_template.Execute(mail_content, struct {
			HOSTNAME  string
			URL       string
			RECIPIENT string
		}{
			HOSTNAME:  vm.Hostname,
			URL:       url,
			RECIPIENT: receivers[0],
		})
		if err != nil {
			return fmt.Errorf("Failed create VM usage survey: Failed to execute email template: %v", err)
		}

		// Authentication.
		// fmt.Println(config.AppConfig.SMTP_USER, config.AppConfig.SMTP_PASSWORD, config.AppConfig.SMTP_HOST, config.AppConfig.SMTP_PORT)
		// TODO: Add startup check
		auth := smtp.PlainAuth("", config.AppConfig.SMTP_USER, config.AppConfig.SMTP_PASSWORD, config.AppConfig.SMTP_HOST)

		// Sending email.
		err = smtp.SendMail(config.AppConfig.SMTP_HOST+":"+config.AppConfig.SMTP_PORT, auth, config.AppConfig.SMTP_SENDER, receivers, mail_content.Bytes())
		if err != nil {
			return fmt.Errorf("Failed create VM usage survey: Failed to send email: %v", err)
		}
		emails_sent++
	}

	err = notifier.NotifyVMUsageSurvey(surveyID, fmt.Sprintf("Sent %d emails for VM usage survey", emails_sent))
	if err != nil {
		return fmt.Errorf("Failed to send VM usage survey notification: %v", err)
	}
	return nil
}

func generateSurveys() ([]SurveyVM, error) {
	vms, err := proxmox.GetAllClusterVMs()
	if err != nil {
		return nil, fmt.Errorf("Failed to get VM list: %v", err.Error())
	}

	surveyList := make([]SurveyVM, 0)

	for _, m := range *vms {

		vmConfig, err := proxmox.GetNodeVMConfig(m.Node, m.Vmid)
		if err != nil {
			log.Printf("Failed to get VM config for VM %d: %v", m.Vmid, err.Error())
			continue
		}

		nethz, err := getDescriptionField(vmConfig.Description, "nethz=")
		if err != nil {
			log.Printf("Failed to get nethz field: %v", err.Error())
		}
		mail, err := getDescriptionField(vmConfig.Description, "uni_contact=")
		if err != nil {
			log.Printf("Failed to get uni_contact field: %v", err.Error())
		}
		externalMail, err := getDescriptionField(vmConfig.Description, "contact=")
		if err != nil {
			log.Printf("Failed to get contact field: %v", err.Error())
		}

		vm := SurveyVM{
			Hostname:     m.Name,
			VMID:         m.Vmid,
			Nethz:        nethz,
			Mail:         mail,
			ExternalMail: externalMail,
		}
		surveyList = append(surveyList, vm)
	}

	return surveyList, nil
}

func getDescriptionField(description string, field string) (string, error) {
	lines := strings.Split(description, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, field) {
			return strings.TrimSpace(strings.TrimPrefix(line, field)), nil
		}
	}
	return "", fmt.Errorf("Field '%s' not found in description", field)
}
