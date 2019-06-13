// Copyright 2014 The go-ethereum Authors
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
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/p2p/discover"
	"github.com/ethereumproject/go-ethereum/p2p/nat"
	"github.com/spf13/afero"
)

var (
	datadirPrivateKey   = "nodekey"            // Path within the datadir to the node's private key
	datadirStaticNodes  = "static-nodes.json"  // Path within the datadir to the static node list
	datadirTrustedNodes = "trusted-nodes.json" // Path within the datadir to the trusted node list
	datadirNodeDatabase = "nodes"              // Path within the datadir to store the node infos
)

// fs wraps afero.FS, used as a type of it's own so that we can take it's address
// and set a zero-value default.
type fs struct {
	afero.Fs
}

// Config represents a small collection of configuration values to fine tune the
// P2P network layer of a protocol stack. These values can be further extended by
// all registered services.
type Config struct {
	// DataDir is the file system folder the node should use for any data storage
	// requirements. The configured data directory will not be directly shared with
	// registered services, instead those can use utility methods to create/access
	// databases or flat files. This enables ephemeral nodes which can fully reside
	// in memory.
	DataDir string

	// IPCPath is the requested location to place the IPC endpoint. If the path is
	// a simple file name, it is placed inside the chaindata directory (or on the root
	// pipe path on Windows), whereas if it's a resolvable path name (absolute or
	// relative), then that specific path is enforced. An empty path disables IPC.
	IPCPath string

	// fs is an abstracted file system.
	// In normal use, it points to a thin wrapper around the standard os FS package,
	// and can be swapped for an abstracted in-mem map during tests, which helps
	// ensure test reliability by removing sometimes-laggy OS FS R/Ws.
	// The var is currently private because it is not set to inMem outside of this package tests.
	fs *fs

	// This field should be a valid secp256k1 private key that will be used for both
	// remote peer identification as well as network traffic encryption. If no key
	// is configured, the preset one is loaded from the data dir, generating it if
	// needed.
	PrivateKey *ecdsa.PrivateKey

	// Name sets the node name of this server.
	Name string

	// NoDiscovery specifies whether the peer discovery mechanism should be started
	// or not. Disabling is usually useful for protocol debugging (manual topology).
	NoDiscovery bool

	// Bootstrap nodes used to establish connectivity with the rest of the network.
	BootstrapNodes []*discover.Node

	// Network interface address on which the node should listen for inbound peers.
	ListenAddr string

	// If set to a non-nil value, the given NAT port mapper is used to make the
	// listening port available to the Internet.
	NAT nat.Interface

	// If Dialer is set to a non-nil value, the given Dialer is used to dial outbound
	// peer connections.
	Dialer *net.Dialer

	// If NoDial is true, the node will not dial any peers.
	NoDial bool

	// MaxPeers is the maximum number of peers that can be connected. If this is
	// set to zero, then only the configured static and trusted peers can connect.
	MaxPeers int

	// MaxPendingPeers is the maximum number of peers that can be pending in the
	// handshake phase, counted separately for inbound and outbound connections.
	// Zero defaults to preset values.
	MaxPendingPeers int

	// HTTPHost is the host interface on which to start the HTTP RPC server. If this
	// field is empty, no HTTP API endpoint will be started.
	HTTPHost string

	// HTTPPort is the TCP port number on which to start the HTTP RPC server. The
	// default zero value is/ valid and will pick a port number randomly (useful
	// for ephemeral nodes).
	HTTPPort int

	// HTTPCors is the Cross-Origin Resource Sharing header to send to requesting
	// clients. Please be aware that CORS is a browser enforced security, it's fully
	// useless for custom HTTP clients.
	HTTPCors string

	// HTTPModules is a list of API modules to expose via the HTTP RPC interface.
	// If the module list is empty, all RPC API endpoints designated public will be
	// exposed.
	HTTPModules []string

	// WSHost is the host interface on which to start the websocket RPC server. If
	// this field is empty, no websocket API endpoint will be started.
	WSHost string

	// WSPort is the TCP port number on which to start the websocket RPC server. The
	// default zero value is/ valid and will pick a port number randomly (useful for
	// ephemeral nodes).
	WSPort int

	// WSOrigins is the list of domain to accept websocket requests from. Please be
	// aware that the server can only act upon the HTTP request the client sends and
	// cannot verify the validity of the request header.
	WSOrigins string

	// WSModules is a list of API modules to expose via the websocket RPC interface.
	// If the module list is empty, all RPC API endpoints designated public will be
	// exposed.
	WSModules []string
}

