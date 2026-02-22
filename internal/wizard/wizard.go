// Package wizard provides interactive wizard functionality for content creation.
package wizard

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ContentType represents the type of content to create.
type ContentType string

const (
	ContentTypeMarketing  ContentType = "marketing"
	ContentTypeEmail      ContentType = "email"
	ContentTypeReport     ContentType = "report"
	ContentTypeScript     ContentType = "script"
	ContentTypeWhitepaper ContentType = "whitepaper"
	ContentTypeStory      ContentType = "story"
	ContentTypePoem       ContentType = "poem"
)

// Wizard represents an interactive wizard session.
type Wizard struct {
	UserID      int64
	ContentType ContentType
	Answers     map[string]string
	Step        int
	StartedAt   time.Time
	mu          sync.RWMutex
}

// Manager manages wizard sessions for users.
type Manager struct {
	mu       sync.RWMutex
	sessions map[int64]*Wizard
	timeout  time.Duration
}

// NewManager creates a new wizard manager.
func NewManager(timeout time.Duration) *Manager {
	return &Manager{
		sessions: make(map[int64]*Wizard),
		timeout:  timeout,
	}
}

// StartWizard starts a new wizard session for a user.
func (m *Manager) StartWizard(userID int64, contentType ContentType) *Wizard {
	m.mu.Lock()
	defer m.mu.Unlock()

	wizard := &Wizard{
		UserID:      userID,
		ContentType: contentType,
		Answers:     make(map[string]string),
		Step:        0,
		StartedAt:   time.Now(),
	}

	m.sessions[userID] = wizard
	return wizard
}

// GetWizard returns the wizard session for a user, if exists.
func (m *Manager) GetWizard(userID int64) (*Wizard, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	wizard, exists := m.sessions[userID]
	if !exists {
		return nil, false
	}

	// Check timeout
	if time.Since(wizard.StartedAt) > m.timeout {
		m.mu.RUnlock()
		m.mu.Lock()
		delete(m.sessions, userID)
		m.mu.Unlock()
		return nil, false
	}

	return wizard, true
}

// EndWizard ends a wizard session for a user.
func (m *Manager) EndWizard(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, userID)
}

// CancelWizard cancels a wizard session for a user (alias for EndWizard).
func (m *Manager) CancelWizard(userID int64) {
	m.EndWizard(userID)
}

// GetProgress returns the current progress as a string.
func (w *Wizard) GetProgress() string {
	steps := GetSteps(w.ContentType)
	if steps == nil {
		return ""
	}
	return fmt.Sprintf("(Step %d of %d)", w.GetStep()+1, len(steps))
}

// SetAnswer sets an answer for the current step.
func (w *Wizard) SetAnswer(key, value string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Answers[key] = value
	w.Step++
}

// GetAnswer gets an answer by key.
func (w *Wizard) GetAnswer(key string) string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Answers[key]
}

// GetStep returns the current step.
func (w *Wizard) GetStep() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Step
}

// GetAnswers returns all answers.
func (w *Wizard) GetAnswers() map[string]string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	answers := make(map[string]string)
	for k, v := range w.Answers {
		answers[k] = v
	}
	return answers
}

// WizardStep represents a question in the wizard.
type WizardStep struct {
	Key      string
	Question string
	Validate func(answer string) error
}

// GetMarketingSteps returns the wizard steps for marketing content.
func GetMarketingSteps() []WizardStep {
	return []WizardStep{
		{
			Key:      "website_name",
			Question: "What is the name of your website or business?",
		},
		{
			Key:      "website_url",
			Question: "What is the URL of your website?",
		},
		{
			Key:      "target_audience",
			Question: "Who is your target audience? (e.g., small business owners, tech enthusiasts)",
		},
		{
			Key:      "key_benefits",
			Question: "What are the key benefits or features of your product/service?",
		},
		{
			Key:      "tone",
			Question: "What tone would you like? (e.g., professional, friendly, urgent, humorous)",
		},
		{
			Key:      "length",
			Question: "What length would you like? (short/medium/long)",
		},
		{
			Key:      "topic",
			Question: "What specific topic or angle should the marketing copy focus on?",
		},
		{
			Key:      "cta",
			Question: "What call-to-action should be included? (e.g., Sign up now, Learn more, Contact us)",
		},
	}
}

