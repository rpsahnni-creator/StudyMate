package devstub

import "testing"

func TestTopicsMatchingTextFindsTopicName(t *testing.T) {
	text := "Topic: Photosynthesis. Photosynthesis is the process..."
	matched := TopicsMatchingText(text)
	if len(matched) != 1 || matched[0].Name != "Photosynthesis" {
		t.Fatalf("expected photosynthesis topic, got %+v", matched)
	}
}

func TestTopicForSeedDeterministic(t *testing.T) {
	a := TopicForSeed("job:42:page:1")
	b := TopicForSeed("job:42:page:1")
	if a.Name != b.Name {
		t.Fatalf("expected deterministic topic")
	}
}

func TestParagraphForSeedIncludesTopic(t *testing.T) {
	p := ParagraphForSeed("seed-1")
	if !containsFold(p, "Topic:") {
		t.Fatalf("expected topic prefix in paragraph: %q", p)
	}
}

func TestTopicForChapterHintInterjections(t *testing.T) {
	topic := TopicForChapterHint("Interjections", "job:1")
	if topic.Name != "Interjections" {
		t.Fatalf("expected Interjections topic, got %q", topic.Name)
	}
	p := ParagraphForChapterHint("Interjections", "job:1")
	if !containsFold(p, "Interjections") {
		t.Fatalf("expected interjections in paragraph: %q", p)
	}
}