// IPCEndpoint resolves an IPC endpoint based on a configured value, taking into
// account the set data folders as well as the designated platform we're currently
// running on.
func (c *Config) IPCEndpoint() string {
	// Short circuit if IPC has not been enabled
	if c.IPCPath == "" {
		return ""
	}
	// just for safety, but should be already initialized
	if c.fs == nil {
		c.fs = &fs{afero.NewOsFs()}
	}
	// On windows we can only use plain top-level pipes
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(c.IPCPath, `\\.\pipe\`) {
			return c.IPCPath
		}
		return `\\.\pipe\` + c.IPCPath
	}
	// Resolve names into the data directory full paths otherwise
	if filepath.Base(c.IPCPath) == c.IPCPath {
		if c.DataDir == "" {
			// Use afs.MkdirAll instead of afero.TempDir because with the latter,
			// afero creates temporary™ sub dir under the normal temp dir. Not sure if bug or expected, but
			// just using plain os temp dir is ok.
			// Anyways, all afero ~is doing~ do should do is just mkdir -p the os-specific tmp dir path anyway
			tempDir := os.TempDir()
			err := c.fs.MkdirAll(tempDir, os.ModeTemporary)
			if err != nil && !os.IsExist(err) {
				glog.Fatal(err)
			}
			return filepath.Join(tempDir, c.IPCPath)
		}
		return filepath.Join(c.DataDir, c.IPCPath)
	}
	return c.IPCPath
}

// DefaultIPCEndpoint returns the IPC path used by default.
func DefaultIPCEndpoint(chainDir string) string {
	config := &Config{DataDir: chainDir, IPCPath: common.DefaultIPCSocket}
	return config.IPCEndpoint()
}

// HTTPEndpoint resolves an HTTP endpoint based on the configured host interface
// and port parameters.
func (c *Config) HTTPEndpoint() string {
	if c.HTTPHost == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.HTTPHost, c.HTTPPort)
}

// WSEndpoint resolves an websocket endpoint based on the configured host interface
// and port parameters.
func (c *Config) WSEndpoint() string {
	if c.WSHost == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.WSHost, c.WSPort)
}

// NodeKey retrieves the currently configured private key of the node, checking
// first any manually set key, falling back to the one found in the configured
// data folder. If no key can be found, a new one is generated.
func (c *Config) NodeKey() *ecdsa.PrivateKey {
	// Use any specifically configured key
	if c.PrivateKey != nil {
		return c.PrivateKey
	}
	// just for safety, but should be already initialized
	if c.fs == nil {
		c.fs = &fs{afero.NewOsFs()}
	}
	// Generate ephemeral key if no datadir is being used
	if c.DataDir == "" {
		key, err := crypto.GenerateKey()
		if err != nil {
			glog.Fatalf("Failed to generate ephemeral node key: %v", err)
		}
		return key
	}
	// Fall back to persistent key from the data directory
	keyfile := filepath.Join(c.DataDir, datadirPrivateKey)
	f, err := c.fs.Open(keyfile)
	if err == nil {
		// file open error was nil, attempt to load key, fatal on any error
		defer f.Close()
		key, err := crypto.LoadECDSA(f)
		if err == nil {
			return key
		}

		glog.Fatalf("could not load key file: %v", err)
	}

	// there was an error opening an existing key file; there's nothing we can do about this
	if !os.IsNotExist(err) {
		glog.Fatalf("could not load key file: %v", err)
		return nil
	}

	// No key file found, generate and store a new one
	key, err := crypto.GenerateKey()
	if err != nil {
		glog.Fatalf("Failed to generate node key: %v", err)
	}

	f, err = c.fs.Create(keyfile)
	if err != nil {
		glog.Fatalf("failed to open node key file: %v", err)
	}
	defer f.Close()
	if _, err := crypto.WriteECDSAKey(f, key); err != nil {
		glog.V(logger.Error).Infof("Failed to persist node key: %v", err)
	}
	return key
}

// StaticNodes returns a list of node enode URLs configured as static nodes.
func (c *Config) StaticNodes() []*discover.Node {
	return c.parsePersistentNodes(datadirStaticNodes)
}

// TrusterNodes returns a list of node enode URLs configured as trusted nodes.
func (c *Config) TrusterNodes() []*discover.Node {
	return c.parsePersistentNodes(datadirTrustedNodes)
}

// parsePersistentNodes parses a list of discovery node URLs loaded from a .json
// file from within the data directory.
func (c *Config) parsePersistentNodes(file string) []*discover.Node {
	// Short circuit if no node config is present
	if c.DataDir == "" {
		return nil
	}
	// just for safety, but should be already initialized
	if c.fs == nil {
		c.fs = &fs{afero.NewOsFs()}
	}
	path := filepath.Join(c.DataDir, file)
	if _, err := c.fs.Stat(path); err != nil {
		return nil
	}
	// Load the nodes from the config file
	blob, err := afero.ReadFile(c.fs, path)
	if err != nil {
		glog.V(logger.Error).Infof("Failed to access nodes: %v", err)
		return nil
	}
	nodelist := []string{}
	if err := json.Unmarshal(blob, &nodelist); err != nil {
		glog.V(logger.Error).Infof("Failed to load nodes: %v", err)
		return nil
	}
	// Interpret the list as a discovery node array
	var nodes []*discover.Node
	for _, url := range nodelist {
		if url == "" {
			continue
		}
		node, err := discover.ParseNode(url)
		if err != nil {
			glog.V(logger.Error).Infof("Node URL %s: %v\n", url, err)
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes
}
