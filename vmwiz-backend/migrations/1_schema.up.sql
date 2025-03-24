CREATE TABLE request(
  ID BIGSERIAL PRIMARY KEY,
  created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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