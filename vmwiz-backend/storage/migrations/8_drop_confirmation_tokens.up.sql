-- confirmation_tokens was never used: the confirmation flow keeps tokens in
-- memory, and no query ever touched this table. Drop it.
DROP TABLE IF EXISTS confirmation_tokens;
