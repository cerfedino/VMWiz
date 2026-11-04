CREATE TABLE survey(
  id BIGSERIAL PRIMARY KEY,
  date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE survey_email(
  id BIGSERIAL PRIMARY KEY,
  recipient text NOT NULL,
  surveyId BIGSERIAL REFERENCES survey(id),
  vmid INT NOT NULL,
  hostname text NOT NULL,
  uuid text NOT NULL,
  email_sent boolean DEFAULT false,
  still_used boolean
);