package proxy

import (
	"errors"
	"gateway/core/cache"
	"gateway/model"
	"sync"
)

// Caches is a collection of functions for retrieving cached data for the proxy.
type Caches interface {
	Endpoint(CacheCriteria) (*model.ProxyEndpoint, error)
	Libraries(CacheCriteria) ([]*model.Library, error)
	Plan(CacheCriteria) (*model.Plan, error)
	Host(CacheCriteria) (*model.Host, error)
}

// DataSourceCache adds functions for retrieving data from a cache.
type DataSourceCache interface {
	Get(CacheCriteria) (interface{}, error)
	Remove(CacheCriteria) bool
	Contains(CacheCriteria) bool
}

// ModelDataSource adds functions for retrieving specific models for the cache, specifically these
// functions will be called when there's a cache miss and the data needs to be retrieved.
type ModelDataSource interface {
	FindProxyEndpointForProxy(int64, string) (*model.ProxyEndpoint, error)
	AllLibrariesForProxy(int64) ([]*model.Library, error)
	FindPlanByAccountID(int64) (*model.Plan, error)
	FindHostForHostname(string) (*model.Host, error)
}

// CacheCriteria is the criteria for retrieving data from a cache. Usually a model's ID or something
// similar.
type CacheCriteria interface{}

type baseCache struct {
	cache cache.Cacher
	sync.RWMutex
	dataSource ModelDataSource
}

// Contains returns true if the cache contains an entry for the given criteria. Satisfies the DataSourceCache interface.
func (b *baseCache) Contains(criteria CacheCriteria) bool {
	b.RLock()
	defer b.RUnlock()
	return b.cache.Contains(criteria)
}

// Remove removes the entry for the given criteria. Satisfies the DataSourceCache interface.
func (b *baseCache) Remove(criteria CacheCriteria) bool {
	b.Lock()
	defer b.Unlock()
	return b.cache.Remove(criteria)
}

type endpointCache struct {
	baseCache
}

func newEndpointCache(endpointSource ModelDataSource, cache cache.Cacher) *endpointCache {
	return &endpointCache{baseCache: baseCache{cache: cache, dataSource: endpointSource}}
}

// Get returns a ProxyEndpoint based on the supplied criteria. Satisfies the DataSourceCache interface.
func (e *endpointCache) Get(criteria CacheCriteria) (interface{}, error) {
	e.RLock()
	if val, ok := e.cache.Get(criteria); ok {
		e.RUnlock()
		return val, nil
	}
	e.RUnlock()

	id, ok := criteria.(int64)
	if !ok {
		return nil, errors.New("criteria should be id of type int64")
	}
	val, err := e.dataSource.FindProxyEndpointForProxy(id, model.ProxyEndpointTypeHTTP)
	if err != nil {
		return nil, err
	}

	e.Lock()
	defer e.Unlock()

	e.cache.Add(id, val)
	return val, nil
}

func (e *endpointCache) Remove(criteria CacheCriteria) bool {
	e.Lock()
	defer e.Unlock()
	return e.cache.Remove(criteria)
}

type libraryCache struct {
	baseCache
}

func newLibraryCache(librarySource ModelDataSource, cache cache.Cacher) *libraryCache {
	return &libraryCache{baseCache: baseCache{cache: cache, dataSource: librarySource}}
}

func (l *libraryCache) Get(criteria CacheCriteria) (interface{}, error) {
	l.RLock()
	if val, ok := l.cache.Get(criteria); ok {
		l.RUnlock()
		return val, nil
	}
	l.RUnlock()

	apiid, ok := criteria.(int64)
	if !ok {
		return nil, errors.New("criteria should be APIID of type int64")
	}
	val, err := l.dataSource.AllLibrariesForProxy(apiid)
	if err != nil {
		return nil, err
	}

	l.Lock()
	defer l.Unlock()

	l.cache.Add(apiid, val)
	return val, nil
}

func (l *libraryCache) Remove(criteria CacheCriteria) bool {
	l.Lock()
	defer l.Unlock()
	return l.cache.Remove(criteria)
}

type planCache struct {
	baseCache
}

func newPlanCache(planSource ModelDataSource, cache cache.Cacher) *planCache {
	return &planCache{baseCache: baseCache{cache: cache, dataSource: planSource}}
}

func (p *planCache) Get(criteria CacheCriteria) (interface{}, error) {
	p.RLock()
	if val, ok := p.cache.Get(criteria); ok {
		p.RUnlock()
		return val, nil
	}
	p.RUnlock()

	accountid, ok := criteria.(int64)
	if !ok {
		return nil, errors.New("criteria should be accountID of type int64")
	}
	val, err := p.dataSource.FindPlanByAccountID(accountid)
	if err != nil {
		return nil, err
	}

	p.Lock()
	defer p.Unlock()

	p.cache.Add(accountid, val)
	return val, nil

}

func (p *planCache) Remove(criteria CacheCriteria) bool {
	p.Lock()
	defer p.Unlock()
	return p.cache.Remove(criteria)
}

type hostCache struct {
	baseCache
}

func newHostCache(hostSource ModelDataSource, cache cache.Cacher) *hostCache {
	return &hostCache{baseCache: baseCache{cache: cache, dataSource: hostSource}}
}

func (h *hostCache) Get(criteria CacheCriteria) (interface{}, error) {
	h.RLock()
	if val, ok := h.cache.Get(criteria); ok {
		h.RUnlock()
		return val, nil
	}
	h.RUnlock()

	hostname, ok := criteria.(string)
	if !ok {
		return nil, errors.New("criteria should be hostname of type string")
	}
	val, err := h.dataSource.FindHostForHostname(hostname)
	if err != nil {
		return nil, err
	}

	h.Lock()
	defer h.Unlock()

	h.cache.Add(hostname, val)
	return val, nil
}

func (h *hostCache) Remove(criteria CacheCriteria) bool {
	h.Lock()
	defer h.Unlock()
	return h.cache.Remove(criteria)
}
