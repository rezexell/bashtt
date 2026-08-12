package templates

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/rezexell/bashtt/internal/domain"
)

//go:embed template1.sh template2.sh
var templateFS embed.FS

type Provider struct{}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) Get(
	ctx context.Context,
	template domain.Template,
) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var filename string

	switch template {
	case domain.Template1:
		filename = "template1.sh"

	case domain.Template2:
		filename = "template2.sh"

	default:
		return nil, fmt.Errorf(
			"get template %q: %w",
			template,
			domain.ErrInvalidTemplate,
		)
	}

	data, err := fs.ReadFile(templateFS, filename)
	if err != nil {
		return nil, fmt.Errorf(
			"read embedded template %q: %w",
			filename,
			err,
		)
	}

	return []byte(strings.TrimSpace(string(data)) + "\n"), nil
}
