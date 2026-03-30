package auto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookupTemplate_ByID(t *testing.T) {
	t.Parallel()
	tmpl := LookupTemplate("bugfix")
	require.NotNil(t, tmpl)
	require.Equal(t, "Bug Fix", tmpl.Name)
}

func TestLookupTemplate_ByName(t *testing.T) {
	t.Parallel()
	tmpl := LookupTemplate("Feature")
	require.NotNil(t, tmpl)
	require.Equal(t, "feature", tmpl.ID)
}

func TestLookupTemplate_CaseInsensitive(t *testing.T) {
	t.Parallel()
	tmpl := LookupTemplate("HOTFIX")
	require.NotNil(t, tmpl)
	require.Equal(t, "hotfix", tmpl.ID)
}

func TestLookupTemplate_NotFound(t *testing.T) {
	t.Parallel()
	tmpl := LookupTemplate("nonexistent")
	require.Nil(t, tmpl)
}

func TestListTemplateNames(t *testing.T) {
	t.Parallel()
	names := ListTemplateNames()
	require.Len(t, names, 5)
	require.Contains(t, names[0], "bugfix")
}

func TestTemplateCatalog_AllHaveSlices(t *testing.T) {
	t.Parallel()
	for _, tmpl := range TemplateCatalog {
		require.NotEmpty(t, tmpl.Slices, "template %s has no slices", tmpl.ID)
		for _, s := range tmpl.Slices {
			require.NotEmpty(t, s.Tasks, "slice %q in template %s has no tasks", s.Title, tmpl.ID)
		}
	}
}
