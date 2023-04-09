package vm

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"testing"
	
	"git.defalsify.org/festive/cache"
	"git.defalsify.org/festive/render"
	"git.defalsify.org/festive/resource"
	"git.defalsify.org/festive/state"
)

var dynVal = "three"

type TestResource struct {
	resource.MenuResource
	state *state.State
}

func getOne(ctx context.Context) (string, error) {
	return "one", nil
}

func getTwo(ctx context.Context) (string, error) {
	return "two", nil
}

func getDyn(ctx context.Context) (string, error) {
	return dynVal, nil
}

type TestStatefulResolver struct {
	state *state.State
}

func (r TestResource) GetTemplate(sym string) (string, error) {
	switch sym {
	case "foo":
		return "inky pinky blinky clyde", nil
	case "bar":
		return "inky pinky {{.one}} blinky {{.two}} clyde", nil
	case "baz":
		return "inky pinky {{.baz}} blinky clyde", nil
	case "three":
		return "{{.one}} inky pinky {{.three}} blinky clyde {{.two}}", nil
	case "_catch":
		return "aiee", nil
	}
	panic(fmt.Sprintf("unknown symbol %s", sym))
	return "", fmt.Errorf("unknown symbol %s", sym)
}

func (r TestResource) FuncFor(sym string) (resource.EntryFunc, error) {
	switch sym {
	case "one":
		return getOne, nil
	case "two":
		return getTwo, nil
	case "dyn":
		return getDyn, nil
	case "arg":
		return r.getInput, nil
	}
	return nil, fmt.Errorf("invalid function: '%s'", sym)
}

func(r TestResource) getInput(ctx context.Context) (string, error) {
	v, err := r.state.GetInput()
	return string(v), err
}

func(r TestResource) GetCode(sym string) ([]byte, error) {
	var b []byte
	if sym == "_catch" {
		b = NewLine(b, MOUT, []string{"0", "repent"}, nil, nil)
		b = NewLine(b, HALT, nil, nil, nil)
	}
	return b, nil
}

func TestRun(t *testing.T) {
	st := state.NewState(5)
	rs := TestResource{}
	ca := cache.NewCache()
	vm := NewVm(&st, &rs, ca, nil, nil)

	b := NewLine(nil, MOVE, []string{"foo"}, nil, nil)
	b = NewLine(b, HALT, nil, nil, nil)
	_, err := vm.Run(b, context.TODO())
	if err != nil {
		t.Errorf("run error: %v", err)	
	}

	b = []byte{0x01, 0x02}
	_, err = vm.Run(b, context.TODO())
	if err == nil {
		t.Errorf("no error on invalid opcode")	
	}
}

func TestRunLoadRender(t *testing.T) {
	st := state.NewState(5)
	rs := TestResource{}
	ca := cache.NewCache()
	pg := render.NewPage(ca, rs)
	vm := NewVm(&st, &rs, ca, nil, pg)

	st.Down("barbarbar")

	var err error
	b := NewLine(nil, LOAD, []string{"one"}, []byte{0x0a}, nil)
	b = NewLine(b, HALT, nil, nil, nil)
	b, err = vm.Run(b, context.TODO())
	if err != nil {
		t.Error(err)
	}
	m, err := ca.Get()
	if err != nil {
		t.Error(err)
	}
	r, err := pg.RenderTemplate("foo", m, 0)
	if err != nil {
		t.Error(err)
	}
	expect := "inky pinky blinky clyde"
	if r != expect {
		t.Errorf("Expected %v, got %v", []byte(expect), []byte(r))
	}

	r, err = pg.RenderTemplate("bar", m, 0)
	if err == nil {
		t.Errorf("expected error for render of bar: %v" ,err)
	}

	b = NewLine(nil, LOAD, []string{"two"}, []byte{0x0a}, nil)
	b = NewLine(b, HALT, nil, nil, nil)
	b, err = vm.Run(b, context.TODO())
	if err != nil {
		t.Error(err)
	}
	b = NewLine(nil, MAP, []string{"one"}, nil, nil)
	b = NewLine(b, HALT, nil, nil, nil)
	_, err = vm.Run(b, context.TODO())
	if err != nil {
		t.Error(err)
	}
	m, err = ca.Get()
	if err != nil {
		t.Error(err)
	}
	r, err = pg.RenderTemplate("bar", m, 0)
	if err != nil {
		t.Error(err)
	}
	expect = "inky pinky one blinky two clyde"
	if r != expect {
		t.Errorf("Expected %v, got %v", expect, r)
	}
}

