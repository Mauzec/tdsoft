package ui

import (
	"context"
	"log"
	"reflect"
	"sync"

	"fyne.io/fyne/v2"
)

type ScreenFactory func(r *Router) fyne.CanvasObject
type ScreenID string

type Router struct {
	win       fyne.Window
	factories map[ScreenID]ScreenFactory

	// dependency registry
	services map[reflect.Type]any
	// per-screen last passed params (ShowWith)
	params map[ScreenID]any
	mu     sync.RWMutex

	// screen-scoped context, cancelled whenever Show switches screen
	screenCtx context.Context
	cancel    context.CancelFunc
}

func NewRouter(win fyne.Window) *Router {
	return &Router{
		win:       win,
		factories: make(map[ScreenID]ScreenFactory),
		services:  make(map[reflect.Type]any),
		params:    make(map[ScreenID]any),
		screenCtx: context.Background(),
		cancel:    func() {},
	}
}

func (r *Router) Register(id ScreenID, f ScreenFactory) {
	r.factories[id] = f
}

// Show shows the screen with given ID.
func (r *Router) Show(id ScreenID) {
	f, ok := r.factories[id]
	if !ok {
		log.Printf("trying to show ScreenID %s, but not registered", id)
		return
	}

	r.mu.Lock()
	if r.cancel != nil {
		r.cancel()
	}
	r.screenCtx, r.cancel = context.WithCancel(context.Background())
	r.mu.Unlock()

	if fyne.CurrentApp() != nil {
		fyne.Do(func() { r.win.SetContent(f(r)) })
		return
	}
	r.win.SetContent(f(r))
}

// ShowWith shows the screen with given ID, passing param to it.
func (r *Router) ShowWith(id ScreenID, param any) {
	r.mu.Lock()
	r.params[id] = param
	r.mu.Unlock()
	r.Show(id)
}

// Param retrieves the last param passed to ShowWith for a screen
func (r *Router) Param(id ScreenID) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.params[id]
	return v, ok
}

// ParamAs assigns the last param for the ScreenID into targetPtr
// if the value's type is assginable to the target type.
//
// targetPtr must be a non-nil pointer to a variable(pointer)(e.g. &V or **V).
//
// Returns false if there is no param, param is nil, targetPtr is invalid,
// or the types are not assignable.
func (r *Router) ParamAs(id ScreenID, targetPtr any) (ret bool) {
	v, ok := r.Param(id)
	if !ok {
		return
	}
	if v == nil {
		return
	}

	rv := reflect.ValueOf(targetPtr)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return
	}

	vv := reflect.ValueOf(v)
	if !vv.Type().AssignableTo(rv.Elem().Type()) {
		return
	}
	rv.Elem().Set(vv)
	return true
}

// PutService regiesters a dependency (service) by its statis type
func (r *Router) PutService(v any) {
	if v == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[reflect.TypeOf(v)] = v
}

// GetService retrieves a dependecy (service) of type T.
func (r *Router) GetService(T reflect.Type) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.services[T]
	return v, ok
}

// GetServiceAs loads a dependency into targetPtr via assignability.
//
// targetPtr must be a non-nil pointer of the desired type (e.g. *MyType, *MyInterface)
//
// Avoid **T and *interface{}
func (r *Router) GetServiceAs(targetPtr any) (ret bool) {
	rv := reflect.ValueOf(targetPtr)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return
	}

	T := rv.Elem().Type()
	r.mu.RLock()
	for st, svc := range r.services {
		if st.AssignableTo(T) {
			rv.Elem().Set(reflect.ValueOf(svc))
			r.mu.RUnlock()
			return true
		}

		// interface match
		if T.Kind() == reflect.Interface && st.Implements(T) {
			rv.Elem().Set(reflect.ValueOf(svc))
			r.mu.RUnlock()
			return true
		}
	}
	r.mu.RUnlock()
	return
}

func (r *Router) MustGetService(T reflect.Type) any {
	r.mu.RLock()
	v, ok := r.services[T]
	r.mu.RUnlock()
	if !ok {
		panic("router: missing service of type " + T.String())
	}
	return v
}

func (r *Router) Window() fyne.Window { return r.win }

// func (r *Router) Events() <-chan
// func (r *Router)Emit(...)

func (r *Router) ScreenContext() context.Context {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.screenCtx
}
