# ============================================
# luceneq — Makefile
# ============================================

GO        := go
GOCMD     := $(shell which $(GO) 2>/dev/null || echo "go")
GOFLAGS   ?=
LINT      := golangci-lint
TOOLS     := github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Цвета для вывода
COLOR_RED    := \033[31m
COLOR_GREEN  := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE   := \033[34m
COLOR_RESET  := \033[0m

.PHONY: all test lint fmt vet clean help \
        tidy mod check pre-commit \
        lint-install test-verbose test-race

# По умолчанию — проверить всё
all: check

# --------------------------------------------
# Сборка
# --------------------------------------------

## build: собрать библиотеку
build:
	@printf "$(COLOR_BLUE)[BUILD]$(COLOR_RESET) собираю библиотеку...\n"
	@$(GOCMD) build ./...
	@printf "$(COLOR_GREEN)[OK]$(COLOR_RESET) сборка успешна\n"

## tidy: обновить go.mod и go.sum
tidy:
	@printf "$(COLOR_BLUE)[TIDY]$(COLOR_RESET) обновляю зависимости...\n"
	@$(GOCMD) mod tidy
	@printf "$(COLOR_GREEN)[OK]$(COLOR_RESET) зависимости обновлены\n"

## mod: показать модульную информацию
mod:
	@$(GOCMD) mod graph
	@$(GOCMD) mod verify

# --------------------------------------------
# Тесты
# --------------------------------------------

## test: запустить все тесты
test:
	@printf "$(COLOR_BLUE)[TEST]$(COLOR_RESET) запускаю тесты...\n"
	@$(GOCMD) test ./... $(GOFLAGS) -count=1
	@printf "$(COLOR_GREEN)[OK]$(COLOR_RESET) все тесты прошли\n"

## test-verbose: запустить тесты с подробным выводом
test-verbose:
	@$(GOCMD) test ./... -v -count=1

## test-race: запустить тесты с детектором гонок
test-race:
	@printf "$(COLOR_BLUE)[TEST]$(COLOR_RESET) запускаю тесты с -race...\n"
	@$(GOCMD) test ./... -race -count=1

## test-cover: запустить тесты с отчётом о покрытии
test-cover:
	@printf "$(COLOR_BLUE)[TEST]$(COLOR_RESET) запускаю тесты с покрытием...\n"
	@$(GOCMD) test ./... -coverprofile=coverage.out -count=1
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@printf "$(COLOR_GREEN)[OK]$(COLOR_RESET) отчёт: coverage.html\n"

## test-bench: запустить бенчмарки
test-bench:
	@$(GOCMD) test ./... -bench=. -run=^$ -benchmem

# --------------------------------------------
# Линтинг и качество кода
# --------------------------------------------

## lint: запустить golangci-lint
lint:
	@printf "$(COLOR_BLUE)[LINT]$(COLOR_RESET) запускаю golangci-lint...\n"
	@if ! command -v $(LINT) &>/dev/null; then \
		printf "$(COLOR_YELLOW)[WARN]$(COLOR_RESET) golangci-lint не найден, устанавливаю...\n"; \
		$(MAKE) lint-install; \
	fi
	@$(LINT) run ./...
	@printf "$(COLOR_GREEN)[OK]$(COLOR_RESET) линтинг прошёл успешно\n"

## lint-install: установить golangci-lint через go install
lint-install:
	@printf "$(COLOR_BLUE)[INSTALL]$(COLOR_RESET) устанавливаю $(LINT)...\n"
	@$(GOCMD) install $(TOOLS)
	@printf "$(COLOR_GREEN)[OK]$(COLOR_RESET) $(LINT) установлен\n"

## lint-fix: запустить линтер с автоисправлениями
lint-fix:
	@if ! command -v $(LINT) &>/dev/null; then \
		$(MAKE) lint-install; \
	fi
	@$(LINT) run ./... --fix

## vet: запустить go vet
vet:
	@printf "$(COLOR_BLUE)[VET]$(COLOR_RESET) запускаю go vet...\n"
	@$(GOCMD) vet ./...
	@printf "$(COLOR_GREEN)[OK]$(COLOR_RESET) vet прошёл успешно\n"

## fmt: форматировать код
fmt:
	@printf "$(COLOR_BLUE)[FMT]$(COLOR_RESET) форматирую код...\n"
	@gofmt -s -w .
	@printf "$(COLOR_GREEN)[OK]$(COLOR_RESET) код отформатирован\n"

## check: полная проверка (lint + vet + test)
check: lint vet test

## pre-commit: подготовка перед коммитом
pre-commit: fmt lint vet test

# --------------------------------------------
# Очистка
# --------------------------------------------

## clean: удалить временные файлы
clean:
	@printf "$(COLOR_YELLOW)[CLEAN]$(COLOR_RESET) удаляю временные файлы...\n"
	@rm -f coverage.out coverage.html
	@printf "$(COLOR_GREEN)[OK]$(COLOR_RESET) очистка завершена\n"

# --------------------------------------------
# Справка
# --------------------------------------------

## help: показать справку
help:
	@printf "\n$(COLOR_BLUE)╔══════════════════════════════════════════╗$(COLOR_RESET)\n"
	@printf "$(COLOR_BLUE)║         luceneq — Makefile Help          ║$(COLOR_RESET)\n"
	@printf "$(COLOR_BLUE)╚══════════════════════════════════════════╝$(COLOR_RESET)\n\n"
	@printf "$(COLOR_YELLOW)Использование: make [цель]$(COLOR_RESET)\n\n"
	@printf "$(COLOR_YELLOW)Цели:$(COLOR_RESET)\n"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | \
		awk -F: '{printf "  $(COLOR_GREEN)%-16s$(COLOR_RESET) %s\n", $$1, $$2}' | \
		sort | \
		column -c2 -t -s :
	@printf "\n"
