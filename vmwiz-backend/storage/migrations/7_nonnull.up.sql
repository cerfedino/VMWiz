-- make some more fields non-null


UPDATE request SET requestCreatedAt = CURRENT_TIMESTAMP WHERE requestCreatedAt IS NULL;
ALTER TABLE request ALTER COLUMN requestCreatedAt SET NOT NULL;

UPDATE request SET isOrganization = false WHERE isOrganization IS NULL;
ALTER TABLE request ALTER COLUMN isOrganization SET DEFAULT false;
ALTER TABLE request ALTER COLUMN isOrganization SET NOT NULL;

UPDATE survey SET date = CURRENT_TIMESTAMP WHERE date IS NULL;
ALTER TABLE survey ALTER COLUMN date SET NOT NULL;

ALTER TABLE survey_email ALTER COLUMN surveyId SET NOT NULL;

UPDATE survey_email SET email_sent = false WHERE email_sent IS NULL;
ALTER TABLE survey_email ALTER COLUMN email_sent SET NOT NULL;

UPDATE confirmation_tokens SET used = false WHERE used IS NULL;
ALTER TABLE confirmation_tokens ALTER COLUMN used SET NOT NULL;

UPDATE confirmation_tokens SET created = CURRENT_TIMESTAMP WHERE created IS NULL;
ALTER TABLE confirmation_tokens ALTER COLUMN created SET NOT NULL;
