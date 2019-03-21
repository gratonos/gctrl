package controller

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
)

type funcMeta struct {
	fn   reflect.Value
	in   []reflect.Type
	out  []reflect.Type
	name string
	desc string
}

type Controller struct {
	funcs  map[string]*funcMeta
	rwlock sync.RWMutex
}

const unixPerm = 0770

var defaultController = New()

func New() *Controller {
	return &Controller{
		funcs: make(map[string]*funcMeta),
	}
}

func (ctrl *Controller) Register(fn interface{}, name, desc string) error {
	errfn := genErrFunc("Register")

	if fn == nil {
		return errfn("fn must not be nil")
	}
	if name == "" {
		return errfn("name must not be empty")
	}
	meta, err := genFuncMeta(fn, name, desc)
	if err != nil {
		return errfn(err.Error())
	}

	ctrl.rwlock.Lock()
	defer ctrl.rwlock.Unlock()

	if _, ok := ctrl.funcs[name]; ok {
		return errfn(fmt.Sprintf("name '%s' has been registered", name))
	}
	ctrl.funcs[name] = meta
	return nil
}

func (ctrl *Controller) MustRegister(fn interface{}, name, desc string) {
	if err := ctrl.Register(fn, name, desc); err != nil {
		panic(err)
	}
}

func (ctrl *Controller) Serve(rw io.ReadWriter) error {
	prompt(rw)
	scanner := bufio.NewScanner(rw)
	for scanner.Scan() {
		cmd := strings.TrimSpace(scanner.Text())
		if cmd == "" {
			continue
		}

		ctrl.rwlock.RLock() // todo

		if builtin(cmd) {
			ctrl.handleBuiltin(cmd, rw)
		} else {
			ctrl.handleFuncCall(cmd, rw)
		}

		ctrl.rwlock.RUnlock()
	}
	return scanner.Err()
}

func genFuncMeta(fn interface{}, name, desc string) (*funcMeta, error) {
	typ := reflect.TypeOf(fn)
	if typ.Kind() != reflect.Func {
		return nil, errors.New("fn is not a function")
	}

	var in []reflect.Type
	for i := 0; i < typ.NumIn(); i++ {
		inType := typ.In(i)
		if !checkType(inType.Kind()) {
			return nil, errors.New("supported parameter types are bool, int64, " +
				"uint64, float64 and string")
		}
		in = append(in, inType)
	}

	var out []reflect.Type
	for i := 0; i < typ.NumOut(); i++ {
		outType := typ.Out(i)
		out = append(out, outType)
	}

	meta := &funcMeta{
		fn:   reflect.ValueOf(fn),
		in:   in,
		out:  out,
		name: name,
		desc: desc,
	}
	return meta, nil
}

func checkType(kind reflect.Kind) bool {
	switch kind {
	case reflect.Bool, reflect.Int64, reflect.Uint64, reflect.Float64, reflect.String:
		return true
	default:
		return false
	}
}