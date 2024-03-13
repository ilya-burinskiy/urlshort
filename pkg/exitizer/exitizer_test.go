package exitizer_test

import (
	"testing"

	"github.com/ilya-burinskiy/urlshort/pkg/exitizer"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestExitizer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), exitizer.Analyzer, "./...")
}
