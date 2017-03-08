package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"gateway/config"
	"gateway/logreport"
	"net"
	"net/http"
	"strings"
	"sync"

	apsql "gateway/sql"

	"github.com/gorilla/mux"
)

type parsedCertCache struct {
	sync.RWMutex
	dataSource  proxyDataSource
	proxyDomain string
	adminDomain string
	// map of hostname:certs
	certs map[string]*tls.Certificate
	// map of hostID:hostname
	idHostName map[int64]string
}

func newParsedCertCache(dataSource proxyDataSource, proxyDomain, adminDomain string) *parsedCertCache {
	c := make(map[string]*tls.Certificate)
	idHostName := make(map[int64]string)
	return &parsedCertCache{
		dataSource:  dataSource,
		proxyDomain: proxyDomain,
		adminDomain: adminDomain,
		certs:       c,
		idHostName:  idHostName,
	}
}

func (p *parsedCertCache) add(serverName string, cert *tls.Certificate) {
	p.Lock()
	defer p.Unlock()

	p.certs[serverName] = cert
}

func (p *parsedCertCache) get(serverName string) (*tls.Certificate, error) {
	p.RLock()
	if p.isProxySubdomain(serverName) {
		// return the proxy domain cert
		defer p.RUnlock()
		if val, ok := p.certs[p.proxyDomain]; ok {
			return val, nil
		}
		return nil, errors.New("no cert found for proxy")
	}

	if p.isAdminDomain(serverName) {
		// return admin domain cert
		defer p.RUnlock()
		if val, ok := p.certs[p.adminDomain]; ok {
			return val, nil
		}
		return nil, errors.New("no cert found for admin")
	}

	cached, ok := p.certs[serverName]
	p.RUnlock()

	if ok {
		return cached, nil
	}

	host, err := p.dataSource.Host(serverName)
	if err != nil {
		return nil, err
	}

	// check if the host has custom certs
	if host.CertContents() == "" || host.PrivateKeyContents() == "" {
		return nil, fmt.Errorf("SSL not configured for %s", host.Hostname)
	}

	p.Lock()
	defer p.Unlock()

	p.idHostName[host.ID] = host.Hostname

	// parse cert
	cert, err := tls.X509KeyPair([]byte(host.CertContents()), []byte(host.PrivateKeyContents()))
	if err != nil {
		return nil, err
	}

	p.certs[host.Hostname] = &cert
	return &cert, nil
}

func (p *parsedCertCache) isProxySubdomain(domain string) bool {
	return strings.Contains(domain, p.proxyDomain)
}

func (p *parsedCertCache) isAdminDomain(domain string) bool {
	return strings.Contains(domain, p.adminDomain)
}

func (p *parsedCertCache) Notify(n *apsql.Notification) {
	if n.Table != "hosts" {
		return
	}

	p.Lock()
	defer p.Unlock()

	if hostname, ok := p.idHostName[n.ID]; ok {
		// remove cached parsed certs from certs
		delete(p.certs, hostname)
		// remove ID to hostname mapping
		delete(p.idHostName, n.ID)
	}
}

func (p *parsedCertCache) Reconnect() {
	p.Lock()
	defer p.Unlock()

	p.certs = make(map[string]*tls.Certificate)
}

type TlsServer struct {
	server *Server
	router *mux.Router
	certs  *parsedCertCache
}

func NewTlsServer(s *Server, router *mux.Router) *TlsServer {
	cache := newParsedCertCache(s.proxyData, s.Conf.Proxy.Domain, s.Conf.Admin.Host)
	s.OwnDb.RegisterListener(cache)
	return &TlsServer{s, router, cache}
}

func (s *TlsServer) Listen() {
	laddr := fmt.Sprintf("%s:%d", s.server.Conf.Proxy.Host, s.server.Conf.Proxy.TLSPort)

	addProxyCerts := func(domain string, cert, key []byte) error {
		c, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return err
		}

		s.certs.add(domain, &c)
		return nil
	}

	// Parse proxy cert
	proxyCert, _ := base64.StdEncoding.DecodeString(s.server.Conf.Proxy.TLSCertContent)
	proxyKey, _ := base64.StdEncoding.DecodeString(s.server.Conf.Proxy.TLSKeyContent)

	err := addProxyCerts(s.server.Conf.Proxy.Domain, proxyCert, proxyKey)
	if err != nil {
		logreport.Fatal(fmt.Sprintf("failed to start tls: %s", err))
		return
	}

	// Parse admin cert
	adminCert, _ := base64.StdEncoding.DecodeString(s.server.Conf.Admin.TLSCertContent)
	adminKey, _ := base64.StdEncoding.DecodeString(s.server.Conf.Admin.TLSKeyContent)

	err = addProxyCerts(s.server.Conf.Admin.Host, adminCert, adminKey)
	if err != nil {
		logreport.Fatal(fmt.Sprintf("failed to start tls: %s", err))
		return
	}

	// create underlying tcp listener
	tcpListener, err := net.Listen("tcp", laddr)
	if err != nil {
		logreport.Fatal(fmt.Sprintf("failed to start tls: %s", err))
		return
	}

	tlsConfig := &tls.Config{
		GetCertificate: func(h *tls.ClientHelloInfo) (*tls.Certificate, error) {
			res, err := s.certs.get(h.ServerName)
			return res, err
		},
	}

	// if a ca cert is configured setup a cert pool
	if s.server.Conf.Proxy.TLSCacertContent != "" {
		caPool := x509.NewCertPool()
		caCertContents, _ := base64.StdEncoding.DecodeString(s.server.Conf.Proxy.TLSCacertContent)
		ok := caPool.AppendCertsFromPEM(caCertContents)
		if !ok {
			fmt.Println("failed to parse CA Cert contents from configuration")
		} else {
			tlsConfig.RootCAs = caPool
		}
	}

	//create tls listener to wrap tcp listener
	tlsListener := tls.NewListener(tcpListener, tlsConfig)

	logreport.Printf("%s HTTPS available on port %d", config.Admin, s.server.Conf.Proxy.TLSPort)
	http.Serve(tlsListener, s.router)
}

type tlsRedirectRouter struct {
	*mux.Router
	conf       *config.Configuration
	dataSource proxyDataSource
}

func (r *tlsRedirectRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h := strings.Split(req.Host, ":")
	hostName := h[0]
	// redirect for the admin
	if hostName == r.conf.Admin.Host {
		r.redirect(hostName, w, req)
		return
	}

	// redirect for custom domains if ForceSSL is true
	host, err := r.dataSource.Host(hostName)
	if err != nil {
		fmt.Println(err)
		return
	}
	if host.ForceSSL {
		r.redirect(hostName, w, req)
		return
	}

	r.Router.ServeHTTP(w, req)
}

func (r *tlsRedirectRouter) redirect(host string, w http.ResponseWriter, req *http.Request) {
	// update the requests scheme to https and the host to the TLS port and redirect
	req.URL.Scheme = "https"
	req.URL.Host = fmt.Sprintf("%s:%d", host, r.conf.Proxy.TLSPort)

	// Send a 307 in dev mode so browsers do not cache the redirect for all eternity.
	if r.conf.DevMode() {
		http.Redirect(w, req, req.URL.String(), 307)
		return
	}

	// use a 301 so the client can cache the redirect
	http.Redirect(w, req, req.URL.String(), 301)
}
