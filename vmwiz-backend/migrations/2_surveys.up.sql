CREATE TABLE survey(
  id BIGSERIAL PRIMARY KEY,
  date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE survey_question(
  id BIGSERIAL PRIMARY KEY,
  surveyID BIGSERIAL REFERENCES survey(id),
  vmid INT NOT NULL,
  hostname text NOT NULL,
  uuid text NOT NULL,
  still_used boolean
);