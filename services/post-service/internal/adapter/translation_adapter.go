package adapter

type TranslationAdapter interface {
	TranslateSingle(text string) (string, error)
	TranslateMarkdown(content string) (string, error)
}
