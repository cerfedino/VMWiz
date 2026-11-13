DROP TABLE IF EXISTS resource_change_closure;
ALTER TABLE vm_request_waitlist RENAME TO request_waitlist;
ALTER INDEX vm_request_closure_open RENAME TO request_closure_open;
ALTER TABLE vm_request_closure RENAME TO request_closure;
