package api

import (
	"context"
	"fmt"
	"tkbastion/pkg/guacd"
	"tkbastion/pkg/log"

	"github.com/gorilla/websocket"
)

type TunnelHandler struct {
	ws     *websocket.Conn
	tunnel *guacd.Tunnel
	ctx    context.Context
	cancel context.CancelFunc
}

func NewTunnelHandler(ws *websocket.Conn, tunnel *guacd.Tunnel) *TunnelHandler {
	ctx, cancel := context.WithCancel(context.Background())
	return &TunnelHandler{
		ws:     ws,
		tunnel: tunnel,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (r TunnelHandler) Start() {
	go func() {
		for {
			select {
			case <-r.ctx.Done():
				return
			default:
				instruction, err := r.tunnel.Read()
				if err != nil {
					fmt.Println("from guacd: ", string(instruction))
					disConnectNewSession(r.ws, TunnelClosed, "远程连接已关闭")
					return
				}
				if len(instruction) == 0 {
					continue
				}
				err = r.ws.WriteMessage(websocket.TextMessage, instruction)
				if err != nil {
					log.Debugf("WebSocket写入失败，即将关闭Guacd连接...")
					return
				}
			}
		}
	}()
}

func (r TunnelHandler) Stop() {
	r.cancel()
}
