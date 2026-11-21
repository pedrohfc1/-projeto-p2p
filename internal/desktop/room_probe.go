package desktop

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"projetop2p/internal/roomprobe"
	"projetop2p/internal/voice"
)

type roomProbeState struct {
	sync.Mutex
	approved  peer.ID
	session   string
	until     time.Time
	epoch     uint64
	receiving bool
	cancel    context.CancelFunc
}

func findRoomProbePeer(view voice.View, alias string) (peer.ID, error) {
	if !view.Joined || view.Authorization != "valid" || view.Expires <= time.Now().Unix() {
		return "", errors.New("a sala precisa de autorização válida")
	}
	for _, participant := range view.Participants {
		if participant.Self {
			continue
		}
		sum := sha256.Sum256([]byte(participant.ID))
		if fmt.Sprintf("%x", sum[:6]) == alias {
			return peer.Decode(participant.ID)
		}
	}
	return "", errors.New("o participante não faz parte da sala")
}

func roomProbeAuthorized(service *voice.Service, session string, id peer.ID) bool {
	view := service.View()
	if !view.Joined || view.Session != session || view.Authorization != "valid" || view.Expires <= time.Now().Unix() {
		return false
	}
	for _, participant := range view.Participants {
		if !participant.Self && participant.ID == id.String() {
			return true
		}
	}
	return false
}

func (c *Core) ApproveRoomProbe(alias string) error {
	service, err := c.voiceService()
	if err != nil {
		return err
	}
	view := service.View()
	id, err := findRoomProbePeer(view, alias)
	if err != nil {
		return err
	}
	p := &c.roomProbe
	p.Lock()
	defer p.Unlock()
	if p.cancel != nil {
		return errors.New("teste de diagnóstico já em execução")
	}
	p.epoch++
	p.approved = id
	p.session = view.Session
	p.until = time.Now().Add(90 * time.Second)
	p.receiving = false
	return nil
}

func (c *Core) acceptRoomProbe(service *voice.Service, id peer.ID) (func() bool, bool) {
	p := &c.roomProbe
	p.Lock()
	if p.cancel != nil || p.approved != id || p.receiving || time.Now().After(p.until) {
		p.Unlock()
		return nil, false
	}
	session, epoch := p.session, p.epoch
	p.receiving = true
	p.Unlock()
	return func() bool {
		p.Lock()
		authorized := p.epoch == epoch && time.Now().Before(p.until)
		p.Unlock()
		return authorized && roomProbeAuthorized(service, session, id)
	}, true
}

func (c *Core) CancelRoomProbe() {
	p := &c.roomProbe
	p.Lock()
	defer p.Unlock()
	p.epoch++
	p.approved = ""
	p.until = time.Time{}
	if p.cancel != nil {
		p.cancel()
	}
}

func (c *Core) RunRoomProbe(alias string, seconds int) (roomprobe.Result, error) {
	service, err := c.voiceService()
	if err != nil {
		return roomprobe.Result{}, err
	}
	view := service.View()
	id, err := findRoomProbePeer(view, alias)
	if err != nil {
		return roomprobe.Result{}, err
	}
	c.mu.Lock()
	node := c.node
	c.mu.Unlock()
	if node == nil {
		return roomprobe.Result{}, errors.New("conexão indisponível")
	}
	p := &c.roomProbe
	p.Lock()
	if p.cancel != nil || (p.approved != "" && time.Now().Before(p.until)) {
		p.Unlock()
		return roomprobe.Result{}, errors.New("cancele o teste ou a permissão anterior")
	}
	ctx, cancel := context.WithCancel(c.ctx)
	p.cancel = cancel
	p.Unlock()
	defer func() {
		cancel()
		p.Lock()
		p.cancel = nil
		p.Unlock()
	}()
	return roomprobe.Run(ctx, node.Host, id, seconds, func() bool {
		return roomProbeAuthorized(service, view.Session, id)
	})
}
