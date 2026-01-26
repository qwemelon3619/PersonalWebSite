package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	quill "github.com/chocobits/go-delta-json-to-html"
	"golang.org/x/net/html"
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

// deltaToHTML converts Quill Delta JSON to HTML string.
func (t *translationAdapterImpl) deltaToHTML(deltaStr string) (string, error) {
	var delta map[string]interface{}
	if err := json.Unmarshal([]byte(deltaStr), &delta); err != nil {
		return "", err
	}
	ops, ok := delta["ops"]
	if !ok {
		return "", fmt.Errorf("invalid delta: missing ops")
	}
	opsBytes, err := json.Marshal(ops)
	if err != nil {
		return "", err
	}
	htmlBytes, err := quill.Render(opsBytes)
	if err != nil {
		return "", err
	}
	return string(htmlBytes), nil
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

func (t *translationAdapterImpl) htmlToDelta(htmlStr string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	var ops []map[string]interface{}

	var traverse func(*html.Node, map[string]interface{})
	traverse = func(n *html.Node, inheritedAttrs map[string]interface{}) {
		currentAttrs := make(map[string]interface{})
		for k, v := range inheritedAttrs {
			currentAttrs[k] = v
		}

		if n.Type == html.ElementNode {
			var class string
			var style string
			for _, a := range n.Attr {
				if a.Key == "class" {
					class = a.Val
				}
				if a.Key == "style" {
					style = a.Val
				}
			}

			switch n.Data {
			case "span":
				// style 속성에서 color 및 background-color 추출
				if style != "" {
					styles := strings.Split(style, ";")
					for _, s := range styles {
						kv := strings.Split(s, ":")
						if len(kv) == 2 {
							key := strings.TrimSpace(kv[0])
							val := strings.TrimSpace(kv[1])
							if key == "color" {
								currentAttrs["color"] = val
							} else if key == "background-color" {
								currentAttrs["background"] = val
							}
						}
					}
				}
			case "strong", "b":
				currentAttrs["bold"] = true
			case "em", "i":
				currentAttrs["italic"] = true
			case "u":
				currentAttrs["underline"] = true
			case "code":
				if !isInside(n, "pre") && !strings.Contains(class, "ql-code-block") {
					currentAttrs["code"] = true
				}
			case "pre", "h1", "h2", "h3", "h4", "h5", "h6", "li", "p", "div":
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					traverse(c, currentAttrs)
				}

				attr := make(map[string]interface{})
				if n.Data == "pre" || strings.Contains(class, "ql-code-block") {
					attr["code-block"] = true
				} else if len(n.Data) == 2 && n.Data[0] == 'h' {
					attr["header"] = int(n.Data[1] - '0')
				} else if n.Data == "li" {
					attr["list"] = "bullet"
					if n.Parent != nil && n.Parent.Data == "ol" {
						attr["list"] = "ordered"
					}
				}

				lastIdx := len(ops) - 1
				if lastIdx >= 0 && ops[lastIdx]["insert"] == "\n" {
					if len(attr) > 0 {
						ops[lastIdx]["attributes"] = attr
					}
				} else {
					newOp := map[string]interface{}{"insert": "\n"}
					if len(attr) > 0 {
						newOp["attributes"] = attr
					}
					ops = append(ops, newOp)
				}
				return
			case "br":
				ops = append(ops, map[string]interface{}{"insert": "\n"})
				return
			case "img":
				for _, attr := range n.Attr {
					if attr.Key == "src" {
						ops = append(ops, map[string]interface{}{
							"insert": map[string]interface{}{"image": attr.Val},
						})
						break
					}
				}
				return
			}
		} else if n.Type == html.TextNode {
			text := strings.ReplaceAll(n.Data, "\n", "")
			if text != "" {
				op := map[string]interface{}{"insert": text}
				if len(currentAttrs) > 0 {
					op["attributes"] = currentAttrs
				}
				ops = append(ops, op)
			}
			return
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c, currentAttrs)
		}
	}

	traverse(doc, nil)

	deltaBytes, _ := json.Marshal(map[string]interface{}{"ops": ops})
	return string(deltaBytes), nil
}

// 헬퍼: 특정 태그 내부에 있는지 확인
func isInside(n *html.Node, tag string) bool {
	p := n.Parent
	for p != nil {
		if p.Data == tag {
			return true
		}
		p = p.Parent
	}
	return false
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

// TranslateDelta translates only text inserts inside a Quill Delta via HTML conversion.
func (t *translationAdapterImpl) TranslateDelta(content string) (string, error) {
	// Convert Delta to HTML
	htmlStr, err := t.deltaToHTML(content)
	if err != nil {
		return "", fmt.Errorf("failed to convert delta to HTML: %w", err)
	}

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

	// Convert back to Delta
	translatedDelta, err := t.htmlToDelta(translatedHTML)
	if err != nil {
		return "", fmt.Errorf("failed to convert HTML back to delta: %w", err)
	}

	return translatedDelta, nil
}
