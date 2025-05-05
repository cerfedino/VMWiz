package survey

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"strconv"
	"strings"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/notifier"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/proxmox"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/google/uuid"
)

// Creates a new VM Usage Survey in the database and sends out the emails to the users.
// This function may return error if any email fails to send: in that case, the surveyId is still returned such that the missed emails can be retried later.
func CreateVMUsageSurvey(restrict_pool []string) (*int64, error) {
	vms, err := generateSurveys(restrict_pool)
	if err != nil {
		return nil, err
	}

	// We create a new survey
	surveyId, err := storage.DB.SurveyCreateNew()
	if err != nil {
		return nil, err
	}

	// Notify about the new survey getting created
	err = notifier.NotifyVMUsageSurvey(surveyId, fmt.Sprintf("Created new VM usage survey with ID %d", surveyId))
	if err != nil {
		return &surveyId, fmt.Errorf("Failed create VM usage survey: %v", err)
	}

	// Estabilish all the emails that need to be sent and store them in the database
	for idx, vm := range vms {

		uuidString := uuid.New().String()

		receivers := []string{}
		if config.AppConfig.SMTP_RECEIVER_OVERRIDE != "" {
			// Override the receiver email address with the one from the config if present
			receivers = []string{config.AppConfig.SMTP_RECEIVER_OVERRIDE}
		} else {
			receivers = []string{vm.University_email}
		}

		// Store the new survey question in the database
		_, err := storage.DB.SurveyEmailStore(receivers[0], surveyId, vm.Vmid, vm.Hostname, uuidString, false, nil)

		if err != nil {
			msg := fmt.Sprintf("Failed create VM usage survey %v: Failed to estabilish all emails that need to be sent (Stopped at VM %v out of %v): %v", surveyId, idx, len(vms), err)
			notifier.NotifyVMUsageSurvey(surveyId, msg)
			return &surveyId, fmt.Errorf(msg)
		}
	}

	// Retrieve all emails that need to be sent from the database
	surveyEmails, err := storage.DB.SurveyEmailGetAllNotAnsweredOrUnsentBySurveyID(surveyId)
	if err != nil {
		msg := fmt.Sprintf("Failed send VM usage survey %v: Failed to get all emails that need to be sent from db: %v", surveyId, err)
		notifier.NotifyVMUsageSurvey(surveyId, msg)
		return &surveyId, fmt.Errorf(msg)
	}

	err = sendVMUsageSurvey(surveyId, *surveyEmails)
	if err != nil {
		return &surveyId, fmt.Errorf("Failed create VM usage survey %v: Failed to send emails: %v", surveyId, err)
	}

	return &surveyId, nil
}

func RetryUnsentEmails(surveyId int64) error {

	err := notifier.NotifyVMUsageSurvey(surveyId, fmt.Sprintf("Retrying unsent emails for VM usage survey %d", surveyId))
	if err != nil {
		return fmt.Errorf("Failed to retry unsent emails for VM usage survey %v: %v", surveyId, err)
	}

	// Retrieve all unsent emails
	surveyEmails, err := storage.DB.SurveyEmailGetAllUnsentBySurveyID(surveyId)
	if err != nil {
		msg := fmt.Sprintf("Failed to retry unsent emails for VM usage survey %v: Failed to get all emails that need to be sent from db: %v", surveyId, err)
		notifier.NotifyVMUsageSurvey(surveyId, msg)
		return fmt.Errorf(msg)
	}

	err = sendVMUsageSurvey(surveyId, *surveyEmails)
	if err != nil {
		return fmt.Errorf("Failed to retry unsent emails for VM usage survey %v: Failed to send emails: %v", surveyId, err)
	}

	return nil
}

