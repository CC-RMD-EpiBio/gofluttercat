package frontend

import (
	"context"
	"fmt"
	"log"
	"net/http"

	conf "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/config"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/web/handlers"
	badger "github.com/dgraph-io/badger/v4"
)

// AssessmentMetaView holds display metadata for an instrument.
type AssessmentMetaView struct {
	Name        string
	Description string
	Scales      map[string]string
}

// FrontendHandler serves the HTML frontend.
type FrontendHandler struct {
	db          *badger.DB
	instruments map[string]*handlers.InstrumentRegistry
	ctx         context.Context
	metas       map[string]AssessmentMetaView
	catCfg      conf.CatConfig
}

func NewFrontendHandler(db *badger.DB, instruments map[string]*handlers.InstrumentRegistry,
	ctx context.Context, metas map[string]AssessmentMetaView, catCfg conf.CatConfig) *FrontendHandler {
	return &FrontendHandler{
		db:          db,
		instruments: instruments,
		ctx:         ctx,
		metas:       metas,
		catCfg:      catCfg,
	}
}

func (fh *FrontendHandler) HandleHome(w http.ResponseWriter, r *http.Request) {
	page := HomePage(fh.instruments, fh.metas)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.Render(w); err != nil {
		log.Printf("render error: %v", err)
	}
}

func (fh *FrontendHandler) HandleStartAssessment(w http.ResponseWriter, r *http.Request) {
	instrumentID := r.FormValue("instrument")
	if instrumentID == "" {
		instrumentID = "rwa"
	}

	ctx := fh.ctx
	sess, err := handlers.CreateSession(instrumentID, fh.instruments, fh.db, &ctx, fh.catCfg)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		ErrorAlert(err.Error()).Render(w)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/ui/assess?sid=%s", sess.SessionId))
	w.WriteHeader(http.StatusOK)
}

func (fh *FrontendHandler) HandleAssessmentPage(w http.ResponseWriter, r *http.Request) {
	sid := r.URL.Query().Get("sid")
	if sid == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	ctx := fh.ctx
	item, err := handlers.GetNextItem(sid, fh.db, &ctx, fh.instruments)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		page := Page("Error", Navbar(), ErrorAlert(err.Error()))
		page.Render(w)
		return
	}

	// Count existing responses for progress display
	numResponses := 0
	rehydrated, err := irtcat.SessionStateFromId(sid, fh.db, &ctx)
	if err == nil {
		numResponses = len(rehydrated.Responses)
	}

	if item == nil {
		http.Redirect(w, r, fmt.Sprintf("/ui/results?sid=%s", sid), http.StatusSeeOther)
		return
	}

	page := AssessmentPage(sid, item, numResponses)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.Render(w); err != nil {
		log.Printf("render error: %v", err)
	}
}

func (fh *FrontendHandler) HandleSubmitResponse(w http.ResponseWriter, r *http.Request) {
	sid := r.FormValue("sid")
	itemName := r.FormValue("item_name")
	valueStr := r.FormValue("value")

	if sid == "" || itemName == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		ErrorAlert("Missing required fields").Render(w)
		return
	}

	var value int
	fmt.Sscanf(valueStr, "%d", &value)

	req := irtcat.SkinnyResponse{
		ItemName: itemName,
		Value:    value,
	}

	ctx := fh.ctx
	err := handlers.ProcessResponse(sid, req, fh.db, &ctx, fh.instruments)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		ErrorAlert(err.Error()).Render(w)
		return
	}

	// Get next item
	item, err := handlers.GetNextItem(sid, fh.db, &ctx, fh.instruments)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		ErrorAlert(err.Error()).Render(w)
		return
	}

	if item == nil {
		w.Header().Set("HX-Redirect", fmt.Sprintf("/ui/results?sid=%s", sid))
		w.WriteHeader(http.StatusOK)
		return
	}

	// Count responses for progress
	numResponses := 0
	rehydrated, err := irtcat.SessionStateFromId(sid, fh.db, &ctx)
	if err == nil {
		numResponses = len(rehydrated.Responses)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fragment := QuestionCard(sid, item, numResponses)
	if err := fragment.Render(w); err != nil {
		log.Printf("render error: %v", err)
	}
}

func (fh *FrontendHandler) HandleResultsPage(w http.ResponseWriter, r *http.Request) {
	sid := r.URL.Query().Get("sid")
	if sid == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	ctx := fh.ctx
	summary, err := handlers.GetSummary(sid, fh.db, &ctx, fh.instruments)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		page := Page("Error", Navbar(), ErrorAlert(err.Error()))
		page.Render(w)
		return
	}

	// Look up scale display names from the session's instrument
	scaleNames := make(map[string]string)
	rehydrated, err := irtcat.SessionStateFromId(sid, fh.db, &ctx)
	if err == nil {
		if meta, ok := fh.metas[rehydrated.InstrumentID]; ok {
			scaleNames = meta.Scales
		}
	}

	page := ResultsPage(sid, summary, scaleNames)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.Render(w); err != nil {
		log.Printf("render error: %v", err)
	}
}
