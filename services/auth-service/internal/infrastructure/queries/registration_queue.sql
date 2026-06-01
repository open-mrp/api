-- name: CreateRegistrationQueueEntry :execrows
INSERT IGNORE INTO registration_queue (email, name, plan_code, registration_session_id)
VALUES (?, ?, ?, ?);
