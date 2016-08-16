package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"

	"github.com/kyokomi/goslash/goslash"
	"github.com/kyokomi/goslash/plugins"
	"github.com/kyokomi/goslash/plugins/akari"
	"github.com/kyokomi/goslash/plugins/echo"
	"github.com/kyokomi/goslash/plugins/lgtm"
	"github.com/kyokomi/goslash/plugins/suddendeath"
	"github.com/kyokomi/goslash/plugins/time"

	"github.com/unrolled/render"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

func init() {
	renderer := render.New(render.Options{})
	token := os.Getenv("SLACK_TOKEN")

	http.HandleFunc("/v1/cmd", func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)

		req, err := goslash.ParseFormSlashCommandRequest(r)
		if err != nil {
			renderer.JSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		if req.Token != token {
			log.Errorf(ctx, "received invalid token:%v from %v", req.Token, r.RemoteAddr)
			renderer.JSON(w, http.StatusForbidden, "invalid token")
			return
		}

		slashPlugins := map[string]plugins.Plugin{
			"echo":  echo.New(),
			"time":  time.New(),
			"突然":    suddendeath.New(),
			"LGTM":  lgtm.New(urlfetch.Client(ctx)),
			"akari": akari.New(),
		}

		slashCmd := plugins.New(urlfetch.Client(ctx), slashPlugins)

		if appengine.IsDevAppServer() {
			// development
			cmd, _ := req.CmdArgs()
			p, ok := slashPlugins[cmd]
			if !ok {
				renderer.JSON(w, http.StatusNotFound, "cmd not found")
				return
			}

			msg := p.Do(req)
			var jsonData bytes.Buffer
			if err := json.NewEncoder(&jsonData).Encode(&msg); err != nil {
				renderer.JSON(w, http.StatusInternalServerError, err.Error())
				return
			}
			renderer.JSON(w, http.StatusOK, jsonData.String())
		} else {
			// production
			renderer.Text(w, http.StatusOK, slashCmd.Execute(req))
		}
	})
}
