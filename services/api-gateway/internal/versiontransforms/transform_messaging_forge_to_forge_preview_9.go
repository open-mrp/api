package versiontransforms

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge reshaped conversations and messages:
//   - Entity references became record stubs: conversation topic → related.record, message and announcement
//     resource → related.record, link and attachment resource → record. Requests name a record as {type, id}.
//   - conversation unread → unread_count; a caller's hidden state moved from the conversation's status to their
//     participant's membership.
//   - message agent_run_failed + agent_error_code → status failed + error {code, message}; a deleted message reads
//     status deleted.
//   - create takes participants [{account_user_id, role?}] and derives type; the list filters by memberships[],
//     record_type and record_id, and no longer takes q.
//
// preview.9 callers send and read the old shapes. What forge no longer keeps is not rebuilt: a conversation's status
// never reads hidden, and creating a direct message no longer returns an existing one.
type messagingForgeToPreview9 struct {
	forgeToPreview9
}

func init() {
	version.RegisterTransformer(messagingForgeToPreview9{})
}

func (messagingForgeToPreview9) ObjectTypes() []object.Type {
	return []object.Type{
		constants.ObjectTypeConversation, constants.ObjectTypeChatMessage, constants.ObjectTypeConversationLink,
		constants.ObjectTypeConversationParticipant, constants.ObjectTypeMessageAttachment, constants.ObjectTypeAttachmentUploadTarget,
		constants.ObjectTypeAnnouncement, constants.ObjectTypeMessagingGroup, constants.ObjectTypeMessagingBlock,
	}
}

// ForcedIncludes keeps a link's record, which preview.9 always returned as resource.
func (messagingForgeToPreview9) ForcedIncludes(objectType object.Type) []string {
	if objectType == constants.ObjectTypeConversationLink {
		return []string{"record"}
	}
	return nil
}

func (messagingForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		switch obj["object"] {
		case string(constants.ObjectTypeConversation):
			if v, ok := obj["unread_count"]; ok {
				obj["unread"] = v
				delete(obj, "unread_count")
			}
			obj["topic"] = relatedRecordEntity(obj)
			delete(obj, "related")
		case string(constants.ObjectTypeChatMessage):
			obj["resource"] = relatedRecordEntity(obj)
			delete(obj, "related")
			downgradeMessageStatus(obj)
		case string(constants.ObjectTypeAnnouncement):
			obj["resource"] = relatedRecordEntity(obj)
			delete(obj, "related")
		case string(constants.ObjectTypeConversationLink), string(constants.ObjectTypeMessageAttachment):
			if _, ok := obj["record"]; ok {
				obj["resource"] = recordEntity(obj["record"])
				delete(obj, "record")
			}
		case string(constants.ObjectTypeConversationParticipant):
			if obj["membership"] == string(constants.ParticipantMembershipHidden) {
				obj["membership"] = string(constants.ParticipantMembershipActive)
			}
		}
	})
	return data
}

// relatedRecordEntity is the preview.9 entity for related.record, or nil when it wasn't included or is gone.
func relatedRecordEntity(obj map[string]any) any {
	related, ok := obj["related"].(map[string]any)
	if !ok {
		return nil
	}
	return recordEntity(related["record"])
}

// recordEntity turns a record stub into preview.9's entity: the record's number is its display name.
func recordEntity(v any) any {
	rec, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return map[string]any{"id": rec["id"], "object": "entity", "type": rec["type"], "name": rec["number"], "handle": nil}
}

// downgradeMessageStatus restores preview.9's agent failure flags and reads a deleted message as sent.
func downgradeMessageStatus(obj map[string]any) {
	failed := false
	var code any
	if errObj, ok := obj["error"].(map[string]any); ok {
		failed, code = true, errObj["code"]
		obj["status"] = string(constants.MessageStatusSent)
	}
	if obj["status"] == string(constants.MessageStatusDeleted) {
		obj["status"] = string(constants.MessageStatusSent)
	}
	obj["agent_run_failed"] = failed
	obj["agent_error_code"] = code
	delete(obj, "error")
}

func (messagingForgeToPreview9) TransformRequest(_ object.Type, data map[string]any) map[string]any {
	return data
}

const (
	conversationsRoute = "/v1/messaging/conversations"
	messagesRoute      = "/v1/messaging/conversations/{id}/messages"
	linksRoute         = "/v1/messaging/conversations/{id}/links"
	announcementsRoute = "/v1/messaging/announcements"
)

