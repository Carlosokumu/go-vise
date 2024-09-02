package engine

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"git.defalsify.org/vise.git/resource"
	"git.defalsify.org/vise.git/state"
	"git.defalsify.org/vise.git/vm"
)

func getNull() io.WriteCloser {
	nul, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0700)
	if err != nil {
		panic(err)
	}
	return nul
}

func codeGet(ctx context.Context, s string) ([]byte, error) {
	var b []byte
	var err error
	switch s {
		case "root":
			b = vm.NewLine(nil, vm.HALT, nil, nil, nil)
			b = vm.NewLine(b, vm.LOAD, []string{"foo"}, []byte{0x0}, nil)
		default:
			err = fmt.Errorf("unknown code symbol '%s'", s)
	}
	return b, err
}

func flagSet(ctx context.Context, nodeSym string, input []byte) (resource.Result, error) {
	return resource.Result{
		Content: "xyzzy",
		FlagSet: []uint32{state.FLAG_USERSTART},
	}, nil
}

func TestDbEngineMinimal(t *testing.T) {
	ctx := context.Background()
	cfg := Config{}
	rs := resource.NewMenuResource()
	en := NewDbEngine(cfg, rs)
	cont, err := en.Init(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cont {
		t.Fatalf("expected not continue")
	}
}

func TestDbEngineRoot(t *testing.T) {
	nul := getNull()
	defer nul.Close()
	ctx := context.Background()
	cfg := Config{}
	rs := resource.NewMenuResource()
	rs.WithCodeGetter(codeGet)
	en := NewDbEngine(cfg, rs)
	cont, err := en.Init(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !cont {
		t.Fatalf("expected continue")
	}

	cont, err = en.Exec(ctx, []byte{0x30})
	if err == nil {
		t.Fatalf("expected loadfail")
	}

	_, err = en.WriteResult(ctx, nul) 
	if err != nil {
		t.Fatal(err)
	}

	cont, err = en.Exec(ctx, []byte{0x30})
	if err == nil {
		t.Fatalf("expected nocode")
	}
}

func TestDbEnginePersist(t *testing.T) {
	nul := getNull()
	defer nul.Close()
	ctx := context.Background()
	cfg := Config{
		FlagCount: 1,
		SessionId: "bar",
	}
	rs := resource.NewMenuResource()
	rs.WithCodeGetter(codeGet)
	rs.AddLocalFunc("foo", flagSet)
	en := NewDbEngine(cfg, rs)
	cont, err := en.Init(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !cont {
		t.Fatalf("expected continue")
	}

	cont, err = en.Exec(ctx, []byte{0x30})
	if err != nil {
		t.Fatal(err)
	}

	_, err = en.WriteResult(ctx, nul) 
	if err != nil {
		t.Fatal(err)
	}

}
