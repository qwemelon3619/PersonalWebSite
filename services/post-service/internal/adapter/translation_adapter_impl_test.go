package adapter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"seungpyo.lee/PersonalWebSite/services/post-service/internal/config"
)

func TestDoRequest_APIKeyMissing(t *testing.T) {
	a := NewTranslationAdapter(&config.PostConfig{TranslationAPIURL: "http://example.com"}).(*translationAdapterImpl)
	if _, err := a.doRequest([]byte(`{}`)); err == nil || !strings.Contains(err.Error(), "translation API key is not configured") {
		t.Fatalf("expected api key missing error, got %v", err)
	}
}

func TestDoRequest_Non200AndParseError(t *testing.T) {
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer s1.Close()
	a1 := NewTranslationAdapter(&config.PostConfig{TranslationAPIURL: s1.URL, TranslationAPIKey: "k"}).(*translationAdapterImpl)
	if _, err := a1.doRequest([]byte(`{}`)); err == nil || !strings.Contains(err.Error(), "translation API error") {
		t.Fatalf("expected non-200 error, got %v", err)
	}
}

func TestCallBatch_ParsesResponse(t *testing.T) {
	var captured map[string]any
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"translations": []map[string]any{
				{"detected_source_language": "KO", "text": "hello"},
				{"detected_source_language": "KO", "text": "world"},
			},
		})
	}))
	defer s.Close()
	a := NewTranslationAdapter(&config.PostConfig{TranslationAPIURL: s.URL, TranslationAPIKey: "k"}).(*translationAdapterImpl)
	out, err := a.CallBatch([]string{"a", "b"}, "en", true, []string{"code"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(out) != 2 || out[0] != "hello" || out[1] != "world" {
		t.Fatalf("unexpected output: %v", out)
	}
	if captured["tag_handling"] != "html" || captured["tag_handling_version"] != "v2" {
		t.Fatalf("expected html mode fields, got %+v", captured)
	}
}

func TestCallBatch_ParseFailureAndEmptyTranslations(t *testing.T) {
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer s1.Close()
	a1 := NewTranslationAdapter(&config.PostConfig{TranslationAPIURL: s1.URL, TranslationAPIKey: "k"}).(*translationAdapterImpl)
	if _, err := a1.CallBatch([]string{"a"}, "EN", false, nil); err == nil || !strings.Contains(err.Error(), "failed to parse DeepL response") {
		t.Fatalf("expected parse error, got %v", err)
	}

	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"translations": []any{}})
	}))
	defer s2.Close()
	a2 := NewTranslationAdapter(&config.PostConfig{TranslationAPIURL: s2.URL, TranslationAPIKey: "k"}).(*translationAdapterImpl)
	if _, err := a2.CallBatch([]string{"a"}, "EN", false, nil); err == nil || !strings.Contains(err.Error(), "empty translations") {
		t.Fatalf("expected empty translations error, got %v", err)
	}
}

func TestTranslateSingle(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"translations": []map[string]any{{"text": "hello"}},
		})
	}))
	defer s.Close()
	a := NewTranslationAdapter(&config.PostConfig{TranslationAPIURL: s.URL, TranslationAPIKey: "k"})
	out, err := a.TranslateSingle("안녕")
	if err != nil || out != "hello" {
		t.Fatalf("expected hello, got out=%q err=%v", out, err)
	}
}

func TestTranslateMarkdown_MasksAndReturnsHTML(t *testing.T) {
	var captured map[string]any
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"translations": []map[string]any{{"text": "<h1>EN</h1><img src=\"/a.png\" translate=\"no\"><pre translate=\"no\">x</pre>"}},
		})
	}))
	defer s.Close()
	a := NewTranslationAdapter(&config.PostConfig{TranslationAPIURL: s.URL, TranslationAPIKey: "k"})
	out, err := a.TranslateMarkdown("# 안녕\n\n![a](/a.png)\n\n```go\nx\n```")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(out, "<h1>EN</h1>") {
		t.Fatalf("expected translated html output, got %q", out)
	}
	texts, _ := captured["text"].([]any)
	if len(texts) != 1 {
		t.Fatalf("expected single text payload")
	}
	payload := texts[0].(string)
	if !strings.Contains(payload, `translate="no"`) {
		t.Fatalf("expected masked html, got %q", payload)
	}
}

func TestTranslateMarkdown_Failure(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer s.Close()
	a := NewTranslationAdapter(&config.PostConfig{TranslationAPIURL: s.URL, TranslationAPIKey: "k"})
	if _, err := a.TranslateMarkdown("x"); err == nil || !strings.Contains(err.Error(), "failed to translate HTML") {
		t.Fatalf("expected translate markdown failure, got %v", err)
	}
}
