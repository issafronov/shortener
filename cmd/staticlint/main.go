/*
Package staticlint запускает кастомный multichecker, состоящий из следующих анализаторов:

1. Стандартные анализаторы:
  - printf, structtag, errorsas, sortslice, httpresponse

2. Анализаторы Staticcheck (https://staticcheck.io):
  - Все SA-анализаторы (предупреждения об ошибках)
  - Один из других классов (например, S1000 — стиль кода)

3. Сторонние анализаторы:
  - asciicheck: запрещает использование не-ASCII символов
  - shadow: проверка на затенение переменных

4. Собственный анализатор:
  - noosexit: запрещает прямой вызов os.Exit внутри main функции пакета main.

Запуск:

	go run ./cmd/staticlint

Вывод будет содержать список всех проблем, найденных анализаторами.

Перед коммитом убедитесь, что ваш проект не вызывает ошибок при прогоне этого multichecker.
*/
package main

import (
	"github.com/issafronov/shortener/cmd/staticlint/noosexit"
	"github.com/tdakkota/asciicheck"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"

	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/structtag"

	"honnef.co/go/tools/staticcheck"

	"strings"
)

func main() {
	var analyzers []*analysis.Analyzer
	seen := make(map[string]bool)

	stdAnalyzers := []*analysis.Analyzer{
		printf.Analyzer,
		structtag.Analyzer,
		errorsas.Analyzer,
		sortslice.Analyzer,
		httpresponse.Analyzer,
	}
	for _, a := range stdAnalyzers {
		analyzers = append(analyzers, a)
		seen[a.Name] = true
	}

	for _, a := range staticcheck.Analyzers {
		name := a.Analyzer.Name
		if strings.HasPrefix(name, "SA") || name == "shadow" {
			if !seen[name] {
				analyzers = append(analyzers, a.Analyzer)
				seen[name] = true
			}
		}
	}

	if !seen[asciicheck.NewAnalyzer().Name] {
		analyzers = append(analyzers, asciicheck.NewAnalyzer())
		seen[asciicheck.NewAnalyzer().Name] = true
	}

	if !seen[noosexit.Analyzer.Name] {
		analyzers = append(analyzers, noosexit.Analyzer)
		seen[noosexit.Analyzer.Name] = true
	}

	multichecker.Main(analyzers...)
}