// GetEmailSteps returns the wizard steps for email content.
func GetEmailSteps() []WizardStep {
	return []WizardStep{
		{
			Key:      "subject",
			Question: "What is the subject line of the email?",
		},
		{
			Key:      "recipient",
			Question: "Who is the recipient? (e.g., potential customers, existing clients)",
		},
		{
			Key:      "purpose",
			Question: "What is the purpose of this email? (e.g., newsletter, promotion, announcement)",
		},
		{
			Key:      "tone",
			Question: "What tone would you like? (e.g., formal, casual, friendly)",
		},
		{
			Key:      "key_message",
			Question: "What is the key message or offer you want to convey?",
		},
		{
			Key:      "cta",
			Question: "What action should the recipient take? (e.g., Click here, Reply, Visit)",
		},
	}
}

// GetReportSteps returns the wizard steps for report content.
func GetReportSteps() []WizardStep {
	return []WizardStep{
		{
			Key:      "title",
			Question: "What is the title of the report?",
		},
		{
			Key:      "audience",
			Question: "Who is the target audience for this report?",
		},
		{
			Key:      "topic",
			Question: "What is the main topic or subject of the report?",
		},
		{
			Key:      "scope",
			Question: "What is the scope of the report? (e.g., industry analysis, market research)",
		},
		{
			Key:      "key_points",
			Question: "What are the key points or findings to include?",
		},
		{
			Key:      "length",
			Question: "What length would you like? (brief/medium/comprehensive)",
		},
		{
			Key:      "format",
			Question: "What format would you prefer? (e.g., executive summary, detailed analysis)",
		},
	}
}

// GetScriptSteps returns the wizard steps for script content.
func GetScriptSteps() []WizardStep {
	return []WizardStep{
		{
			Key:      "type",
			Question: "What type of script? (e.g., video, podcast, advertisement)",
		},
		{
			Key:      "topic",
			Question: "What is the main topic or subject?",
		},
		{
			Key:      "duration",
			Question: "What is the desired duration? (e.g., 30 seconds, 5 minutes)",
		},
		{
			Key:      "audience",
			Question: "Who is the target audience?",
		},
		{
			Key:      "tone",
			Question: "What tone would you like? (e.g., serious, humorous, inspirational)",
		},
		{
			Key:      "key_message",
			Question: "What is the key message to convey?",
		},
		{
			Key:      "cta",
			Question: "What call-to-action should be included?",
		},
	}
}

// GetWhitepaperSteps returns the wizard steps for whitepaper content.
func GetWhitepaperSteps() []WizardStep {
	return []WizardStep{
		{
			Key:      "title",
			Question: "What is the title of the whitepaper?",
		},
		{
			Key:      "topic",
			Question: "What is the main topic or research question?",
		},
		{
			Key:      "audience",
			Question: "Who is the target audience?",
		},
		{
			Key:      "problem",
			Question: "What problem or challenge does it address?",
		},
		{
			Key:      "solution",
			Question: "What is the proposed solution or findings?",
		},
		{
			Key:      "length",
			Question: "What length would you like? (short/medium/long)",
		},
		{
			Key:      "tone",
			Question: "What tone would you like? (e.g., academic, professional, accessible)",
		},
	}
}

// GetStorySteps returns the wizard steps for story content.
func GetStorySteps() []WizardStep {
	return []WizardStep{
		{
			Key:      "genre",
			Question: "What genre? (e.g., sci-fi, fantasy, romance, mystery, literary)",
		},
		{
			Key:      "premise",
			Question: "What is the premise or plot idea?",
		},
		{
			Key:      "characters",
			Question: "Describe the main characters (optional):",
		},
		{
			Key:      "setting",
			Question: "What is the setting? (e.g., modern city, medieval kingdom, space station)",
		},
		{
			Key:      "tone",
			Question: "What tone? (e.g., dark, uplifting, suspenseful, humorous)",
		},
		{
			Key:      "length",
			Question: "What length? (short story/novella/novel excerpt)",
		},
	}
}

