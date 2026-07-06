package devstub

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// Topic is a coherent study snippet used by OCR + quiz stubs in local dev.
type Topic struct {
	Name        string
	Paragraph   string
	Question    string
	Correct     string
	Distractors [3]string
	Explanation string
}

var Topics = []Topic{
	{
		Name:        "Photosynthesis",
		Paragraph:   "Photosynthesis is the process by which green plants use sunlight, water, and carbon dioxide to produce glucose and release oxygen. This mainly happens inside chloroplasts.",
		Question:    "Where does photosynthesis mainly occur in plant cells?",
		Correct:     "Chloroplasts",
		Distractors: [3]string{"Mitochondria", "Nucleus", "Cell wall"},
		Explanation: "Chloroplasts contain chlorophyll and are the site where photosynthesis takes place.",
	},
	{
		Name:        "Gravity",
		Paragraph:   "Gravity is a force that pulls objects with mass toward each other. On Earth, it gives weight to physical objects and causes unsupported objects to fall toward the ground.",
		Question:    "What does gravity cause unsupported objects to do near Earth's surface?",
		Correct:     "Fall toward the ground",
		Distractors: [3]string{"Float upward", "Move in circles only", "Lose all mass"},
		Explanation: "Earth's gravity pulls objects toward its center, which we observe as falling.",
	},
	{
		Name:        "Water cycle",
		Paragraph:   "The water cycle describes how water evaporates from surfaces, forms clouds through condensation, and returns to Earth as precipitation such as rain or snow.",
		Question:    "Which step turns water vapor into clouds in the water cycle?",
		Correct:     "Condensation",
		Distractors: [3]string{"Evaporation", "Precipitation", "Photosynthesis"},
		Explanation: "Condensation is the process where water vapor cools and forms tiny droplets in clouds.",
	},
	{
		Name:        "Fractions",
		Paragraph:   "A fraction represents a part of a whole. The numerator shows how many parts are taken, and the denominator shows how many equal parts the whole is divided into.",
		Question:    "In a fraction, what does the denominator represent?",
		Correct:     "Total equal parts in the whole",
		Distractors: [3]string{"Parts we selected", "The answer after adding", "A random whole number"},
		Explanation: "The denominator tells us into how many equal parts the whole is split.",
	},
	{
		Name:        "Electric circuits",
		Paragraph:   "A simple electric circuit needs a power source, conducting wires, and a load such as a bulb. Current flows when the circuit is complete and closed.",
		Question:    "When does electric current flow in a simple circuit?",
		Correct:     "When the circuit is complete and closed",
		Distractors: [3]string{"Only when wires are cut", "Without any power source", "When the switch is always open"},
		Explanation: "Current needs a closed path from the source, through the load, and back.",
	},
	{
		Name:        "Interjections",
		Paragraph:   "Interjections are short words or phrases that express sudden feelings or emotions, such as joy, surprise, or pain. Examples include wow, oh, alas, and hurray. They are often followed by an exclamation mark.",
		Question:    "Which word is an interjection that shows surprise?",
		Correct:     "Wow!",
		Distractors: [3]string{"Quickly", "Because", "Under"},
		Explanation: "Wow! expresses sudden surprise or amazement, which is the role of an interjection.",
	},
	{
		Name:        "Nouns",
		Paragraph:   "A noun is a naming word. It can name a person, place, animal, or thing. Examples: Ram, Delhi, dog, book.",
		Question:    "Which of these is a proper noun?",
		Correct:     "Delhi",
		Distractors: [3]string{"city", "river", "school"},
		Explanation: "Delhi is the specific name of a place, so it is a proper noun.",
	},
}

func TopicForSeed(seed string) Topic {
	sum := sha256.Sum256([]byte(seed))
	return Topics[int(binary.BigEndian.Uint32(sum[:4]))%len(Topics)]
}

func ParagraphForSeed(seed string) string {
	t := TopicForSeed(seed)
	return fmt.Sprintf("Topic: %s. %s", t.Name, t.Paragraph)
}

// ParagraphForChapterHint picks a topic that matches the chapter title (dev stub only).
func ParagraphForChapterHint(chapterTitle, fallbackSeed string) string {
	t := TopicForChapterHint(chapterTitle, fallbackSeed)
	return fmt.Sprintf("Chapter: %s. Topic: %s. %s", chapterTitle, t.Name, t.Paragraph)
}

// TopicForChapterHint matches chapter title keywords to a study topic.
func TopicForChapterHint(chapterTitle, fallbackSeed string) Topic {
	if matched := TopicsMatchingText(chapterTitle); len(matched) > 0 {
		return matched[0]
	}
	for _, topic := range Topics {
		if containsFold(chapterTitle, topic.Name) {
			return topic
		}
	}
	return TopicForSeed(fallbackSeed + ":" + chapterTitle)
}

func TopicsMatchingText(text string) []Topic {
	if text == "" {
		return nil
	}
	lower := text
	matched := make([]Topic, 0, len(Topics))
	seen := make(map[string]struct{}, len(Topics))
	for _, topic := range Topics {
		if _, ok := seen[topic.Name]; ok {
			continue
		}
		if containsFold(lower, topic.Name) {
			seen[topic.Name] = struct{}{}
			matched = append(matched, topic)
		}
	}
	return matched
}

func containsFold(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		(stringIndexFold(haystack, needle) >= 0)
}

func stringIndexFold(s, sub string) int {
	// small helper — avoids strings.ToLower alloc on full haystack
	for i := 0; i+len(sub) <= len(s); i++ {
		if equalFold(s[i:i+len(sub)], sub) {
			return i
		}
	}
	return -1
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func FillBlankText(t Topic) string {
	return fmt.Sprintf("%s mainly relates to _____.", t.Name)
}

func TrueStatement(t Topic) string {
	return fmt.Sprintf("%s: %s", t.Name, t.Paragraph)
}

func FalseStatement(t Topic) string {
	return fmt.Sprintf("%s never happens in real life and has no scientific basis.", t.Name)
}

func SimpleHindiExplanation(t Topic) string {
	return fmt.Sprintf("Sahi jawab: %s. %s — Example: school book se yaad karo ki %s simple words mein samjhaya gaya hai.", t.Correct, t.Explanation, t.Name)
}
