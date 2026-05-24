// Package server provides the browser UI for DESi Paper Review.
package server

import (
	"context"
	"html/template"
	"io"
	"net/http"
	"time"

	"github.com/hstre/desi-paper-review-web/internal/desi"
	"github.com/hstre/desi-paper-review-web/internal/review"
)

// SamplePaper is a built-in demonstration paper with deliberate
// overclaims and reproducibility gaps.
const SamplePaper = `# A Novel Method That Solves Generalization and Proves Robust Across Tasks

## Abstract
We present the first method that solves the long-standing problem of
generalization in learning systems. Our novel approach generalizes to
any task and proves robust under arbitrary conditions. We claim
significant gains and argue that the method establishes a new standard
for the field.

## Introduction
Earlier approaches struggle to generalize beyond the setting they were
designed for. We introduce a method that, we argue, solves this
difficulty once and for all. The idea is novel, and we expect it to
prove robust where previous ideas fall short.

## Method
Our method applies a sequence of transformations to the input and
returns a refined representation. The procedure is simple and, we
believe, broadly applicable across any domain.

## Results
Our approach achieves high accuracy and strong performance on a dataset.
The method generalizes and remains robust throughout. We observe
consistent gains in our experiments, and the improvements look
significant relative to what one would naively expect.

## Discussion
The method appears to solve a hard problem and proves effective in the
settings we considered. We expect it to generalize widely and to remain
robust as conditions change.

## Limitations
However, the method does not yet handle noisy inputs well, and a careful
treatment of failure modes remains future work.

## Conclusion
We have presented the first approach that solves generalization, stays
robust, and proves significant in practice. The method is novel and, we
argue, generalizes to any future task.
`

// Server serves the UI and runs reviews via the pipeline.
type Server struct {
	pipeline *review.Pipeline
	desi     *desi.Client
	tmpl     *template.Template
}

// New builds a server that talks to the DESi service at desiURL.
func New(desiURL string, cfg review.Config) *Server {
	client := desi.New(desiURL)
	return &Server{
		pipeline: &review.Pipeline{Desi: client, Config: cfg},
		desi:     client,
		tmpl:     template.Must(template.New("page").Parse(pageTemplate)),
	}
}

// Handler returns the HTTP handler (mux) for the server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/review", s.handleReview)
	mux.HandleFunc("/download", s.handleDownload)
	mux.HandleFunc("/healthz", s.handleHealthz)
	return mux
}

type pageData struct {
	Health         *desi.Health
	HealthErr      string
	Input          string
	Result         *review.Result
	MainClaims     []review.Claim
	ReportMarkdown string
	Error          string
}

func (s *Server) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 20*time.Second)
}

func (s *Server) loadHealth(data *pageData) {
	ctx, cancel := s.ctx()
	defer cancel()
	h, err := s.desi.Health(ctx)
	if err != nil {
		data.HealthErr = err.Error()
		return
	}
	data.Health = h
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data := &pageData{}
	if r.URL.Query().Get("sample") == "1" {
		data.Input = SamplePaper
	}
	s.loadHealth(data)
	s.render(w, data)
}

func (s *Server) handleReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	data := &pageData{}
	s.loadHealth(data)

	// Limit total upload to 8 MiB.
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		// Fall back to plain form parsing (textarea only).
		_ = r.ParseForm()
	}

	text := r.FormValue("paper")
	if file, _, err := r.FormFile("file"); err == nil {
		defer file.Close()
		if b, rerr := io.ReadAll(io.LimitReader(file, 8<<20)); rerr == nil && len(b) > 0 {
			text = string(b)
		}
	}
	data.Input = text

	if text == "" {
		data.Error = "Please paste a paper or upload a .md/.txt file."
		s.render(w, data)
		return
	}

	ctx, cancel := s.ctx()
	defer cancel()
	res, err := s.pipeline.Review(ctx, text)
	if err != nil {
		data.Error = err.Error()
		s.render(w, data)
		return
	}
	data.Result = res
	data.ReportMarkdown = review.RenderReport(res.Artifact)
	for _, c := range res.Artifact.Claims {
		switch c.Category {
		case review.CatMain, review.CatNovelty, review.CatGeneralization, review.CatResult:
			data.MainClaims = append(data.MainClaims, c)
		}
	}
	s.render(w, data)
}

// handleDownload re-runs the review on the submitted text (deterministic)
// and returns the JSON artifact or Markdown report as a file download.
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	text := r.FormValue("paper")
	format := r.FormValue("format")
	if text == "" {
		http.Error(w, "no paper text", http.StatusBadRequest)
		return
	}

	ctx, cancel := s.ctx()
	defer cancel()
	res, err := s.pipeline.Review(ctx, text)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="review_output.json"`)
		io.WriteString(w, res.JSON)
	case "md":
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="review_report.md"`)
		io.WriteString(w, review.RenderReport(res.Artifact))
	default:
		http.Error(w, "unknown format (want json or md)", http.StatusBadRequest)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := s.ctx()
	defer cancel()
	h, err := s.desi.Health(ctx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, `{"ok":false,"error":"desi service unreachable"}`)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if id, _ := h.CoreIdentity.Float64(); id == 1.0 {
		io.WriteString(w, `{"ok":true,"library":"`+h.Library+`","version":"`+h.Version+`","core_identity":`+h.CoreIdentity.String()+`}`)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	io.WriteString(w, `{"ok":false,"error":"core_identity != 1.0"}`)
}

func (s *Server) render(w http.ResponseWriter, data *pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
