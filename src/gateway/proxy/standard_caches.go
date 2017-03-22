package proxy

import (
	"gateway/core/cache"
	"gateway/model"
	apsql "gateway/sql"
	"sync"
)

// StandardCaches is the default LRU cache that is injected with a ModelDataSource and an upper bound for the cache size.
type StandardCaches struct {
	endpoints, libraries, plans, hosts DataSourceCache
	planAccountIDs                     map[int64][]int64 //plan ID -> []AccountID
	planAccountIDsMutex                sync.RWMutex
	apiEndpointIDs                     map[int64][]int64
	apiEndpointIDsMutex                sync.RWMutex
	apiHostnames                       map[int64][]string
	apiHostnamesMutex                  sync.RWMutex
}

func noop(key, value interface{}) {}

func newCaches(dataSource ModelDataSource, cacheSize int) Caches {
	return &StandardCaches{
		endpoints:      newEndpointCache(dataSource, cache.NewLRUCache(cacheSize, noop)),
		libraries:      newLibraryCache(dataSource, cache.NewLRUCache(cacheSize, noop)),
		hosts:          newHostCache(dataSource, cache.NewLRUCache(cacheSize, noop)),
		plans:          newPlanCache(dataSource, cache.NewLRUCache(cacheSize, noop)),
		planAccountIDs: make(map[int64][]int64),
		apiEndpointIDs: make(map[int64][]int64),
		apiHostnames:   make(map[int64][]string),
	}
}

// Endpoint returns an endpoint for the given criteria. Satisfies the Caches interface.
func (s *StandardCaches) Endpoint(criteria CacheCriteria) (*model.ProxyEndpoint, error) {
	if s.endpoints.Contains(criteria) {
		val, err := s.endpoints.Get(criteria)
		if err != nil {
			return nil, err
		}
		return val.(*model.ProxyEndpoint), nil
	}

	val, err := s.endpoints.Get(criteria)
	if err != nil {
		return nil, err
	}
	endpoint := val.(*model.ProxyEndpoint)
	id := criteria.(int64)

	// update maps
	s.apiEndpointIDsMutex.Lock()
	defer s.apiEndpointIDsMutex.Unlock()

	s.apiEndpointIDs[endpoint.APIID] = append(s.apiEndpointIDs[endpoint.APIID], id)
	return endpoint, nil
}

// Libraries returns a slice of libraries for the given criteria. Satisfies the Caches interface.
func (s *StandardCaches) Libraries(criteria CacheCriteria) ([]*model.Library, error) {
	val, err := s.libraries.Get(criteria)
	if err != nil {
		return nil, err
	}
	return val.([]*model.Library), nil
}

// Plan returns a plan for the given criteria. Satisfies the Caches interface.
func (s *StandardCaches) Plan(criteria CacheCriteria) (*model.Plan, error) {
	if s.plans.Contains(criteria) {
		val, err := s.plans.Get(criteria)
		if err != nil {
			return nil, err
		}
		return val.(*model.Plan), nil
	}

	val, err := s.plans.Get(criteria)
	if err != nil {
		return nil, err
	}

	plan := val.(*model.Plan)
	accountID := criteria.(int64)

	// Update planIDtoAccountIDs map
	s.planAccountIDsMutex.Lock()
	defer s.planAccountIDsMutex.Unlock()
	s.planAccountIDs[plan.ID] = append(s.planAccountIDs[plan.ID], accountID)

	return plan, nil
}

// Host returns a host for the given criteria. Satisfies the Caches interface.
func (s *StandardCaches) Host(criteria CacheCriteria) (*model.Host, error) {
	if s.hosts.Contains(criteria) {
		val, err := s.hosts.Get(criteria)
		if err != nil {
			return nil, err
		}
		return val.(*model.Host), nil
	}

	val, err := s.hosts.Get(criteria)
	if err != nil {
		return nil, err
	}
	host := val.(*model.Host)

	s.apiHostnamesMutex.Lock()
	defer s.apiHostnamesMutex.Unlock()
	s.apiHostnames[host.APIID] = append(s.apiHostnames[host.APIID], host.Hostname)
	return host, nil
}

// Notify satisfies the Listener interface.
func (s *StandardCaches) Notify(n *apsql.Notification) {
	switch {
	case n.Table == "accounts":
		s.handleAccountNotification(n.AccountID)
	case n.Table == "plans":
		s.handlePlanNotification(n.ID)
	case n.Table == "hosts":
		fallthrough
	case n.Table == "apis" && (n.Event == apsql.Update || n.Event == apsql.Delete):
		fallthrough
	case n.Table == "environments" && (n.Event == apsql.Update || n.Event == apsql.Delete):
		fallthrough
	case n.Table == "libraries":
		fallthrough
	case n.Table == "proxy_endpoint_schemas":
		fallthrough
	case n.Table == "proxy_endpoint_components" && (n.Event == apsql.Update || n.Event == apsql.Delete):
		fallthrough
	case n.Table == "remote_endpoints" && (n.Event == apsql.Update || n.Event == apsql.Delete):
		fallthrough
	case n.Table == "proxy_endpoints":
		s.handleAPINotification(n.APIID)
	}
}