func TestRunMultiple(t *testing.T) {
	st := state.NewState(5)
	rs := TestResource{}
	ca := cache.NewCache()
	vm := NewVm(&st, &rs, ca, nil, nil)

	b := NewLine(nil, MOVE, []string{"test"}, nil, nil)
	b = NewLine(b, LOAD, []string{"one"}, []byte{0x00}, nil)
	b = NewLine(b, LOAD, []string{"two"}, []byte{42}, nil)
	b = NewLine(b, HALT, nil, nil, nil)
	b, err := vm.Run(b, context.TODO())
	if err != nil {
		t.Error(err)
	}
	if len(b) > 0 {
		t.Errorf("expected empty code")
	}
}

func TestRunReload(t *testing.T) {
	st := state.NewState(5)
	rs := TestResource{}
	ca := cache.NewCache()
	pg := render.NewPage(ca, rs)
	vm := NewVm(&st, &rs, ca, nil, pg)

	b := NewLine(nil, MOVE, []string{"root"}, nil, nil)
	b = NewLine(b, LOAD, []string{"dyn"}, nil, []uint8{0})
	b = NewLine(b, MAP, []string{"dyn"}, nil, nil)
	b = NewLine(b, HALT, nil, nil, nil)
	_, err := vm.Run(b, context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	r, err := pg.Val("dyn")
	if err != nil {
		t.Fatal(err)
	}
	if r != "three" {
		t.Fatalf("expected result 'three', got %v", r)
	}
	dynVal = "baz"
	b = NewLine(nil, RELOAD, []string{"dyn"}, nil, nil)
	b = NewLine(b, HALT, nil, nil, nil)
	_, err = vm.Run(b, context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	r, err = pg.Val("dyn")
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("dun now %s", r)
	if r != "baz" {
		t.Fatalf("expected result 'baz', got %v", r)
	}
}

func TestHalt(t *testing.T) {
	st := state.NewState(5)
	rs := TestResource{}
	ca := cache.NewCache()
	vm := NewVm(&st, &rs, ca, nil, nil)

	b := NewLine(nil, MOVE, []string{"root"}, nil, nil)
	b = NewLine(b, LOAD, []string{"one"}, nil, []uint8{0})
	b = NewLine(b, HALT, nil, nil, nil)
	b = NewLine(b, MOVE, []string{"foo"}, nil, nil)
	var err error
	b, err = vm.Run(b, context.TODO())
	if err != nil {
		t.Error(err)
	}
	r, _ := st.Where()
	if r == "foo" {
		t.Fatalf("Expected where-symbol not to be 'foo'")
	}
	if !bytes.Equal(b[:2], []byte{0x00, MOVE}) {
		t.Fatalf("Expected MOVE instruction, found '%v'", b)
	}
}

func TestRunArg(t *testing.T) {
	st := state.NewState(5)
	rs := TestResource{}
	ca := cache.NewCache()
	vm := NewVm(&st, &rs, ca, nil, nil)

	input := []byte("bar")
	_ = st.SetInput(input)

	bi := NewLine(nil, INCMP, []string{"bar", "baz"}, nil, nil)
	bi = NewLine(bi, HALT, nil, nil, nil)
	b, err := vm.Run(bi, context.TODO())
	if err != nil {
		t.Error(err)	
	}
	l := len(b)
	if l != 0 {
		t.Errorf("expected empty remainder, got length %v: %v", l, b)
	}
	r, _ := st.Where()
	if r != "baz" {
		t.Errorf("expected where-state baz, got %v", r)
	}
}

func TestRunInputHandler(t *testing.T) {
	st := state.NewState(5)
	rs := TestResource{}
	ca := cache.NewCache()
	pg := render.NewPage(ca, rs)
	vm := NewVm(&st, &rs, ca, nil, pg)

	_ = st.SetInput([]byte("baz"))

	bi := NewLine([]byte{}, INCMP, []string{"bar", "aiee"}, nil, nil)
	bi = NewLine(bi, INCMP, []string{"baz", "foo"}, nil, nil)
	bi = NewLine(bi, LOAD, []string{"one"}, []byte{0x00}, nil)
	bi = NewLine(bi, LOAD, []string{"two"}, []byte{0x03}, nil)
	bi = NewLine(bi, MAP, []string{"one"}, nil, nil)
	bi = NewLine(bi, MAP, []string{"two"}, nil, nil)
	bi = NewLine(bi, HALT, nil, nil, nil)

	var err error
	_, err = vm.Run(bi, context.TODO())
	if err != nil {
		t.Fatal(err)	
	}
	r, _ := st.Where()
	if r != "foo" {
		t.Fatalf("expected where-sym 'foo', got '%v'", r)
	}
}

func TestRunArgInvalid(t *testing.T) {
	st := state.NewState(5)
	rs := TestResource{}
	ca := cache.NewCache()
	mn := render.NewMenu()
	vm := NewVm(&st, &rs, ca, mn, nil)

	_ = st.SetInput([]byte("foo"))

	var err error
	
	st.Down("root")
	b := NewLine(nil, INCMP, []string{"bar", "baz"}, nil, nil)

	b, err = vm.Run(b, context.TODO())
	if err != nil {
		t.Fatal(err)	
	}
	r, _ := st.Where()
	if r != "_catch" {
		t.Fatalf("expected where-state _catch, got %v", r)
	}
}

func TestRunMenu(t *testing.T) {
	st := state.NewState(5)
	rs := TestResource{}
	ca := cache.NewCache()
	mn := render.NewMenu()
	vm := NewVm(&st, &rs, ca, mn, nil)

	var err error

	b := NewLine(nil, MOVE, []string{"foo"}, nil, nil)
	b = NewLine(b, MOUT, []string{"0", "one"}, nil, nil)
	b = NewLine(b, MOUT, []string{"1", "two"}, nil, nil)
	b = NewLine(b, HALT, nil, nil, nil)

	b, err = vm.Run(b, context.TODO())
	if err != nil {
		t.Error(err)	
	}
	l := len(b)
	if l != 0 {
		t.Errorf("expected empty remainder, got length %v: %v", l, b)
	}
	
	r, err := mn.Render(0)
	if err != nil {
		t.Fatal(err)
	}
	expect := "0:one\n1:two"
	if r != expect {
		t.Fatalf("expected:\n\t%s\ngot:\n\t%s\n", expect, r)
	}
}

func TestRunMenuBrowse(t *testing.T) {
	log.Printf("This test is incomplete, it must check the output of a menu browser once one is implemented. For now it only checks whether it can execute the runner endpoints for the instrucitons.")
	st := state.NewState(5)
	rs := TestResource{}
	ca := cache.NewCache()
	mn := render.NewMenu()
	vm := NewVm(&st, &rs, ca, mn, nil)

	var err error

	b := NewLine(nil, MOVE, []string{"foo"}, nil, nil)
	b = NewLine(b, MOUT, []string{"0", "one"}, nil, nil)
	b = NewLine(b, MOUT, []string{"1", "two"}, nil, nil)
	b = NewLine(b, HALT, nil, nil, nil)

	b, err = vm.Run(b, context.TODO())
	if err != nil {
		t.Error(err)	
	}
	l := len(b)
	if l != 0 {
		t.Errorf("expected empty remainder, got length %v: %v", l, b)
	}
	
	r, err := mn.Render(0)
	if err != nil {
		t.Fatal(err)
	}
	expect := "0:one\n1:two"
	if r != expect {
		t.Fatalf("expected:\n\t%s\ngot:\n\t%s\n", expect, r)
	}
}

