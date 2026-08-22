#!/usr/bin/env bash
#
# Provisions the DEV AWS resources for the chat<->email bridge so the inbound pipeline can be tested
# in minikube before it ships. Everything lives in us-east-1 (SES email receiving is not offered in
# us-east-2). Resources are dev-prefixed and independent of prod.
#
# What it creates:
#   - S3 bucket            augno-notification-inbound-email-dev   (raw inbound .eml storage)
#   - SQS queue + DLQ      augno-notification-inbound-email-dev   (S3 ObjectCreated events)
#   - S3 -> SQS event notification
#   - an IAM policy attached to your existing dev IAM user (the one behind the k8s aws-credentials
#     secret) granting S3 read + SQS consume + SES send/identity-verify
#
# Inbound can be tested WITHOUT real email: drop a raw .eml into the bucket (see the "simulate" line
# at the end) and the S3 event fans it onto SQS exactly as a real SES delivery would. Real email
# receiving (SES receipt rule + a verified domain + MX) is the optional Part B at the bottom.
#
# Usage:
#   DEV_IAM_USER=<your-dev-iam-username> ./scripts/dev-email-bridge-setup.sh
#
# Requires: awscli v2, credentials for the dev account, jq.

set -euo pipefail

REGION="us-east-1"
BUCKET="augno-notification-inbound-email-dev"
QUEUE="augno-notification-inbound-email-dev"
DLQ="augno-notification-inbound-email-dev-dlq"
POLICY_NAME="AugnoDevNotificationInboundEmail"
: "${DEV_IAM_USER:?Set DEV_IAM_USER to the IAM user behind the k8s aws-credentials secret}"

ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
echo "Account: ${ACCOUNT_ID}  Region: ${REGION}"

# --- S3 bucket (us-east-1 takes no LocationConstraint) ---
aws s3api create-bucket --bucket "${BUCKET}" --region "${REGION}" 2>/dev/null \
  || echo "bucket ${BUCKET} already exists, continuing"
aws s3api put-public-access-block --bucket "${BUCKET}" \
  --public-access-block-configuration BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
aws s3api put-bucket-lifecycle-configuration --bucket "${BUCKET}" --lifecycle-configuration '{
  "Rules":[{"ID":"expire-processed-emails","Status":"Enabled","Filter":{},"Expiration":{"Days":30}}]
}'

# --- SQS DLQ + main queue ---
DLQ_URL="$(aws sqs create-queue --region "${REGION}" --queue-name "${DLQ}" \
  --attributes MessageRetentionPeriod=1209600 --query QueueUrl --output text)"
DLQ_ARN="$(aws sqs get-queue-attributes --region "${REGION}" --queue-url "${DLQ_URL}" \
  --attribute-names QueueArn --query 'Attributes.QueueArn' --output text)"

QUEUE_URL="$(aws sqs create-queue --region "${REGION}" --queue-name "${QUEUE}" \
  --attributes "{\"VisibilityTimeout\":\"120\",\"MessageRetentionPeriod\":\"345600\",\"RedrivePolicy\":\"{\\\"deadLetterTargetArn\\\":\\\"${DLQ_ARN}\\\",\\\"maxReceiveCount\\\":\\\"5\\\"}\"}" \
  --query QueueUrl --output text)"
QUEUE_ARN="$(aws sqs get-queue-attributes --region "${REGION}" --queue-url "${QUEUE_URL}" \
  --attribute-names QueueArn --query 'Attributes.QueueArn' --output text)"
BUCKET_ARN="arn:aws:s3:::${BUCKET}"

# --- Let this S3 bucket publish ObjectCreated events to the queue ---
aws sqs set-queue-attributes --region "${REGION}" --queue-url "${QUEUE_URL}" --attributes "{
  \"Policy\": \"$(jq -c -n --arg q "${QUEUE_ARN}" --arg b "${BUCKET_ARN}" --arg acct "${ACCOUNT_ID}" '{
    Version:"2012-10-17",
    Statement:[{Sid:"AllowS3Publish",Effect:"Allow",Principal:{Service:"s3.amazonaws.com"},
      Action:"sqs:SendMessage",Resource:$q,
      Condition:{ArnEquals:{"aws:SourceArn":$b},StringEquals:{"aws:SourceAccount":$acct}}}]}' | sed 's/"/\\"/g')\"
}"

# --- Wire the S3 -> SQS notification (queue policy must exist first) ---
aws s3api put-bucket-notification-configuration --bucket "${BUCKET}" \
  --notification-configuration "{\"QueueConfigurations\":[{\"QueueArn\":\"${QUEUE_ARN}\",\"Events\":[\"s3:ObjectCreated:*\"]}]}"

