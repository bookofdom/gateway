package proxy

import (
	apsql "gateway/sql"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

func (s *CacheSuite) TestStandardCachesGetters(c *gc.C) {
	caches := newCaches(&dataSourceMock{}, 1)

	//Should return an Endpoint
	endpoint, err := caches.Endpoint(int64(1))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(endpoint, gc.NotNil)

	//Should return a Host
	host, err := caches.Host("foo")
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(host, gc.NotNil)

	//Should return Libraries
	libraries, err := caches.Libraries(int64(1))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(libraries, gc.NotNil)

	//Should return a Plan
	plan, err := caches.Plan(int64(1))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(plan, gc.NotNil)
}

func (s *CacheSuite) TestPlanNotifications(c *gc.C) {
	caches := newCaches(&dataSourceMock{}, 10)
	stdCaches := caches.(*StandardCaches)

	// Fill the cache with a few entries
	for _, v := range []int64{1, 2, 3} {
		val, err := caches.Plan(int64(v))
		c.Assert(err, jc.ErrorIsNil)
		c.Assert(val, gc.NotNil)
	}
	criteria := int64(1)

	// Ensure the planIDtoAccountIDs map is updated
	c.Assert(len(stdCaches.planAccountIDs[criteria]), gc.Equals, 3)

	notification := &apsql.Notification{Table: "plans", ID: 1}
	stdCaches.Notify(notification)

	planCache := stdCaches.plans.(*planCache)

	// Ensure that the cache value for the plan was removed
	c.Assert(planCache.cache.Len(), gc.Equals, 0)
	c.Assert(planCache.Contains(criteria), jc.IsFalse)

	// Ensure that the planAccountIDs map has 0 results for the plan ID
	c.Assert(len(stdCaches.planAccountIDs[criteria]), gc.Equals, 0)
}

func (s *CacheSuite) TestAccountNotification(c *gc.C) {
	caches := newCaches(&dataSourceMock{}, 10)
	stdCaches := caches.(*StandardCaches)

	// Fill the cache with a few entries
	for _, v := range []int64{1, 2, 3} {
		val, err := caches.Plan(int64(v))
		c.Assert(err, jc.ErrorIsNil)
		c.Assert(val, gc.NotNil)
	}
	criteria := int64(1)

	// Ensure the planIDtoAccountIDs map is updated
	c.Assert(len(stdCaches.planAccountIDs[criteria]), gc.Equals, 3)

	notification := &apsql.Notification{Table: "accounts", AccountID: criteria}
	stdCaches.Notify(notification)

	planCache := stdCaches.plans.(*planCache)

	// Ensure that the cache value for the account's plan was removed
	c.Assert(planCache.cache.Len(), gc.Equals, 2)
	c.Assert(planCache.Contains(criteria), jc.IsFalse)

	// Ensure that the planAccountIDs map was updated and the account ID was removed from the
	// plan's account ID slice.
	c.Assert(len(stdCaches.planAccountIDs[criteria]), gc.Equals, 2)
}

func (s *CacheSuite) TestAPINotification(c *gc.C) {
	apiid := int64(1)
	endpointids := []int64{1, 2, 3}
	hostnames := []string{"foo", "bar", "baz"}

	for i, t := range []struct {
		should        string
		notification  *apsql.Notification
		causeAPIPurge bool
	}{{
		should:        "purge API cache on hosts change",
		notification:  &apsql.Notification{Table: "hosts", APIID: apiid},
		causeAPIPurge: true,
	}, {
		should:        "purge API cache on apis update",
		notification:  &apsql.Notification{Table: "apis", APIID: apiid, Event: apsql.Update},
		causeAPIPurge: true,
	}, {
		should:        "purge API cache on apis delete",
		notification:  &apsql.Notification{Table: "apis", APIID: apiid, Event: apsql.Delete},
		causeAPIPurge: true,
	}, {
		should:        "not purge API cache on apis insert",
		notification:  &apsql.Notification{Table: "apis", APIID: apiid, Event: apsql.Insert},
		causeAPIPurge: false,
	}, {
		should:        "purge API cache on environments update",
		notification:  &apsql.Notification{Table: "environments", APIID: apiid, Event: apsql.Update},
		causeAPIPurge: true,
	}, {
		should:        "purge API cache on environments delete",
		notification:  &apsql.Notification{Table: "environments", APIID: apiid, Event: apsql.Delete},
		causeAPIPurge: true,
	}, {
		should:        "not purge API cache on environments insert",
		notification:  &apsql.Notification{Table: "environments", APIID: apiid, Event: apsql.Insert},
		causeAPIPurge: false,
	}, {
		should:        "purge API cache on libraries change",
		notification:  &apsql.Notification{Table: "libraries", APIID: apiid},
		causeAPIPurge: true,
	}, {
		should:        "purge API cache on proxy_endpoint_schemas change",
		notification:  &apsql.Notification{Table: "proxy_endpoint_schemas", APIID: apiid},
		causeAPIPurge: true,
	}, {
		should:        "purge API cache on proxy_endpoint_components update",
		notification:  &apsql.Notification{Table: "proxy_endpoint_components", APIID: apiid, Event: apsql.Update},
		causeAPIPurge: true,
	}, {
		should:        "purge API cache on proxy_endpoint_components delete",
		notification:  &apsql.Notification{Table: "proxy_endpoint_components", APIID: apiid, Event: apsql.Delete},
		causeAPIPurge: true,
	}, {
		should:        "not purge API cache on proxy_endpoint_components insert",
		notification:  &apsql.Notification{Table: "proxy_endpoint_components", APIID: apiid, Event: apsql.Insert},
		causeAPIPurge: false,
	}, {
		should:        "purge API cache on remote_endpoints update",
		notification:  &apsql.Notification{Table: "remote_endpoints", APIID: apiid, Event: apsql.Update},
		causeAPIPurge: true,
	}, {
		should:        "purge API cache on remote_endpoints delete",
		notification:  &apsql.Notification{Table: "remote_endpoints", APIID: apiid, Event: apsql.Delete},
		causeAPIPurge: true,
	}, {
		should:        "not purge API cache on remote_endpoints insert",
		notification:  &apsql.Notification{Table: "remote_endpoints", APIID: apiid, Event: apsql.Insert},
		causeAPIPurge: false,
	}, {
		should:        "purge API cache on proxy_endpoints change",
		notification:  &apsql.Notification{Table: "proxy_endpoints", APIID: apiid},
		causeAPIPurge: true,
	}} {
		c.Logf("Test %d: should %s", i, t.should)

		caches := newCaches(&dataSourceMock{}, 10)
		stdCaches := caches.(*StandardCaches)
		libCache := stdCaches.libraries.(*libraryCache)
		endpointCache := stdCaches.endpoints.(*endpointCache)
		hostCache := stdCaches.hosts.(*hostCache)

		// Fill up the caches
		libraries, err := caches.Libraries(apiid)
		c.Assert(err, jc.ErrorIsNil)
		c.Assert(libraries, gc.NotNil)

		for _, v := range endpointids {
			val, err := caches.Endpoint(v)
			c.Assert(err, jc.ErrorIsNil)
			c.Assert(val, gc.NotNil)
		}

		for _, hostname := range hostnames {
			val, err := caches.Host(hostname)
			c.Assert(err, jc.ErrorIsNil)
			c.Assert(val, gc.NotNil)
		}

		// apiEndpointIDs map should be updated with the endpoint previously retrieved from the cache
		c.Assert(len(stdCaches.apiEndpointIDs), gc.Equals, 1)
		c.Assert(len(stdCaches.apiEndpointIDs[apiid]), gc.Equals, len(endpointids))

		// apiHostnames map should be updated with retrieved cache values
		c.Assert(len(stdCaches.apiHostnames), gc.Equals, 1)
		c.Assert(len(stdCaches.apiHostnames[apiid]), gc.Equals, len(hostnames))

		// Ensure caches are correct
		c.Assert(libCache.cache.Len(), gc.Equals, 1)
		c.Assert(endpointCache.cache.Len(), gc.Equals, len(endpointids))
		c.Assert(hostCache.cache.Len(), gc.Equals, len(hostnames))

		stdCaches.Notify(t.notification)

		if t.causeAPIPurge {
			// Ensure caches are empty
			c.Assert(libCache.cache.Len(), gc.Equals, 0)
			c.Assert(endpointCache.cache.Len(), gc.Equals, 0)
			c.Assert(hostCache.cache.Len(), gc.Equals, 0)

			// Ensure maps are updated
			c.Assert(len(stdCaches.apiEndpointIDs), gc.Equals, 0)
			c.Assert(stdCaches.apiEndpointIDs[apiid], gc.IsNil)
			c.Assert(len(stdCaches.apiHostnames), gc.Equals, 0)
			c.Assert(len(stdCaches.apiHostnames), gc.Equals, 0)
		} else {
			// Ensure caches did not change
			c.Assert(libCache.cache.Len(), gc.Equals, 1)
			c.Assert(endpointCache.cache.Len(), gc.Equals, len(endpointids))
			c.Assert(hostCache.cache.Len(), gc.Equals, len(hostnames))

			// Ensure maps were not updated
			c.Assert(len(stdCaches.apiEndpointIDs), gc.Equals, 1)
			c.Assert(len(stdCaches.apiEndpointIDs[apiid]), gc.Equals, len(endpointids))
			c.Assert(len(stdCaches.apiHostnames), gc.Equals, 1)
			c.Assert(len(stdCaches.apiHostnames[apiid]), gc.Equals, len(hostnames))
		}
	}
}