// GetPoemSteps returns the wizard steps for poem content.
func GetPoemSteps() []WizardStep {
	return []WizardStep{
		{
			Key:      "style",
			Question: "What style of poem? (e.g., haiku, sonnet, free verse, limerick, ballad)",
		},
		{
			Key:      "topic",
			Question: "What is the topic or theme?",
		},
		{
			Key:      "mood",
			Question: "What mood? (e.g., melancholy, joyful, reflective, romantic)",
		},
		{
			Key:      "length",
			Question: "How many lines? (e.g., 4, 8, 16, 32)",
		},
		{
			Key:      "structure",
			Question: "Any specific structure or rhyming scheme? (optional)",
		},
	}
}

// GetSteps returns the wizard steps for a content type.
func GetSteps(contentType ContentType) []WizardStep {
	switch contentType {
	case ContentTypeMarketing:
		return GetMarketingSteps()
	case ContentTypeEmail:
		return GetEmailSteps()
	case ContentTypeReport:
		return GetReportSteps()
	case ContentTypeScript:
		return GetScriptSteps()
	case ContentTypeWhitepaper:
		return GetWhitepaperSteps()
	case ContentTypeStory:
		return GetStorySteps()
	case ContentTypePoem:
		return GetPoemSteps()
	default:
		return nil
	}
}

// GetCurrentQuestion returns the current question for a wizard.
func (w *Wizard) GetCurrentQuestion() string {
	steps := GetSteps(w.ContentType)
	if steps == nil {
		return ""
	}

	step := w.GetStep()
	if step >= len(steps) {
		return ""
	}

	return steps[step].Question
}

// GetCurrentKey returns the current answer key for a wizard.
func (w *Wizard) GetCurrentKey() string {
	steps := GetSteps(w.ContentType)
	if steps == nil {
		return ""
	}

	step := w.GetStep()
	if step >= len(steps) {
		return ""
	}

	return steps[step].Key
}

// IsComplete returns true if the wizard is complete.
func (w *Wizard) IsComplete() bool {
	steps := GetSteps(w.ContentType)
	if steps == nil {
		return true
	}

	return w.GetStep() >= len(steps)
}

// BuildPrompt builds the generation prompt from wizard answers.
func (w *Wizard) BuildPrompt() string {
	answers := w.GetAnswers()

	switch w.ContentType {
	case ContentTypeMarketing:
		return buildMarketingPrompt(answers)
	case ContentTypeEmail:
		return buildEmailPrompt(answers)
	case ContentTypeReport:
		return buildReportPrompt(answers)
	case ContentTypeScript:
		return buildScriptPrompt(answers)
	case ContentTypeWhitepaper:
		return buildWhitepaperPrompt(answers)
	case ContentTypeStory:
		return buildStoryPrompt(answers)
	case ContentTypePoem:
		return buildPoemPrompt(answers)
	default:
		return ""
	}
}

func buildMarketingPrompt(answers map[string]string) string {
	return fmt.Sprintf(`Create marketing copy with the following details:

Website/Business: %s
URL: %s
Target Audience: %s
Key Benefits: %s
Tone: %s
Length: %s
Topic/Angle: %s
Call-to-Action: %s

Please create compelling marketing copy that incorporates all these elements.`,
		answers["website_name"],
		answers["website_url"],
		answers["target_audience"],
		answers["key_benefits"],
		answers["tone"],
		answers["length"],
		answers["topic"],
		answers["cta"])
}

func buildEmailPrompt(answers map[string]string) string {
	return fmt.Sprintf(`Write an email with the following details:

Subject: %s
Recipient: %s
Purpose: %s
Tone: %s
Key Message: %s
Call-to-Action: %s

Please create a complete email incorporating all these elements.`,
		answers["subject"],
		answers["recipient"],
		answers["purpose"],
		answers["tone"],
		answers["key_message"],
		answers["cta"])
}

func buildReportPrompt(answers map[string]string) string {
	return fmt.Sprintf(`Create a report with the following details:

Title: %s
Target Audience: %s
Topic: %s
Scope: %s
Key Points: %s
Length: %s
Format: %s

Please create a comprehensive report incorporating all these elements.`,
		answers["title"],
		answers["audience"],
		answers["topic"],
		answers["scope"],
		answers["key_points"],
		answers["length"],
		answers["format"])
}

