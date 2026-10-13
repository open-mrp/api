-- Local/dev email bridge: verified domain + inbox on mail.dev.openmrp.ai.
-- AWS side (SES identity, MX, DKIM, S3/SQS, receipt rule) is Terraform in open-mrp/infra
-- email_inbound_dev.tf. After make seed-core + Tilt, mail to support@mail.dev.openmrp.ai
-- lands in messaging; PLATFORM=development sends replies to dev@openmrp.ai.

INSERT IGNORE INTO email_domain (id, account_id, domain, status, dkim_tokens, mail_from_domain, verified_at, created_at, updated_at) VALUES
    ('emdom_01seeddevdomain00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'mail.dev.openmrp.ai', 'verified',
     '[]', NULL, NOW(), NOW(), NOW());

INSERT IGNORE INTO email_inbox (id, account_id, email_domain_id, address, from_name, status, agent_config_id, created_at, updated_at) VALUES
    ('eminb_01seeddevinbox000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'emdom_01seeddevdomain00', 'support@mail.dev.openmrp.ai', 'Acme Support', 'active', NULL, NOW(), NOW());
