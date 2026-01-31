package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/gomarkdown/markdown"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/config"
)

type batchRequest struct {
	Text               []string `json:"text"`
	TargetLang         string   `json:"target_lang"`
	TagHandling        string   `json:"tag_handling,omitempty"`
	TagHandlingVersion string   `json:"tag_handling_version,omitempty"`
	IgnoreTags         []string `json:"ignore_tags,omitempty"`
}

type translationAdapterImpl struct {
	cfg *config.PostConfig
}

func NewTranslationAdapter(cfg *config.PostConfig) TranslationAdapter {
	return &translationAdapterImpl{cfg: cfg}
}

// CallBatch translates a batch of texts using the configured translation API.
func (t *translationAdapterImpl) CallBatch(texts []string, targetLang string, htmlMode bool, ignoreTags []string) ([]string, error) {
	reqBody := batchRequest{Text: texts, TargetLang: strings.ToUpper(targetLang)}
	if htmlMode {
		reqBody.TagHandling = "html"
		reqBody.TagHandlingVersion = "v2"
		reqBody.IgnoreTags = ignoreTags
	}
	bb, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// send request and get raw response bytes
	respBytes, err := t.doRequest(bb)
	if err != nil {
		return nil, err
	}

	// Expect DeepL-style response: { "translations": [ {"detected_source_language":"EN","text":"..."}, ... ] }
	type deeplResp struct {
		Translations []struct {
			DetectedSourceLanguage string `json:"detected_source_language"`
			Text                   string `json:"text"`
		} `json:"translations"`
	}
	var dr deeplResp
	if err := json.Unmarshal(respBytes, &dr); err != nil {
		return nil, fmt.Errorf("failed to parse DeepL response: %w; body=%s", err, string(respBytes))
	}
	if len(dr.Translations) == 0 {
		return nil, fmt.Errorf("empty translations in response: %s", string(respBytes))
	}
	out := make([]string, 0, len(dr.Translations))
	for _, it := range dr.Translations {
		out = append(out, it.Text)
	}
	return out, nil
}

// doRequest performs the HTTP POST and returns response body bytes.
func (t *translationAdapterImpl) doRequest(body []byte) ([]byte, error) {
	if t.cfg.TranslationAPIKey == "" {
		return nil, fmt.Errorf("translation API key is not configured")
	}

	req, err := http.NewRequest("POST", t.cfg.TranslationAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "DeepL-Auth-Key "+t.cfg.TranslationAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call translation API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bb, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("translation API error (status %d): %s", resp.StatusCode, string(bb))
	}

	bb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read translation response: %w", err)
	}
	return bb, nil
}

// markdownToHTML converts Markdown string to HTML string.
func (t *translationAdapterImpl) markdownToHTML(markdownStr string) string {
	htmlBytes := markdown.ToHTML([]byte(markdownStr), nil, nil)
	return string(htmlBytes)
}

// TranslateSingle is the preferred single-text translation method.
func (t *translationAdapterImpl) TranslateSingle(text string) (string, error) {
	res, err := t.CallBatch([]string{text}, "EN", false, nil)
	if err != nil {
		return "", err
	}
	if len(res) == 0 {
		return "", fmt.Errorf("empty translation result")
	}
	return res[0], nil
}

func (t *translationAdapterImpl) htmlToMarkdown(htmlStr string) (string, error) {
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(htmlStr)
	if err != nil {
		return "", fmt.Errorf("failed to convert HTML to Markdown: %w", err)
	}

	// Fix extra newlines before code blocks
	markdown = strings.ReplaceAll(markdown, "\n\n```", "\n```")

	return markdown, nil
}

// maskHTML adds translate="no" to img and code/pre tags to prevent translation.
func (t *translationAdapterImpl) maskHTML(htmlStr string) string {
	// Add translate="no" to img tags
	imgRegex := regexp.MustCompile(`<img([^>]*)>`)
	htmlStr = imgRegex.ReplaceAllStringFunc(htmlStr, func(match string) string {
		if strings.Contains(match, `translate="no"`) {
			return match
		}
		return strings.Replace(match, `<img`, `<img translate="no"`, 1)
	})

	// Add translate="no" to code/pre tags
	codeRegex := regexp.MustCompile(`<(code|pre)([^>]*)>`)
	htmlStr = codeRegex.ReplaceAllStringFunc(htmlStr, func(match string) string {
		if strings.Contains(match, `translate="no"`) {
			return match
		}
		tag := regexp.MustCompile(`<(code|pre)`).FindStringSubmatch(match)[1]
		return strings.Replace(match, `<`+tag, `<`+tag+` translate="no"`, 1)
	})

	return htmlStr
}

// TranslateMarkdown translates Markdown via HTML conversion.
func (t *translationAdapterImpl) TranslateMarkdown(content string) (string, error) {
	// Convert Markdown to HTML
	htmlStr := t.markdownToHTML(content)

	// Add translate="no" to non-translatable tags
	maskedHTML := t.maskHTML(htmlStr)

	// Translate HTML using DeepL
	translatedHTMLs, err := t.CallBatch([]string{maskedHTML}, "EN", true, nil)
	if err != nil {
		return "", fmt.Errorf("failed to translate HTML: %w", err)
	}
	if len(translatedHTMLs) == 0 {
		return "", fmt.Errorf("empty HTML translation result")
	}
	translatedHTML := translatedHTMLs[0]

	// Convert back to Markdown
	translatedMarkdown, err := t.htmlToMarkdown(translatedHTML)
	if err != nil {
		return "", fmt.Errorf("failed to convert HTML back to Markdown: %w", err)
	}

	return translatedMarkdown, nil
}
