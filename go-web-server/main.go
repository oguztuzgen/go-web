package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var env map[string]string = GetEnv()

type Person struct {
	Id       uint64 `json:"id"`
	Name     string `json:"name"`
	Age      int    `json:"age"`
	Password string `json:"-"`
}

type Repository[T any] interface {
	save(T) error
	find(int) (T, error)
	findAll() []T
}

type PersonRepository struct {
	increment uint64
	ds        []Person
}

func (pr *PersonRepository) save(p Person) error {
	p.Id = pr.increment
	pr.ds = append(pr.ds, p)
	pr.increment += 1
	return nil
}

func (pr PersonRepository) findAll() []Person {
	return pr.ds
}

func (pr PersonRepository) find(id uint64) (Person, error) {
	for i := 0; i < len(pr.ds); i++ {
		if pr.ds[i].Id == id {
			return pr.ds[i], nil
		}
	}
	return Person{}, errors.New("Person not found")
}

type Authenticator interface {
	Authorize(h http.Header) bool
}

type BearerAuthenticator struct {
	pr PersonRepository
}

func (ba BearerAuthenticator) Authorize(h http.Header) bool {
	auth := h.Values("authorization")
	if len(auth) == 0 {
		return false
	}
	authSplit := strings.Split(auth[0], " ")
	if authSplit[0] != "Bearer" {
		return false
	}
	tokenString := authSplit[1]
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}

		return env["JWT_SECRET"], nil
	})

	if err != nil {
		return false
	}

	if claims, ok := token.Claims.(jwt.MapClaims); !ok || !token.Valid {
		return false
	} else {
		id, ok := claims["id"].(uint64)
		if !ok {
			return false
		}
		name, ok := claims["name"].(string)
		if !ok {
			return false
		}

		p, err := ba.pr.find(id)
		if err != nil {
			return false
		}
		if p.Name != name {
			return false
		}

		return true
	}

	// return jwt.NewW !(token != "key")
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

type PersonService struct {
	personRepository *PersonRepository
}

func (ds PersonService) Hello(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		ctx := r.Context()
		defer fmt.Println("hello get handler ended")

		p := strings.Split(r.URL.Path, "/")

		if p[1] != "" {
			id, err := strconv.ParseUint(p[1], 10, 0)
			if err != nil {
				panic(err)
			}

			select {
			case <-time.After(1 * time.Second):
				person, err := ds.personRepository.find(id)
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
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(p.Password), 11)
				p.Password = string(hashedPassword)
				ds.personRepository.save(p)
				token := jwt.NewWithClaims(jwt.GetSigningMethod(jwt.SigningMethodES256.Name), jwt.MapClaims{
					"Name": p.Name,
				})

				tokenString, err := token.SignedString(env["JWT_SECRET"])
				if err != nil {
					panic(err)
				}

				resBody, _ := json.Marshal(struct {
					Token string
				}{
					Token: tokenString,
				})

				w.Write(resBody)
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
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("123"), 11)
	ds := PersonService{&PersonRepository{increment: 1, ds: []Person{}}}
	ds.personRepository.save(Person{0, "Oguz", 15, string(hashedPassword)})

	middleware := []Middleware{
		AuthMiddleware,
	}

	mux.HandleFunc("/", CompileMiddleware(ds.Hello, middleware))

	err := http.ListenAndServe(":80", mux)
	if err != nil {
		panic(err)
	}
}
