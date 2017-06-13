package runner

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

type apiHookHandler func(hook string, jobID string) error

type api struct {
	net.Listener
	authTokens  map[string]string
	hookHandler apiHookHandler
}

func (a *api) RegisterJob(jobID string) (string, error) {
	data := make([]byte, 10)
	_, err := rand.Read(data)
	if err != nil {
		return "", err
	}
	token := fmt.Sprintf("%x", sha256.Sum256(data))
	a.authTokens[token] = jobID
	return token, nil
}

func (a *api) handleNotifyHook(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	jobID := req.Context().Value("JobID").(string)
	hook := vars["hook"]
	debugf("job %s reported hook %s", jobID, hook)

	if a.hookHandler != nil {
		if err := a.hookHandler(hook, jobID); err != nil {
			debugf("hook handler for %s failed: %v", hook, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode("OK")
}

// authenticate provides Authentication middleware for handlers
func (a *api) authenticate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string

		// Get token from the Authorization header
		// format: Authorization: Bearer
		tokens, ok := r.Header["Authorization"]
		if ok && len(tokens) >= 1 {
			token = tokens[0]
			token = strings.TrimPrefix(token, "Bearer ")
		}

		// Check if we have the token in our auth table
		jobId, ok := a.authTokens[token]
		if !ok {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "JobID", jobId)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", err
}

func apiListen(listenOn string, handler apiHookHandler) (*api, error) {
	if listenOn == "" {
		addr, err := getLocalIP()
		if err != nil {
			return nil, err
		}
		listenOn = addr + ":0"
	}

	l, err := net.Listen("tcp", listenOn)
	if err != nil {
		return nil, err
	}

	server := &api{
		Listener:    l,
		authTokens:  map[string]string{},
		hookHandler: handler,
	}

	router := mux.NewRouter()

	router.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode("OK")
	})

	router.HandleFunc("/notify/hook/{hook}",
		server.authenticate(server.handleNotifyHook)).
		Methods("POST")

	go func() {
		debugf("API server listening on %s", l.Addr().String())
		fmt.Println(http.Serve(l, router))
		os.Exit(1)
	}()

	return server, nil
}
