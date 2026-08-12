package templates_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rezexell/bashtt/internal/domain"
	"github.com/rezexell/bashtt/internal/templates"
)

func TestProvider_Get(t *testing.T) {
	provider := templates.New()

	tests := []struct {
		name     string
		template domain.Template
		want     string
	}{
		{
			name:     "template1",
			template: domain.Template1,
			want: `#!/bin/bash

echo "Template 1"
date
hostname
`,
		},
		{
			name:     "template2",
			template: domain.Template2,
			want: `#!/bin/bash

echo "Template 2"
whoami
pwd
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := provider.Get(
				context.Background(),
				tt.template,
			)

			require.NoError(t, err)
			require.Equal(t, tt.want, string(got))
		})
	}
}

func TestProvider_Get_InvalidTemplate(t *testing.T) {
	provider := templates.New()

	_, err := provider.Get(
		context.Background(),
		domain.Template("unknown"),
	)

	require.ErrorIs(
		t,
		err,
		domain.ErrInvalidTemplate,
	)
}

func TestProvider_Get_ContextCanceled(t *testing.T) {
	provider := templates.New()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := provider.Get(
		ctx,
		domain.Template1,
	)

	require.ErrorIs(t, err, context.Canceled)
}
