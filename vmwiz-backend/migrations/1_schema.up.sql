CREATE TYPE request_status AS ENUM ('accepted', 'rejected', 'pending');


CREATE TABLE request(
  requestID BIGSERIAL PRIMARY KEY,
  requestCreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  requestStatus request_status NOT NULL DEFAULT 'pending',
  email text NOT NULL,
  personalEmail text NOT NULL,
  isOrganization boolean,
  orgName text,
  hostname text NOT NULL,
  image text NOT NULL,
  cores int NOT NULL,
  ramGB int NOT NULL,
  diskGB int NOT NULL,
  sshPubkeys text[] NOT NULL,
  comments text
)

CREATE TABLE survey(
  id SERIAL PRIMARY KEY,
  date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
)

CREATE TABLE survey_question(
  id SERIAL PRIMARY KEY,
  vmid INT NOT NULL,
  hostname text NOT NULL,
  surveyID INT REFERENCES survey(id),
  uuid text NOT NULL,
  still_used boolean
)