# --- IAM: grant the dev user S3 read + SQS consume + SES send/verify ---
aws iam put-user-policy --user-name "${DEV_IAM_USER}" --policy-name "${POLICY_NAME}" --policy-document "$(jq -c -n \
  --arg bucket "${BUCKET_ARN}" --arg queue "${QUEUE_ARN}" '{
  Version:"2012-10-17",
  Statement:[
    {Effect:"Allow",Action:["s3:GetObject","s3:PutObject","s3:ListBucket"],Resource:[$bucket, ($bucket+"/*")]},
    {Effect:"Allow",Action:["sqs:ReceiveMessage","sqs:DeleteMessage","sqs:GetQueueAttributes"],Resource:$queue},
    {Effect:"Allow",Action:["ses:SendEmail","ses:SendRawEmail","ses:VerifyDomainIdentity","ses:VerifyDomainDkim","ses:GetIdentityDkimAttributes","ses:GetIdentityVerificationAttributes"],Resource:"*"}
  ]}')"

echo
echo "=== DONE. Put this in infra/development/kubernetes/config/app-config.yaml ==="
echo "INBOUND_EMAIL_BUCKET:    ${BUCKET}"
echo "INBOUND_EMAIL_QUEUE_URL: ${QUEUE_URL}"
echo "INBOUND_EMAIL_REGION:    ${REGION}"
echo
echo "=== Simulate an inbound email (once the Phase 3 consumer is running) ==="
echo "  aws s3 cp sample.eml s3://${BUCKET}/inbound/test-\$(date +%s).eml"
echo "  # -> S3 fires an event -> SQS -> notification-service inbound consumer threads it into chat."

# ----------------------------------------------------------------------------------------------------
# Part B (OPTIONAL) — real email receiving via SES.
#
# !!! DEV AND PROD SHARE ONE AWS ACCOUNT. There is exactly ONE active SES receipt rule set per
# !!! account+region. Do NOT create/activate a separate dev rule set — `set-active-receipt-rule-set`
# !!! would DEACTIVATE prod's and silently break prod inbound mail. The s3 cp simulation above needs
# !!! none of this and is the recommended dev path.
#
# If you truly need real inbound mail in dev, ADD A RULE to the SINGLE active rule set (the one prod's
# Terraform creates, `augno-notification-inbound`), scoped by a dev-only recipient domain, routing to
# the dev bucket. This requires prod's email_inbound.tf to be applied first so the rule set exists.
# Adding the rule by CLI drifts from Terraform state — prefer adding it to email_inbound.tf instead.
#
#   ACTIVE_RS="$(aws ses describe-active-receipt-rule-set --region us-east-1 --query Metadata.Name --output text)"
#   # 1. Verify a DEV-ONLY subdomain for receiving + DKIM (never a real customer/prod domain):
#   aws ses verify-domain-identity --region us-east-1 --domain mail.dev.openmrp.ai
#   aws ses verify-domain-dkim     --region us-east-1 --domain mail.dev.openmrp.ai   # publish the 3 CNAMEs
#   # 2. Publish MX:  mail.dev.openmrp.ai  MX 10 inbound-smtp.us-east-1.amazonaws.com
#   # 3. Allow SES to write to the dev bucket:
#   aws s3api put-bucket-policy --bucket "${BUCKET}" --policy "$(jq -c -n --arg b "arn:aws:s3:::${BUCKET}" --arg acct "${ACCOUNT_ID}" '{Version:"2012-10-17",Statement:[{Sid:"AllowSESPuts",Effect:"Allow",Principal:{Service:"ses.amazonaws.com"},Action:"s3:PutObject",Resource:($b+"/*"),Condition:{StringEquals:{"AWS:SourceAccount":$acct}}}]}')"
#   # 4. Add a DEV rule to the EXISTING active rule set (recipient-scoped so it can't catch prod mail):
#   aws ses create-receipt-rule --region us-east-1 --rule-set-name "${ACTIVE_RS}" \
#     --rule "{\"Name\":\"dev-to-s3\",\"Enabled\":true,\"ScanEnabled\":true,\"Recipients\":[\"mail.dev.openmrp.ai\"],\"Actions\":[{\"S3Action\":{\"BucketName\":\"${BUCKET}\",\"ObjectKeyPrefix\":\"inbound/\"}}]}"
#
# Part C (OPTIONAL) — outbound send test. SES starts in sandbox (can only send to verified addresses).
#   aws ses verify-email-identity --region us-east-1 --email-address dev@augno.com
# ----------------------------------------------------------------------------------------------------
