package admin

import (
	"gateway/config"
	"gateway/core"
	"gateway/core/vm"
	"gateway/core/vm/advanced"
	"gateway/logreport"
	"gateway/repl"
	"net"
	"time"

	aphttp "gateway/http"

	"golang.org/x/net/websocket"
)

type ReplController struct {
	BaseController
	conf                config.ProxyAdmin
	keyStore            vm.DataSource
	remoteEndpointStore vm.DataSource
	preparer            advanced.RequestPreparer
}

func RouteRepl(controller *ReplController,
	path string,
	router aphttp.Router,
	conf config.ProxyAdmin,
	keyStore, remoteEndpointStore vm.DataSource,
	preparer advanced.RequestPreparer) {

	controller.conf = conf
	controller.keyStore = keyStore
	controller.remoteEndpointStore = remoteEndpointStore
	controller.preparer = preparer

	router.Handle(path, websocket.Handler(controller.replHandler))
}

func (c *ReplController) replHandler(ws *websocket.Conn) {
	r := repl.NewRepl(core.VMCopy(c.accountID(ws.Request()), c.apiID(ws.Request()), c.environmentID(ws.Request()), c.keyStore, c.remoteEndpointStore, c.preparer, nil))

	go func() {
		stopReader := make(chan int, 1)
		stopWriter := make(chan int, 1)

		// start a ticker for the websocket heartbeat
		heartbeatTicker := time.NewTicker(time.Duration(c.conf.WsHeartbeatInterval) * time.Second)
		// when the function finishes stop the read loop go routine, stop the ticker, stop the repl and close the websocket
		defer func() {
			stopReader <- 0
			heartbeatTicker.Stop()
			r.Stop()
			ws.Close()
		}()

		// websocket read loop in a separate go routine
		go func() {
			for {
				select {
				case <-stopReader:
					return
				default:
					m := make([]byte, c.conf.ReplMaximumFrameSize)
					ws.SetReadDeadline(time.Now().Add(time.Duration(c.conf.WsReadDeadline) * time.Second))
					n, err := ws.Read(m)
					if err != nil {
						switch err.(type) {
						case *net.OpError:
							// swallow OpError, which occurrs when the ReadDeadline is hit. This way
							// the select loop keeps spinning instead of blocking on ws.Read indefinitely.
						default:
							// anything else stop the writer
							logreport.Println(err)
							stopWriter <- 0
						}
					} else {
						// push the input to the repl
						r.Input <- m[:n]
					}
				}
			}
		}()

		// main repl feedback loop to push repl results to the websocket and send heartbeats
		for {
			select {
			case out := <-r.Output:
				// write to socket
				ws.SetWriteDeadline(time.Now().Add(time.Duration(c.conf.WsWriteDeadline) * time.Second))
				if _, err := ws.Write(out); err != nil {
					logreport.Println(err)
					return
				}
			case <-heartbeatTicker.C:
				// send heartbeat
				f := &repl.Frame{Type: repl.HEARBEAT}
				ws.SetWriteDeadline(time.Now().Add(time.Duration(c.conf.WsWriteDeadline) * time.Second))
				if _, err := ws.Write(f.JSON()); err != nil {
					logreport.Println(err)
					return
				}
			case <-stopWriter:
				return
			}
		}

	}()

	// Run is blocking so this will wait to return
	r.Run()
}