func SendSurveyReminder(surveyId int64) error {

	err := notifier.NotifyVMUsageSurvey(surveyId, fmt.Sprintf("Retrying unsent emails for VM usage survey %d", surveyId))
	if err != nil {
		return fmt.Errorf("Failed to retry unsent emails for VM usage survey %v: %v", surveyId, err)
	}

	// Retrieve all unsent emails
	surveyEmails, err := storage.DB.SurveyEmailGetAllNotAnsweredBySurveyID(surveyId)
	if err != nil {
		msg := fmt.Sprintf("Failed to retry unsent emails for VM usage survey %v: Failed to get all emails that need to be sent from db: %v", surveyId, err)
		notifier.NotifyVMUsageSurvey(surveyId, msg)
		return fmt.Errorf(msg)
	}

	err = sendVMUsageSurveyReminder(surveyId, *surveyEmails)
	if err != nil {
		return fmt.Errorf("Failed to retry unsent emails for VM usage survey %v: Failed to send emails: %v", surveyId, err)
	}

	return nil
}

func sendVMUsageSurveyReminder(surveyId int64, surveyEmails []storage.SQLUsageSurveyEmail) error {
	// Process the email template for VM Usage Survey
	VMUSAGE_SURVEY_REMINDER_TEMPLATE_PATH := "survey/vmusage_survey_reminder.tmpl"
	vmusage_survey_reminder_template, err := template.ParseFiles(VMUSAGE_SURVEY_REMINDER_TEMPLATE_PATH)
	if err != nil {
		return fmt.Errorf("Failed send VM usage survey reminder: Failed to parse email template: %v", err)
	}

	// Send each email
	emails_sent := 0
	for _, surveyEmail := range surveyEmails {
		if emails_sent%10 == 0 {
			// TODO: Startup check for checking production ^ SMTP disabled
			if config.AppConfig.SMTP_ENABLE {
				log.Printf("Sending emails ... (%v / %v)", emails_sent, len(surveyEmails))
			} else {
				log.Printf("Dry-run Sending emails ... (%v / %v) (SMTP disabled)", emails_sent, len(surveyEmails))
			}
		}

		mail_content := new(bytes.Buffer)
		err = vmusage_survey_reminder_template.Execute(mail_content, struct {
			HOSTNAME string
			URL      string
			REPLYTO  string
		}{
			HOSTNAME: surveyEmail.Hostname,
			URL:      config.AppConfig.VMWIZ_SCHEME + "://" + config.AppConfig.VMWIZ_HOSTNAME + ":" + strconv.Itoa(config.AppConfig.VMWIZ_PORT) + "/survey?id=" + surveyEmail.Uuid + "&hostname=" + surveyEmail.Hostname,
			REPLYTO:  config.AppConfig.SMTP_REPLYTO,
		})
		if err != nil {
			return fmt.Errorf("Failed send VM usage survey reminder: Failed to execute email template: %v", err)
		}

		// TODO: Add startup check for checking wether we can send emails
		err = notifier.SendEmail("VSOS VM Usage Survey: Reminder", mail_content.Bytes(), []string{surveyEmail.Recipient})
		if err != nil {
			log.Printf("Failed send VM usage survey reminder: Failed to send email: %v", err)
			continue
		}

		emails_sent++
	}
	var msg string
	if config.AppConfig.SMTP_ENABLE {
		msg = fmt.Sprintf("Sent %d reminder emails for VM usage survey %v", emails_sent, surveyId)
	} else {
		msg = fmt.Sprintf("Dry-run Sent %d reminder emails for VM usage survey %v (SMTP disabled)", emails_sent, surveyId)
	}

	log.Printf("[+] " + msg)

	// Notify about the survey getting sent
	err = notifier.NotifyVMUsageSurvey(surveyId, msg)
	if err != nil {
		return fmt.Errorf("Failed to send VM usage survey notification: %v", err)
	}
	return nil
}