func (s *StandardCaches) handleAPINotification(apiid int64) {
	// remove API's cached libraries
	s.libraries.Remove(apiid)

	// remove cached endpoints
	s.apiEndpointIDsMutex.RLock()
	for _, id := range s.apiEndpointIDs[apiid] {
		s.endpoints.Remove(id)
	}
	s.apiEndpointIDsMutex.RUnlock()

	// remove API entry from apiEndpointIDs map
	s.apiEndpointIDsMutex.Lock()
	delete(s.apiEndpointIDs, apiid)
	s.apiEndpointIDsMutex.Unlock()

	// remove API's cached hosts
	s.apiHostnamesMutex.RLock()
	for _, hostname := range s.apiHostnames[apiid] {
		s.hosts.Remove(hostname)
	}
	s.apiHostnamesMutex.RUnlock()

	s.apiHostnamesMutex.Lock()
	delete(s.apiHostnames, apiid)
	s.apiHostnamesMutex.Unlock()
}

func (s *StandardCaches) handleAccountNotification(accountid int64) {
	if !s.plans.Contains(accountid) {
		// account info is not cached in plans
		return
	}

	val, err := s.plans.Get(accountid)
	if err != nil {
		return
	}
	plan := val.(*model.Plan)

	s.planAccountIDsMutex.Lock()
	defer s.planAccountIDsMutex.Unlock()

	// remove account id from the plan -> account id map
	for i, id := range s.planAccountIDs[plan.ID] {
		if id == accountid {
			// move it to the end of the slice
			s.planAccountIDs[plan.ID][i] = s.planAccountIDs[plan.ID][len(s.planAccountIDs[plan.ID])-1]
			// chop the end of the slice off
			s.planAccountIDs[plan.ID] = s.planAccountIDs[plan.ID][:len(s.planAccountIDs[plan.ID])-1]
			break
		}
	}

	s.plans.Remove(accountid)
}

func (s *StandardCaches) handlePlanNotification(id int64) {
	s.planAccountIDsMutex.RLock()
	accountIDs, exist := s.planAccountIDs[id]

	if !exist {
		// plan is not cached
		s.planAccountIDsMutex.RUnlock()
		return
	}

	removedAccountIDs := make([]int64, 0)
	if accountIDs != nil {
		for _, accountID := range accountIDs {
			removedAccountIDs = append(removedAccountIDs, accountID)
		}
	}
	s.planAccountIDsMutex.RUnlock()

	for _, accountID := range removedAccountIDs {
		s.plans.Remove(accountID)
	}

	s.planAccountIDsMutex.Lock()
	defer s.planAccountIDsMutex.Unlock()

	delete(s.planAccountIDs, id)
}

// Reconnect satisfies the Listener interface.
func (s *StandardCaches) Reconnect() {
	s.planAccountIDsMutex.Lock()
	s.planAccountIDs = make(map[int64][]int64)
	s.planAccountIDsMutex.Unlock()

	s.apiEndpointIDsMutex.Lock()
	s.apiEndpointIDs = make(map[int64][]int64)
	s.apiEndpointIDsMutex.Unlock()

	s.apiHostnamesMutex.Lock()
	s.apiHostnames = make(map[int64][]string)
	s.apiHostnamesMutex.Unlock()

	s.endpoints.Purge()
	s.hosts.Purge()
	s.libraries.Purge()
}

// dbDataSource is a proxy pattern that satisfies the ModelDataSource interface. It also
// satisfies the Caches interface and can be used as a "pass-through" cache in which all
// calls are passed directly to the database and nothing is stored in-memory.
type dbDataSource struct {
	db *apsql.DB
}

func asModelDataSource(db *apsql.DB) *dbDataSource {
	return &dbDataSource{db}
}

// Endpoint satisfies the Caches interface.
func (d *dbDataSource) Endpoint(criteria CacheCriteria) (*model.ProxyEndpoint, error) {
	return d.FindProxyEndpointForProxy(criteria.(int64), model.ProxyEndpointTypeHTTP)
}

// Host satisfies the Caches interface.
func (d *dbDataSource) Host(criteria CacheCriteria) (*model.Host, error) {
	return d.FindHostForHostname(criteria.(string))
}

// Libraries satisfies the Caches interface.
func (d *dbDataSource) Libraries(criteria CacheCriteria) ([]*model.Library, error) {
	return d.AllLibrariesForProxy(criteria.(int64))
}

// Plan satisfies the Caches interface.
func (d *dbDataSource) Plan(criteria CacheCriteria) (*model.Plan, error) {
	return d.FindPlanByAccountID(criteria.(int64))
}

// FindProxyEndpointForProxy satisfies the ModelDataSource interface.
func (d *dbDataSource) FindProxyEndpointForProxy(id int64, endpointType string) (*model.ProxyEndpoint, error) {
	return model.FindProxyEndpointForProxy(d.db, id, endpointType)
}

// AllLibrariesForProxy satisfies the ModelDataSource interface.
func (d *dbDataSource) AllLibrariesForProxy(apiid int64) ([]*model.Library, error) {
	return model.AllLibrariesForProxy(d.db, apiid)
}

// FindPlanbyAccountID satisfies the ModelDataSource interface.
func (d *dbDataSource) FindPlanByAccountID(accountid int64) (*model.Plan, error) {
	return model.FindPlanByAccountID(d.db, accountid)
}

// FindHostsForHostname satisfies the ModelDataSource interface.
func (d *dbDataSource) FindHostForHostname(hostname string) (*model.Host, error) {
	return model.FindHostForHostname(d.db, hostname)
}
