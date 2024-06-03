package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.opentelemetry.io/otel"
)

type Response struct {
	TempC float64 `json:"temp_c"`
	TempF float64 `json:"temp_f"`
	TempK float64 `json:"temp_k"`
	City  string  `json:"city"`
}

var postData struct {
	Cep string `json:"cep"`
}

func main() {
	startZipkin()

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	r.Post("/", SearchCepHandler)

	http.ListenAndServe(":8080", r)
}

func SearchCepHandler(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error reading request body"))
		return
	}

	err = json.Unmarshal(body, &postData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error unmarshalling request body"))
		return
	}

	if postData.Cep == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("cep parameter is required"))
		return
	}

	validate := regexp.MustCompile(`^[0-9]{8}$`)
	if !validate.MatchString(postData.Cep) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("invalid zipcode"))
		return
	}

	temperature, err := CallServiceB(postData.Cep, r.Context())

	if err != nil {
		errorStr := err.Error()
		if errorStr == "can not find zipcode" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(errorStr))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error while searching for cep: " + errorStr))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if temperature != nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(temperature)
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("can not find temperature"))
	}
}

func CallServiceB(cep string, ctx context.Context) (*Response, error) {
	_, span := otel.Tracer("service-a").Start(ctx, "call-to-service-b")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://goapp-service-b:8081/?cep="+cep, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("can not find zipcode")
	}

	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var data Response
	err = json.Unmarshal(res, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}
