package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	req, err := http.NewRequest("POST", t.cfg.TranslationAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if t.cfg.TranslationAPIKey != "" {
		req.Header.Set("Authorization", "DeepL-Auth-Key "+t.cfg.TranslationAPIKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bb, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("translation service responded %d: %s", resp.StatusCode, string(bb))
	}
	bb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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
	var segments []string
	for _, op := range ops {
		opMap, ok := op.(map[string]interface{})
		if !ok {
			continue
		}
		insert, ok := opMap["insert"]
		if !ok {
			continue
		}
		if sStr, ok := insert.(string); ok {
			segments = append(segments, sStr)
		}
	}
	if len(segments) == 0 {
		return content, nil
	}
	// Split segments by \n to preserve line breaks
	var allLines []string
	var segmentIndices []int // index in segments for each line
	for i, seg := range segments {
		lines := strings.Split(seg, "\n")
		allLines = append(allLines, lines...)
		for range lines {
			segmentIndices = append(segmentIndices, i)
		}
	}
	translatedLines, err := t.CallBatch(allLines, "EN")
	if err != nil {
		return "", err
	}
	if len(translatedLines) != len(allLines) {
		return "", fmt.Errorf("translation line count mismatch: got %d, want %d", len(translatedLines), len(allLines))
	}
	// Reconstruct segments
	translatedSegments := make([]string, len(segments))
	lineIdx := 0
	for i := range segments {
		var lines []string
		for segmentIndices[lineIdx] == i {
			lines = append(lines, translatedLines[lineIdx])
			lineIdx++
			if lineIdx >= len(segmentIndices) {
				break
			}
		}
		translatedSegments[i] = strings.Join(lines, "\n")
	}
	if len(translatedSegments) != len(segments) {
		return "", fmt.Errorf("reconstructed segment count mismatch: got %d, want %d", len(translatedSegments), len(segments))
	}
	si := 0
	for i, op := range ops {
		opMap, ok := op.(map[string]interface{})
		if !ok {
			continue
		}
		insert, ok := opMap["insert"]
		if !ok {
			continue
		}
		if _, ok := insert.(string); ok {
			opMap["insert"] = translatedSegments[si]
			ops[i] = opMap
			si++
		}
	}
	delta["ops"] = ops
	updated, err := json.Marshal(delta)
	if err != nil {
		return "", err
	}
	return string(updated), nil
}
