package adapter

type TranslationAdapter interface {
	TranslateSingle(text string) (string, error)
	TranslateDelta(content string) (string, error)
}
