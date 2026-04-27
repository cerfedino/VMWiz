-- rollback tested on sql playground

UPDATE request 
    SET requestStatus = 'pending' 
    WHERE requestStatus = 'hold';


-- 1. Remove the existing default value first
ALTER TABLE request
  ALTER COLUMN requestStatus DROP DEFAULT;


ALTER TYPE request_status RENAME TO status_old;
CREATE TYPE request_status AS ENUM ('accepted', 'rejected', 'pending');

-- 3. Update the table column to use the new type
-- (Postgres requires an explicit cast for this)
ALTER TABLE request 
  ALTER COLUMN requestStatus TYPE request_status 
  USING requestStatus::text::request_status;

-- 4. Re-apply the default value for the new type
ALTER TABLE request 
  ALTER COLUMN requestStatus SET DEFAULT 'pending'::request_status;


-- 5. Drop the old version of the type
DROP TYPE status_old;