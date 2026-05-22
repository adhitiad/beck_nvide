package usecase

import (
	"context"
	"regexp"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type compiledWord struct {
	ID            domain.UUID
	Word          string
	SeverityLevel int
	Language      string
	IsRegex       bool
	Reg           *regexp.Regexp
}

func (u *moderationUseCase) ReloadWordlist(ctx context.Context) error {
	u.wordlistMu.Lock()
	defer u.wordlistMu.Unlock()

	words, err := u.repo.GetWordlist(ctx)
	if err != nil {
		return err
	}

	var compiled []*compiledWord
	for _, w := range words {
		var r *regexp.Regexp
		if w.IsRegex {
			r, err = regexp.Compile("(?i)" + w.Word)
			if err != nil {
				u.logger.Warn("Failed to compile regex word", zap.String("word", w.Word), zap.Error(err))
				continue
			}
		} else {
			// exact word border matching
			r, err = regexp.Compile("(?i)\\b" + regexp.QuoteMeta(w.Word) + "\\b")
			if err != nil {
				continue
			}
		}

		compiled = append(compiled, &compiledWord{
			ID:            w.ID,
			Word:          w.Word,
			SeverityLevel: w.SeverityLevel,
			Language:      w.Language,
			IsRegex:       w.IsRegex,
			Reg:           r,
		})
	}

	u.wordlistRegex = compiled
	u.logger.Info("Toxicity wordlist compiled successfully", zap.Int("count", len(compiled)))
	return nil
}

// Wordlist CRUD
func (u *moderationUseCase) GetWordlist(ctx context.Context) ([]*domain.ModerationWordlist, error) {
	return u.repo.GetWordlist(ctx)
}

func (u *moderationUseCase) AddWord(ctx context.Context, word string, severity int, lang string, isRegex bool) error {
	w := &domain.ModerationWordlist{
		ID:            domain.NewUUID(),
		Word:          word,
		SeverityLevel: severity,
		Language:      lang,
		IsRegex:       isRegex,
	}
	err := u.repo.AddWord(ctx, w)
	if err != nil {
		return err
	}
	return u.ReloadWordlist(ctx)
}

func (u *moderationUseCase) DeleteWord(ctx context.Context, word string) error {
	err := u.repo.DeleteWord(ctx, word)
	if err != nil {
		return err
	}
	return u.ReloadWordlist(ctx)
}
