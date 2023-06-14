package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Person struct {
	Name string `json:"name"`
	Age int	`json:"age"`
}

type Repository[T any] interface {
	save(T) error
	find(int) (T, error)
	findAll() []T
}

type PersonRepository struct {
	ds []Person
}

func (pr *PersonRepository) save(p Person) error {
	pr.ds = append(pr.ds, p)
	return nil;
}

func (pr PersonRepository) findAll() []Person {
	return pr.ds
}

func (pr PersonRepository) find(index int) (Person, error) {
	if (len(pr.ds) <= index) {
		return Person{}, errors.New("Person not found")
	}
	return pr.ds[index], nil
}

type Authenticator interface {
	Authorize(h http.Header) bool
}

type BearerAuthenticator struct{}

func (BearerAuthenticator) Authorize(h http.Header) bool {
	auth := h.Values("authorization")
	// replace key with jwt
	return !(len(auth) == 0 || strings.Split(auth[0], " ")[1] != "key")
}

func AuthMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ba := BearerAuthenticator{}
		if !ba.Authorize(r.Header) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized!"))
			return
		}
		h.ServeHTTP(w, r)
	})
}

type Service interface {
	Hello(w http.ResponseWriter, r *http.Request)
}

type PersonService struct{
	personRepository *PersonRepository
}

func (ds PersonService) Hello(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		ctx := r.Context()
		defer fmt.Println("hello get handler ended")

		p := strings.Split(r.URL.Path, "/")

		if (p[1] != "") {
			index, err := strconv.ParseInt(p[1], 10, 0)
			if err != nil {
				panic(err)
			}
			
			select {
			case <-time.After(1 * time.Second):
				person, err := ds.personRepository.find(int(index))
				if err != nil {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				w.WriteHeader(http.StatusOK)
				if result, err := json.Marshal(person); err != nil {
					panic(err)
				} else {
					fmt.Fprint(w, string(result))
				}
			case <-ctx.Done():
				err := ctx.Err()
				fmt.Println("server: ", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			
		} else {
			select {
			case <-time.After(1 * time.Second):
				w.WriteHeader(http.StatusOK)
				if result, err := json.Marshal(ds.personRepository.findAll()); err != nil {
					panic(err)
				} else {
					fmt.Fprint(w, string(result))
				}
			case <-ctx.Done():
				err := ctx.Err()
				fmt.Println("server: ", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}

	case "POST":
		ctx := r.Context()
		defer fmt.Println("hello post handler ended")

		select {
		case <-time.After(1 * time.Second):
			w.WriteHeader(http.StatusCreated)
			var p Person
			body, err := io.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			if err := json.Unmarshal(body, &p); err != nil {
				panic(err)
			} else {
				ds.personRepository.save(p)
			}

		case <-ctx.Done():
			err := ctx.Err()
			fmt.Println("server: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

}

type Middleware func(http.HandlerFunc) http.HandlerFunc

func CompileMiddleware(h http.HandlerFunc, m []Middleware) http.HandlerFunc {
	if len(m) < 1 {
		return h
	}

	wrapped := h

	for i := len(m) - 1; i >= 0; i-- {
		wrapped = m[i](wrapped)
	}

	return wrapped
}

func main() {
	mux := http.NewServeMux()
	ds := PersonService{&PersonRepository{ds: []Person{ {"Oguz", 15} }}}
	srv := &http.Server{
		Handler: mux,
	}

	middleware := []Middleware{
		AuthMiddleware,
	}

	mux.HandleFunc("/", CompileMiddleware(ds.Hello, middleware))

	ln, err := net.Listen("tcp", ":80")
	if err != nil {
		panic(err)
	}

	srv.Serve(ln)
}
