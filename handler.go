package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/satori/go.uuid"
	"net/http"
)

type logger interface {
	Printf(format string, v ...interface{})
}

type httpHandler struct {
	client Accessor
	logger logger
}

type message struct {
	Message string
}

func (h *httpHandler) Get(writer http.ResponseWriter, req *http.Request) {
	var sp opentracing.Span
	opName := req.URL.Path
	// Attempt to join a trace by getting trace context from the headers.
	wireContext, err := opentracing.GlobalTracer().Extract(
		opentracing.TextMap,
		opentracing.HTTPHeadersCarrier(req.Header))
	if err != nil {
		sp = opentracing.StartSpan(opName) // Start a span using the global, in this case noop, tracer
		fmt.Printf("Started new root span: %v\n", sp)
	} else {
		sp = opentracing.StartSpan(opName, opentracing.ChildOf(wireContext))
		fmt.Printf("Started new child span: %v\n", sp)
	}
	defer sp.Finish()

	UUID := mux.Vars(req)["uuid"]
	if uuid.FromStringOrNil(UUID) == uuid.Nil {
		writer.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(writer).Encode(&message{"UUID should be a valid UUID according to RFC 4122"})
		return
	}
	sp.LogFields(log.String("uuid", UUID))

	content, found, err := h.client.Get("holiday", UUID)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(writer).Encode(&message{err.Error()})
		return
	}

	if !found {
		writer.WriteHeader(http.StatusNotFound)
		json.NewEncoder(writer).Encode(&message{"Resource not found"})
		return
	}

	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(&content)
}

func (h *httpHandler) Write(writer http.ResponseWriter, req *http.Request) {
	UUID := mux.Vars(req)["uuid"]
	if uuid.FromStringOrNil(UUID) == uuid.Nil {
		writer.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(writer).Encode(&message{"UUID should be a valid UUID according to RFC 4122"})
		return
	}

	var bsonResource map[string]interface{}
	json.NewDecoder(req.Body).Decode(&bsonResource)
	uuidInPayload, found := bsonResource["uuid"]
	if !found {
		writer.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(writer).Encode(&message{"UUID is missing from payload"})
		return
	}

	if uuidInPayloadString, ok := uuidInPayload.(string); !ok || uuidInPayloadString != UUID {
		writer.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(writer).Encode(&message{"UUID in payload is not the same as the UUID in the request path"})
		return
	}

	err, inserted := h.client.Write(UUID, bsonResource)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(writer).Encode(&message{err.Error()})
		return
	}

	if inserted {
		writer.WriteHeader(http.StatusCreated)
		json.NewEncoder(writer).Encode(&message{fmt.Sprintf("Data inserted for %s", UUID)})
		return
	}

	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(&message{fmt.Sprintf("Data updated for %s", UUID)})
}

func (h *httpHandler) Ping(writer http.ResponseWriter, req *http.Request) {
	var sp opentracing.Span
	opName := req.URL.Path
	// Attempt to join a trace by getting trace context from the headers.
	wireContext, err := opentracing.GlobalTracer().Extract(
		opentracing.TextMap,
		opentracing.HTTPHeadersCarrier(req.Header))
	if err != nil {
		sp = opentracing.StartSpan(opName) // Start a span using the global, in this case noop, tracer
		fmt.Printf("Started new root span: %v\n", sp)
	} else {
		sp = opentracing.StartSpan(opName, opentracing.ChildOf(wireContext))
		fmt.Printf("Started new child span: %v\n", sp)
	}
	defer sp.Finish()

	return
	err = h.client.Ping()
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(writer).Encode(&message{err.Error()})
		return
	}
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(&message{"Connection to Mongo established"})
}
