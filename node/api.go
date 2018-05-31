// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package node

import (
	"fmt"
	"strings"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/p2p"
	"github.com/ethereumproject/go-ethereum/p2p/discover"
	"github.com/ethereumproject/go-ethereum/rpc"
	"math/rand"
	"path/filepath"
	"time"

	kf "github.com/rotblauer/go-kf"
)

// PrivateAdminAPI is the collection of administrative API methods exposed only
// over a secure RPC channel.
type PrivateAdminAPI struct {
	node *Node // Node interfaced by this API
}

// NewPrivateAdminAPI creates a new API definition for the private admin methods
// of the node itself.
func NewPrivateAdminAPI(node *Node) *PrivateAdminAPI {
	return &PrivateAdminAPI{node: node}
}

// AddPeer requests connecting to a remote node, and also maintaining the new
// connection at all times, even reconnecting if it is lost.
func (api *PrivateAdminAPI) AddPeer(url string) (bool, error) {
	// Make sure the server is running, fail otherwise
	server := api.node.Server()
	if server == nil {
		return false, ErrNodeStopped
	}
	// Try to add the url as a static peer and return
	node, err := discover.ParseNode(url)
	if err != nil {
		return false, fmt.Errorf("invalid enode: %v", err)
	}
	server.AddPeer(node)
	return true, nil
}

func (api *PrivateAdminAPI) QueryUnknownPeers() (bool, error) {
	go func() {
		tick := time.Tick(6 * time.Second)
		refreshMemTick := time.Tick(30 * time.Minute)
		var refreshing bool
		var unchecked = []string{}
		// matching should be smaller
		appendIfMissing := func(slice []string, s string) []string {
			for _, ele := range slice {
				if ele == s {
					return slice
				}
			}
			return append(slice, s)
		}
		uniq := func(removeFrom []string, matching []string) (unique []string) {
			unique = []string{}
			//glog.V(logger.Error).Errorln("unique", len(unique))
			for _, s := range removeFrom {
				var match bool
				for _, m := range matching {
					if strings.Contains(s, m) {
						match = true
						break
						//if i == 0 {
						//	unique = unique[0:]
						//} else if i == len(removeFrom)-1 {
						//	unique = unique[:i]
						//} else {
						//	unique = append(unique[:i], unique[i+1:]...)
						//}
						//continue outer
					}
				}
				if !match {
					unique = append(unique, s)
				}
			}
			return unique
		}
		doRefresh := func() {
			refreshing = true
			s, e := kf.NewStore(&kf.StoreConfig{
				BaseDir: filepath.Join(common.HomeDir(), "Library", "etc-peering"),
				Locking: true,
				KV:      true,
			})
			if e != nil {
				glog.V(logger.Error).Errorln("error opening KF", e)
				return
			}
			defer s.Close()
			nodesReportingNeighbors, e := s.GetKeys("findnode")
			if e != nil {
				panic(e.Error())
			}
			glog.V(logger.Error).Errorln("nodesReportingNeighbors", len(nodesReportingNeighbors))
			for _, n := range nodesReportingNeighbors {
				v, e := s.GetValue(filepath.Join("findnode", n))
				if e != nil {
					glog.V(logger.Error).Errorln("error getting neighbors data", e)
					return
				}
				// split lines, each has a reported neighbor
				vs := strings.Split(string(v), "\n")
				//glog.V(logger.Error).Errorln("vs", len(vs))
				for _, v := range vs {
					unchecked = appendIfMissing(unchecked, v)
				}
			}
			//glog.D(logger.Info).Infof("Found %d nonunique unchecked neighbors", len(unchecked))
			//var checker = []string{}
			//unchecked = uniq(checker, unchecked)
			checkedOK, e := s.GetKeys(filepath.Join("reqHash", "OK"))
			if e != nil {
				glog.V(logger.Error).Errorln("error ", e)
			} else {
				unchecked = uniq(unchecked, checkedOK)
			}
			checkedFAIL, e := s.GetKeys(filepath.Join("reqHash", "FAIL"))
			if e != nil {
				glog.V(logger.Error).Errorln("error ", e)
			} else {
				unchecked = uniq(unchecked, checkedFAIL)
			}

			//DiscRequested:           "Disconnect requested",
			//	DiscNetworkError:        "Network error",
			//		DiscProtocolError:       "Breach of protocol",
			//		DiscUselessPeer:         "Useless peer",
			//		DiscTooManyPeers:        "Too many peers",
			//		DiscAlreadyConnected:    "Already connected",
			//		DiscIncompatibleVersion: "Incompatible P2P protocol version",
			//		DiscInvalidIdentity:     "Invalid node identity",
			//		DiscQuitting:            "Client quitting",
			//		DiscUnexpectedIdentity:  "Unexpected identity",
			//		DiscSelf:                "Connected to self",
			//		DiscReadTimeout:         "Read timeout",
			//		DiscSubprotocolError:    "Subprotocol error",

			badDisconnects := []string{
				"breach-of-protocol",
				"network-error",
				"useless-peer",
				"incompatible-p2p-protocol-version",
				"invalid-node-identity",
				"unexpected-identity",
				"subprotocol-error",
			}
			conns := []string{}
			countsBad := []int{}
			for _, b := range badDisconnects {
				bb := filepath.Join("disconnect", b)
				v, e := s.GetKeys(bb)
				if e != nil {
					glog.V(logger.Error).Errorln("error ", e, b)
					continue
				}
				conns = append(conns, b)
				countsBad = append(countsBad, len(v))
				unchecked = uniq(unchecked, v)
			}
			str := fmt.Sprintf(`REFRESH PEERS: 
totalUnique=%d 
reqOK=%d 
reqF=%d
`,
				len(unchecked),
				len(checkedOK),
				len(checkedFAIL),
			)
			for i, s := range conns {
				str += fmt.Sprintf(`disconnect.%s=%d
`, s, countsBad[i])
			}
			glog.D(logger.Warn).Warnln(str)
			refreshing = false
		}
		doRefresh()
		for {
			select {
			case <-tick:
				if refreshing {
					continue
				}
				if unchecked == nil {
					panic("uncheck slice is nil")
				}
				l := len(unchecked)
				if l != 0 {
					i := rand.Intn(l)
					p := unchecked[i]
					ok, e := api.AddPeer(p)
					if e != nil {
						glog.D(logger.Info).Infoln("Add peer", p, "err: ", e)
						continue
					}
					glog.D(logger.Info).Infoln("Add peer", p, ok)
				}

			case <-refreshMemTick:
				doRefresh()
			}
		}
	}()
	return true, nil
}

