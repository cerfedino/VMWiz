CREATE TABLE confirmation_tokens(
    token text PRIMARY KEY,
    used boolean DEFAULT FALSE,
    created timestamp DEFAULT CURRENT_TIMESTAMP
);