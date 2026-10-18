package protocol

import "time"

// The person decides in the orchestrator's conversation, through a card: the
// questions of one kind that wait together, one item each, shown one at a
// time, each with its options, its recommendation selected, and room for the
// person's own words. The card is a view of the queue: answering an item is
// answering its question, and only the person's POST answers.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.5.

// AnswerOther is the answer in the person's own words, the words as the
// note. On a design question it is the answer; on a merge or a ceiling
// question, whose answers the harness acts on, the words go to the
// orchestrator and the question keeps waiting (CardItemRelayed).
const AnswerOther = "other"

// The states of an item of a card.
const (
	// CardItemPending waits for the person.
	CardItemPending = "pending"
	// CardItemSkipped is one the person passed over when they confirmed the
	// card: its question still waits.
	CardItemSkipped = "skipped"
	// CardItemRelayed is one the person answered in their own words where
	// the harness acts on an option: the words went to the orchestrator,
	// which proposes answers, and the question still waits.
	CardItemRelayed = "relayed"
	// CardItemAnswered is one whose question has its answer, from this card
	// or from any other place the person answered it.
	CardItemAnswered = "answered"
)

// The status of one answer of a batch (AnswerResult.Status): the item's new
// state, or refused with the code of why.
const AnswerRefused = "refused"

// QuestionCard groups the questions of one kind that wait together — three
// merges ready, say — in the order they were asked. A card is proposed when
// the orchestrator made it from the person's words: its items come with the
// answers it understood selected, and they are answers only once the person
// confirms them with their own POST.
type QuestionCard struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
	// Proposed says the selections are the orchestrator's proposal, and From
	// is the person's words it came from.
	Proposed bool   `json:"proposed,omitempty"`
	From     string `json:"from,omitempty"`
	// Why is the orchestrator's reason for a proposed card.
	Why   string     `json:"why,omitempty"`
	Items []CardItem `json:"items"`
	// Pending is how many items still wait — pending, skipped or relayed:
	// the card needs the person while it is above zero.
	Pending int        `json:"pending"`
	Created time.Time  `json:"created"`
	Updated time.Time  `json:"updated"`
	Closed  *time.Time `json:"closed,omitempty"`
}

// CardItem is one question of a card, flat, with where it is in the card.
// Selected is the option shown selected: the recommendation, or on a
// proposed card the answer the orchestrator understood, with its note.
type CardItem struct {
	Question     `tstype:",extends"`
	State        string `json:"state"`
	Selected     string `json:"selected"`
	SelectedNote string `json:"selected_note,omitempty"`
	// Other is the person's own words, on an item that was relayed.
	Other string `json:"other,omitempty"`
}

// QuestionEvidence is what the harness measured of a child, read from the
// decision log's lines: the reviewer's verdict and the gate's result, each
// with the line it came from. What the child said of its work is no part of it.
type QuestionEvidence struct {
	// Verdict is the reviewer's, on ReviewedCommit; Findings how many it
	// pointed at. ReviewSeq is the review.verdict line.
	Verdict        string `json:"verdict"`
	Reviewer       string `json:"reviewer,omitempty"`
	ReviewedCommit string `json:"reviewed_commit"`
	Findings       int    `json:"findings,omitempty"`
	ReviewSeq      uint64 `json:"review_seq"`
	// Gate is the result of Command on Commit, over main at Base. GateSeq is
	// the gate.measured line.
	Gate       string `json:"gate"`
	Command    string `json:"command"`
	Commit     string `json:"commit"`
	Base       string `json:"base"`
	ExitCode   int    `json:"exit_code"`
	DurationMS int    `json:"duration_ms"`
	GateSeq    uint64 `json:"gate_seq"`
}

// AnswerItem is one answer of a batch.
type AnswerItem struct {
	QuestionID string `json:"question_id"`
	Answer     string `json:"answer"`
	Note       string `json:"note,omitempty"`
}

// AnswerQuestionsRequest is the person's confirm of a card: every answer
// they gave, in one POST, and the items they skipped, which keep waiting.
// Skipped needs CardID, the card they were skipped in.
type AnswerQuestionsRequest struct {
	CardID  string       `json:"card_id,omitempty"`
	Answers []AnswerItem `json:"answers"`
	Skipped []string     `json:"skipped,omitempty"`
}

// AnswerResult is what became of one answer of a batch: answered, relayed,
// skipped, or refused with the code and why. Question is the question as it
// is now, when there is one.
type AnswerResult struct {
	QuestionID string    `json:"question_id"`
	Status     string    `json:"status"`
	Code       string    `json:"code,omitempty"`
	Message    string    `json:"message,omitempty"`
	Question   *Question `json:"question,omitempty"`
}

// AnswerQuestionsResponse answers the batch: one result per answer and per
// skip, in the order sent, and the cards they changed. The first answer to a
// question wins, inside the batch too.
type AnswerQuestionsResponse struct {
	Results []AnswerResult `json:"results"`
	Cards   []QuestionCard `json:"cards"`
}