func sendVMUsageSurvey(surveyId int64, surveyEmails []storage.SQLUsageSurveyEmail) error {
	// Process the email template for VM Usage Survey
	VMUSAGE_SURVEY_TEMPLATE_PATH := "survey/vmusage_survey.tmpl"
	vmusage_survey_template, err := template.ParseFiles(VMUSAGE_SURVEY_TEMPLATE_PATH)
	if err != nil {
		return fmt.Errorf("Failed send VM usage survey: Failed to parse email template: %v", err)
	}

	// Send each email
	emails_sent := 0
	for _, surveyEmail := range surveyEmails {
		if emails_sent%10 == 0 {
			// TODO: Startup check for checking production ^ SMTP disabled
			if config.AppConfig.SMTP_ENABLE {
				log.Printf("Sending emails ... (%v / %v)", emails_sent, len(surveyEmails))
			} else {
				log.Printf("Dry-run Sending emails ... (%v / %v) (SMTP disabled)", emails_sent, len(surveyEmails))
			}
		}

		mail_content := new(bytes.Buffer)
		err = vmusage_survey_template.Execute(mail_content, struct {
			HOSTNAME string
			URL      string
			REPLYTO  string
		}{
			HOSTNAME: surveyEmail.Hostname,
			URL:      config.AppConfig.VMWIZ_SCHEME + "://" + config.AppConfig.VMWIZ_HOSTNAME + ":" + strconv.Itoa(config.AppConfig.VMWIZ_PORT) + "/survey?id=" + surveyEmail.Uuid + "&hostname=" + surveyEmail.Hostname,
			REPLYTO:  config.AppConfig.SMTP_REPLYTO,
		})
		if err != nil {
			return fmt.Errorf("Failed send VM usage survey: Failed to execute email template: %v", err)
		}

		// TODO: Add startup check for checking wether we can send emails
		err = notifier.SendEmail("VSOS VM Usage Survey: Response needed", mail_content.Bytes(), []string{surveyEmail.Recipient})
		if err != nil {
			log.Printf("Failed send VM usage survey: Failed to send email: %v", err)
			continue
		}

		if config.AppConfig.SMTP_ENABLE {
			err = storage.DB.SurveyEmailMarkAsSent(surveyEmail.Uuid)
			if err != nil {
				log.Printf("Failed send VM usage survey: Failed to set EmailMarkAsSent %v", err)
				continue
			}
		}

		emails_sent++
	}
	var msg string
	if config.AppConfig.SMTP_ENABLE {
		msg = fmt.Sprintf("Sent %d emails for VM usage survey %v", emails_sent, surveyId)
	} else {
		msg = fmt.Sprintf("Dry-run Sent %d emails for VM usage survey %v (SMTP disabled)", emails_sent, surveyId)
	}

	log.Printf("[+] " + msg)

	// Notify about the survey getting sent
	err = notifier.NotifyVMUsageSurvey(surveyId, msg)
	if err != nil {
		return fmt.Errorf("Failed to send VM usage survey notification: %v", err)
	}
	return nil
}

type vmNotesInfo struct {
	Hostname         string
	Vmid             int
	Nethz            string
	University_email string
	ExternalMail     string
}

func generateSurveys(restrict_pool []string) ([]vmNotesInfo, error) {
	vms, err := proxmox.GetAllClusterVMs()
	if err != nil {
		return nil, fmt.Errorf("Failed to get VM list: %v", err.Error())
	}

	surveyList := make([]vmNotesInfo, 0)

	for _, m := range *vms {
		// If restrict_pool is not empty, check if the VM is is one of the allowed pools
		if len(restrict_pool) > 0 {
			found := false
			for _, pool := range restrict_pool {
				if m.Pool == pool {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		vmConfig, err := proxmox.GetNodeVMConfig(m.Node, m.Vmid)
		if err != nil {
			continue
		}

		nethz, err := getDescriptionField(vmConfig.Description, "nethz=")
		if err != nil {
			continue
		}
		mail, err := getDescriptionField(vmConfig.Description, "uni_contact=")
		if err != nil {
			continue
		}
		externalMail, err := getDescriptionField(vmConfig.Description, "contact=")
		if err != nil {
			continue
		}

		vm := vmNotesInfo{
			Hostname:         m.Name,
			Vmid:             m.Vmid,
			Nethz:            nethz,
			University_email: mail,
			ExternalMail:     externalMail,
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
