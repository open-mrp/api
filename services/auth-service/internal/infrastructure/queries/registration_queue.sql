-- name: CreateRegistrationQueueEntry :exec
INSERT INTO registration_queue (email, name, plan_code, registration_session_id)
VALUES (?, ?, ?, ?);