// TransformRequestBody upgrades preview.9's create-conversation, send-message and add-link bodies.
func (messagingForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	if method != http.MethodPost {
		return data
	}
	switch route {
	case conversationsRoute:
		upgradeCreateConversation(data)
	case messagesRoute:
		moveRecord(data, "link_resource_type", "link_resource_id", "record")
		if atts, ok := data["attachments"].([]any); ok {
			for _, a := range atts {
				if att, ok := a.(map[string]any); ok {
					moveRecord(att, "resource_type", "resource_id", "record")
				}
			}
		}
	case linksRoute:
		moveRecord(data, "resource_type", "resource_id", "record")
	}
	return data
}

// upgradeCreateConversation folds preview.9's participant lists into participants and its topic into record. A
// preview.9 group with one other member is still a group: the member is given the member role, which forge reads
// as a group.
func upgradeCreateConversation(data map[string]any) {
	moveRecord(data, "topic_resource_type", "topic_resource_id", "record")
	var participants []any
	ids, _ := data["participant_account_user_ids"].([]any)
	roles, _ := data["participants"].([]any)
	delete(data, "participant_account_user_ids")
	roleOf := map[any]bool{}
	for _, r := range roles {
		if m, ok := r.(map[string]any); ok {
			roleOf[m["account_user_id"]] = true
		}
	}
	isGroup := data["type"] == string(constants.ConversationTypeGroup)
	for _, id := range ids {
		if roleOf[id] {
			continue
		}
		p := map[string]any{"account_user_id": id}
		if isGroup && len(ids) == 1 && len(roles) == 0 && data["group_id"] == nil && data["record"] == nil {
			p["role"] = string(constants.ParticipantRoleMember)
		}
		participants = append(participants, p)
	}
	participants = append(participants, roles...)
	if participants != nil {
		data["participants"] = participants
	}
	delete(data, "type")
}

// moveRecord replaces a type and id pair with a {type, id} record under key, when both were sent.
func moveRecord(data map[string]any, typeKey, idKey, key string) {
	t, hasType := data[typeKey]
	id, hasID := data[idKey]
	delete(data, typeKey)
	delete(data, idKey)
	if hasType && hasID && t != nil && id != nil {
		data[key] = map[string]any{"type": t, "id": id}
	}
}

// TransformQuery upgrades preview.9's list filters and the include keys that named entities.
func (messagingForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	switch route {
	case conversationsRoute:
		if status := query.Get("status"); status != "" {
			query.Set("memberships[]", status)
			query.Del("status")
		}
		renameQuery(query, "topic_resource_type", "record_type")
		renameQuery(query, "topic_resource_id", "record_id")
		query.Del("q")
	case messagesRoute, announcementsRoute:
		query.Del("q")
	}
	for _, param := range []string{"include", "include[]"} {
		values, ok := query[param]
		if !ok {
			continue
		}
		var keys []string
		for _, v := range values {
			for _, key := range strings.Split(v, ",") {
				if key = strings.TrimSpace(key); key != "" {
					keys = append(keys, preview9MessagingIncludeKey(route, key))
				}
			}
		}
		if param == "include" {
			query[param] = []string{strings.Join(keys, ",")}
		} else {
			query[param] = keys
		}
	}
	return query
}

func renameQuery(query url.Values, from, to string) {
	if v, ok := query[from]; ok {
		query[to] = v
		query.Del(from)
	}
}

// preview9MessagingIncludeKey maps a preview.9 include key to forge's: topic and a message's or announcement's
// resource are related.record; an attachment's or link's resource is record.
func preview9MessagingIncludeKey(route, key string) string {
	segs := strings.Split(key, ".")
	out := make([]string, 0, len(segs)+1)
	for i, seg := range segs {
		switch {
		case seg == "topic":
			out = append(out, "related", "record")
		case seg == "resource" && i > 0 && (segs[i-1] == "attachments" || segs[i-1] == "attachment"):
			out = append(out, "record")
		case seg == "resource" && i == 0 && strings.HasPrefix(route, linksRoute):
			out = append(out, "record")
		case seg == "resource":
			out = append(out, "related", "record")
		default:
			out = append(out, seg)
		}
	}
	return strings.Join(out, ".")
}
