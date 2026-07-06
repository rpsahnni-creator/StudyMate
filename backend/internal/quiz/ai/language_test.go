package ai

import "testing"

func TestDetectContentLanguage_English(t *testing.T) {
	got := DetectContentLanguage("Photosynthesis is the process by which green plants make food.")
	if got != "english" {
		t.Fatalf("expected english, got %s", got)
	}
}

func TestDetectContentLanguage_Hindi(t *testing.T) {
	got := DetectContentLanguage("प्रकाश संश्लेषण वह प्रक्रिया है जिससे हरे पौधे अपना भोजन बनाते हैं।")
	if got != "hindi" {
		t.Fatalf("expected hindi, got %s", got)
	}
}

func TestExplanationRulesForLanguage_English(t *testing.T) {
	rules := ExplanationRulesForLanguage("english")
	if rules.Language != "english" {
		t.Fatalf("expected english rules, got %s", rules.Language)
	}
}

func TestExplanationRulesForLanguage_AutoUsesMatchPage(t *testing.T) {
	rules := ExplanationRulesForLanguage(LanguageAuto)
	if rules.Language != "match_page" {
		t.Fatalf("expected match_page, got %s", rules.Language)
	}
}

func TestExplanationDBCode(t *testing.T) {
	if ExplanationDBCode("english") != "en" {
		t.Fatal("expected en")
	}
	if ExplanationDBCode("hindi") != "hi" {
		t.Fatal("expected hi")
	}
}