func buildScriptPrompt(answers map[string]string) string {
	return fmt.Sprintf(`Write a script with the following details:

Type: %s
Topic: %s
Duration: %s
Target Audience: %s
Tone: %s
Key Message: %s
Call-to-Action: %s

Please create a complete script incorporating all these elements.`,
		answers["type"],
		answers["topic"],
		answers["duration"],
		answers["audience"],
		answers["tone"],
		answers["key_message"],
		answers["cta"])
}

func buildWhitepaperPrompt(answers map[string]string) string {
	return fmt.Sprintf(`Create a whitepaper with the following details:

Title: %s
Topic: %s
Target Audience: %s
Problem/Challenge: %s
Solution/Findings: %s
Length: %s
Tone: %s

Please create a comprehensive whitepaper incorporating all these elements.`,
		answers["title"],
		answers["topic"],
		answers["audience"],
		answers["problem"],
		answers["solution"],
		answers["length"],
		answers["tone"])
}

func buildStoryPrompt(answers map[string]string) string {
	return fmt.Sprintf(`Write a story with the following details:

Genre: %s
Premise: %s
Characters: %s
Setting: %s
Tone: %s
Length: %s

Please create an engaging story incorporating all these elements.`,
		answers["genre"],
		answers["premise"],
		answers["characters"],
		answers["setting"],
		answers["tone"],
		answers["length"])
}

func buildPoemPrompt(answers map[string]string) string {
	return fmt.Sprintf(`Write a poem with the following details:

Style: %s
Topic: %s
Mood: %s
Length: %s lines
Structure: %s

Please create a poem incorporating all these elements.`,
		answers["style"],
		answers["topic"],
		answers["mood"],
		answers["length"],
		answers["structure"])
}

// ParseFlags parses command flags from input string.
func ParseFlags(input string) (map[string]string, string) {
	flags := make(map[string]string)
	var args string

	// Remove leading slash if present
	input = strings.TrimPrefix(input, "/")

	parts := strings.Fields(input)
	remaining := make([]string, 0)
	collectArgs := false

	for _, part := range parts {
		if collectArgs {
			remaining = append(remaining, part)
			continue
		}

		if strings.HasPrefix(part, "-") && len(part) > 1 {
			// It's a flag
			flag := strings.TrimPrefix(part, "-")
			flags[flag] = "true"
		} else if strings.HasPrefix(part, "-") && len(part) == 1 {
			// Next part is value
			collectArgs = true
		} else if len(flagValue(part)) > 0 {
			// It's a flag with value like -temail@example.com
			flagParts := strings.SplitN(part, "=", 2)
			if len(flagParts) == 2 {
				flags[flagParts[0]] = flagParts[1]
			} else {
				remaining = append(remaining, part)
			}
		} else {
			remaining = append(remaining, part)
		}
	}

	// Handle -flag value format
	if len(remaining) >= 2 {
		for i := 0; i < len(remaining)-1; i++ {
			if strings.HasPrefix(remaining[i], "-") {
				flag := strings.TrimPrefix(remaining[i], "-")
				flags[flag] = remaining[i+1]
				i++ // Skip next since it's used as value
			}
		}
	}

	// Get remaining as args
	for _, part := range remaining {
		if !strings.HasPrefix(part, "-") {
			args = part
			break
		}
	}

	return flags, args
}

func flagValue(s string) string {
	if strings.HasPrefix(s, "-") && len(s) > 2 {
		return s[2:]
	}
	return ""
}

// GetContentTypeFromArgs extracts content type from command arguments.
func GetContentTypeFromArgs(args string) (ContentType, string) {
	parts := strings.Fields(args)
	if len(parts) == 0 {
		return "", ""
	}

	first := strings.ToLower(parts[0])
	remaining := ""
	if len(parts) > 1 {
		remaining = strings.Join(parts[1:], " ")
	}

	switch first {
	case "marketing":
		return ContentTypeMarketing, remaining
	case "email":
		return ContentTypeEmail, remaining
	case "report":
		return ContentTypeReport, remaining
	case "script":
		return ContentTypeScript, remaining
	case "whitepaper":
		return ContentTypeWhitepaper, remaining
	case "story":
		return ContentTypeStory, remaining
	case "poem":
		return ContentTypePoem, remaining
	default:
		return "", args
	}
}