// StartRPC starts the HTTP RPC API server.
func (api *PrivateAdminAPI) StartRPC(host *string, port *rpc.HexNumber, cors *string, apis *string) (bool, error) {
	api.node.lock.Lock()
	defer api.node.lock.Unlock()

	if api.node.httpHandler != nil {
		return false, fmt.Errorf("HTTP RPC already running on %s", api.node.httpEndpoint)
	}

	if host == nil {
		h := common.DefaultHTTPHost
		if api.node.httpHost != "" {
			h = api.node.httpHost
		}
		host = &h
	}
	if port == nil {
		port = rpc.NewHexNumber(api.node.httpPort)
	}
	if cors == nil {
		cors = &api.node.httpCors
	}

	modules := api.node.httpWhitelist
	if apis != nil {
		modules = nil
		for _, m := range strings.Split(*apis, ",") {
			modules = append(modules, strings.TrimSpace(m))
		}
	}

	if err := api.node.startHTTP(fmt.Sprintf("%s:%d", *host, port.Int()), api.node.rpcAPIs, modules, *cors); err != nil {
		return false, err
	}
	return true, nil
}

// StopRPC terminates an already running HTTP RPC API endpoint.
func (api *PrivateAdminAPI) StopRPC() (bool, error) {
	api.node.lock.Lock()
	defer api.node.lock.Unlock()

	if api.node.httpHandler == nil {
		return false, fmt.Errorf("HTTP RPC not running")
	}
	api.node.stopHTTP()
	return true, nil
}

// StartWS starts the websocket RPC API server.
func (api *PrivateAdminAPI) StartWS(host *string, port *rpc.HexNumber, allowedOrigins *string, apis *string) (bool, error) {
	api.node.lock.Lock()
	defer api.node.lock.Unlock()

	if api.node.wsHandler != nil {
		return false, fmt.Errorf("WebSocket RPC already running on %s", api.node.wsEndpoint)
	}

	if host == nil {
		h := common.DefaultWSHost
		if api.node.wsHost != "" {
			h = api.node.wsHost
		}
		host = &h
	}
	if port == nil {
		port = rpc.NewHexNumber(api.node.wsPort)
	}
	if allowedOrigins == nil {
		allowedOrigins = &api.node.wsOrigins
	}

	modules := api.node.wsWhitelist
	if apis != nil {
		modules = nil
		for _, m := range strings.Split(*apis, ",") {
			modules = append(modules, strings.TrimSpace(m))
		}
	}

	if err := api.node.startWS(fmt.Sprintf("%s:%d", *host, port.Int()), api.node.rpcAPIs, modules, *allowedOrigins); err != nil {
		return false, err
	}
	return true, nil
}

// StopRPC terminates an already running websocket RPC API endpoint.
func (api *PrivateAdminAPI) StopWS() (bool, error) {
	api.node.lock.Lock()
	defer api.node.lock.Unlock()

	if api.node.wsHandler == nil {
		return false, fmt.Errorf("WebSocket RPC not running")
	}
	api.node.stopWS()
	return true, nil
}

// PublicAdminAPI is the collection of administrative API methods exposed over
// both secure and unsecure RPC channels.
type PublicAdminAPI struct {
	node *Node // Node interfaced by this API
}

// NewPublicAdminAPI creates a new API definition for the public admin methods
// of the node itself.
func NewPublicAdminAPI(node *Node) *PublicAdminAPI {
	return &PublicAdminAPI{node: node}
}

// Peers retrieves all the information we know about each individual peer at the
// protocol granularity.
func (api *PublicAdminAPI) Peers() ([]*p2p.PeerInfo, error) {
	server := api.node.Server()
	if server == nil {
		return nil, ErrNodeStopped
	}
	return server.PeersInfo(), nil
}

// NodeInfo retrieves all the information we know about the host node at the
// protocol granularity.
func (api *PublicAdminAPI) NodeInfo() (*p2p.NodeInfo, error) {
	server := api.node.Server()
	if server == nil {
		return nil, ErrNodeStopped
	}
	return server.NodeInfo(), nil
}

// Datadir retrieves the current data directory the node is using.
func (api *PublicAdminAPI) Datadir() string {
	return api.node.DataDir()
}

// PublicWeb3API offers helper utils
type PublicWeb3API struct {
	stack *Node
}

// NewPublicWeb3API creates a new Web3Service instance
func NewPublicWeb3API(stack *Node) *PublicWeb3API {
	return &PublicWeb3API{stack}
}

// ClientVersion returns the node name
func (s *PublicWeb3API) ClientVersion() string {
	return s.stack.Server().Name
}

// Sha3 applies the ethereum sha3 implementation on the input.
// It assumes the input is hex encoded.
func (s *PublicWeb3API) Sha3(input string) string {
	return common.ToHex(crypto.Keccak256(common.FromHex(input)))
}
