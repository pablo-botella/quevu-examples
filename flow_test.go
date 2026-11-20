package main

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	qhub "github.com/pablo-botella/quevu-hub"
	quevu "github.com/pablo-botella/quevu-kernel"
	qpanel "github.com/pablo-botella/quevu-panel"
	qsqlite "github.com/pablo-botella/quevu-sql/sqlite"
	qtxt "github.com/pablo-botella/quevu-txt"
)

// The whole way, with the real pieces: several producers write into one
// hub through inlets, and a text file, a SQLite table and a screen each
// receive from a channel of their own.
func TestFromProducersToEveryWriter(t *testing.T) {
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "flow.log")
	dbPath := filepath.Join(dir, "flow.db")

	file, err := qtxt.Open(txtPath, qtxt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	db, err := qsqlite.Open(dbPath, qsqlite.Options{Table: "flow", Buffer: 50, FlushEvery: 50 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	scr := qpanel.New("flow", qpanel.WithMemory(100))

	h, err := qhub.New("flow",
		qhub.WithWriterOptions(file, qhub.Options{Name: "file", Buffer: 64}),
		qhub.WithWriterOptions(db, qhub.Options{Name: "db", Buffer: 64}),
		qhub.WithWriterOptions(scr, qhub.Options{Name: "screen", OnFull: qhub.Skip}),
		qhub.WithInput(16))
	if err != nil {
		t.Fatal(err)
	}

	const producers, each = 4, 500
	var wg sync.WaitGroup
	for p := 0; p < producers; p++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			in := quevu.NewInlet(h)
			for i := 0; i < each; i++ {
				fmt.Fprintf(in, "p%d line %d\n", p, i)
			}
		}()
	}
	wg.Wait()
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	total := producers * each
	for _, st := range h.Stats() {
		if st.Name != "screen" && (st.Written != uint64(total) || st.Skipped != 0 || st.Failed != 0) {
			t.Fatalf("%+v", st)
		}
	}

	// Both stores hold everything, in the same order and with the same
	// positions.
	check := func(name string, p quevu.Provider) []quevu.Chunk {
		t.Helper()
		var all []quevu.Chunk
		seq := uint64(0)
		for {
			got, err := p.After(seq, 300)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if len(got) == 0 {
				break
			}
			all = append(all, got...)
			seq = got[len(got)-1].Seq
		}
		if len(all) != total {
			t.Fatalf("%s holds %d of %d", name, len(all), total)
		}
		for i, c := range all {
			if c.Seq != uint64(i+1) {
				t.Fatalf("%s: position %d holds %d", name, i, c.Seq)
			}
		}
		return all
	}
	fromFile := check("file", qtxt.NewReader(txtPath))
	reader, err := qsqlite.NewReader(dbPath, "flow")
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	fromDB := check("db", reader)
	for i := range fromFile {
		if string(fromFile[i].Data) != string(fromDB[i].Data) {
			t.Fatalf("position %d differs: %q and %q", i+1, fromFile[i].Data, fromDB[i].Data)
		}
	}

	// A second run continues the positions where the stores left them.
	file2, err := qtxt.Open(txtPath, qtxt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	h2, err := qhub.New("flow", qhub.WithWriter(file2))
	if err != nil {
		t.Fatal(err)
	}
	quevu.NewInlet(h2).Append([]byte("after the restart"))
	if err := h2.Close(); err != nil {
		t.Fatal(err)
	}
	last, err := qtxt.NewReader(txtPath).Tail(1)
	if err != nil || len(last) != 1 || last[0].Seq != uint64(total+1) || string(last[0].Data) != "after the restart" {
		t.Fatalf("after the restart: %+v %v", last, err)
	}
}
