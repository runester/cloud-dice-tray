// Package web provides the cloud-dice-tray HTTP application.
package web

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/runester/cloud-dice-tray/internal/dice"
)

//go:embed templates/*.html static/*
var assets embed.FS

type Server struct {
	templates *template.Template
	router    http.Handler
}

func New() (*Server, error) {
	templates, err := template.ParseFS(assets, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	static, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, fmt.Errorf("load static assets: %w", err)
	}

	server := &Server{templates: templates}
	router := chi.NewRouter()
	router.Get("/", server.index)
	router.Post("/dice/validate", server.validate)
	router.Post("/dice/roll", server.roll)
	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	server.router = router
	return server, nil
}

func (s *Server) Handler() http.Handler { return s.router }

func (s *Server) index(w http.ResponseWriter, _ *http.Request) {
	s.render(w, http.StatusOK, "index", nil)
}

func (s *Server) validate(w http.ResponseWriter, r *http.Request) {
	expression, ok := formExpression(w, r)
	if !ok {
		return
	}
	if err := dice.Validate(expression); err != nil {
		s.renderDiceError(w, err)
		return
	}
	s.render(w, http.StatusOK, "valid", nil)
}

func (s *Server) roll(w http.ResponseWriter, r *http.Request) {
	expression, ok := formExpression(w, r)
	if !ok {
		return
	}
	result, err := dice.Evaluate(expression)
	if err != nil {
		s.renderDiceError(w, err)
		return
	}

	rolls := make([]rollView, len(result.Rolls))
	for i, roll := range result.Rolls {
		values := dice.List(roll.Values...)
		rolls[i] = rollView{Notation: roll.Notation, Values: values.String()}
	}
	s.render(w, http.StatusOK, "result", resultView{
		Expression: result.Expression,
		Rolls:      rolls,
		Value:      result.Value.String(),
	})
}

func formExpression(w http.ResponseWriter, r *http.Request) (string, bool) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return "", false
	}
	return strings.TrimSpace(r.FormValue("expression")), true
}

type rollView struct{ Notation, Values string }
type resultView struct {
	Expression string
	Rolls      []rollView
	Value      string
}
type errorView struct{ Message, Position string }

func (s *Server) renderDiceError(w http.ResponseWriter, err error) {
	view := errorView{Message: err.Error()}
	var expressionError *dice.Error
	if errors.As(err, &expressionError) {
		view.Message = expressionError.Message
		if expressionError.Start >= 0 {
			view.Position = fmt.Sprintf("Characters %d–%d · %s", expressionError.Start+1, expressionError.End, expressionError.Code)
		}
	}
	// HTMX does not swap error responses by default. Expression errors are a
	// normal, successfully handled application response, so return 200 and let
	// the error fragment replace any stale validation or roll feedback.
	s.render(w, http.StatusOK, "error", view)
}

func (s *Server) render(w http.ResponseWriter, status int, name string, data any) {
	var output strings.Builder
	if err := s.templates.ExecuteTemplate(&output, name, data); err != nil {
		http.Error(w, "could not render page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(output.String()))
}
