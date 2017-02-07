package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"gopkg.in/mgo.v2"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"sourcegraph.com/sourcegraph/appdash"
	"net"

	"sourcegraph.com/sourcegraph/appdash/traceapp"
	appdashot "sourcegraph.com/sourcegraph/appdash/opentracing"
	"net/url"
	"github.com/opentracing/opentracing-go"
)

func main() {
	app := cli.App("mongo-store-api", "Testing go interaction with MongoDB")

	dbName := app.String(cli.StringOpt{
		Name:   "db-name",
		Value:  "",
		EnvVar: "DB_NAME",
	})
	dbAddress := app.String(cli.StringOpt{
		Name:   "db-address",
		Value:  "",
		EnvVar: "DB_ADDRESS",
	})
	userName := app.String(cli.StringOpt{
		Name:   "db-user",
		Value:  "",
		EnvVar: "USER",
	})
	pwd := app.String(cli.StringOpt{
		Name:   "password",
		Value:  "",
		EnvVar: "PASSWORD",
	})
	timeout := app.Int(cli.IntOpt{
		Name:   "timeout",
		Value:  5,
		EnvVar: "TIMEOUT",
	})
	port := app.Int(cli.IntOpt{
		Name:   "port",
		Value:  8080,
		EnvVar: "PORT",
	})

	var addr []string
	app.Before = func() {

		if *dbName == "" || *userName == "" || *pwd == "" {
			app.PrintHelp()
			os.Exit(1)
		}

		addr = strings.Split(*dbAddress, ",")
		if len(addr) == 1 && addr[0] == "" {
			app.PrintHelp()
			os.Exit(1)
		}
	}
	app.Before()
	//command args or env vars
	app.Action = func() {


		info := &mgo.DialInfo{
			Addrs:    addr,
			Timeout:  time.Duration(*timeout) * time.Second,
			Database: *dbName,
			Username: *userName,
			Password: *pwd,
		}
		mgoSession, err := mgo.DialWithInfo(info)
		if err != nil {
			log.Fatalf("ERROR - MongoDB session could not be created: %v\n", err)
		}

		h := httpHandler{
			client: &dbClient{
				session: mgoSession,
			},
			logger: log.New(os.Stdout, "", log.LUTC),
		}

		r := mux.NewRouter()

		r.HandleFunc("/store/{collection}/{uuid}", h.Get).Methods(http.MethodGet)
		r.HandleFunc("/store/{collection}/{uuid}", h.Write).Methods(http.MethodPost)
		r.HandleFunc("/store/__healthy", h.Ping).Methods(http.MethodGet)

		setupTracing(r)
		srv := &http.Server{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			//IdleTimeout:  120 * time.Second,
			Handler:      r,
			Addr:         fmt.Sprintf(":%d", *port),
		}


		log.Printf("Application starting on port: %d\n", *port)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("ERROR - Failed to start http server: %v\n", err)
		}

	}

	app.Run(os.Args)
}

func setupTracing(router *mux.Router) string {
	appdashPort := 8700

	store := appdash.NewMemoryStore()

	// Listen on any available TCP port locally.
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		log.Fatal(err)
	}
	collectorPort := l.Addr().(*net.TCPAddr).Port
	fmt.Printf("Collector port: %d\n", collectorPort)
	// Start an Appdash collection server that will listen for spans and
	// annotations and add them to the local collector (stored in-memory).
	cs := appdash.NewServer(l, appdash.NewLocalCollector(store))
	go cs.Start()

	// Print the URL at which the web UI will be running.
	appdashURLStr := fmt.Sprintf("http://localhost:%d", appdashPort)
	appdashURL, err := url.Parse(appdashURLStr)
	if err != nil {
		log.Fatalf("Error parsing %s: %s", appdashURLStr, err)
	}
	fmt.Printf("To see your traces, go to %s/traces\n", appdashURL)

	// Start the web UI in a separate goroutine.
	tapp, err := traceapp.New(nil, appdashURL)
	if err != nil {
		log.Fatalf("Error creating traceapp: %v", err)
	}
	tapp.Store = store
	tapp.Queryer = store
	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", appdashPort), tapp))
	}()
	tracer := appdashot.NewTracer(appdash.NewRemoteCollector(fmt.Sprintf(":%d", collectorPort)))
	opentracing.InitGlobalTracer(tracer)

	return fmt.Sprintf(":%d", collectorPort)
}
