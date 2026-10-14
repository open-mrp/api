-- +goose Up

-- The async-job endpoint tools were renamed with their routes (/v1/core/jobs -> /v1/core/async-jobs):
-- cancel_job -> cancel_async_job, retrieve_job -> retrieve_async_job. Carry the old slugs in an agent's
-- endpoint_tool_slugs and endpoint_tool_review over to the new ones, so a granted tool stays granted
-- instead of being pruned as unknown. A slug only ever appears quoted, as an array element or a key.

UPDATE agent_definition
SET config = replace(replace(config::text, '"cancel_job"', '"cancel_async_job"'), '"retrieve_job"', '"retrieve_async_job"')::jsonb
WHERE config::text LIKE '%"cancel_job"%' OR config::text LIKE '%"retrieve_job"%';

UPDATE agent_config
SET config = replace(replace(config::text, '"cancel_job"', '"cancel_async_job"'), '"retrieve_job"', '"retrieve_async_job"')::jsonb
WHERE config::text LIKE '%"cancel_job"%' OR config::text LIKE '%"retrieve_job"%';

-- +goose Down

UPDATE agent_definition
SET config = replace(replace(config::text, '"cancel_async_job"', '"cancel_job"'), '"retrieve_async_job"', '"retrieve_job"')::jsonb
WHERE config::text LIKE '%"cancel_async_job"%' OR config::text LIKE '%"retrieve_async_job"%';

UPDATE agent_config
SET config = replace(replace(config::text, '"cancel_async_job"', '"cancel_job"'), '"retrieve_async_job"', '"retrieve_job"')::jsonb
WHERE config::text LIKE '%"cancel_async_job"%' OR config::text LIKE '%"retrieve_async_job"%';
