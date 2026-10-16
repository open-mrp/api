package constants

// NoteType is what kind of thing a note says. Agents recall account-wide notes of every type but `comment`.
type NoteType string

const (
	// NoteTypeComment is a remark.
	NoteTypeComment NoteType = "comment"
	// NoteTypePreference is how someone likes things done.
	NoteTypePreference NoteType = "preference"
	// NoteTypeFact is a durable detail.
	NoteTypeFact NoteType = "fact"
	// NoteTypeInstruction is standing guidance to follow.
	NoteTypeInstruction NoteType = "instruction"
)

func (t NoteType) IsValid() bool {
	switch t {
	case NoteTypeComment, NoteTypePreference, NoteTypeFact, NoteTypeInstruction:
		return true
	default:
		return false
	}
}

func (t NoteType) EnumValues() []string {
	return []string{string(NoteTypeComment), string(NoteTypePreference), string(NoteTypeFact), string(NoteTypeInstruction)}
}

func (t *NoteType) StringPtr() *string {
	if t == nil {
		return nil
	}
	v := string(*t)
	return &v
}

// NoteImportance orders the notes an agent recalls: it reads the most important first.
type NoteImportance string

const (
	NoteImportanceLow    NoteImportance = "low"
	NoteImportanceNormal NoteImportance = "normal"
	NoteImportanceHigh   NoteImportance = "high"
)

func (i NoteImportance) IsValid() bool {
	switch i {
	case NoteImportanceLow, NoteImportanceNormal, NoteImportanceHigh:
		return true
	default:
		return false
	}
}

func (i NoteImportance) EnumValues() []string {
	return []string{string(NoteImportanceLow), string(NoteImportanceNormal), string(NoteImportanceHigh)}
}

func (i *NoteImportance) StringPtr() *string {
	if i == nil {
		return nil
	}
	v := string(*i)
	return &v
}

// Rank is the importance as the note table stores it, so recall can order by it: low 1, normal 2, high 3.
func (i NoteImportance) Rank() int32 {
	switch i {
	case NoteImportanceLow:
		return 1
	case NoteImportanceHigh:
		return 3
	default:
		return 2
	}
}

// NoteImportanceFromRank is the importance a stored rank holds.
func NoteImportanceFromRank(rank int32) NoteImportance {
	switch {
	case rank <= 1:
		return NoteImportanceLow
	case rank >= 3:
		return NoteImportanceHigh
	default:
		return NoteImportanceNormal
	}
}

// NoteStatus is whether a note is still in use. It is derived from `expires_at`, never set.
type NoteStatus string

const (
	// NoteStatusActive has no `expires_at`, or one still in the future.
	NoteStatusActive NoteStatus = "active"
	// NoteStatusExpired is past its `expires_at`. Agents no longer recall it.
	NoteStatusExpired NoteStatus = "expired"
)

func (s NoteStatus) IsValid() bool {
	switch s {
	case NoteStatusActive, NoteStatusExpired:
		return true
	default:
		return false
	}
}

func (s NoteStatus) EnumValues() []string {
	return []string{string(NoteStatusActive), string(NoteStatusExpired)}
}

func (s *NoteStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

// NoteImportanceFromScore reads a 0–1 importance score, as agent memories stored it: below 0.4 low, from 0.7 high.
func NoteImportanceFromScore(score float64) NoteImportance {
	switch {
	case score < 0.4:
		return NoteImportanceLow
	case score >= 0.7:
		return NoteImportanceHigh
	default:
		return NoteImportanceNormal
	}
}
