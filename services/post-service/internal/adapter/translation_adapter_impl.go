package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"seungpyo.lee/PersonalWebSite/services/post-service/internal/config"
)

type batchRequest struct {
	Text       []string `json:"text"`
	TargetLang string   `json:"target_lang"`
}

type translationAdapterImpl struct {
	cfg *config.PostConfig
}

func NewTranslationAdapter(cfg *config.PostConfig) TranslationAdapter {
	return &translationAdapterImpl{cfg: cfg}
}

// splitIntoSentences splits text into sentences, preserving line breaks
func (t *translationAdapterImpl) splitIntoSentences(text string) []string {
	// Handle empty text
	if strings.TrimSpace(text) == "" {
		return []string{text}
	}

	var sentences []string

	// Split by line breaks first to preserve structure
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			// Add line break between lines
			sentences = append(sentences, "")
		}
		line = strings.TrimSpace(line)
		if line == "" {
			// Preserve empty lines (line breaks)
			sentences = append(sentences, "")
			continue
		}

		// Split line into sentences using regex
		// Regex to match sentence endings (., !, ?) followed by space (at least one)
		sentenceRegex := regexp.MustCompile(`([.!?]+)\s+`)
		parts := sentenceRegex.Split(line, -1)
		delimiters := sentenceRegex.FindAllString(line, -1)

		for j, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			sentence := part
			if j < len(delimiters) {
				sentence += delimiters[j]
			}

			if sentence != "" {
				sentences = append(sentences, sentence)
			}
		}
	}

	return sentences
}

// maskImgSrc masks img src attributes to prevent translation
func (t *translationAdapterImpl) maskImgSrc(text string) (string, []string) {
	re := regexp.MustCompile(`<img[^>]*src="([^"]*)"[^>]*>`)
	matches := re.FindAllStringSubmatch(text, -1)
	placeholders := make([]string, len(matches))
	masked := text
	for i, match := range matches {
		src := match[1]
		placeholders[i] = src
		placeholder := fmt.Sprintf("IMG_SRC_PLACEHOLDER_%d", i)
		// Replace the src value with placeholder
		masked = strings.Replace(masked, `src="`+src+`"`, `src="`+placeholder+`"`, 1)
	}
	return masked, placeholders
}

// unmaskImgSrc restores img src attributes after translation
func (t *translationAdapterImpl) unmaskImgSrc(text string, placeholders []string) string {
	for i, src := range placeholders {
		placeholder := fmt.Sprintf("IMG_SRC_PLACEHOLDER_%d", i)
		text = strings.Replace(text, placeholder, src, -1)
	}
	return text
}

// CallBatch translates a batch of texts using the configured translation API.
func (t *translationAdapterImpl) CallBatch(texts []string, targetLang string) ([]string, error) {
	reqBody := batchRequest{Text: texts, TargetLang: strings.ToUpper(targetLang)}
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

// TranslateSingle is the preferred single-text translation method.
func (t *translationAdapterImpl) TranslateSingle(text string) (string, error) {
	res, err := t.CallBatch([]string{text}, "EN")
	if err != nil {
		return "", err
	}
	if len(res) == 0 {
		return "", fmt.Errorf("empty translation result")
	}
	return res[0], nil
}

// TranslateDelta translates only text inserts inside a Quill Delta.
func (t *translationAdapterImpl) TranslateDelta(content string) (string, error) {
	var delta map[string]interface{}
	if err := json.Unmarshal([]byte(content), &delta); err != nil {
		return "", fmt.Errorf("invalid delta format: %w", err)
	}
	ops, ok := delta["ops"].([]interface{})
	if !ok {
		return "", fmt.Errorf("invalid delta format: missing ops")
	}
	for i, op := range ops {
		opMap, ok := op.(map[string]interface{})
		if !ok {
			continue
		}
		insert, ok := opMap["insert"]
		if !ok {
			continue
		}
		if sStr, ok := insert.(string); ok && sStr != "" {
			// 문장 단위로 나누기
			sentences := t.splitIntoSentences(sStr)
			var translatedSentences []string
			for _, sentence := range sentences {
				if strings.TrimSpace(sentence) == "" {
					// 줄바꿈 유지
					translatedSentences = append(translatedSentences, sentence)
				} else {
					// img 태그 제외하고 번역
					masked, placeholders := t.maskImgSrc(sentence)
					translated, err := t.TranslateSingle(masked)
					if err != nil {
						return "", err
					}
					unmasked := t.unmaskImgSrc(translated, placeholders)
					translatedSentences = append(translatedSentences, unmasked)
				}
			}
			// 재구성하여 글 형식 보존
			var result strings.Builder
			for j, sent := range translatedSentences {
				if sent == "" {
					// 줄바꿈
					if j < len(translatedSentences)-1 {
						result.WriteString("\n")
					}
				} else {
					if j > 0 && translatedSentences[j-1] != "" {
						result.WriteString(" ")
					}
					result.WriteString(sent)
				}
			}
			opMap["insert"] = result.String()
			ops[i] = opMap
		}
	}
	delta["ops"] = ops
	updated, err := json.Marshal(delta)
	if err != nil {
		return "", err
	}
	return string(updated), nil
}
