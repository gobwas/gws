package main
import (
	"net/http"
	"github.com/gorilla/websocket"
	"log"
	"io/ioutil"
	"flag"
)

var listen = flag.String("l", ":8888", "listen addr")

func main() {
	flag.Parse()
	log.SetFlags(0)

	h := make(http.Header)
	h.Add("X-Echo-Server", "true")

	upgrader := websocket.Upgrader{}

	log.Println("ready to listen", *listen)
	log.Fatal(http.ListenAndServe(*listen, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, h)
		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close()

		for {
			t, r, err := conn.NextReader()
			if err != nil {
				log.Println(err)
				return
			}

			msg, err := ioutil.ReadAll(r)
			if err != nil {
				log.Println(err)
				return
			}

			resp := []rune(string(msg))
			for i, l := 0, len(resp)-1; i < len(resp)/2; i, l = i+1, l-1 {
				resp[i], resp[l] = resp[l], resp[i]
			}

			if t == websocket.TextMessage {
				writer, err := conn.NextWriter(t)
				if err != nil {
					log.Println(err)
					return
				}

				_, err = writer.Write([]byte(string(resp)))
				if err != nil {
					log.Println(err)
					return
				}

				err = writer.Close()
				if err != nil {
					log.Println(err)
					return
				}
			}
		}
	})))
}


