package main

import (
	"github.com/spf13/viper"
	"log"
	"./config"
	"flag"
	"net/url"
	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	"net/http"
	"os"
	"time"
	"net/http/httputil"
	"strconv"
	"encoding/json"
	"sync"
)

type GuiConfig struct {
   OraUrl string `json:"oraAddress"`
   OraPort int `json:"oraPort"`
   Dms    bool   `json:"documentManagementService"`
}

type routerSwapper struct {
	mu sync.Mutex
	root *mux.Router
}

func main() {
	var configFile string

	flag.StringVar(&configFile, "config", "./config.yml", "Specifies the config file location")
	flag.Parse()

	viper.SetConfigFile(configFile)

	var conf config.Config

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	err := viper.Unmarshal(&conf)
	if err != nil {
		log.Fatalf("unable to decode into struct, %v", err)
	}
	log.Printf("Ora connection URL is %s", conf.Ora.ConnectionUri)
	log.Printf("Server listens on %s:%d", conf.Server.BindAddress, conf.Server.Port)


	oraRemote, err := url.Parse(conf.Ora.ConnectionUri)
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(oraRemote)

	r := mux.NewRouter()

	// Note: In a larger application, we'd likely extract our route-building logic into our handlers
	// package, given the coupling between them.

	// Static routes for serving serving files
	r.Path("/favicon.ico").HandlerFunc(staticFileHandler("./favicon.ico"))
	r.Path("/manifest.json").HandlerFunc(staticFileHandler("./manifest.json"))
	r.Path("/config").HandlerFunc(jsonTypeHandler(GuiConfig{OraUrl: conf.Server.ApiExposeUrl, OraPort: conf.Server.ApiExposePort,
																Dms: conf.Server.DocumentManagementService}))

	// Serve static assets directly.
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.PathPrefix("/rest/").HandlerFunc(proxyHandler(proxy))

	// It's important that this is before your catch-all route ("/")
	// api := r.PathPrefix("/api/v1/").Subrouter()
	// api.HandleFunc("/users", GetUsersHandler).Methods("GET")

	// Optional: Use a custom 404 handler for our API paths.
	// api.NotFoundHandler = JSONNotFound

	// Catch-all: Serve our JavaScript application's entry-point (index.html).
	r.PathPrefix("/").HandlerFunc(staticFileHandler("./index.html"))

	srv := &http.Server{
		Handler: handlers.LoggingHandler(os.Stdout, r),
		Addr:    conf.Server.BindAddress + ":" +strconv.Itoa(conf.Server.Port),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: time.Duration(conf.Server.WriteTimeout) * time.Second,
		ReadTimeout:  time.Duration(conf.Server.ReadTimeout) * time.Second,
	}

	log.Fatal(srv.ListenAndServe())

	log.Println("Itt vagyok")
}


func staticFileHandler(entrypoint string) func(w http.ResponseWriter, r *http.Request) {
	fn := func(w http.ResponseWriter, r *http.Request) {
		log.Println(entrypoint)
		http.ServeFile(w, r, entrypoint)
	}

	return http.HandlerFunc(fn)
}

func proxyHandler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("ORA proxy: " + r.RequestURI)
		r.Header.Del("Cookie")
		p.ServeHTTP(w, r)
	}
}

func jsonTypeHandler(data interface{}) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(data)
	}
}